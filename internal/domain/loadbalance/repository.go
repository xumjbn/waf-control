package loadbalance

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

// --- VIPs ---

func (r *Repository) ListVIPs(ctx context.Context) ([]VIP, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, COALESCE(description,''), address, protocol,
		protocol_port, pool_id, connection_limit, session_persistence, admin_state_up, created_at, updated_at
		FROM lb_vips ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list vips: %w", err)
	}
	defer rows.Close()

	var vips []VIP
	for rows.Next() {
		var v VIP
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.Address, &v.Protocol,
			&v.ProtocolPort, &v.PoolID, &v.ConnectionLimit, &v.SessionPersistence,
			&v.AdminStateUp, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vip: %w", err)
		}
		vips = append(vips, v)
	}
	return vips, nil
}

func (r *Repository) CreateVIP(ctx context.Context, req CreateVIPRequest) (*VIP, error) {
	protocol := req.Protocol
	if protocol == "" {
		protocol = "HTTP"
	}
	port := req.ProtocolPort
	if port == 0 {
		port = 80
	}

	var v VIP
	err := r.pool.QueryRow(ctx, `INSERT INTO lb_vips (name, description, address, protocol, protocol_port, pool_id, connection_limit, session_persistence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, COALESCE(description,''), address, protocol, protocol_port, pool_id, connection_limit, session_persistence, admin_state_up, created_at, updated_at`,
		req.Name, req.Description, req.Address, protocol, port, req.PoolID, req.ConnectionLimit, req.SessionPersistence).Scan(
		&v.ID, &v.Name, &v.Description, &v.Address, &v.Protocol, &v.ProtocolPort,
		&v.PoolID, &v.ConnectionLimit, &v.SessionPersistence, &v.AdminStateUp, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create vip: %w", err)
	}
	return &v, nil
}

func (r *Repository) UpdateVIP(ctx context.Context, id int64, req UpdateVIPRequest) (*VIP, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Address != nil {
		sets = append(sets, fmt.Sprintf("address = $%d", argIdx))
		args = append(args, *req.Address)
		argIdx++
	}
	if req.Protocol != nil {
		sets = append(sets, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, *req.Protocol)
		argIdx++
	}
	if req.ProtocolPort != nil {
		sets = append(sets, fmt.Sprintf("protocol_port = $%d", argIdx))
		args = append(args, *req.ProtocolPort)
		argIdx++
	}
	if req.PoolID != nil {
		sets = append(sets, fmt.Sprintf("pool_id = $%d", argIdx))
		args = append(args, *req.PoolID)
		argIdx++
	}
	if req.ConnectionLimit != nil {
		sets = append(sets, fmt.Sprintf("connection_limit = $%d", argIdx))
		args = append(args, *req.ConnectionLimit)
		argIdx++
	}
	if req.SessionPersistence != nil {
		sets = append(sets, fmt.Sprintf("session_persistence = $%d", argIdx))
		args = append(args, *req.SessionPersistence)
		argIdx++
	}
	if req.AdminStateUp != nil {
		sets = append(sets, fmt.Sprintf("admin_state_up = $%d", argIdx))
		args = append(args, *req.AdminStateUp)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE lb_vips SET %s WHERE id = $%d
		RETURNING id, name, COALESCE(description,''), address, protocol, protocol_port, pool_id, connection_limit, session_persistence, admin_state_up, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var v VIP
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&v.ID, &v.Name, &v.Description, &v.Address, &v.Protocol, &v.ProtocolPort,
		&v.PoolID, &v.ConnectionLimit, &v.SessionPersistence, &v.AdminStateUp, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update vip: %w", err)
	}
	return &v, nil
}

func (r *Repository) GetVIP(ctx context.Context, id int64) (*VIP, error) {
	var v VIP
	err := r.pool.QueryRow(ctx, `SELECT id, name, COALESCE(description,''), address, protocol,
		protocol_port, pool_id, connection_limit, session_persistence, admin_state_up, created_at, updated_at
		FROM lb_vips WHERE id = $1`, id).Scan(
		&v.ID, &v.Name, &v.Description, &v.Address, &v.Protocol, &v.ProtocolPort,
		&v.PoolID, &v.ConnectionLimit, &v.SessionPersistence, &v.AdminStateUp, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get vip: %w", err)
	}
	return &v, nil
}

func (r *Repository) DeleteVIP(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM lb_vips WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete vip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("vip not found")
	}
	return nil
}

// --- Pools ---

func (r *Repository) ListPools(ctx context.Context) ([]Pool, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, COALESCE(description,''), protocol, lb_method, admin_state_up, created_at, updated_at
		FROM lb_pools ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list pools: %w", err)
	}
	defer rows.Close()

	var pools []Pool
	for rows.Next() {
		var p Pool
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Protocol, &p.LBMethod, &p.AdminStateUp, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan pool: %w", err)
		}
		pools = append(pools, p)
	}
	return pools, nil
}

