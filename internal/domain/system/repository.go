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

// EnsureSchema 启动时幂等补齐 settings.description + licenses 完整字段（migration 000015）。
func (r *Repository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE system_settings ADD COLUMN IF NOT EXISTS description VARCHAR(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS customer      VARCHAR(128) NOT NULL DEFAULT ''`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS contact_email VARCHAR(128) NOT NULL DEFAULT ''`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS grace_until   TIMESTAMPTZ`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS issued_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
		`ALTER TABLE licenses ADD COLUMN IF NOT EXISTS edition       VARCHAR(32) NOT NULL DEFAULT 'community'`,
		`CREATE INDEX IF NOT EXISTS idx_licenses_is_active ON licenses(is_active)`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure system schema (%q): %w", s, err)
		}
	}
	// 默认设置（幂等）
	defaults := [][3]string{
		{"platform.name", "OpenWAF", "平台名称"},
		{"platform.timezone", "Asia/Shanghai", "默认时区"},
		{"platform.lang", "zh-CN", "默认语言"},
		{"alert.retention_days", "90", "告警保留天数"},
		{"log.retention_days", "30", "日志保留天数"},
		{"security.session_timeout", "3600", "会话超时秒"},
	}
	for _, d := range defaults {
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO system_settings (key, value, category, description)
			 VALUES ($1, $2, 'basic', $3)
			 ON CONFLICT (key) DO NOTHING`, d[0], d[1], d[2]); err != nil {
			return fmt.Errorf("seed setting %s: %w", d[0], err)
		}
	}
	return nil
}

// --- Settings ---

const settingCols = `id, key, value, COALESCE(category,''), COALESCE(description,''), created_at, updated_at`

func scanSetting(rs interface {
	Scan(...interface{}) error
}, s *Setting) error {
	return rs.Scan(&s.ID, &s.Key, &s.Value, &s.Category, &s.Description, &s.CreatedAt, &s.UpdatedAt)
}

func (r *Repository) ListSettings(ctx context.Context, category string) ([]Setting, error) {
	query := `SELECT ` + settingCols + ` FROM system_settings`
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
		if err := scanSetting(rows, &s); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings = append(settings, s)
	}
	return settings, nil
}

func (r *Repository) UpsertSetting(ctx context.Context, req UpsertSettingRequest) (*Setting, error) {
	var s Setting
	q := `INSERT INTO system_settings (key, value, category, description)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO UPDATE SET
		  value = EXCLUDED.value,
		  category = EXCLUDED.category,
		  description = EXCLUDED.description,
		  updated_at = NOW()
		RETURNING ` + settingCols
	if err := scanSetting(r.pool.QueryRow(ctx, q, req.Key, req.Value, req.Category, req.Description), &s); err != nil {
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

const licenseCols = `id, license_key, COALESCE(product_name,''), COALESCE(edition,'community'),
	COALESCE(customer,''), COALESCE(contact_email,''), max_nodes,
	COALESCE(issued_at, created_at), expires_at, grace_until, is_active, created_at`

func scanLicense(rs interface {
	Scan(...interface{}) error
}, l *License) error {
	return rs.Scan(&l.ID, &l.LicenseKey, &l.ProductName, &l.Edition,
		&l.Customer, &l.ContactEmail, &l.MaxNodes,
		&l.IssuedAt, &l.ExpiresAt, &l.GraceUntil, &l.IsActive, &l.CreatedAt)
}

func (r *Repository) ListLicenses(ctx context.Context) ([]License, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+licenseCols+` FROM licenses ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	var licenses []License
	for rows.Next() {
		var l License
		if err := scanLicense(rows, &l); err != nil {
			return nil, fmt.Errorf("scan license: %w", err)
		}
		licenses = append(licenses, l)
	}
	return licenses, nil
}

// CurrentLicense 返回当前激活的 license（is_active = true）。NW · 09 系统页"许可证详情"卡使用。
func (r *Repository) CurrentLicense(ctx context.Context) (*License, error) {
	var l License
	if err := scanLicense(r.pool.QueryRow(ctx,
		`SELECT `+licenseCols+` FROM licenses WHERE is_active = TRUE ORDER BY id DESC LIMIT 1`), &l); err != nil {
		return nil, fmt.Errorf("current license: %w", err)
	}
	return &l, nil
}

func (r *Repository) CreateLicense(ctx context.Context, req CreateLicenseRequest) (*License, error) {
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid expires_at format: %w", err)
	}
	var graceUntil *time.Time
	if req.GraceUntil != "" {
		t, err := time.Parse(time.RFC3339, req.GraceUntil)
		if err != nil {
			return nil, fmt.Errorf("invalid grace_until format: %w", err)
		}
		graceUntil = &t
	}
	edition := req.Edition
	if edition == "" {
		edition = "community"
	}

	var l License
	q := `INSERT INTO licenses (license_key, product_name, edition, customer, contact_email,
		max_nodes, expires_at, grace_until, issued_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, NOW())
		RETURNING ` + licenseCols
	if err := scanLicense(r.pool.QueryRow(ctx, q,
		req.LicenseKey, req.ProductName, edition, req.Customer, req.ContactEmail,
		req.MaxNodes, expiresAt, graceUntil), &l); err != nil {
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
