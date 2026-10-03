// Package httpapi is the JSON HTTP API.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sorin-work/estates/api/internal/auth"
	"github.com/sorin-work/estates/api/internal/billing"
	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

// Store is everything the handlers need from the data layer. Each feature
// declares its own slice of it next to its handlers.
type Store interface {
	Ping(ctx context.Context) error
	userStore
	propertyStore
	unitStore
	tenantStore
	leaseStore
	paymentStore
	maintenanceStore
	expenseStore
	depositStore
	vendorStore
	recurringStore
	renewalStore
	reportStore
	activityStore
	documentStore
	searchStore
	dashboardStore
}

// Billing generates charges and works out lease balances.
type Billing interface {
	GenerateRent(ctx context.Context, l store.Lease, asOf domain.Date) (int, error)
	BillOwner(ctx context.Context, ownerID string, asOf domain.Date) (billing.Result, error)
	Balances(ctx context.Context, ownerID string, asOf domain.Date) ([]billing.Balance, error)
	GenerateRecurring(ctx context.Context, r store.RecurringExpense, asOf domain.Date) (int, error)
}

type Server struct {
	store       Store
	billing     Billing
	tokens      *auth.Tokens
	log         *slog.Logger
	corsOrigins []string
	// today returns the current date; tests replace it.
	today func() domain.Date
}

type Options struct {
	Store       Store
	Billing     Billing
	Tokens      *auth.Tokens
	Logger      *slog.Logger
	CORSOrigins []string
	Today       func() domain.Date
}

func New(o Options) *Server {
	s := &Server{
		store:       o.Store,
		billing:     o.Billing,
		tokens:      o.Tokens,
		log:         o.Logger,
		corsOrigins: o.CORSOrigins,
		today:       o.Today,
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	if s.today == nil {
		s.today = domain.Today
	}
	return s
}

// Handler returns the router with all routes and middleware.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(s.logRequests)
	r.Use(s.recoverPanics)
	r.Use(s.cors)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	r.Get("/healthz", s.health)

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", s.handle(s.register))
		r.Post("/auth/login", s.handle(s.login))

		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/auth/me", s.handle(s.me))
			r.Put("/auth/me", s.handle(s.updateMe))
			r.Put("/auth/password", s.handle(s.changePassword))
			r.Route("/properties", s.propertyRoutes)
			r.Route("/units", s.unitRoutes)
			r.Route("/tenants", s.tenantRoutes)
			r.Route("/leases", s.leaseRoutes)
			r.Route("/charges", s.chargeRoutes)
			r.Route("/payments", s.paymentRoutes)
			r.Route("/rent", s.rentRoutes)
			r.Route("/maintenance", s.maintenanceRoutes)
			r.Route("/expenses", s.expenseRoutes)
			r.Route("/deposit-transactions", s.depositRoutes)
			r.Route("/vendors", s.vendorRoutes)
			r.Route("/recurring-expenses", s.recurringRoutes)
			r.Route("/rent-changes", s.rentChangeRoutes)
			r.Route("/reports", s.reportRoutes)
			r.Route("/documents", s.documentRoutes)
			r.Get("/dashboard", s.handle(s.dashboard))
			r.Get("/activity", s.handle(s.listActivity))
			r.Get("/reminders", s.handle(s.reminders))
			r.Get("/search", s.handle(s.search))
		})
	})
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		s.log.Error("health check failed", "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
