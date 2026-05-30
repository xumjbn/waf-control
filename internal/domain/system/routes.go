package system

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes 注册系统管理路由。fleet 可空 —— 给定时升级流程在 rollout
// 阶段下发真实 sync_rules 到在线节点的能力（由 agent.Service 实现）。
func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	registerWith(r, pool, nil)
}

// RegisterRoutesWithFleet 在注册系统路由的同时注入集群命令下发能力。
func RegisterRoutesWithFleet(r chi.Router, pool *pgxpool.Pool, fleet FleetCommander) {
	registerWith(r, pool, fleet)
}

func registerWith(r chi.Router, pool *pgxpool.Pool, fleet FleetCommander) {
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
	if fleet != nil {
		taskH.SetFleet(fleet)
	}

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
