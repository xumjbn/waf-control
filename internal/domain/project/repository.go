package project

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context) ([]Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, domain_id, parent_id, is_domain, enabled, created_at, updated_at
		FROM projects ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	out := make([]Project, 0)
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.DomainID, &p.ParentID, &p.IsDomain, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (*Project, error) {
	var p Project
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, domain_id, parent_id, is_domain, enabled, created_at, updated_at
		FROM projects WHERE id = $1`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.DomainID, &p.ParentID, &p.IsDomain, &p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, p *Project) error {
	if p.DomainID == "" {
		p.DomainID = "default"
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO projects (name, description, domain_id, parent_id, is_domain, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		p.Name, p.Description, p.DomainID, p.ParentID, p.IsDomain, p.Enabled,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, p *Project) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE projects
		SET name = $1, description = $2, parent_id = $3, is_domain = $4, enabled = $5, updated_at = NOW()
		WHERE id = $6`,
		p.Name, p.Description, p.ParentID, p.IsDomain, p.Enabled, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

// === project-user-role assignments ===

func (r *Repository) Assign(ctx context.Context, projectID, userID, roleID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_user_roles (project_id, user_id, role_id)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		projectID, userID, roleID,
	)
	if err != nil {
		return fmt.Errorf("assign project user role: %w", err)
	}
	return nil
}

func (r *Repository) Revoke(ctx context.Context, projectID, userID, roleID int64) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM project_user_roles
		WHERE project_id = $1 AND user_id = $2 AND role_id = $3`,
		projectID, userID, roleID,
	)
	if err != nil {
		return fmt.Errorf("revoke project user role: %w", err)
	}
	return nil
}

// ListProjectUserIDs 返回某 project 下出现过的不重复 user_id 列表。
func (r *Repository) ListProjectUserIDs(ctx context.Context, projectID int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT user_id FROM project_user_roles WHERE project_id = $1 ORDER BY user_id`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list project users: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		ids = append(ids, uid)
	}
	return ids, nil
}
