package alert

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		slog.Warn("alert ensure schema failed", "error", err)
	}
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
			r.Get("/stats/hourly", h.StatsHourly) // 近 24h 按小时分布（监控大屏）
		})

		r.Route("/channels", func(r chi.Router) {
			r.Get("/", h.ListChannels)
			r.Post("/", h.CreateChannel)
			r.Put("/{id}", h.UpdateChannel)
			r.Delete("/{id}", h.DeleteChannel)
			r.Post("/{id}/test", h.TestChannel)
		})
	})
}
