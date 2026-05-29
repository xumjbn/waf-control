package policy

// bot_api.go —— Bot 挑战模式 + API 端点 schema 检查 两组后端
//
// 与 UI policy 页 Bot Tab + API Tab 一一对应（migration 000022）。
//
// 端点：
//   GET    /api/v1/policy/sites/{siteId}/bot-challenges       该站点 5 种挑战模式
//   PUT    /api/v1/policy/sites/{siteId}/bot-challenges/{ch}  开关 / config 单条
//
//   GET    /api/v1/policy/sites/{siteId}/api-endpoints        该站点已注册 API 端点
//   POST   /api/v1/policy/sites/{siteId}/api-endpoints        登记端点（手工或 Swagger 导入）
//   PUT    /api/v1/policy/api-endpoints/{id}
//   DELETE /api/v1/policy/api-endpoints/{id}
//   GET    /api/v1/policy/sites/{siteId}/api-kpi              4 个 KPI 派生

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/pkg/httputil"
)

// ---------- Bot 挑战模式 ----------

var KnownChallenges = []string{"js", "tls", "dev", "slider", "behave"}

type BotChallenge struct {
	SiteID    int64                  `json:"site_id"`
	Challenge string                 `json:"challenge"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]any         `json:"config"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type BotStore struct {
	pool *pgxpool.Pool
}

func NewBotStore(pool *pgxpool.Pool) *BotStore {
	return &BotStore{pool: pool}
}

func (s *BotStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS bot_challenges (
			site_id    BIGINT      NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
			challenge  VARCHAR(32) NOT NULL,
			enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
			config     JSONB       NOT NULL DEFAULT '{}'::jsonb,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (site_id, challenge)
		)`)
	return err
}

func (s *BotStore) GetForSite(ctx context.Context, siteID int64) ([]BotChallenge, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT site_id, challenge, enabled, config, updated_at
		   FROM bot_challenges WHERE site_id=$1`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	have := map[string]BotChallenge{}
	for rows.Next() {
		var b BotChallenge
		var cfgRaw []byte
		if err := rows.Scan(&b.SiteID, &b.Challenge, &b.Enabled, &cfgRaw, &b.UpdatedAt); err != nil {
			return nil, err
		}
		if len(cfgRaw) > 0 {
			_ = json.Unmarshal(cfgRaw, &b.Config)
		}
		if b.Config == nil {
			b.Config = map[string]any{}
		}
		have[b.Challenge] = b
	}
	// 补齐缺省值（默认 js/tls/dev 开，slider/behave 关）
	defaults := map[string]bool{
		"js": true, "tls": true, "dev": true, "slider": false, "behave": false,
	}
	out := make([]BotChallenge, 0, len(KnownChallenges))
	for _, c := range KnownChallenges {
		if ex, ok := have[c]; ok {
			out = append(out, ex)
		} else {
			out = append(out, BotChallenge{
				SiteID:    siteID,
				Challenge: c,
				Enabled:   defaults[c],
				Config:    map[string]any{},
			})
		}
	}
	return out, rows.Err()
}

func (s *BotStore) Upsert(ctx context.Context, b BotChallenge) error {
	for _, c := range KnownChallenges {
		if c == b.Challenge {
			cfg, _ := json.Marshal(b.Config)
			if len(cfg) == 0 {
				cfg = []byte("{}")
			}
			_, err := s.pool.Exec(ctx, `
				INSERT INTO bot_challenges (site_id, challenge, enabled, config, updated_at)
				VALUES ($1, $2, $3, $4::jsonb, NOW())
				ON CONFLICT (site_id, challenge) DO UPDATE
				   SET enabled = EXCLUDED.enabled,
				       config  = EXCLUDED.config,
				       updated_at = NOW()`,
				b.SiteID, b.Challenge, b.Enabled, string(cfg))
			return err
		}
	}
	return fmt.Errorf("unknown challenge %q (want %v)", b.Challenge, KnownChallenges)
}

type BotHandler struct{ store *BotStore }

func NewBotHandler(s *BotStore) *BotHandler { return &BotHandler{store: s} }

func (h *BotHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "siteId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}
	list, err := h.store.GetForSite(r.Context(), siteID)
	if err != nil {
		slog.Error("list bot challenges", "site", siteID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (h *BotHandler) Put(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "siteId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}
	ch := chi.URLParam(r, "challenge")
	var body struct {
		Enabled *bool          `json:"enabled"`
		Config  map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	cfg := body.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	if err := h.store.Upsert(r.Context(), BotChallenge{
		SiteID: siteID, Challenge: ch, Enabled: enabled, Config: cfg,
	}); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- API 端点 ----------

type APIEndpoint struct {
	ID           int64     `json:"id"`
	SiteID       int64     `json:"site_id"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	AuthType     string    `json:"auth_type"`
	RateLimit    string    `json:"rate_limit"`
	SchemaStatus string    `json:"schema_status"`
	QPS          int       `json:"qps"`
	Status       string    `json:"status"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type APIStore struct{ pool *pgxpool.Pool }

func NewAPIStore(pool *pgxpool.Pool) *APIStore { return &APIStore{pool: pool} }

func (s *APIStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS api_endpoints (
			id            BIGSERIAL PRIMARY KEY,
			site_id       BIGINT       NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
			method        VARCHAR(10)  NOT NULL,
			path          VARCHAR(512) NOT NULL,
			auth_type     VARCHAR(32)  NOT NULL DEFAULT 'JWT',
			rate_limit    VARCHAR(64)  NOT NULL DEFAULT '',
			schema_status VARCHAR(16)  NOT NULL DEFAULT 'pending',
			qps           INTEGER      NOT NULL DEFAULT 0,
			status        VARCHAR(16)  NOT NULL DEFAULT 'ok',
			description   TEXT         NOT NULL DEFAULT '',
			created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			UNIQUE (site_id, method, path)
		)`)
	return err
}

const apiCols = `id, site_id, method, path, auth_type, rate_limit, schema_status,
	qps, status, description, created_at, updated_at`

func (s *APIStore) List(ctx context.Context, siteID int64) ([]APIEndpoint, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+apiCols+`
		FROM api_endpoints WHERE site_id=$1 ORDER BY method, path`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]APIEndpoint, 0)
	for rows.Next() {
		var a APIEndpoint
		if err := rows.Scan(&a.ID, &a.SiteID, &a.Method, &a.Path, &a.AuthType, &a.RateLimit,
			&a.SchemaStatus, &a.QPS, &a.Status, &a.Description, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *APIStore) Get(ctx context.Context, id int64) (*APIEndpoint, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+apiCols+` FROM api_endpoints WHERE id=$1`, id)
	var a APIEndpoint
	if err := row.Scan(&a.ID, &a.SiteID, &a.Method, &a.Path, &a.AuthType, &a.RateLimit,
		&a.SchemaStatus, &a.QPS, &a.Status, &a.Description, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (s *APIStore) Create(ctx context.Context, a APIEndpoint) (*APIEndpoint, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO api_endpoints (site_id, method, path, auth_type, rate_limit,
		                          schema_status, description, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,COALESCE($8,'ok'))
		RETURNING `+apiCols,
		a.SiteID, a.Method, a.Path, a.AuthType, a.RateLimit, a.SchemaStatus, a.Description, a.Status)
	var out APIEndpoint
	if err := row.Scan(&out.ID, &out.SiteID, &out.Method, &out.Path, &out.AuthType, &out.RateLimit,
		&out.SchemaStatus, &out.QPS, &out.Status, &out.Description, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *APIStore) Update(ctx context.Context, id int64, patch APIEndpoint) (*APIEndpoint, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE api_endpoints
		   SET auth_type=COALESCE(NULLIF($2,''), auth_type),
		       rate_limit=COALESCE(NULLIF($3,''), rate_limit),
		       schema_status=COALESCE(NULLIF($4,''), schema_status),
		       status=COALESCE(NULLIF($5,''), status),
		       description=COALESCE(NULLIF($6,''), description),
		       qps=COALESCE($7, qps),
		       updated_at=NOW()
		 WHERE id=$1
		 RETURNING `+apiCols,
		id, patch.AuthType, patch.RateLimit, patch.SchemaStatus, patch.Status, patch.Description, patch.QPS)
	var out APIEndpoint
	if err := row.Scan(&out.ID, &out.SiteID, &out.Method, &out.Path, &out.AuthType, &out.RateLimit,
		&out.SchemaStatus, &out.QPS, &out.Status, &out.Description, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *APIStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM api_endpoints WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api endpoint %d not found", id)
	}
	return nil
}

// APIKPI 派生：总数 / 未授权拦截近 24h / JWT 重放 / 敏感脱敏
type APIKPI struct {
	Registered           int `json:"registered"`
	UnauthorizedBlocks24h int `json:"unauthorized_blocks_24h"`
	JWTReplayBlocks24h    int `json:"jwt_replay_blocks_24h"`
	SensitiveMasked24h    int `json:"sensitive_masked_24h"`
}

func (s *APIStore) KPI(ctx context.Context, siteID int64) (APIKPI, error) {
	var k APIKPI
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM api_endpoints WHERE site_id=$1`, siteID).Scan(&k.Registered); err != nil {
		return k, err
	}

	// 三个 24h 统计：从 attack_logs 派生。
	// 旧实现 `site=fmt.Sprintf("site-%d", siteID)` 完全错配 —— attack_logs.site
	// 列存的是站点 name（agent 上报时传 site 字符串名），不是 "site-N" 这种 ID 格式，
	// 导致计数永远为 0。修正：从 sites 表取真名，再按 name 计数。
	var siteName string
	if err := s.pool.QueryRow(ctx,
		`SELECT name FROM sites WHERE id=$1`, siteID).Scan(&siteName); err != nil {
		// 站点不存在或 db 出错 —— 计数维持 0，但记日志便于排查
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("api kpi lookup site name", "site_id", siteID, "err", err)
		}
		return k, nil
	}

	tryCount := func(where string) int {
		var n int
		// 只吞 pg『表/列缺失』错误（容忍 attack_logs 还未建立）；其他错误记 warn 便于排查
		err := s.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM attack_logs
			   WHERE occurred_at >= NOW() - INTERVAL '24 hours'
			     AND site = $1 AND `+where,
			siteName).Scan(&n)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && (pgErr.Code == "42P01" || pgErr.Code == "42703") {
				return 0
			}
			slog.Warn("api kpi count", "site", siteName, "where", where, "err", err)
		}
		return n
	}
	k.UnauthorizedBlocks24h = tryCount("attack_type ILIKE '%unauth%'")
	k.JWTReplayBlocks24h = tryCount("attack_type ILIKE '%jwt%replay%'")
	k.SensitiveMasked24h = tryCount(
		"(attack_type ILIKE '%sensitive%' OR attack_type ILIKE '%pii%')")
	return k, nil
}

type APIHandler struct{ store *APIStore }

func NewAPIHandler(s *APIStore) *APIHandler { return &APIHandler{store: s} }

func (h *APIHandler) List(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "siteId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}
	list, err := h.store.List(r.Context(), siteID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (h *APIHandler) Create(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "siteId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}
	var body APIEndpoint
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Method == "" || body.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "method 和 path 必填"})
		return
	}
	body.SiteID = siteID
	if body.AuthType == "" {
		body.AuthType = "JWT"
	}
	if body.SchemaStatus == "" {
		body.SchemaStatus = "pending"
	}
	out, err := h.store.Create(r.Context(), body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *APIHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var patch APIEndpoint
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	out, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		slog.Error("bot/api handler db error", "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *APIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandler) KPI(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "siteId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}
	kpi, err := h.store.KPI(r.Context(), siteID)
	if err != nil {
		slog.Error("bot/api handler db error", "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, kpi)
}
