package security

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

// --- Auth Hosts ---

func (r *Repository) ListAuthHosts(ctx context.Context) ([]AuthHost, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, host, COALESCE(description,''), created_at, updated_at
		FROM auth_hosts ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list auth hosts: %w", err)
	}
	defer rows.Close()

	var items []AuthHost
	for rows.Next() {
		var a AuthHost
		if err := rows.Scan(&a.ID, &a.Host, &a.Description, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan auth host: %w", err)
		}
		items = append(items, a)
	}
	return items, nil
}

func (r *Repository) CreateAuthHost(ctx context.Context, req CreateAuthHostRequest) (*AuthHost, error) {
	var a AuthHost
	err := r.pool.QueryRow(ctx, `INSERT INTO auth_hosts (host, description)
		VALUES ($1, $2)
		RETURNING id, host, COALESCE(description,''), created_at, updated_at`,
		req.Host, req.Description).Scan(
		&a.ID, &a.Host, &a.Description, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create auth host: %w", err)
	}
	return &a, nil
}

func (r *Repository) GetAuthHost(ctx context.Context, id int64) (*AuthHost, error) {
	var a AuthHost
	err := r.pool.QueryRow(ctx, `SELECT id, host, COALESCE(description,''), created_at, updated_at
		FROM auth_hosts WHERE id = $1`, id).Scan(
		&a.ID, &a.Host, &a.Description, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get auth host: %w", err)
	}
	return &a, nil
}

func (r *Repository) UpdateAuthHost(ctx context.Context, id int64, req UpdateAuthHostRequest) (*AuthHost, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Host != nil {
		sets = append(sets, fmt.Sprintf("host = $%d", argIdx))
		args = append(args, *req.Host)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE auth_hosts SET %s WHERE id = $%d
		RETURNING id, host, COALESCE(description,''), created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var a AuthHost
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&a.ID, &a.Host, &a.Description, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update auth host: %w", err)
	}
	return &a, nil
}

func (r *Repository) DeleteAuthHost(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM auth_hosts WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete auth host: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("auth host not found")
	}
	return nil
}

// --- Auth Host Config ---

func (r *Repository) GetAuthHostConfig(ctx context.Context) (*AuthHostConfig, error) {
	var c AuthHostConfig
	err := r.pool.QueryRow(ctx, `SELECT id, enabled, max_attempts, lockout_duration, updated_at
		FROM auth_host_config LIMIT 1`).Scan(
		&c.ID, &c.Enabled, &c.MaxAttempts, &c.LockoutDuration, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get auth host config: %w", err)
	}
	return &c, nil
}

func (r *Repository) UpsertAuthHostConfig(ctx context.Context, req UpdateAuthHostConfigRequest) (*AuthHostConfig, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Enabled != nil {
		sets = append(sets, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *req.Enabled)
		argIdx++
	}
	if req.MaxAttempts != nil {
		sets = append(sets, fmt.Sprintf("max_attempts = $%d", argIdx))
		args = append(args, *req.MaxAttempts)
		argIdx++
	}
	if req.LockoutDuration != nil {
		sets = append(sets, fmt.Sprintf("lockout_duration = $%d", argIdx))
		args = append(args, *req.LockoutDuration)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")

	_, err := r.pool.Exec(ctx, `INSERT INTO auth_host_config (enabled, max_attempts, lockout_duration)
		VALUES (true, 5, 1800) ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return nil, fmt.Errorf("ensure config row: %w", err)
	}

	updateQuery := fmt.Sprintf(`UPDATE auth_host_config SET %s WHERE id = 1
		RETURNING id, enabled, max_attempts, lockout_duration, updated_at`,
		strings.Join(sets, ", "))

	var c AuthHostConfig
	err = r.pool.QueryRow(ctx, updateQuery, args...).Scan(
		&c.ID, &c.Enabled, &c.MaxAttempts, &c.LockoutDuration, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert auth host config: %w", err)
	}
	return &c, nil
}

// --- Password Status ---

func (r *Repository) GetPasswordStatus(ctx context.Context, userID int64) (*PasswordStatus, error) {
	var ps PasswordStatus
	err := r.pool.QueryRow(ctx, `SELECT u.id, u.username,
		CASE WHEN u.password_updated_at IS NULL OR u.password_updated_at < NOW() - INTERVAL '90 days' THEN true ELSE false END as expired,
		COALESCE(to_char(u.password_updated_at, 'YYYY-MM-DD HH24:MI:SS'), 'never') as last_reset
		FROM users u WHERE u.id = $1`, userID).Scan(
		&ps.UserID, &ps.Username, &ps.Expired, &ps.LastReset)
	if err != nil {
		return nil, fmt.Errorf("get password status: %w", err)
	}
	return &ps, nil
}
