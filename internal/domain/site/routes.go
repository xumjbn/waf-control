package site

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/agent"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	registerWith(r, pool, nil)
}

func RegisterRoutesWithAgent(r chi.Router, pool *pgxpool.Pool, agentSvc *agent.Service) {
	registerWith(r, pool, agentSvc)
}

func registerWith(r chi.Router, pool *pgxpool.Pool, agentSvc *agent.Service) {
	repo := NewRepository(pool)
	// 自愈 schema：000010 的 ALTER 在启动时再跑一次，部署窗口期 race
	// 时让 SELECT rps/blocked_rate/... 不会因为缺列直接 500。
	if pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := repo.EnsureSchema(ctx); err != nil {
			slog.Warn("site ensure schema failed", "err", err)
		}
	}
	h := NewHandlerWithAgent(repo, agentSvc)

	// 站点级防护模块（site_modules）
	var modH *ModuleHandler
	if pool != nil {
		modStore := NewModuleStore(pool)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := modStore.EnsureSchema(ctx); err != nil {
			slog.Warn("site_modules ensure schema failed", "err", err)
		}
		modH = NewModuleHandler(modStore)
	}

	r.Route("/sites", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Put("/{id}/metrics", h.UpdateMetrics)
		r.Get("/{id}/devices", h.ListDevices)
		r.Post("/{id}/devices", h.BindDevice)
		r.Delete("/{id}/devices/{deviceId}", h.UnbindDevice)
		if modH != nil {
			r.Get("/{id}/modules", modH.List)
			r.Put("/{id}/modules/{module}", modH.Put)
		}
	})
}