func (r *Repository) CreatePool(ctx context.Context, req CreatePoolRequest) (*Pool, error) {
	protocol := req.Protocol
	if protocol == "" {
		protocol = "HTTP"
	}
	method := req.LBMethod
	if method == "" {
		method = "ROUND_ROBIN"
	}

	var p Pool
	err := r.pool.QueryRow(ctx, `INSERT INTO lb_pools (name, description, protocol, lb_method)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, COALESCE(description,''), protocol, lb_method, admin_state_up, created_at, updated_at`,
		req.Name, req.Description, protocol, method).Scan(
		&p.ID, &p.Name, &p.Description, &p.Protocol, &p.LBMethod, &p.AdminStateUp, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &p, nil
}

func (r *Repository) UpdatePool(ctx context.Context, id int64, req UpdatePoolRequest) (*Pool, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Protocol != nil {
		sets = append(sets, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, *req.Protocol)
		argIdx++
	}
	if req.LBMethod != nil {
		sets = append(sets, fmt.Sprintf("lb_method = $%d", argIdx))
		args = append(args, *req.LBMethod)
		argIdx++
	}
	if req.AdminStateUp != nil {
		sets = append(sets, fmt.Sprintf("admin_state_up = $%d", argIdx))
		args = append(args, *req.AdminStateUp)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE lb_pools SET %s WHERE id = $%d
		RETURNING id, name, COALESCE(description,''), protocol, lb_method, admin_state_up, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var p Pool
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&p.ID, &p.Name, &p.Description, &p.Protocol, &p.LBMethod, &p.AdminStateUp, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update pool: %w", err)
	}
	return &p, nil
}

func (r *Repository) GetPool(ctx context.Context, id int64) (*Pool, error) {
	var p Pool
	err := r.pool.QueryRow(ctx, `SELECT id, name, COALESCE(description,''), protocol, lb_method, admin_state_up, created_at, updated_at
		FROM lb_pools WHERE id = $1`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Protocol, &p.LBMethod, &p.AdminStateUp, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get pool: %w", err)
	}
	return &p, nil
}

func (r *Repository) DeletePool(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM lb_pools WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete pool: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pool not found")
	}
	return nil
}

// --- Members ---

func (r *Repository) ListMembers(ctx context.Context, poolID int64) ([]Member, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, pool_id, address, protocol_port, weight, admin_state_up, status, created_at
		FROM lb_members WHERE pool_id = $1 ORDER BY id`, poolID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.PoolID, &m.Address, &m.ProtocolPort, &m.Weight, &m.AdminStateUp, &m.Status, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *Repository) CreateMember(ctx context.Context, req CreateMemberRequest) (*Member, error) {
	weight := req.Weight
	if weight <= 0 {
		weight = 1
	}

	var m Member
	err := r.pool.QueryRow(ctx, `INSERT INTO lb_members (pool_id, address, protocol_port, weight)
		VALUES ($1, $2, $3, $4)
		RETURNING id, pool_id, address, protocol_port, weight, admin_state_up, status, created_at`,
		req.PoolID, req.Address, req.ProtocolPort, weight).Scan(
		&m.ID, &m.PoolID, &m.Address, &m.ProtocolPort, &m.Weight, &m.AdminStateUp, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create member: %w", err)
	}
	return &m, nil
}

func (r *Repository) GetMember(ctx context.Context, id int64) (*Member, error) {
	var m Member
	err := r.pool.QueryRow(ctx, `SELECT id, pool_id, address, protocol_port, weight, admin_state_up, status, created_at
		FROM lb_members WHERE id = $1`, id).Scan(
		&m.ID, &m.PoolID, &m.Address, &m.ProtocolPort, &m.Weight, &m.AdminStateUp, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get member: %w", err)
	}
	return &m, nil
}

func (r *Repository) UpdateMember(ctx context.Context, id int64, req UpdateMemberRequest) (*Member, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Address != nil {
		sets = append(sets, fmt.Sprintf("address = $%d", argIdx))
		args = append(args, *req.Address)
		argIdx++
	}
	if req.ProtocolPort != nil {
		sets = append(sets, fmt.Sprintf("protocol_port = $%d", argIdx))
		args = append(args, *req.ProtocolPort)
		argIdx++
	}
	if req.Weight != nil {
		sets = append(sets, fmt.Sprintf("weight = $%d", argIdx))
		args = append(args, *req.Weight)
		argIdx++
	}
	if req.AdminStateUp != nil {
		sets = append(sets, fmt.Sprintf("admin_state_up = $%d", argIdx))
		args = append(args, *req.AdminStateUp)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`UPDATE lb_members SET %s WHERE id = $%d
		RETURNING id, pool_id, address, protocol_port, weight, admin_state_up, status, created_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var m Member
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&m.ID, &m.PoolID, &m.Address, &m.ProtocolPort, &m.Weight, &m.AdminStateUp, &m.Status, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update member: %w", err)
	}
	return &m, nil
}

