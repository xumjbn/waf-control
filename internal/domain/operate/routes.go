package operate

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/operation-logs", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
	})
}
