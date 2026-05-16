package device

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

func (r *Repository) List(ctx context.Context, p ListParams) ([]Device, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if p.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, p.Status)
		argIdx++
	}
	if p.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR serial_no ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+p.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM devices " + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count devices: %w", err)
	}

	offset := (p.Page - 1) * p.PageSize
	query := fmt.Sprintf(`SELECT id, name, COALESCE(serial_no,''), COALESCE(model,''), status,
		COALESCE(ip_address,''), COALESCE(version,''), COALESCE(description,''), created_at, updated_at
		FROM devices %s ORDER BY id DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, p.PageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.SerialNo, &d.Model, &d.Status,
			&d.IPAddress, &d.Version, &d.Description, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Device, error) {
	var d Device
	err := r.pool.QueryRow(ctx, `SELECT id, name, COALESCE(serial_no,''), COALESCE(model,''), status,
		COALESCE(ip_address,''), COALESCE(version,''), COALESCE(description,''), created_at, updated_at
		FROM devices WHERE id = $1`, id).Scan(
		&d.ID, &d.Name, &d.SerialNo, &d.Model, &d.Status,
		&d.IPAddress, &d.Version, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}
	return &d, nil
}

func (r *Repository) Create(ctx context.Context, req CreateRequest) (*Device, error) {
	var d Device
	err := r.pool.QueryRow(ctx, `INSERT INTO devices (name, serial_no, model, ip_address, version, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, COALESCE(serial_no,''), COALESCE(model,''), status,
		COALESCE(ip_address,''), COALESCE(version,''), COALESCE(description,''), created_at, updated_at`,
		req.Name, req.SerialNo, req.Model, req.IPAddress, req.Version, req.Description).Scan(
		&d.ID, &d.Name, &d.SerialNo, &d.Model, &d.Status,
		&d.IPAddress, &d.Version, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}
	return &d, nil
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRequest) (*Device, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.SerialNo != nil {
		sets = append(sets, fmt.Sprintf("serial_no = $%d", argIdx))
		args = append(args, *req.SerialNo)
		argIdx++
	}
	if req.Model != nil {
		sets = append(sets, fmt.Sprintf("model = $%d", argIdx))
		args = append(args, *req.Model)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if req.IPAddress != nil {
		sets = append(sets, fmt.Sprintf("ip_address = $%d", argIdx))
		args = append(args, *req.IPAddress)
		argIdx++
	}
	if req.Version != nil {
		sets = append(sets, fmt.Sprintf("version = $%d", argIdx))
		args = append(args, *req.Version)
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
	query := fmt.Sprintf(`UPDATE devices SET %s WHERE id = $%d
		RETURNING id, name, COALESCE(serial_no,''), COALESCE(model,''), status,
		COALESCE(ip_address,''), COALESCE(version,''), COALESCE(description,''), created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var d Device
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&d.ID, &d.Name, &d.SerialNo, &d.Model, &d.Status,
		&d.IPAddress, &d.Version, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}
	return &d, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM devices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}
