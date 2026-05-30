package logs

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/domain/acl"
)

// RegisterRoutes 保留兼容旧调用方。新调用方建议使用 RegisterRoutesWithACL。
func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	registerWith(r, pool, nil)
}

// RegisterRoutesWithACL 在挂载日志路由时同时启用 ban / whitelist 这两个 acl 联动端点。
func RegisterRoutesWithACL(r chi.Router, pool *pgxpool.Pool, aclRepo *acl.Repository) {
	registerWith(r, pool, aclRepo)
}

func registerWith(r chi.Router, pool *pgxpool.Pool, aclRepo *acl.Repository) {
	repo := NewRepository(pool)

	// 启动时自愈攻击日志表的 UI 富字段。
	if err := repo.EnsureSchema(context.Background()); err != nil {
		slog.Warn("logs ensure schema failed", "error", err)
	}

	var h *Handler
	if aclRepo != nil {
		h = NewHandlerWithACL(repo, aclRepo)
	} else {
		h = NewHandler(repo)
	}

	r.Route("/logs", func(r chi.Router) {
		r.Route("/attack", func(r chi.Router) {
			r.Get("/", h.ListAttackLogs)
			r.Post("/", h.IngestAttackLog) // 内部 / agent 上报
			r.Get("/count", h.CountAttackLogs)
			r.Get("/trend", h.AttackTrend) // 按小时分桶（可选 site/rule_id）
			r.Delete("/", h.ClearAttackLogs)
			r.Get("/{id}", h.GetAttackLog)
			r.Get("/{id}/related", h.RelatedEvents)
			r.Post("/{id}/ban", h.BanIP)
			r.Post("/{id}/whitelist", h.WhitelistIP)
		})
		r.Route("/antivirus", func(r chi.Router) {
			r.Get("/", h.ListAntivirusLogs)
			r.Get("/{id}", h.GetAntivirusLog)
			r.Get("/count", h.CountAntivirusLogs)
			r.Delete("/", h.ClearAntivirusLogs)
		})
		r.Route("/antitamper", func(r chi.Router) {
			r.Get("/", h.ListAntitamperLogs)
			r.Get("/{id}", h.GetAntitamperLog)
			r.Get("/count", h.CountAntitamperLogs)
			r.Delete("/", h.ClearAntitamperLogs)
		})
	})
}
