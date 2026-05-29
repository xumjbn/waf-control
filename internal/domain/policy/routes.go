package policy

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		slog.Warn("policy ensure schema failed", "error", err)
	}

	// 启动期自动 sync 一次（数据源自动选择：env / disk / embed）。
	// embed 永远可用，所以这里不再可能因为找不到规则目录而失败。
	ins, upd, total, source, err := repo.SyncFromFS(context.Background())
	if err != nil {
		slog.Warn("modsec rules sync failed", "source", source, "err", err)
	} else {
		slog.Info("modsec rules synced",
			"source", source, "inserted", ins, "updated", upd, "total", total)
	}

	// Bot 挑战模式 + API 端点（migration 000022）—— 每个站点独立配置
	botStore := NewBotStore(pool)
	if err := botStore.EnsureSchema(context.Background()); err != nil {
		slog.Warn("bot_challenges ensure schema failed", "err", err)
	}
	apiStore := NewAPIStore(pool)
	if err := apiStore.EnsureSchema(context.Background()); err != nil {
		slog.Warn("api_endpoints ensure schema failed", "err", err)
	}
	botH := NewBotHandler(botStore)
	apiH := NewAPIHandler(apiStore)
	r.Route("/policy/sites/{siteId}", func(r chi.Router) {
		r.Get("/bot-challenges", botH.List)
		r.Put("/bot-challenges/{challenge}", botH.Put)
		r.Get("/api-endpoints", apiH.List)
		r.Post("/api-endpoints", apiH.Create)
		r.Get("/api-kpi", apiH.KPI)
	})
	r.Put("/policy/api-endpoints/{id}", apiH.Update)
	r.Delete("/policy/api-endpoints/{id}", apiH.Delete)

	h := NewHandler(repo)

	r.Route("/policy-categories", func(r chi.Router) {
		r.Get("/", h.ListCategories)
		r.Post("/", h.CreateCategory)
		r.Put("/{id}", h.UpdateCategory)
		r.Delete("/{id}", h.DeleteCategory)
	})

	r.Route("/policies", func(r chi.Router) {
		r.Get("/", h.ListPolicies)
		r.Post("/", h.CreatePolicy)
		// 试运行（无 id 也能调，编辑中尚未保存的规则也可试）
		r.Post("/dry-run", h.DryRun)
		r.Post("/sync-builtin", func(w http.ResponseWriter, req *http.Request) {
			ins, upd, total, source, err := repo.SyncFromFS(req.Context())
			if err != nil {
				writeJSON(w, http.StatusInternalServerError,
					map[string]string{"error": err.Error(), "source": source})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"source":   source,
				"inserted": ins,
				"updated":  upd,
				"total":    total,
			})
		})
		r.Get("/{id}", h.GetPolicy)
		r.Put("/{id}", h.UpdatePolicy)
		r.Delete("/{id}", h.DeletePolicy)
		r.Post("/{id}/hit", h.IncrementHits)
		r.Get("/{id}/rules", h.ListRules)
		r.Post("/{id}/rules", h.CreateRule)
		r.Delete("/{id}/rules/{ruleId}", h.DeleteRule)
		r.Get("/{id}/history", h.ListHistory)
	})
}
