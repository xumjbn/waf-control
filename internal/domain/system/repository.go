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

func (r *Repository) ListUpgrades(ctx context.Context) ([]Upgrade, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, version, file_name, file_size, status, created_at, updated_at
		FROM system_upgrades ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list upgrades: %w", err)
	}
	defer rows.Close()

	var upgrades []Upgrade
	for rows.Next() {
		var u Upgrade
		if err := rows.Scan(&u.ID, &u.Version, &u.FileName, &u.FileSize, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan upgrade: %w", err)
		}
		upgrades = append(upgrades, u)
	}
	return upgrades, nil
}

func (r *Repository) CreateUpgrade(ctx context.Context, req CreateUpgradeRequest) (*Upgrade, error) {
	var u Upgrade
	err := r.pool.QueryRow(ctx, `INSERT INTO system_upgrades (version, file_name, file_size)
		VALUES ($1, $2, $3)
		RETURNING id, version, file_name, file_size, status, created_at, updated_at`,
		req.Version, req.FileName, req.FileSize).Scan(
		&u.ID, &u.Version, &u.FileName, &u.FileSize, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create upgrade: %w", err)
	}
	return &u, nil
}

func (r *Repository) TriggerUpgrade(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "UPDATE system_upgrades SET status = 'in_progress', updated_at = NOW() WHERE id = $1 AND status = 'pending'", id)
	if err != nil {
		return fmt.Errorf("trigger upgrade: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("upgrade not found or not in pending status")
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
