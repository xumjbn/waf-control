package alert

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/alert", func(r chi.Router) {
		r.Route("/policies", func(r chi.Router) {
			r.Get("/", h.ListPolicies)
			r.Post("/", h.CreatePolicy)
			r.Put("/{id}", h.UpdatePolicy)
			r.Delete("/{id}", h.DeletePolicy)
		})

		r.Route("/events", func(r chi.Router) {
			r.Get("/", h.ListEvents)
			r.Post("/", h.CreateEvent)
			r.Put("/{id}/status", h.UpdateEventStatus)
			r.Post("/mark-all-read", h.MarkAllRead)
			r.Get("/stats", h.Stats)
		})

		r.Route("/channels", func(r chi.Router) {
			r.Get("/", h.ListChannels)
			r.Put("/{id}", h.UpdateChannel)
		})
	})
}
