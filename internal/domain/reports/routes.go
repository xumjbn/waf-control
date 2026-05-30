package reports

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		slog.Warn("reports ensure schema failed", "error", err)
	}
	h := NewHandler(repo)
	if err := h.gen.EnsureSchema(context.Background()); err != nil {
		slog.Warn("report_outputs ensure schema failed", "error", err)
	}

	r.Route("/reports", func(r chi.Router) {
		// 统一列表 + 执行 + 下载（NW · 08 报表中心首页消费）
		r.Get("/all", h.ListAll)
		r.Post("/{type}/{id}/run", h.RunReport)
		r.Get("/{type}/{id}/download", h.DownloadReport)

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
			r.Post("/{id}/enabled", h.SetTimingEnabled) // 启用/停用（调度器据此跑）
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
