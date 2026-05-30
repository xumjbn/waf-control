package reports

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// UnifiedItem 是前端 PageReport 报表中心列表的单行（custom/combined/timing/manual 合一）。
type UnifiedItem struct {
	ID          int64      `json:"id"`
	Type        string     `json:"type"` // custom / combined / timing / manual
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Schedule    string     `json:"schedule"` // cron / 间隔 / 一次性
	IsEnabled   bool       `json:"is_enabled"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	NextRunAt   *time.Time `json:"next_run_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// EnsureSchema 启动时幂等补齐报表执行追踪列（migration 000014）。
func (r *Repository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE report_custom   ADD COLUMN IF NOT EXISTS last_run_at  TIMESTAMPTZ`,
		`ALTER TABLE report_custom   ADD COLUMN IF NOT EXISTS next_run_at  TIMESTAMPTZ`,
		`ALTER TABLE report_custom   ADD COLUMN IF NOT EXISTS is_enabled   BOOLEAN NOT NULL DEFAULT TRUE`,
		`ALTER TABLE report_combined ADD COLUMN IF NOT EXISTS last_run_at  TIMESTAMPTZ`,
		`ALTER TABLE report_combined ADD COLUMN IF NOT EXISTS is_enabled   BOOLEAN NOT NULL DEFAULT TRUE`,
		`ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS cron         VARCHAR(64) NOT NULL DEFAULT '0 0 * * *'`,
		`ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS last_run_at  TIMESTAMPTZ`,
		`ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS next_run_at  TIMESTAMPTZ`,
		`ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS is_enabled   BOOLEAN NOT NULL DEFAULT TRUE`,
		`ALTER TABLE report_manual   ADD COLUMN IF NOT EXISTS file_path    VARCHAR(512) NOT NULL DEFAULT ''`,
		`ALTER TABLE report_manual   ADD COLUMN IF NOT EXISTS file_size    BIGINT       NOT NULL DEFAULT 0`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure reports schema (%q): %w", s, err)
		}
	}
	return nil
}

// ListAll 聚合四类报表，按 created_at DESC 排序。
func (r *Repository) ListAll(ctx context.Context) ([]UnifiedItem, error) {
	q := `
		SELECT id, 'custom'   AS type, name, COALESCE(description,'') AS description,
		       COALESCE(schedule,'') AS schedule, COALESCE(is_enabled,true) AS is_enabled,
		       last_run_at, next_run_at, created_at
		  FROM report_custom
		UNION ALL
		SELECT id, 'combined' AS type, name, COALESCE(description,'') AS description,
		       '' AS schedule, COALESCE(is_enabled,true) AS is_enabled,
		       last_run_at, NULL::timestamptz AS next_run_at, created_at
		  FROM report_combined
		UNION ALL
		SELECT id, 'timing'   AS type, name, COALESCE(description,'') AS description,
		       COALESCE(cron,'0 0 * * *') AS schedule, COALESCE(is_enabled,true) AS is_enabled,
		       last_run_at, next_run_at, created_at
		  FROM report_timing
		UNION ALL
		SELECT id, 'manual'   AS type, name, COALESCE(description,'') AS description,
		       '' AS schedule, true AS is_enabled,
		       created_at AS last_run_at, NULL::timestamptz AS next_run_at, created_at
		  FROM report_manual
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list unified reports: %w", err)
	}
	defer rows.Close()
	out := make([]UnifiedItem, 0)
	for rows.Next() {
		var it UnifiedItem
		if err := rows.Scan(&it.ID, &it.Type, &it.Name, &it.Description, &it.Schedule,
			&it.IsEnabled, &it.LastRunAt, &it.NextRunAt, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan unified report: %w", err)
		}
		out = append(out, it)
	}
	return out, nil
}

// MarkRun 把某类报表的 last_run_at 置为 NOW（占位实现，真实计算 next_run_at 由 cron 调度器维护）。
func (r *Repository) MarkRun(ctx context.Context, kind string, id int64) error {
	var table string
	switch kind {
	case "custom":
		table = "report_custom"
	case "combined":
		table = "report_combined"
	case "timing":
		table = "report_timing"
	case "manual":
		table = "report_manual"
	default:
		return fmt.Errorf("invalid report type: %s", kind)
	}
	tag, err := r.pool.Exec(ctx, fmt.Sprintf(`UPDATE %s SET last_run_at = NOW() WHERE id = $1`, table), id)
	if err != nil {
		return fmt.Errorf("mark run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s/%d not found", kind, id)
	}
	return nil
}

// --- handler ---

// ListAll  GET /reports/all
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListAll(r.Context())
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

// RunReport POST /reports/{type}/{id}/run
// 真生成：把窗口内攻击日志聚合成 CSV 产物落 report_outputs，并 MarkRun 记录时间。
func (h *Handler) RunReport(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "type")
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	out, gerr := h.gen.Generate(r.Context(), kind, id)
	if gerr != nil {
		writeJSONErr(w, http.StatusInternalServerError, gerr)
		return
	}
	_ = h.repo.MarkRun(r.Context(), kind, id) // 时间戳记录失败不阻断生成结果
	writeJSON(w, http.StatusAccepted, map[string]any{
		"type":      kind,
		"id":        id,
		"status":    "generated",
		"filename":  out.Filename,
		"row_count": out.RowCount,
		"queued_at": time.Now(),
	})
}

// DownloadReport GET /reports/{type}/{id}/download
// 服务最近一次生成的 CSV 产物；无产物则即时生成一份再返回。
func (h *Handler) DownloadReport(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "type")
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	out, _ := h.gen.LatestOutput(r.Context(), kind, id)
	if out == nil {
		generated, gerr := h.gen.Generate(r.Context(), kind, id)
		if gerr != nil {
			writeJSONErr(w, http.StatusInternalServerError, gerr)
			return
		}
		out = generated
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, out.Filename))
	_, _ = w.Write([]byte(out.Content))
}

func writeJSONErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
