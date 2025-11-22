// Package httpapi is the JSON HTTP API.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sorin-work/estates/api/internal/auth"
	"github.com/sorin-work/estates/api/internal/domain"
)

// Store is everything the handlers need from the data layer. Each feature
// declares its own slice of it next to its handlers.
type Store interface {
	Ping(ctx context.Context) error
	userStore
}

type Server struct {
	store  Store
	tokens *auth.Tokens
	log    *slog.Logger
	// today returns the current date; tests replace it.
	today func() domain.Date
}

type Options struct {
	Store  Store
	Tokens *auth.Tokens
	Logger *slog.Logger
	Today  func() domain.Date
}

func New(o Options) *Server {
	s := &Server{
		store:  o.Store,
		tokens: o.Tokens,
		log:    o.Logger,
		today:  o.Today,
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
	r.Use(middleware.RealIP)
	r.Use(s.logRequests)
	r.Use(s.recoverPanics)

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