func (r *Repository) DeleteMember(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM lb_members WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

// --- Health Monitors ---

func (r *Repository) GetHealthMonitor(ctx context.Context, poolID int64) (*HealthMonitor, error) {
	var hm HealthMonitor
	err := r.pool.QueryRow(ctx, `SELECT id, pool_id, type, delay, timeout, max_retries, COALESCE(http_method,'GET'), COALESCE(url_path,'/'), COALESCE(expected_codes,'200'), admin_state_up, created_at, updated_at
		FROM lb_health_monitors WHERE pool_id = $1`, poolID).Scan(
		&hm.ID, &hm.PoolID, &hm.Type, &hm.Delay, &hm.Timeout, &hm.MaxRetries,
		&hm.HTTPMethod, &hm.URLPath, &hm.ExpectedCodes, &hm.AdminStateUp, &hm.CreatedAt, &hm.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get health monitor: %w", err)
	}
	return &hm, nil
}

func (r *Repository) CreateHealthMonitor(ctx context.Context, req CreateHealthMonitorRequest) (*HealthMonitor, error) {
	typ := req.Type
	if typ == "" {
		typ = "HTTP"
	}
	delay := req.Delay
	if delay <= 0 {
		delay = 5
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 3
	}
	maxRetries := req.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	var hm HealthMonitor
	err := r.pool.QueryRow(ctx, `INSERT INTO lb_health_monitors (pool_id, type, delay, timeout, max_retries, http_method, url_path, expected_codes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, pool_id, type, delay, timeout, max_retries, COALESCE(http_method,'GET'), COALESCE(url_path,'/'), COALESCE(expected_codes,'200'), admin_state_up, created_at, updated_at`,
		req.PoolID, typ, delay, timeout, maxRetries, req.HTTPMethod, req.URLPath, req.ExpectedCodes).Scan(
		&hm.ID, &hm.PoolID, &hm.Type, &hm.Delay, &hm.Timeout, &hm.MaxRetries,
		&hm.HTTPMethod, &hm.URLPath, &hm.ExpectedCodes, &hm.AdminStateUp, &hm.CreatedAt, &hm.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create health monitor: %w", err)
	}
	return &hm, nil
}

func (r *Repository) UpdateHealthMonitor(ctx context.Context, poolID int64, req UpdateHealthMonitorRequest) (*HealthMonitor, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Type != nil {
		sets = append(sets, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, *req.Type)
		argIdx++
	}
	if req.Delay != nil {
		sets = append(sets, fmt.Sprintf("delay = $%d", argIdx))
		args = append(args, *req.Delay)
		argIdx++
	}
	if req.Timeout != nil {
		sets = append(sets, fmt.Sprintf("timeout = $%d", argIdx))
		args = append(args, *req.Timeout)
		argIdx++
	}
	if req.MaxRetries != nil {
		sets = append(sets, fmt.Sprintf("max_retries = $%d", argIdx))
		args = append(args, *req.MaxRetries)
		argIdx++
	}
	if req.HTTPMethod != nil {
		sets = append(sets, fmt.Sprintf("http_method = $%d", argIdx))
		args = append(args, *req.HTTPMethod)
		argIdx++
	}
	if req.URLPath != nil {
		sets = append(sets, fmt.Sprintf("url_path = $%d", argIdx))
		args = append(args, *req.URLPath)
		argIdx++
	}
	if req.ExpectedCodes != nil {
		sets = append(sets, fmt.Sprintf("expected_codes = $%d", argIdx))
		args = append(args, *req.ExpectedCodes)
		argIdx++
	}
	if req.AdminStateUp != nil {
		sets = append(sets, fmt.Sprintf("admin_state_up = $%d", argIdx))
		args = append(args, *req.AdminStateUp)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE lb_health_monitors SET %s WHERE pool_id = $%d
		RETURNING id, pool_id, type, delay, timeout, max_retries, COALESCE(http_method,'GET'), COALESCE(url_path,'/'), COALESCE(expected_codes,'200'), admin_state_up, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, poolID)

	var hm HealthMonitor
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&hm.ID, &hm.PoolID, &hm.Type, &hm.Delay, &hm.Timeout, &hm.MaxRetries,
		&hm.HTTPMethod, &hm.URLPath, &hm.ExpectedCodes, &hm.AdminStateUp, &hm.CreatedAt, &hm.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update health monitor: %w", err)
	}
	return &hm, nil
}

func (r *Repository) DeleteHealthMonitor(ctx context.Context, poolID int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM lb_health_monitors WHERE pool_id = $1", poolID)
	if err != nil {
		return fmt.Errorf("delete health monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("health monitor not found")
	}
	return nil
}
