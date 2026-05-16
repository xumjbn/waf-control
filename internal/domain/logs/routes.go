package logs

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/logs", func(r chi.Router) {
		r.Get("/attack", h.ListAttackLogs)
		r.Get("/antivirus", h.ListAntivirusLogs)
		r.Get("/antitamper", h.ListAntitamperLogs)
	})
}
