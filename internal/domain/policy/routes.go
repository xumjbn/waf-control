package policy

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// modsecRulesDir 决定从哪里读取 modsec .conf 规则做 builtin 种子。
// 优先级：env WAF_MODSEC_RULES_DIR → /etc/waf/modsec-rules（容器） →
// ../deploy/modsec/rules.d（dev：相对 cmd/server/） → ./deploy/modsec/rules.d（dev：项目根）。
func modsecRulesDir() string {
	if v := os.Getenv("WAF_MODSEC_RULES_DIR"); v != "" {
		return v
	}
	for _, p := range []string{
		"/etc/waf/modsec-rules",
		"../../deploy/modsec/rules.d",
		"../deploy/modsec/rules.d",
		"./deploy/modsec/rules.d",
	} {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	if err := repo.EnsureSchema(context.Background()); err != nil {
		slog.Warn("policy ensure schema failed", "error", err)
	}

	// 启动期把 deploy/modsec/rules.d 同步成 builtin policies。
	// dir 找不到也不致命（用户可以纯手工新建规则），只 warn。
	if dir := modsecRulesDir(); dir != "" {
		ins, upd, total, err := repo.SyncFromDir(context.Background(), dir)
		if err != nil {
			slog.Warn("modsec rules sync failed", "dir", dir, "err", err)
		} else {
			slog.Info("modsec rules synced",
				"dir", dir, "inserted", ins, "updated", upd, "total", total)
		}
	} else {
		slog.Warn("no modsec rules dir found; builtin policies will be empty until POST /policies/sync-builtin",
			"env", "WAF_MODSEC_RULES_DIR")
	}

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
		r.Post("/sync-builtin", func(w http.ResponseWriter, req *http.Request) {
			dir := modsecRulesDir()
			if dir == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error": "modsec rules dir not configured (env WAF_MODSEC_RULES_DIR)",
				})
				return
			}
			ins, upd, total, err := repo.SyncFromDir(req.Context(), dir)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"dir":      dir,
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
