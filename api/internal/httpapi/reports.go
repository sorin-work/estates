package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sorin-work/estates/api/internal/domain"
	"github.com/sorin-work/estates/api/internal/store"
)

type reportStore interface {
	PropertyResults(ctx context.Context, ownerID string, from, to domain.Date) ([]store.PropertyResult, error)
}

func (s *Server) reportRoutes(r chi.Router) {
	r.Get("/profit-loss", s.handle(s.profitLoss))
	r.Get("/profit-loss.csv", s.handle(s.profitLossCSV))
}

type propertyResultView struct {
	store.PropertyResult
	NetCents int64 `json:"net_cents"`
}

type profitLossResponse struct {
	From       domain.Date          `json:"from"`
	To         domain.Date          `json:"to"`
	Properties []propertyResultView `json:"properties"`
	Totals     propertyResultView   `json:"totals"`
}

// reportPeriod reads ?from= and ?to=, defaulting to the year to date.
func (s *Server) reportPeriod(r *http.Request) (domain.Date, domain.Date, error) {
	from, err := queryDate(r, "from")
	if err != nil {
		return from, from, err
	}
	to, err := queryDate(r, "to")
	if err != nil {
		return from, to, err
	}
	today := s.today()
	if to.IsZero() {
		to = today
	}
	if from.IsZero() {
		from = domain.NewDate(to.Year(), 1, 1)
	}
	if to.Before(from) {
		return from, to, errBadRequest("to must be on or after from")
	}
	if from.AddMonths(60).Before(to) {
		return from, to, errBadRequest("reports cover at most five years")
	}
	return from, to, nil
}

func (s *Server) buildProfitLoss(r *http.Request) (profitLossResponse, error) {
	from, to, err := s.reportPeriod(r)
	if err != nil {
		return profitLossResponse{}, err
	}
	results, err := s.store.PropertyResults(r.Context(), userID(r), from, to)
	if err != nil {
		return profitLossResponse{}, err
	}
	resp := profitLossResponse{From: from, To: to, Properties: make([]propertyResultView, len(results))}
	totals := map[string]int64{}
	for i, p := range results {
		resp.Properties[i] = propertyResultView{PropertyResult: p, NetCents: p.NetCents()}
		resp.Totals.ChargedCents += p.ChargedCents
		resp.Totals.CollectedCents += p.CollectedCents
		resp.Totals.WithheldCents += p.WithheldCents
		resp.Totals.ExpensesCents += p.ExpensesCents
		for _, c := range p.ByCategory {
			totals[c.Category] += c.AmountCents
		}
	}
	resp.Totals.PropertyName = "All properties"
	resp.Totals.NetCents = resp.Totals.PropertyResult.NetCents()
	resp.Totals.ByCategory = []store.CategoryTotal{}
	for _, c := range domain.ExpenseCategories {
		if totals[c] > 0 {
			resp.Totals.ByCategory = append(resp.Totals.ByCategory, store.CategoryTotal{Category: c, AmountCents: totals[c]})
		}
	}
	return resp, nil
}

// profitLoss is income against expenses per property: what each one
// actually earned over the period.
func (s *Server) profitLoss(w http.ResponseWriter, r *http.Request) error {
	resp, err := s.buildProfitLoss(r)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) profitLossCSV(w http.ResponseWriter, r *http.Request) error {
	resp, err := s.buildProfitLoss(r)
	if err != nil {
		return err
	}
	header := []string{"Property", "Rent and fees charged", "Collected", "Deposits withheld", "Expenses"}
	for _, c := range domain.ExpenseCategories {
		header = append(header, "Expenses: "+c)
	}
	header = append(header, "Net")

	row := func(p propertyResultView) []string {
		byCat := map[string]int64{}
		for _, c := range p.ByCategory {
			byCat[c.Category] = c.AmountCents
		}
		out := []string{p.PropertyName, csvMoney(p.ChargedCents), csvMoney(p.CollectedCents),
			csvMoney(p.WithheldCents), csvMoney(p.ExpensesCents)}
		for _, c := range domain.ExpenseCategories {
			out = append(out, csvMoney(byCat[c]))
		}
		return append(out, csvMoney(p.NetCents))
	}
	rows := make([][]string, 0, len(resp.Properties)+1)
	for _, p := range resp.Properties {
		rows = append(rows, row(p))
	}
	rows = append(rows, row(resp.Totals))
	return writeCSV(w, "profit-loss-"+resp.From.String()+"-to-"+resp.To.String()+".csv", header, rows)
}
