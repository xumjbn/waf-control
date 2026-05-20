package system

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	if err := repo.EnsureUpgradeSchema(context.Background()); err != nil {
		slog.Warn("system_upgrades ensure schema failed", "error", err)
	}
	h := NewHandler(repo)

	r.Route("/system/settings", func(r chi.Router) {
		r.Get("/", h.ListSettings)
		r.Put("/", h.UpsertSetting)
		r.Delete("/{key}", h.DeleteSetting)
	})

	r.Route("/system/licenses", func(r chi.Router) {
		r.Get("/", h.ListLicenses)
		r.Post("/", h.CreateLicense)
		r.Post("/{id}/activate", h.ActivateLicense)
		r.Delete("/{id}", h.DeleteLicense)
	})

	r.Route("/system/upgrades", func(r chi.Router) {
		r.Get("/", h.ListUpgrades)
		r.Get("/check", h.CheckUpgrade) // 当前 + 最新可用
		r.Post("/", h.CreateUpgrade)
		r.Post("/{id}/trigger", h.TriggerUpgrade)
		r.Post("/{id}/apply", h.ApplyUpgrade) // 标记安装完成
		r.Delete("/{id}", h.DeleteUpgrade)
	})
}
