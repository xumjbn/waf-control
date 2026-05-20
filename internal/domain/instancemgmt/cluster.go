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

const clusterCols = `id, name, vip, algo, state, site_count, description, created_at, updated_at`

func (s *ClusterStore) List(ctx context.Context) ([]Cluster, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+clusterCols+` FROM clusters ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Cluster
	for rows.Next() {
		var c Cluster
		if err := rows.Scan(&c.ID, &c.Name, &c.VIP, &c.Algo, &c.State,
			&c.SiteCount, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		ids, _ := s.memberIDs(ctx, c.ID)
		c.NodeIDs = ids
		c.Nodes = len(ids)
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
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ids, _ := s.memberIDs(ctx, c.ID)
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

func (s *ClusterStore) Update(ctx context.Context, c *Cluster) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE clusters SET name=$1, vip=$2, algo=$3, state=$4,
		       site_count=$5, description=$6, updated_at=NOW()
		 WHERE id=$7`,
		c.Name, c.VIP, c.Algo, c.State, c.SiteCount, c.Description, c.ID)
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
	var patch Cluster
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.VIP != "" {
		existing.VIP = patch.VIP
	}
	if patch.Algo != "" {
		existing.Algo = patch.Algo
	}
	if patch.State != "" {
		existing.State = patch.State
	}
	if patch.SiteCount > 0 {
		existing.SiteCount = patch.SiteCount
	}
	if patch.Description != "" {
		existing.Description = patch.Description
	}
	if err := h.store.Update(r.Context(), existing); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cluster": existing})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
