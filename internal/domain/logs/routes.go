package logs

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/logs", func(r chi.Router) {
		r.Route("/attack", func(r chi.Router) {
			r.Get("/", h.ListAttackLogs)
			r.Get("/{id}", h.GetAttackLog)
			r.Get("/count", h.CountAttackLogs)
			r.Delete("/", h.ClearAttackLogs)
		})
		r.Route("/antivirus", func(r chi.Router) {
			r.Get("/", h.ListAntivirusLogs)
			r.Get("/{id}", h.GetAntivirusLog)
			r.Get("/count", h.CountAntivirusLogs)
			r.Delete("/", h.ClearAntivirusLogs)
		})
		r.Route("/antitamper", func(r chi.Router) {
			r.Get("/", h.ListAntitamperLogs)
			r.Get("/{id}", h.GetAntitamperLog)
			r.Get("/count", h.CountAntitamperLogs)
			r.Delete("/", h.ClearAntitamperLogs)
		})
	})
}
