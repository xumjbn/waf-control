package policy

import (
	"context"
	"encoding/json"
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

// --- Categories ---

func (r *Repository) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, COALESCE(description,''), sort_order, created_at
		FROM policy_categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, nil
}

func (r *Repository) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	var c Category
	err := r.pool.QueryRow(ctx, `INSERT INTO policy_categories (name, description, sort_order)
		VALUES ($1, $2, $3)
		RETURNING id, name, COALESCE(description,''), sort_order, created_at`,
		req.Name, req.Description, req.SortOrder).Scan(
		&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return &c, nil
}

func (r *Repository) UpdateCategory(ctx context.Context, id int64, req UpdateCategoryRequest) (*Category, error) {
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
	if req.SortOrder != nil {
		sets = append(sets, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`UPDATE policy_categories SET %s WHERE id = $%d
		RETURNING id, name, COALESCE(description,''), sort_order, created_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var c Category
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&c.ID, &c.Name, &c.Description, &c.SortOrder, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}
	return &c, nil
}

func (r *Repository) DeleteCategory(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM policy_categories WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

// --- Policies ---

func (r *Repository) ListPolicies(ctx context.Context, p ListPolicyParams) ([]Policy, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if p.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIdx))
		args = append(args, *p.CategoryID)
		argIdx++
	}
	if p.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argIdx))
		args = append(args, p.Severity)
		argIdx++
	}
	if p.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, p.Action)
		argIdx++
	}
	if p.IsEnabled != nil {
		conditions = append(conditions, fmt.Sprintf("is_enabled = $%d", argIdx))
		args = append(args, *p.IsEnabled)
		argIdx++
	}
	if p.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+p.Search+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM policies "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count policies: %w", err)
	}

	offset := (p.Page - 1) * p.PageSize
	query := fmt.Sprintf(`SELECT id, name, category_id, severity, action, is_enabled,
		COALESCE(description,''), created_at, updated_at
		FROM policies %s ORDER BY id DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, p.PageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var pol Policy
		if err := rows.Scan(&pol.ID, &pol.Name, &pol.CategoryID, &pol.Severity, &pol.Action,
			&pol.IsEnabled, &pol.Description, &pol.CreatedAt, &pol.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, pol)
	}
	return policies, total, nil
}

func (r *Repository) GetPolicy(ctx context.Context, id int64) (*Policy, error) {
	var pol Policy
	err := r.pool.QueryRow(ctx, `SELECT id, name, category_id, severity, action, is_enabled,
		COALESCE(description,''), created_at, updated_at
		FROM policies WHERE id = $1`, id).Scan(
		&pol.ID, &pol.Name, &pol.CategoryID, &pol.Severity, &pol.Action,
		&pol.IsEnabled, &pol.Description, &pol.CreatedAt, &pol.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}
	return &pol, nil
}

func (r *Repository) CreatePolicy(ctx context.Context, req CreatePolicyRequest) (*Policy, error) {
	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}
	action := req.Action
	if action == "" {
		action = "block"
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	var pol Policy
	err := r.pool.QueryRow(ctx, `INSERT INTO policies (name, category_id, severity, action, is_enabled, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, category_id, severity, action, is_enabled,
		COALESCE(description,''), created_at, updated_at`,
		req.Name, req.CategoryID, severity, action, isEnabled, req.Description).Scan(
		&pol.ID, &pol.Name, &pol.CategoryID, &pol.Severity, &pol.Action,
		&pol.IsEnabled, &pol.Description, &pol.CreatedAt, &pol.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create policy: %w", err)
	}
	return &pol, nil
}

func (r *Repository) UpdatePolicy(ctx context.Context, id int64, req UpdatePolicyRequest) (*Policy, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.CategoryID != nil {
		sets = append(sets, fmt.Sprintf("category_id = $%d", argIdx))
		args = append(args, *req.CategoryID)
		argIdx++
	}
	if req.Severity != nil {
		sets = append(sets, fmt.Sprintf("severity = $%d", argIdx))
		args = append(args, *req.Severity)
		argIdx++
	}
	if req.Action != nil {
		sets = append(sets, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, *req.Action)
		argIdx++
	}
	if req.IsEnabled != nil {
		sets = append(sets, fmt.Sprintf("is_enabled = $%d", argIdx))
		args = append(args, *req.IsEnabled)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetPolicy(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE policies SET %s WHERE id = $%d
		RETURNING id, name, category_id, severity, action, is_enabled,
		COALESCE(description,''), created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var pol Policy
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&pol.ID, &pol.Name, &pol.CategoryID, &pol.Severity, &pol.Action,
		&pol.IsEnabled, &pol.Description, &pol.CreatedAt, &pol.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update policy: %w", err)
	}
	return &pol, nil
}

func (r *Repository) DeletePolicy(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM policies WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("policy not found")
	}
	return nil
}

// --- Rules ---

func (r *Repository) ListRules(ctx context.Context, policyID int64) ([]Rule, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, policy_id, rule_type, field, operator, value, logic, sort_order, created_at
		FROM policy_rules WHERE policy_id = $1 ORDER BY sort_order, id`, policyID)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var ru Rule
		if err := rows.Scan(&ru.ID, &ru.PolicyID, &ru.RuleType, &ru.Field, &ru.Operator,
			&ru.Value, &ru.Logic, &ru.SortOrder, &ru.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		rules = append(rules, ru)
	}
	return rules, nil
}

func (r *Repository) CreateRule(ctx context.Context, req CreateRuleRequest) (*Rule, error) {
	logic := req.Logic
	if logic == "" {
		logic = "AND"
	}

	var ru Rule
	err := r.pool.QueryRow(ctx, `INSERT INTO policy_rules (policy_id, rule_type, field, operator, value, logic, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, policy_id, rule_type, field, operator, value, logic, sort_order, created_at`,
		req.PolicyID, req.RuleType, req.Field, req.Operator, req.Value, logic, req.SortOrder).Scan(
		&ru.ID, &ru.PolicyID, &ru.RuleType, &ru.Field, &ru.Operator,
		&ru.Value, &ru.Logic, &ru.SortOrder, &ru.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create rule: %w", err)
	}
	return &ru, nil
}

func (r *Repository) DeleteRule(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM policy_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

// --- Change History ---

func (r *Repository) ListHistory(ctx context.Context, policyID int64, limit int) ([]ChangeHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `SELECT id, policy_id, user_id, username, action,
		old_value, new_value, created_at
		FROM policy_change_history WHERE policy_id = $1 ORDER BY created_at DESC LIMIT $2`,
		policyID, limit)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var history []ChangeHistory
	for rows.Next() {
		var h ChangeHistory
		if err := rows.Scan(&h.ID, &h.PolicyID, &h.UserID, &h.Username, &h.Action,
			&h.OldValue, &h.NewValue, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		history = append(history, h)
	}
	return history, nil
}

func (r *Repository) RecordChange(ctx context.Context, policyID int64, userID *int64, username, action string, oldVal, newVal interface{}) error {
	oldJSON, _ := json.Marshal(oldVal)
	newJSON, _ := json.Marshal(newVal)

	_, err := r.pool.Exec(ctx, `INSERT INTO policy_change_history (policy_id, user_id, username, action, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		policyID, userID, username, action, oldJSON, newJSON)
	if err != nil {
		return fmt.Errorf("record change: %w", err)
	}
	return nil
}
