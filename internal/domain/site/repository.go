package site

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/domain/identity"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, p ListParams) ([]Site, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	// 多租户 scope —— 非 admin 用户只看自己分配的 project。
	// nil scope（没经过 ScopeMiddleware 的兼容路径）回退到不过滤。
	if scope := identity.ScopeFromContext(ctx); scope != nil && !scope.IsAdmin {
		if len(scope.IDs) == 0 {
			// 没有项目可见 ⇒ 直接返回空列表，不打 SQL。
			return []Site{}, 0, nil
		}
		conditions = append(conditions, fmt.Sprintf("project_id = ANY($%d::bigint[])", argIdx))
		args = append(args, scope.IDs)
		argIdx++
	}

	if p.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, p.Status)
		argIdx++
	}
	if p.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR domain ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+p.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM sites "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sites: %w", err)
	}

	offset := (p.Page - 1) * p.PageSize
	query := fmt.Sprintf(`SELECT id, name, domain, listen_port, ssl_enabled, COALESCE(ssl_cert,''), COALESCE(ssl_key,''),
		upstream, status, waf_enabled, COALESCE(description,''),
		COALESCE(rps, 0), COALESCE(blocked_rate, 0), COALESCE(instance_label, ''),
		metrics_updated_at, created_at, updated_at
		FROM sites %s ORDER BY id DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, p.PageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.Name, &s.Domain, &s.ListenPort, &s.SSLEnabled, &s.SSLCert, &s.SSLKey,
			&s.Upstream, &s.Status, &s.WAFEnabled, &s.Description,
			&s.RPS, &s.BlockedRate, &s.InstanceLabel, &s.MetricsUpdatedAt,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan site: %w", err)
		}
		sites = append(sites, s)
	}
	return sites, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Site, error) {
	// scope 内层校验 —— 不属于当前用户 project 的 site 与"不存在"等价。
	query := `SELECT id, name, domain, listen_port, ssl_enabled, COALESCE(ssl_cert,''), COALESCE(ssl_key,''),
		upstream, status, waf_enabled, COALESCE(description,''),
		COALESCE(rps, 0), COALESCE(blocked_rate, 0), COALESCE(instance_label, ''),
		metrics_updated_at, created_at, updated_at
		FROM sites WHERE id = $1`
	args := []any{id}
	if scope := identity.ScopeFromContext(ctx); scope != nil && !scope.IsAdmin {
		if len(scope.IDs) == 0 {
			return nil, fmt.Errorf("get site: not found in scope")
		}
		query += " AND project_id = ANY($2::bigint[])"
		args = append(args, scope.IDs)
	}
	var s Site
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.Name, &s.Domain, &s.ListenPort, &s.SSLEnabled, &s.SSLCert, &s.SSLKey,
		&s.Upstream, &s.Status, &s.WAFEnabled, &s.Description,
		&s.RPS, &s.BlockedRate, &s.InstanceLabel, &s.MetricsUpdatedAt,
		&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get site: %w", err)
	}
	return &s, nil
}

func (r *Repository) Create(ctx context.Context, req CreateRequest) (*Site, error) {
	upstream := req.Upstream
	if upstream == nil {
		upstream = []byte("[]")
	}
	listenPort := req.ListenPort
	if listenPort == 0 {
		listenPort = 80
	}
	wafEnabled := true
	if req.WAFEnabled != nil {
		wafEnabled = *req.WAFEnabled
	}
	// 新建资源归属：非 admin 落到用户第一个 project；admin / 无 scope 兜底到 default(1)。
	projectID := identity.ScopeFromContext(ctx).DefaultProjectID()

	var s Site
	err := r.pool.QueryRow(ctx, `INSERT INTO sites (name, domain, listen_port, ssl_enabled, ssl_cert, ssl_key, upstream, waf_enabled, description, project_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, name, domain, listen_port, ssl_enabled, COALESCE(ssl_cert,''), COALESCE(ssl_key,''),
		upstream, status, waf_enabled, COALESCE(description,''),
		COALESCE(rps, 0), COALESCE(blocked_rate, 0), COALESCE(instance_label, ''),
		metrics_updated_at, created_at, updated_at`,
		req.Name, req.Domain, listenPort, req.SSLEnabled, req.SSLCert, req.SSLKey, upstream, wafEnabled, req.Description, projectID).Scan(
		&s.ID, &s.Name, &s.Domain, &s.ListenPort, &s.SSLEnabled, &s.SSLCert, &s.SSLKey,
		&s.Upstream, &s.Status, &s.WAFEnabled, &s.Description,
		&s.RPS, &s.BlockedRate, &s.InstanceLabel, &s.MetricsUpdatedAt,
		&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create site: %w", err)
	}
	return &s, nil
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (*Site, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Domain != nil {
		sets = append(sets, fmt.Sprintf("domain = $%d", argIdx))
		args = append(args, *req.Domain)
		argIdx++
	}
	if req.ListenPort != nil {
		sets = append(sets, fmt.Sprintf("listen_port = $%d", argIdx))
		args = append(args, *req.ListenPort)
		argIdx++
	}
	if req.SSLEnabled != nil {
		sets = append(sets, fmt.Sprintf("ssl_enabled = $%d", argIdx))
		args = append(args, *req.SSLEnabled)
		argIdx++
	}
	if req.SSLCert != nil {
		sets = append(sets, fmt.Sprintf("ssl_cert = $%d", argIdx))
		args = append(args, *req.SSLCert)
		argIdx++
	}
	if req.SSLKey != nil {
		sets = append(sets, fmt.Sprintf("ssl_key = $%d", argIdx))
		args = append(args, *req.SSLKey)
		argIdx++
	}
	if req.Upstream != nil {
		sets = append(sets, fmt.Sprintf("upstream = $%d", argIdx))
		args = append(args, *req.Upstream)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.WAFEnabled != nil {
		sets = append(sets, fmt.Sprintf("waf_enabled = $%d", argIdx))
		args = append(args, *req.WAFEnabled)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE sites SET %s WHERE id = $%d
		RETURNING id, name, domain, listen_port, ssl_enabled, COALESCE(ssl_cert,''), COALESCE(ssl_key,''),
		upstream, status, waf_enabled, COALESCE(description,''),
		COALESCE(rps, 0), COALESCE(blocked_rate, 0), COALESCE(instance_label, ''),
		metrics_updated_at, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var s Site
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.Name, &s.Domain, &s.ListenPort, &s.SSLEnabled, &s.SSLCert, &s.SSLKey,
		&s.Upstream, &s.Status, &s.WAFEnabled, &s.Description,
			&s.RPS, &s.BlockedRate, &s.InstanceLabel, &s.MetricsUpdatedAt,
			&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update site: %w", err)
	}
	return &s, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM sites WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete site: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("site not found")
	}
	return nil
}

// --- Protect Assoc ---

func (r *Repository) ListDevices(ctx context.Context, siteID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, "SELECT device_id FROM protect_assoc WHERE site_id = $1", siteID)
	if err != nil {
		return nil, fmt.Errorf("list site devices: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan device id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repository) BindDevice(ctx context.Context, siteID, deviceID int64) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO protect_assoc (site_id, device_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		siteID, deviceID)
	if err != nil {
		return fmt.Errorf("bind device: %w", err)
	}
	return nil
}

func (r *Repository) UnbindDevice(ctx context.Context, siteID, deviceID int64) error {
	_, err := r.pool.Exec(ctx,
		"DELETE FROM protect_assoc WHERE site_id = $1 AND device_id = $2",
		siteID, deviceID)
	if err != nil {
		return fmt.Errorf("unbind device: %w", err)
	}
	return nil
}

// UpdateMetrics 由监控管道调用：把站点 rps/blocked_rate/instance_label 直接写
// 进 sites 表，避免 list 接口跨表 join。NW · 03 UI 列表直接读这些缓存列。
func (r *Repository) UpdateMetrics(ctx context.Context, id int64, req UpdateMetricsRequest) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sites
		   SET rps = $1,
		       blocked_rate = $2,
		       instance_label = COALESCE(NULLIF($3,''), instance_label),
		       metrics_updated_at = NOW()
		 WHERE id = $4`,
		req.RPS, req.BlockedRate, req.InstanceLabel, id)
	if err != nil {
		return fmt.Errorf("update site metrics: %w", err)
	}
	return nil
}

// EnsureSchema 兜底跑 000010 + 000025 的 ALTER —— 部署 race / migration baseline
// 跳过时让 SELECT 不报缺列。
func (r *Repository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS rps           DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS blocked_rate  DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS instance_label VARCHAR(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS metrics_updated_at TIMESTAMPTZ`,
		// 多租户 scope —— migration 000025 加列；这里幂等兜底。
		// 现有行落到 project_id=1（projects 表的 default 项目，see 000005）。
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS project_id BIGINT NOT NULL DEFAULT 1`,
		`CREATE INDEX IF NOT EXISTS idx_sites_project ON sites(project_id)`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure site schema (%s): %w", s, err)
		}
	}
	return nil
}
