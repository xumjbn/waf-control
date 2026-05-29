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
	taskRepo := NewUpgradeTaskRepository(pool)
	if err := taskRepo.EnsureSchema(context.Background()); err != nil {
		slog.Warn("upgrade_tasks ensure schema failed", "error", err)
	}
	h := NewHandler(repo)
	taskH := NewUpgradeTaskHandler(taskRepo, repo)

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
		r.Post("/{id}/apply", h.ApplyUpgrade) // 直接标记完成（无任务流）
		r.Post("/{id}/start", taskH.Start)    // 真升级流程（建 task 并步进）
		r.Delete("/{id}", h.DeleteUpgrade)
	})

	r.Get("/system/upgrade-tasks/{tid}", taskH.Get) // 前端轮询拉进度
}
