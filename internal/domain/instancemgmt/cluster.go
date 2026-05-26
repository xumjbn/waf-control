package instancemgmt

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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/pkg/httputil"
)

// Cluster matches mocks/nebula.ts shape: id/name/vip/algo/state/site_count + nodes.
type Cluster struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	VIP         string    `json:"vip"`
	Algo        string    `json:"algo"`
	State       string    `json:"state"`        // ok / warn / critical
	SiteCount   int       `json:"site_count"`
	Description string    `json:"description,omitempty"`
	Nodes       int       `json:"nodes"`        // derived from cluster_members
	NodeIDs     []string  `json:"node_ids"`     // member node IDs
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ClusterStore wraps the cluster table; instancemgmt currently has no
// repository abstraction (the existing handler is agent-driven), so we put a
// small store here next to the cluster handler rather than introduce a
// separate repository file.
type ClusterStore struct {
	pool *pgxpool.Pool
}

func NewClusterStore(pool *pgxpool.Pool) *ClusterStore {
	return &ClusterStore{pool: pool}
}

// EnsureSchema 启动时兜底：迁移 000009_instance_clusters 没跑也别让 /clusters
// 直接 500。clusters + cluster_members 两张表 + 默认 4 行种子，对应
// migrations/000009_instance_clusters.up.sql。
func (s *ClusterStore) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS clusters (
			id          BIGSERIAL PRIMARY KEY,
			name        VARCHAR(64) NOT NULL UNIQUE,
			vip         VARCHAR(64) NOT NULL DEFAULT '',
			algo        VARCHAR(32) NOT NULL DEFAULT 'round-robin',
			state       VARCHAR(16) NOT NULL DEFAULT 'ok',
			site_count  INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS cluster_members (
			cluster_id BIGINT NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
			node_id    VARCHAR(128) NOT NULL,
			role       VARCHAR(16) NOT NULL DEFAULT 'primary',
			joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (cluster_id, node_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cluster_members_node ON cluster_members(node_id)`,
	}
	for _, q := range stmts {
		if _, err := s.pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	// 默认 4 个集群种子已下放到 migration 000009 —— 仅在初次 fresh DB 时插入一次。
	// 之前在 EnsureSchema 里 ON CONFLICT DO NOTHING 每次重启都试图插，
	// 用户删除『CLU-WWW』重启就回来，UX 反直觉。
	return nil
}

const clusterCols = `id, name, vip, algo, state, site_count, description, created_at, updated_at`

func (s *ClusterStore) List(ctx context.Context) ([]Cluster, error) {
	// 单 SQL 取所有集群 + 成员 node_id 数组 —— 消除之前每个 cluster 调一次 memberIDs
	// 的 N+1 问题。COALESCE 保证空集群返回空数组而非 NULL。
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.name, c.vip, c.algo, c.state, c.site_count, c.description,
		       c.created_at, c.updated_at,
		       COALESCE(
		         (SELECT array_agg(cm.node_id ORDER BY cm.joined_at)
		            FROM cluster_members cm WHERE cm.cluster_id = c.id),
		         ARRAY[]::varchar[]
		       ) AS node_ids
		  FROM clusters c
		 ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Cluster{}
	for rows.Next() {
		var c Cluster
		var nodeIDs []string
		if err := rows.Scan(&c.ID, &c.Name, &c.VIP, &c.Algo, &c.State,
			&c.SiteCount, &c.Description, &c.CreatedAt, &c.UpdatedAt, &nodeIDs); err != nil {
			return nil, err
		}
		c.NodeIDs = nodeIDs
		if c.NodeIDs == nil {
			c.NodeIDs = []string{}
		}
		c.Nodes = len(c.NodeIDs)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *ClusterStore) memberIDs(ctx context.Context, clusterID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT node_id FROM cluster_members WHERE cluster_id=$1 ORDER BY joined_at`, clusterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func (s *ClusterStore) Get(ctx context.Context, id int64) (*Cluster, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+clusterCols+` FROM clusters WHERE id=$1`, id)
	var c Cluster
	if err := row.Scan(&c.ID, &c.Name, &c.VIP, &c.Algo, &c.State,
		&c.SiteCount, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ids, mErr := s.memberIDs(ctx, c.ID)
	if mErr != nil {
		return nil, fmt.Errorf("cluster.Get memberIDs: %w", mErr)
	}
	c.NodeIDs = ids
	c.Nodes = len(ids)
	return &c, nil
}

func (s *ClusterStore) Create(ctx context.Context, c *Cluster) error {
	return s.pool.QueryRow(ctx, `
		INSERT INTO clusters (name, vip, algo, state, site_count, description)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, created_at, updated_at`,
		c.Name, c.VIP, c.Algo, c.State, c.SiteCount, c.Description,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// ClusterPatch 指针字段：只有显式传入的字段才会被覆盖。空字符串等价于"清空"
// 而不是"保持原值"——之前 `if patch.X != ""` 语义无法清空 description/vip。
type ClusterPatch struct {
	Name        *string
	VIP         *string
	Algo        *string
	State       *string
	SiteCount   *int
	Description *string
}

func (s *ClusterStore) Update(ctx context.Context, id int64, p ClusterPatch) error {
	// COALESCE($n, col) 让 NULL 表示不变；非 NULL（含空串）表示覆盖。
	_, err := s.pool.Exec(ctx, `
		UPDATE clusters
		   SET name        = COALESCE($1, name),
		       vip         = COALESCE($2, vip),
		       algo        = COALESCE($3, algo),
		       state       = COALESCE($4, state),
		       site_count  = COALESCE($5, site_count),
		       description = COALESCE($6, description),
		       updated_at  = NOW()
		 WHERE id = $7`,
		p.Name, p.VIP, p.Algo, p.State, p.SiteCount, p.Description, id)
	return err
}

func (s *ClusterStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM clusters WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("cluster %d not found", id)
	}
	return nil
}

func (s *ClusterStore) AssignNode(ctx context.Context, clusterID int64, nodeID, role string) error {
	if role == "" {
		role = "primary"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cluster_members (cluster_id, node_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (cluster_id, node_id) DO UPDATE SET role = EXCLUDED.role`,
		clusterID, nodeID, role)
	return err
}

func (s *ClusterStore) RemoveNode(ctx context.Context, clusterID int64, nodeID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM cluster_members WHERE cluster_id=$1 AND node_id=$2`, clusterID, nodeID)
	return err
}

// ClusterHandler hosts /clusters routes.
type ClusterHandler struct {
	store *ClusterStore
}

func NewClusterHandler(store *ClusterStore) *ClusterHandler {
	return &ClusterHandler{store: store}
}

func (h *ClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.List(r.Context())
	if err != nil {
		slog.Error("list clusters", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"clusters": list})
}

func (h *ClusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if c == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "cluster not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cluster": c})
}

func (h *ClusterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var c Cluster
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if c.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	if c.Algo == "" {
		c.Algo = "round-robin"
	}
	if c.State == "" {
		c.State = "ok"
	}
	if err := h.store.Create(r.Context(), &c); err != nil {
		slog.Error("cluster handler db error", "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"cluster": c})
}

func (h *ClusterHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	existing, err := h.store.Get(r.Context(), id)
	if err != nil || existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "cluster not found"})
		return
	}
	// 接受指针字段 patch —— 空串现在可以清空 description / vip
	var body struct {
		Name        *string `json:"name"`
		VIP         *string `json:"vip"`
		Algo        *string `json:"algo"`
		State       *string `json:"state"`
		SiteCount   *int    `json:"site_count"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.store.Update(r.Context(), id, ClusterPatch{
		Name:        body.Name,
		VIP:         body.VIP,
		Algo:        body.Algo,
		State:       body.State,
		SiteCount:   body.SiteCount,
		Description: body.Description,
	}); err != nil {
		slog.Error("cluster handler db error", "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	// Update 走了 COALESCE patch，existing 已不准确，回拉最新
	updated, gerr := h.store.Get(r.Context(), id)
	if gerr != nil || updated == nil {
		writeJSON(w, http.StatusOK, map[string]any{"cluster": existing})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cluster": updated})
}

func (h *ClusterHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// AssignNode PUTs membership; RemoveNode DELETEs it.
func (h *ClusterHandler) AssignNode(w http.ResponseWriter, r *http.Request) {
	cid, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cluster id"})
		return
	}
	nodeID := chi.URLParam(r, "nodeId")
	role := r.URL.Query().Get("role")
	if err := h.store.AssignNode(r.Context(), cid, nodeID, role); err != nil {
		slog.Error("cluster handler db error", "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ClusterHandler) RemoveNode(w http.ResponseWriter, r *http.Request) {
	cid, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid cluster id"})
		return
	}
	nodeID := chi.URLParam(r, "nodeId")
	if err := h.store.RemoveNode(r.Context(), cid, nodeID); err != nil {
		slog.Error("cluster handler db error", "err", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
