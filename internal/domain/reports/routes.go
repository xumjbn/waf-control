package reports

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/reports", func(r chi.Router) {
		// Custom reports
		r.Route("/custom", func(r chi.Router) {
			r.Get("/", h.ListCustom)
			r.Post("/", h.CreateCustom)
			r.Get("/{id}", h.GetCustom)
			r.Put("/{id}", h.UpdateCustom)
			r.Delete("/{id}", h.DeleteCustom)
			r.Get("/{id}/data", h.CustomReportData)
		})

		// Combined reports
		r.Route("/combined", func(r chi.Router) {
			r.Get("/", h.ListCombined)
			r.Post("/", h.CreateCombined)
			r.Get("/{id}", h.GetCombined)
			r.Put("/{id}", h.UpdateCombined)
			r.Delete("/{id}", h.DeleteCombined)
			r.Get("/{id}/data", h.CombinedReportData)
		})

		// Timing reports
		r.Route("/timing", func(r chi.Router) {
			r.Get("/", h.ListTiming)
			r.Post("/", h.CreateTiming)
			r.Get("/{id}", h.GetTiming)
			r.Put("/{id}", h.UpdateTiming)
			r.Delete("/{id}", h.DeleteTiming)
			r.Get("/{id}/data", h.TimingReportData)
		})

		// Manual reports
		r.Route("/manual", func(r chi.Router) {
			r.Get("/", h.ListManual)
			r.Post("/", h.CreateManual)
			r.Get("/{id}", h.GetManual)
			r.Delete("/{id}", h.DeleteManual)
		})
	})
}
