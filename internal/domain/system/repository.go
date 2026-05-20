package system

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Settings ---

func (r *Repository) ListSettings(ctx context.Context, category string) ([]Setting, error) {
	query := `SELECT id, key, value, COALESCE(category,''), created_at, updated_at FROM system_settings`
	var args []interface{}
	if category != "" {
		query += " WHERE category = $1"
		args = append(args, category)
	}
	query += " ORDER BY category, key"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.Category, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings = append(settings, s)
	}
	return settings, nil
}

func (r *Repository) UpsertSetting(ctx context.Context, req UpsertSettingRequest) (*Setting, error) {
	var s Setting
	err := r.pool.QueryRow(ctx, `INSERT INTO system_settings (key, value, category)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, category = EXCLUDED.category, updated_at = NOW()
		RETURNING id, key, value, COALESCE(category,''), created_at, updated_at`,
		req.Key, req.Value, req.Category).Scan(
		&s.ID, &s.Key, &s.Value, &s.Category, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert setting: %w", err)
	}
	return &s, nil
}

func (r *Repository) DeleteSetting(ctx context.Context, key string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM system_settings WHERE key = $1", key)
	if err != nil {
		return fmt.Errorf("delete setting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("setting not found")
	}
	return nil
}

// --- Licenses ---

func (r *Repository) ListLicenses(ctx context.Context) ([]License, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, license_key, COALESCE(product_name,''), max_nodes, expires_at, is_active, created_at
		FROM licenses ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	var licenses []License
	for rows.Next() {
		var l License
		if err := rows.Scan(&l.ID, &l.LicenseKey, &l.ProductName, &l.MaxNodes, &l.ExpiresAt, &l.IsActive, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan license: %w", err)
		}
		licenses = append(licenses, l)
	}
	return licenses, nil
}

func (r *Repository) CreateLicense(ctx context.Context, req CreateLicenseRequest) (*License, error) {
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid expires_at format: %w", err)
	}

	var l License
	err = r.pool.QueryRow(ctx, `INSERT INTO licenses (license_key, product_name, max_nodes, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, license_key, COALESCE(product_name,''), max_nodes, expires_at, is_active, created_at`,
		req.LicenseKey, req.ProductName, req.MaxNodes, expiresAt).Scan(
		&l.ID, &l.LicenseKey, &l.ProductName, &l.MaxNodes, &l.ExpiresAt, &l.IsActive, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create license: %w", err)
	}
	return &l, nil
}

func (r *Repository) ActivateLicense(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "UPDATE licenses SET is_active = FALSE WHERE is_active = TRUE")
	if err != nil {
		return fmt.Errorf("deactivate licenses: %w", err)
	}
	tag, err := r.pool.Exec(ctx, "UPDATE licenses SET is_active = TRUE WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("activate license: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}

func (r *Repository) DeleteLicense(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM licenses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete license: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}

// --- Upgrades ---

