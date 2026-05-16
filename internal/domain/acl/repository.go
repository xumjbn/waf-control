package acl

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

func (r *Repository) List(ctx context.Context) ([]Rule, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, COALESCE(description,''), direction, action,
		COALESCE(protocol,''), COALESCE(src_ip,''), src_port, COALESCE(dst_ip,''), dst_port,
		priority, is_enabled, created_at, updated_at
		FROM acl_rules ORDER BY priority, id`)
	if err != nil {
		return nil, fmt.Errorf("list acl rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var ru Rule
		if err := rows.Scan(&ru.ID, &ru.Name, &ru.Description, &ru.Direction, &ru.Action,
			&ru.Protocol, &ru.SrcIP, &ru.SrcPort, &ru.DstIP, &ru.DstPort,
			&ru.Priority, &ru.IsEnabled, &ru.CreatedAt, &ru.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan acl rule: %w", err)
		}
		rules = append(rules, ru)
	}
	return rules, nil
}

func (r *Repository) Create(ctx context.Context, req CreateRuleRequest) (*Rule, error) {
	direction := req.Direction
	if direction == "" {
		direction = "inbound"
	}
	action := req.Action
	if action == "" {
		action = "deny"
	}
	priority := req.Priority
	if priority <= 0 {
		priority = 100
	}

	var ru Rule
	err := r.pool.QueryRow(ctx, `INSERT INTO acl_rules (name, description, direction, action, protocol, src_ip, src_port, dst_ip, dst_port, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, name, COALESCE(description,''), direction, action,
		COALESCE(protocol,''), COALESCE(src_ip,''), src_port, COALESCE(dst_ip,''), dst_port,
		priority, is_enabled, created_at, updated_at`,
		req.Name, req.Description, direction, action, req.Protocol,
		req.SrcIP, req.SrcPort, req.DstIP, req.DstPort, priority).Scan(
		&ru.ID, &ru.Name, &ru.Description, &ru.Direction, &ru.Action,
		&ru.Protocol, &ru.SrcIP, &ru.SrcPort, &ru.DstIP, &ru.DstPort,
		&ru.Priority, &ru.IsEnabled, &ru.CreatedAt, &ru.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create acl rule: %w", err)
	}
	return &ru, nil
}

func (r *Repository) Update(ctx context.Context, id int64, req UpdateRuleRequest) (*Rule, error) {
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
	if req.Direction != nil {
		sets = append(sets, fmt.Sprintf("direction = $%d", argIdx))
		args = append(args, *req.Direction)
		argIdx++
	}
	if req.Action != nil {
		sets = append(sets, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, *req.Action)
		argIdx++
	}
	if req.Protocol != nil {
		sets = append(sets, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, *req.Protocol)
		argIdx++
	}
	if req.SrcIP != nil {
		sets = append(sets, fmt.Sprintf("src_ip = $%d", argIdx))
		args = append(args, *req.SrcIP)
		argIdx++
	}
	if req.SrcPort != nil {
		sets = append(sets, fmt.Sprintf("src_port = $%d", argIdx))
		args = append(args, *req.SrcPort)
		argIdx++
	}
	if req.DstIP != nil {
		sets = append(sets, fmt.Sprintf("dst_ip = $%d", argIdx))
		args = append(args, *req.DstIP)
		argIdx++
	}
	if req.DstPort != nil {
		sets = append(sets, fmt.Sprintf("dst_port = $%d", argIdx))
		args = append(args, *req.DstPort)
		argIdx++
	}
	if req.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.IsEnabled != nil {
		sets = append(sets, fmt.Sprintf("is_enabled = $%d", argIdx))
		args = append(args, *req.IsEnabled)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE acl_rules SET %s WHERE id = $%d
		RETURNING id, name, COALESCE(description,''), direction, action,
		COALESCE(protocol,''), COALESCE(src_ip,''), src_port, COALESCE(dst_ip,''), dst_port,
		priority, is_enabled, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var ru Rule
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&ru.ID, &ru.Name, &ru.Description, &ru.Direction, &ru.Action,
		&ru.Protocol, &ru.SrcIP, &ru.SrcPort, &ru.DstIP, &ru.DstPort,
		&ru.Priority, &ru.IsEnabled, &ru.CreatedAt, &ru.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update acl rule: %w", err)
	}
	return &ru, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM acl_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete acl rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("acl rule not found")
	}
	return nil
}
