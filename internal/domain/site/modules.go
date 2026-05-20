package site

// modules.go —— 站点级别防护模块配置（site_modules 表）
//
// 模型：site N:M module。每条 binding 有 enabled + level。
// agent 拉配置时，会把『站点 → 启用的模块 → 模块对应的规则集合』展开成
// 实际下发给 ModSecurity 的 .conf 文件。
//
// 默认 8 个模块（与 deploy/modsec/rules.d/<category>/ 目录一一对应）：
//   sqli / xss / rce / lfi-rfi / bot / rate-limit / ip-reputation / virtual-patches
//
// 端点：
//   GET  /api/v1/sites/{id}/modules                    返回该站点全部模块配置（缺省项以 enabled=true,level='medium' 补齐）
//   PUT  /api/v1/sites/{id}/modules/{module}           更新单个模块的 enabled / level

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 已知模块列表 —— 与 deploy/modsec/rules.d/<category>/ 对齐
var KnownModules = []string{
	"sqli",
	"xss",
	"rce",
	"lfi-rfi",
	"bot",
	"rate-limit",
	"ip-reputation",
	"virtual-patches",
}

func isKnownModule(m string) bool {
	for _, x := range KnownModules {
		if x == m {
			return true
		}
	}
	return false
}

type SiteModule struct {
	SiteID    int64  `json:"site_id"`
	Module    string `json:"module"`
	Enabled   bool   `json:"enabled"`
	Level     string `json:"level"`     // low / medium / high
	UpdatedAt string `json:"updated_at,omitempty"`
}

type ModuleStore struct {
	pool *pgxpool.Pool
}

func NewModuleStore(pool *pgxpool.Pool) *ModuleStore {
	return &ModuleStore{pool: pool}
}

// EnsureSchema 启动期幂等建 site_modules 表。
func (s *ModuleStore) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS site_modules (
			site_id    BIGINT      NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
			module     VARCHAR(32) NOT NULL,
			enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
			level      VARCHAR(8)  NOT NULL DEFAULT 'medium',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (site_id, module)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_site_modules_site ON site_modules(site_id)`,
	}
	for _, q := range stmts {
		if _, err := s.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("site_modules ensure schema: %w", err)
		}
	}
	return nil
}

// GetForSite 返回该站点的全部模块配置；表中没有的项以默认值（enabled=true, level=medium）补齐。
func (s *ModuleStore) GetForSite(ctx context.Context, siteID int64) ([]SiteModule, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT site_id, module, enabled, level, updated_at
		  FROM site_modules WHERE site_id=$1`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	have := map[string]SiteModule{}
	for rows.Next() {
		var m SiteModule
		var updated interface{}
		if err := rows.Scan(&m.SiteID, &m.Module, &m.Enabled, &m.Level, &updated); err != nil {
			return nil, err
		}
		if t, ok := updated.(interface{ Format(string) string }); ok {
			m.UpdatedAt = t.Format("2006-01-02T15:04:05Z07:00")
		}
		have[m.Module] = m
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]SiteModule, 0, len(KnownModules))
	for _, name := range KnownModules {
		if existing, ok := have[name]; ok {
			out = append(out, existing)
			continue
		}
		out = append(out, SiteModule{
			SiteID:  siteID,
			Module:  name,
			Enabled: true,
			Level:   "medium",
		})
	}
	return out, nil
}

// Upsert 写入/更新单个模块配置。
func (s *ModuleStore) Upsert(ctx context.Context, m SiteModule) error {
	if !isKnownModule(m.Module) {
		return fmt.Errorf("unknown module %q (known: %v)", m.Module, KnownModules)
	}
	switch m.Level {
	case "low", "medium", "high":
	default:
		return fmt.Errorf("invalid level %q (want low/medium/high)", m.Level)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO site_modules (site_id, module, enabled, level, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (site_id, module) DO UPDATE
		   SET enabled = EXCLUDED.enabled,
		       level   = EXCLUDED.level,
		       updated_at = NOW()`,
		m.SiteID, m.Module, m.Enabled, m.Level)
	return err
}

// ModuleHandler hosts /sites/{id}/modules routes.
type ModuleHandler struct {
	store *ModuleStore
}

func NewModuleHandler(store *ModuleStore) *ModuleHandler {
	return &ModuleHandler{store: store}
}

func (h *ModuleHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}
	list, err := h.store.GetForSite(r.Context(), siteID)
	if err != nil {
		slog.Error("list site modules", "site_id", siteID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (h *ModuleHandler) Put(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}
	module := chi.URLParam(r, "module")
	if !isKnownModule(module) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "unknown module",
			"allowed": fmt.Sprintf("%v", KnownModules),
		})
		return
	}
	var body struct {
		Enabled *bool   `json:"enabled"`
		Level   *string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	// 从现有状态出发 patch
	current, err := h.store.GetForSite(r.Context(), siteID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	patch := SiteModule{SiteID: siteID, Module: module, Enabled: true, Level: "medium"}
	for _, c := range current {
		if c.Module == module {
			patch = c
			break
		}
	}
	if body.Enabled != nil {
		patch.Enabled = *body.Enabled
	}
	if body.Level != nil {
		patch.Level = *body.Level
	}
	if err := h.store.Upsert(r.Context(), patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, patch)
}
