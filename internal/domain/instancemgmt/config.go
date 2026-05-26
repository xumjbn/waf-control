package instancemgmt

// config.go —— 节点管理员可编辑配置（instance_configs 表）
//
// 与运行时通过 gRPC 上报的 NodeState 解耦：
//   · NodeState：CPU/MEM/IP/状态/版本等观测值，运行时刷新，存内存
//   · InstanceConfig：管理员意图（角色 / 网关 / DNS / 标签 / 启停 /
//     资源限制 / 维护窗口），UI 写入持久化
//
// 与 waf-admin InstanceDetail 配置 Tab 字段一一对应。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/pkg/httputil"
)

type InstanceConfig struct {
	NodeID            string    `json:"node_id"`
	Role              string    `json:"role"`               // data / control / edge
	Gateway           string    `json:"gateway"`
	DNS               string    `json:"dns"`                // 逗号分隔
	Tags              string    `json:"tags"`               // 逗号分隔
	Enabled           bool      `json:"enabled"`
	MaxConnections    int       `json:"max_connections"`
	MaxQPS            int       `json:"max_qps"`
	CPUSoftLimit      int       `json:"cpu_soft_limit"`     // 百分比
	MaintenanceWindow string    `json:"maintenance_window"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// defaultConfig 返回表中没有该 node_id 时的默认值，与 migration 000021 的 DEFAULT 一致。
func defaultConfig(nodeID string) InstanceConfig {
	return InstanceConfig{
		NodeID:            nodeID,
		Role:              "data",
		Gateway:           "",
		DNS:               "",
		Tags:              "",
		Enabled:           true,
		MaxConnections:    50000,
		MaxQPS:            20000,
		CPUSoftLimit:      80,
		MaintenanceWindow: "周日 02:00 - 04:00",
	}
}

type ConfigStore struct {
	pool *pgxpool.Pool
}

func NewConfigStore(pool *pgxpool.Pool) *ConfigStore {
	return &ConfigStore{pool: pool}
}

func (s *ConfigStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS instance_configs (
			node_id            VARCHAR(128) PRIMARY KEY,
			role               VARCHAR(16)  NOT NULL DEFAULT 'data',
			gateway            VARCHAR(64)  NOT NULL DEFAULT '',
			dns                VARCHAR(256) NOT NULL DEFAULT '',
			tags               VARCHAR(256) NOT NULL DEFAULT '',
			enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
			max_connections    INTEGER      NOT NULL DEFAULT 50000,
			max_qps            INTEGER      NOT NULL DEFAULT 20000,
			cpu_soft_limit     INTEGER      NOT NULL DEFAULT 80,
			maintenance_window VARCHAR(64)  NOT NULL DEFAULT '周日 02:00 - 04:00',
			updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("instance_configs ensure schema: %w", err)
	}
	return nil
}

const cfgCols = `node_id, role, gateway, dns, tags, enabled,
	max_connections, max_qps, cpu_soft_limit, maintenance_window, updated_at`

func (s *ConfigStore) Get(ctx context.Context, nodeID string) (InstanceConfig, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+cfgCols+` FROM instance_configs WHERE node_id=$1`, nodeID)
	var c InstanceConfig
	if err := row.Scan(&c.NodeID, &c.Role, &c.Gateway, &c.DNS, &c.Tags, &c.Enabled,
		&c.MaxConnections, &c.MaxQPS, &c.CPUSoftLimit, &c.MaintenanceWindow, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultConfig(nodeID), nil
		}
		return InstanceConfig{}, fmt.Errorf("get instance config %s: %w", nodeID, err)
	}
	return c, nil
}

func (s *ConfigStore) Upsert(ctx context.Context, c InstanceConfig) error {
	// 校验枚举字段
	switch c.Role {
	case "data", "control", "edge":
	default:
		return fmt.Errorf("invalid role %q (want data/control/edge)", c.Role)
	}
	if c.CPUSoftLimit < 0 || c.CPUSoftLimit > 100 {
		return fmt.Errorf("cpu_soft_limit must be 0-100, got %d", c.CPUSoftLimit)
	}
	if c.MaxConnections < 0 || c.MaxQPS < 0 {
		return fmt.Errorf("max_connections / max_qps must be >= 0")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO instance_configs
			(node_id, role, gateway, dns, tags, enabled,
			 max_connections, max_qps, cpu_soft_limit, maintenance_window, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW())
		ON CONFLICT (node_id) DO UPDATE
		   SET role=$2, gateway=$3, dns=$4, tags=$5, enabled=$6,
		       max_connections=$7, max_qps=$8, cpu_soft_limit=$9,
		       maintenance_window=$10, updated_at=NOW()`,
		c.NodeID, c.Role, c.Gateway, c.DNS, c.Tags, c.Enabled,
		c.MaxConnections, c.MaxQPS, c.CPUSoftLimit, c.MaintenanceWindow)
	return err
}

// ConfigHandler hosts /instances/{nodeId}/config routes.
type ConfigHandler struct {
	store *ConfigStore
}

func NewConfigHandler(store *ConfigStore) *ConfigHandler {
	return &ConfigHandler{store: store}
}

func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeId")
	if nodeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id required"})
		return
	}
	cfg, err := h.store.Get(r.Context(), nodeID)
	if err != nil {
		slog.Error("get instance config", "node_id", nodeID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *ConfigHandler) Put(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeId")
	if nodeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id required"})
		return
	}
	// 从现有/默认 config 出发 patch
	cur, err := h.store.Get(r.Context(), nodeID)
	if err != nil {
		slog.Error("get instance config", "node_id", nodeID, "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	var body struct {
		Role              *string `json:"role"`
		Gateway           *string `json:"gateway"`
		DNS               *string `json:"dns"`
		Tags              *string `json:"tags"`
		Enabled           *bool   `json:"enabled"`
		MaxConnections    *int    `json:"max_connections"`
		MaxQPS            *int    `json:"max_qps"`
		CPUSoftLimit      *int    `json:"cpu_soft_limit"`
		MaintenanceWindow *string `json:"maintenance_window"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Role != nil {
		cur.Role = *body.Role
	}
	if body.Gateway != nil {
		cur.Gateway = *body.Gateway
	}
	if body.DNS != nil {
		cur.DNS = *body.DNS
	}
	if body.Tags != nil {
		cur.Tags = *body.Tags
	}
	if body.Enabled != nil {
		cur.Enabled = *body.Enabled
	}
	if body.MaxConnections != nil {
		cur.MaxConnections = *body.MaxConnections
	}
	if body.MaxQPS != nil {
		cur.MaxQPS = *body.MaxQPS
	}
	if body.CPUSoftLimit != nil {
		cur.CPUSoftLimit = *body.CPUSoftLimit
	}
	if body.MaintenanceWindow != nil {
		cur.MaintenanceWindow = *body.MaintenanceWindow
	}
	if err := h.store.Upsert(r.Context(), cur); err != nil {
		// Upsert 返回的是业务校验错误（role/cpu_soft_limit 等），消息可对用户暴露；
		// 但若是底层 db 错误，Sanitize 也能识别。
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) || errors.Is(err, pgx.ErrNoRows) {
			slog.Error("upsert instance config", "node_id", nodeID, "err", err)
			status, msg := httputil.SanitizeDBError(err)
			writeJSON(w, status, map[string]string{"error": msg})
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return
	}
	// 返回最新值
	latest, err := h.store.Get(r.Context(), nodeID)
	if err != nil {
		slog.Error("get instance config (after upsert)", "node_id", nodeID, "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, latest)
}
