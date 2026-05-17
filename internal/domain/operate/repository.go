package operate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const defaultPageSize = 20

func (r *Repository) List(ctx context.Context, filter OperationLogFilter) ([]OperationLog, int, error) {
	var conds []string
	var args []interface{}
	argIdx := 1

	if filter.Username != "" {
		conds = append(conds, fmt.Sprintf("username ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Username+"%")
		argIdx++
	}
	if filter.Method != "" {
		conds = append(conds, fmt.Sprintf("method = $%d", argIdx))
		args = append(args, filter.Method)
		argIdx++
	}
	if filter.Path != "" {
		conds = append(conds, fmt.Sprintf("path ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Path+"%")
		argIdx++
	}
	if filter.MinCode > 0 {
		conds = append(conds, fmt.Sprintf("status_code >= $%d", argIdx))
		args = append(args, filter.MinCode)
		argIdx++
	}
	if filter.MaxCode > 0 {
		conds = append(conds, fmt.Sprintf("status_code <= $%d", argIdx))
		args = append(args, filter.MaxCode)
		argIdx++
	}
	if filter.StartAt != "" {
		t, err := time.Parse(time.RFC3339, filter.StartAt)
		if err == nil {
			conds = append(conds, fmt.Sprintf("created_at >= $%d", argIdx))
			args = append(args, t)
			argIdx++
		}
	}
	if filter.EndAt != "" {
		t, err := time.Parse(time.RFC3339, filter.EndAt)
		if err == nil {
			conds = append(conds, fmt.Sprintf("created_at <= $%d", argIdx))
			args = append(args, t)
			argIdx++
		}
	}

	whereClause := ""
	if len(conds) > 0 {
		whereClause = " WHERE " + strings.Join(conds, " AND ")
	}

	// Count
	var total int
	countQuery := "SELECT COUNT(*) FROM operation_logs" + whereClause
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count operation logs: %w", err)
	}

	// Paginate
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	offset := (page - 1) * pageSize

	query := fmt.Sprintf(`SELECT id, user_id, username, method, path, status_code, duration_ms, client_ip, request_body, response_body, created_at
		FROM operation_logs%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list operation logs: %w", err)
	}
	defer rows.Close()

	var items []OperationLog
	for rows.Next() {
		var o OperationLog
		if err := rows.Scan(&o.ID, &o.UserID, &o.Username, &o.Method, &o.Path,
			&o.StatusCode, &o.DurationMs, &o.ClientIP, &o.RequestBody, &o.ResponseBody, &o.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan operation log: %w", err)
		}
		items = append(items, o)
	}
	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (*OperationLog, error) {
	var o OperationLog
	err := r.pool.QueryRow(ctx, `SELECT id, user_id, username, method, path, status_code, duration_ms, client_ip, request_body, response_body, created_at
		FROM operation_logs WHERE id = $1`, id).Scan(
		&o.ID, &o.UserID, &o.Username, &o.Method, &o.Path,
		&o.StatusCode, &o.DurationMs, &o.ClientIP, &o.RequestBody, &o.ResponseBody, &o.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get operation log %d: %w", id, err)
	}
	return &o, nil
}
