package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, p ListParams) ([]Node, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if p.DeviceID != nil {
		conditions = append(conditions, fmt.Sprintf("device_id = $%d", argIdx))
		args = append(args, *p.DeviceID)
		argIdx++
	}
	if p.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, p.Status)
		argIdx++
	}
	if p.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR hostname ILIKE $%d OR ip_address ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+p.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM nodes "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count nodes: %w", err)
	}

	offset := (p.Page - 1) * p.PageSize
	query := fmt.Sprintf(`SELECT id, device_id, name, COALESCE(hostname,''), ip_address, status,
		cpu_cores, memory_mb, disk_gb, COALESCE(os_version,''), COALESCE(agent_ver,''),
		last_seen, created_at, updated_at
		FROM nodes %s ORDER BY id DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, p.PageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.DeviceID, &n.Name, &n.Hostname, &n.IPAddress, &n.Status,
			&n.CPUCores, &n.MemoryMB, &n.DiskGB, &n.OSVersion, &n.AgentVer,
			&n.LastSeen, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan node: %w", err)
		}
		nodes = append(nodes, n)
	}
	return nodes, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Node, error) {
	var n Node
	err := r.pool.QueryRow(ctx, `SELECT id, device_id, name, COALESCE(hostname,''), ip_address, status,
		cpu_cores, memory_mb, disk_gb, COALESCE(os_version,''), COALESCE(agent_ver,''),
		last_seen, created_at, updated_at
		FROM nodes WHERE id = $1`, id).Scan(
		&n.ID, &n.DeviceID, &n.Name, &n.Hostname, &n.IPAddress, &n.Status,
		&n.CPUCores, &n.MemoryMB, &n.DiskGB, &n.OSVersion, &n.AgentVer,
		&n.LastSeen, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	return &n, nil
}

func (r *Repository) Create(ctx context.Context, req CreateRequest) (*Node, error) {
	var n Node
	err := r.pool.QueryRow(ctx, `INSERT INTO nodes (device_id, name, hostname, ip_address, cpu_cores, memory_mb, disk_gb, os_version, agent_ver)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, device_id, name, COALESCE(hostname,''), ip_address, status,
		cpu_cores, memory_mb, disk_gb, COALESCE(os_version,''), COALESCE(agent_ver,''),
		last_seen, created_at, updated_at`,
		req.DeviceID, req.Name, req.Hostname, req.IPAddress, req.CPUCores, req.MemoryMB, req.DiskGB, req.OSVersion, req.AgentVer).Scan(
		&n.ID, &n.DeviceID, &n.Name, &n.Hostname, &n.IPAddress, &n.Status,
		&n.CPUCores, &n.MemoryMB, &n.DiskGB, &n.OSVersion, &n.AgentVer,
		&n.LastSeen, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create node: %w", err)
	}
	return &n, nil
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (*Node, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Hostname != nil {
		sets = append(sets, fmt.Sprintf("hostname = $%d", argIdx))
		args = append(args, *req.Hostname)
		argIdx++
	}
	if req.IPAddress != nil {
		sets = append(sets, fmt.Sprintf("ip_address = $%d", argIdx))
		args = append(args, *req.IPAddress)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.CPUCores != nil {
		sets = append(sets, fmt.Sprintf("cpu_cores = $%d", argIdx))
		args = append(args, *req.CPUCores)
		argIdx++
	}
	if req.MemoryMB != nil {
		sets = append(sets, fmt.Sprintf("memory_mb = $%d", argIdx))
		args = append(args, *req.MemoryMB)
		argIdx++
	}
	if req.DiskGB != nil {
		sets = append(sets, fmt.Sprintf("disk_gb = $%d", argIdx))
		args = append(args, *req.DiskGB)
		argIdx++
	}
	if req.OSVersion != nil {
		sets = append(sets, fmt.Sprintf("os_version = $%d", argIdx))
		args = append(args, *req.OSVersion)
		argIdx++
	}
	if req.AgentVer != nil {
		sets = append(sets, fmt.Sprintf("agent_ver = $%d", argIdx))
		args = append(args, *req.AgentVer)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE nodes SET %s WHERE id = $%d
		RETURNING id, device_id, name, COALESCE(hostname,''), ip_address, status,
		cpu_cores, memory_mb, disk_gb, COALESCE(os_version,''), COALESCE(agent_ver,''),
		last_seen, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var n Node
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&n.ID, &n.DeviceID, &n.Name, &n.Hostname, &n.IPAddress, &n.Status,
		&n.CPUCores, &n.MemoryMB, &n.DiskGB, &n.OSVersion, &n.AgentVer,
		&n.LastSeen, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update node: %w", err)
	}
	return &n, nil
}

func (r *Repository) ListByDeviceID(ctx context.Context, deviceID int64) ([]NodeBrief, error) {
	nodes, _, err := r.List(ctx, ListParams{
		Page:     1,
		PageSize: 100,
		DeviceID: &deviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("list nodes by device: %w", err)
	}

	briefs := make([]NodeBrief, len(nodes))
	for i, n := range nodes {
		hostname := n.Hostname
		if hostname == "" {
			hostname = n.Name
		}
		briefs[i] = NodeBrief{
			ID:       n.ID,
			DeviceID: n.DeviceID,
			Hostname: hostname,
			IPAddr:   n.IPAddress,
		}
	}
	return briefs, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM nodes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("node not found")
	}
	return nil
}