// EnsureUpgradeSchema 启动时幂等补齐 system_upgrades 表的 NW · 11 字段（migration 000016）。
func (r *Repository) EnsureUpgradeSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS type            VARCHAR(16)  NOT NULL DEFAULT 'patch'`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS channel         VARCHAR(16)  NOT NULL DEFAULT 'stable'`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS notes           TEXT         NOT NULL DEFAULT ''`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS changes_summary TEXT         NOT NULL DEFAULT ''`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS checksum        VARCHAR(128) NOT NULL DEFAULT ''`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS download_url    VARCHAR(512) NOT NULL DEFAULT ''`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS is_current      BOOLEAN      NOT NULL DEFAULT FALSE`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS is_latest       BOOLEAN      NOT NULL DEFAULT FALSE`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS released_at     TIMESTAMPTZ`,
		`ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS applied_at      TIMESTAMPTZ`,
		`CREATE INDEX IF NOT EXISTS idx_system_upgrades_is_current ON system_upgrades(is_current)`,
		`CREATE INDEX IF NOT EXISTS idx_system_upgrades_is_latest  ON system_upgrades(is_latest)`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure system_upgrades schema (%q): %w", s, err)
		}
	}
	return nil
}

const upgradeCols = `id, version, COALESCE(type,'patch'), COALESCE(channel,'stable'),
	file_name, file_size, COALESCE(checksum,''), COALESCE(download_url,''),
	COALESCE(notes,''), COALESCE(changes_summary,''),
	status, COALESCE(is_current,false), COALESCE(is_latest,false),
	released_at, applied_at, created_at, updated_at`

func scanUpgrade(rs interface {
	Scan(...interface{}) error
}, u *Upgrade) error {
	return rs.Scan(&u.ID, &u.Version, &u.Type, &u.Channel,
		&u.FileName, &u.FileSize, &u.Checksum, &u.DownloadURL,
		&u.Notes, &u.ChangesSummary,
		&u.Status, &u.IsCurrent, &u.IsLatest,
		&u.ReleasedAt, &u.AppliedAt, &u.CreatedAt, &u.UpdatedAt)
}

func (r *Repository) ListUpgrades(ctx context.Context) ([]Upgrade, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+upgradeCols+` FROM system_upgrades ORDER BY COALESCE(released_at, created_at) DESC`)
	if err != nil {
		return nil, fmt.Errorf("list upgrades: %w", err)
	}
	defer rows.Close()

	var upgrades []Upgrade
	for rows.Next() {
		var u Upgrade
		if err := scanUpgrade(rows, &u); err != nil {
			return nil, fmt.Errorf("scan upgrade: %w", err)
		}
		upgrades = append(upgrades, u)
	}
	return upgrades, nil
}

// CurrentUpgrade 返回 is_current = true 的当前已安装版本；不存在返回 nil, nil。
func (r *Repository) CurrentUpgrade(ctx context.Context) (*Upgrade, error) {
	var u Upgrade
	err := scanUpgrade(r.pool.QueryRow(ctx,
		`SELECT `+upgradeCols+` FROM system_upgrades WHERE is_current = TRUE ORDER BY id DESC LIMIT 1`), &u)
	if err != nil {
		return nil, nil
	}
	return &u, nil
}

// LatestUpgrade 返回 is_latest = true 的最新可用版本。
func (r *Repository) LatestUpgrade(ctx context.Context) (*Upgrade, error) {
	var u Upgrade
	err := scanUpgrade(r.pool.QueryRow(ctx,
		`SELECT `+upgradeCols+` FROM system_upgrades WHERE is_latest = TRUE ORDER BY id DESC LIMIT 1`), &u)
	if err != nil {
		return nil, nil
	}
	return &u, nil
}

func (r *Repository) CreateUpgrade(ctx context.Context, req CreateUpgradeRequest) (*Upgrade, error) {
	typ := req.Type
	if typ == "" {
		typ = "patch"
	}
	channel := req.Channel
	if channel == "" {
		channel = "stable"
	}
	var releasedAt *time.Time
	if req.ReleasedAt != "" {
		t, err := time.Parse(time.RFC3339, req.ReleasedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid released_at: %w", err)
		}
		releasedAt = &t
	}
	var u Upgrade
	q := `INSERT INTO system_upgrades (
		version, type, channel, file_name, file_size, checksum, download_url,
		notes, changes_summary, is_latest, released_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING ` + upgradeCols
	if err := scanUpgrade(r.pool.QueryRow(ctx, q,
		req.Version, typ, channel, req.FileName, req.FileSize, req.Checksum, req.DownloadURL,
		req.Notes, req.ChangesSummary, req.IsLatest, releasedAt), &u); err != nil {
		return nil, fmt.Errorf("create upgrade: %w", err)
	}
	if req.IsLatest {
		// 同 type 只允许一个 is_latest = true
		_, _ = r.pool.Exec(ctx,
			`UPDATE system_upgrades SET is_latest = FALSE WHERE id <> $1 AND is_latest = TRUE`, u.ID)
	}
	return &u, nil
}

func (r *Repository) TriggerUpgrade(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE system_upgrades SET status = 'in_progress', updated_at = NOW()
		 WHERE id = $1 AND status IN ('pending','failed')`, id)
	if err != nil {
		return fmt.Errorf("trigger upgrade: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("upgrade not found or already running")
	}
	return nil
}

// MarkApplied 模拟 apply 完成 —— 将状态切到 completed、is_current 置 true、其他记录的 is_current 清空。
// 真实环境由 deploy 子模块或 worker 在升级流程结束时调用。
func (r *Repository) MarkApplied(ctx context.Context, id int64) error {
	if _, err := r.pool.Exec(ctx, `UPDATE system_upgrades SET is_current = FALSE WHERE is_current = TRUE`); err != nil {
		return fmt.Errorf("clear current: %w", err)
	}
	tag, err := r.pool.Exec(ctx, `UPDATE system_upgrades SET status = 'completed',
		is_current = TRUE, applied_at = NOW(), updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark applied: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("upgrade not found")
	}
	return nil
}

func (r *Repository) DeleteUpgrade(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM system_upgrades WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete upgrade: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("upgrade not found")
	}
	return nil
}
