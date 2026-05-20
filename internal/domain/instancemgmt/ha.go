package instancemgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HAGroup 与 waf-admin 实例页『HA 主备状态』表对齐：
// HA 组名 / 主 / 备 / VIP / 状态。state 取 ok / warn / critical，
// 前端把 ok 显示为『同步中』、其他显示为『检查中』。
type HAGroup struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	PrimaryNode string     `json:"primary_node"`
	StandbyNode string     `json:"standby_node"`
	VIP         string     `json:"vip"`
	State       string     `json:"state"`
	LastSwitch  *time.Time `json:"last_switch,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type HAStore struct {
	pool *pgxpool.Pool
}

func NewHAStore(pool *pgxpool.Pool) *HAStore {
	return &HAStore{pool: pool}
}

// EnsureSchema 启动时兜底：迁移没跑也别让 /ha-groups 直接 500。
// 真正的字段/索引以 migrations/000017_ha_groups.up.sql 为准。
func (s *HAStore) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ha_groups (
			id           BIGSERIAL PRIMARY KEY,
			name         VARCHAR(32)  NOT NULL UNIQUE,
			primary_node VARCHAR(128) NOT NULL,
			standby_node VARCHAR(128) NOT NULL,
			vip          VARCHAR(64)  NOT NULL,
			state        VARCHAR(16)  NOT NULL DEFAULT 'ok',
			last_switch  TIMESTAMPTZ,
			created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ha_groups_state ON ha_groups(state)`,
	}
	for _, q := range stmts {
		if _, err := s.pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	// 首次部署时 ha_groups 全空，前端表会空着；这里塞 4 条默认数据，
	// 与之前 UI 硬编码 HA-01..HA-04 一致，方便用户直接看到效果，
	// 已经存在的 name 不会被覆盖。
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ha_groups (name, primary_node, standby_node, vip, state) VALUES
			('HA-01', 'waf-01', 'waf-02', '10.0.1.100', 'ok'),
			('HA-02', 'waf-03', 'waf-04', '10.0.2.100', 'ok'),
			('HA-03', 'waf-05', 'waf-06', '10.0.3.100', 'warn'),
			('HA-04', 'waf-07', 'waf-08', '10.0.4.100', 'ok')
		ON CONFLICT (name) DO NOTHING`)
	return err
}

const haCols = `id, name, primary_node, standby_node, vip, state, last_switch, created_at, updated_at`

func (s *HAStore) List(ctx context.Context) ([]HAGroup, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+haCols+` FROM ha_groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HAGroup{}
	for rows.Next() {
		var g HAGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.PrimaryNode, &g.StandbyNode,
			&g.VIP, &g.State, &g.LastSwitch, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *HAStore) Get(ctx context.Context, id int64) (*HAGroup, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+haCols+` FROM ha_groups WHERE id=$1`, id)
	var g HAGroup
	if err := row.Scan(&g.ID, &g.Name, &g.PrimaryNode, &g.StandbyNode,
		&g.VIP, &g.State, &g.LastSwitch, &g.CreatedAt, &g.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &g, nil
}

func (s *HAStore) Create(ctx context.Context, g *HAGroup) error {
	return s.pool.QueryRow(ctx, `
		INSERT INTO ha_groups (name, primary_node, standby_node, vip, state)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at, updated_at`,
		g.Name, g.PrimaryNode, g.StandbyNode, g.VIP, g.State,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func (s *HAStore) Update(ctx context.Context, g *HAGroup) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE ha_groups SET name=$1, primary_node=$2, standby_node=$3,
		       vip=$4, state=$5, updated_at=NOW()
		 WHERE id=$6`,
		g.Name, g.PrimaryNode, g.StandbyNode, g.VIP, g.State, g.ID)
	return err
}

func (s *HAStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM ha_groups WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("ha-group %d not found", id)
	}
	return nil
}

// Switchover 把主备节点对调，写 last_switch 时间。
// 真正的 keepalived/VRRP 切换由部署侧 watchdog 负责，这里只更新视图。
func (s *HAStore) Switchover(ctx context.Context, id int64) (*HAGroup, error) {
	g, err := s.Get(ctx, id)
	if err != nil || g == nil {
		return nil, err
	}
	g.PrimaryNode, g.StandbyNode = g.StandbyNode, g.PrimaryNode
	now := time.Now()
	g.LastSwitch = &now
	_, err = s.pool.Exec(ctx, `
		UPDATE ha_groups SET primary_node=$1, standby_node=$2,
		       last_switch=$3, updated_at=NOW()
		 WHERE id=$4`,
		g.PrimaryNode, g.StandbyNode, now, g.ID)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// HAHandler hosts /ha-groups routes.
type HAHandler struct {
	store *HAStore
}

func NewHAHandler(store *HAStore) *HAHandler {
	return &HAHandler{store: store}
}

func (h *HAHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.List(r.Context())
	if err != nil {
		slog.Error("list ha-groups", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ha_groups": list})
}

func (h *HAHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	g, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if g == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ha-group not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ha_group": g})
}

func (h *HAHandler) Create(w http.ResponseWriter, r *http.Request) {
	var g HAGroup
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if g.Name == "" || g.PrimaryNode == "" || g.StandbyNode == "" || g.VIP == "" {
		writeJSON(w, http.StatusBadRequest,
			map[string]string{"error": "name / primary_node / standby_node / vip 必填"})
		return
	}
	if g.State == "" {
		g.State = "ok"
	}
	if err := h.store.Create(r.Context(), &g); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ha_group": g})
}

func (h *HAHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	existing, err := h.store.Get(r.Context(), id)
	if err != nil || existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ha-group not found"})
		return
	}
	var patch HAGroup
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.PrimaryNode != "" {
		existing.PrimaryNode = patch.PrimaryNode
	}
	if patch.StandbyNode != "" {
		existing.StandbyNode = patch.StandbyNode
	}
	if patch.VIP != "" {
		existing.VIP = patch.VIP
	}
	if patch.State != "" {
		existing.State = patch.State
	}
	if err := h.store.Update(r.Context(), existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ha_group": existing})
}

func (h *HAHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// Switchover 触发主备切换。视图上立刻把 primary / standby 对调。
func (h *HAHandler) Switchover(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	g, err := h.store.Switchover(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if g == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ha-group not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ha_group": g})
}
