package instancemgmt

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/agent"
)

// RegisterRoutes — backwards-compatible signature (used by older call
// sites in server.go). Mounts agent-driven instance routes only.
func RegisterRoutes(r chi.Router, agentSvc *agent.Service) {
	h := NewHandler(agentSvc)
	r.Get("/instances", h.ListInstances)
	r.Get("/instances/detail", h.GetInstance)
	r.Post("/instances/restart", h.RestartInstance)
	r.Post("/instances/register-intent", h.RegisterNodeIntent)
}

// RegisterRoutesWithDB additionally mounts /clusters CRUD + /ha-groups CRUD
// + /instances/{nodeId}/config (needs Postgres). Server bootstrap should
// call this when pool is available.
func RegisterRoutesWithDB(r chi.Router, agentSvc *agent.Service, pool *pgxpool.Pool) {
	RegisterRoutes(r, agentSvc)
	if pool == nil {
		return
	}

	// 实例管理员配置（instance_configs）—— 与 NodeState 运行时观测值解耦
	cfgStore := NewConfigStore(pool)
	if err := cfgStore.EnsureSchema(context.Background()); err != nil {
		slog.Warn("instance_configs ensure schema failed", "err", err)
	}
	cfgH := NewConfigHandler(cfgStore)
	r.Get("/instances/{nodeId}/config", cfgH.Get)
	r.Put("/instances/{nodeId}/config", cfgH.Put)

	clusterStore := NewClusterStore(pool)
	if err := clusterStore.EnsureSchema(context.Background()); err != nil {
		slog.Warn("clusters ensure schema failed", "err", err)
	}
	ch := NewClusterHandler(clusterStore)
	r.Route("/clusters", func(r chi.Router) {
		r.Get("/", ch.List)
		r.Post("/", ch.Create)
		r.Get("/{id}", ch.Get)
		r.Put("/{id}", ch.Update)
		r.Delete("/{id}", ch.Delete)
		r.Put("/{id}/nodes/{nodeId}", ch.AssignNode)
		r.Delete("/{id}/nodes/{nodeId}", ch.RemoveNode)
	})

	haStore := NewHAStore(pool)
	if err := haStore.EnsureSchema(context.Background()); err != nil {
		// 启动期失败时只警告：migrations/000017 跑过的话表已经在；
		// 没跑也别让整个 server 崩掉 —— /ha-groups 后续会自然 500。
		slog.Warn("ha_groups ensure schema failed", "err", err)
	}
	hh := NewHAHandler(haStore)
	r.Route("/ha-groups", func(r chi.Router) {
		r.Get("/", hh.List)
		r.Post("/", hh.Create)
		r.Get("/{id}", hh.Get)
		r.Put("/{id}", hh.Update)
		r.Delete("/{id}", hh.Delete)
		r.Post("/{id}/switchover", hh.Switchover)
	})
}
