package ha

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/ha", func(r chi.Router) {
		r.Get("/config", h.GetConfig)
		r.Put("/config", h.UpsertConfig)
	})
}
