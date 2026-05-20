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

// EnsureSchema 启动时幂等补齐 policies 表的 NW · 04 UI 字段（migration 000012）。
// 真正的内置规则种子由 SyncFromModsec 负责（解析 deploy/modsec/rules.d/）。
func (r *Repository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS scope       VARCHAR(64) NOT NULL DEFAULT '全部站点'`,
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS field       VARCHAR(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS match_value TEXT        NOT NULL DEFAULT ''`,
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS priority    INTEGER     NOT NULL DEFAULT 100`,
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS builtin     BOOLEAN     NOT NULL DEFAULT FALSE`,
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS hits        BIGINT      NOT NULL DEFAULT 0`,
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS last_hit_at TIMESTAMPTZ`,
		`ALTER TABLE policies ADD COLUMN IF NOT EXISTS modsec_id   VARCHAR(32)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_scope    ON policies(scope)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_priority ON policies(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_builtin  ON policies(builtin)`,
		// modsec_id 必须唯一（且允许 NULL —— 用户自建规则没有 modsec_id）
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_policies_modsec_id ON policies(modsec_id) WHERE modsec_id IS NOT NULL`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure policies schema (%q): %w", s, err)
		}
	}
	return nil
}

const policySelectCols = `id, name, category_id, severity, action, is_enabled,
	COALESCE(description,''), created_at, updated_at,
	COALESCE(scope,'全部站点'), COALESCE(field,''), COALESCE(match_value,''),
	COALESCE(priority,100), COALESCE(builtin,false), COALESCE(hits,0), last_hit_at,
	COALESCE(modsec_id,'')`

func scanPolicy(rs interface {
	Scan(...interface{}) error
}, pol *Policy) error {
	return rs.Scan(&pol.ID, &pol.Name, &pol.CategoryID, &pol.Severity, &pol.Action,
		&pol.IsEnabled, &pol.Description, &pol.CreatedAt, &pol.UpdatedAt,
		&pol.Scope, &pol.Field, &pol.Match, &pol.Priority, &pol.Builtin, &pol.Hits, &pol.LastHitAt,
		&pol.ModsecID)
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
	query := fmt.Sprintf(`SELECT %s FROM policies %s ORDER BY priority ASC, id DESC LIMIT $%d OFFSET $%d`,
		policySelectCols, where, argIdx, argIdx+1)
	args = append(args, p.PageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var pol Policy
		if err := scanPolicy(rows, &pol); err != nil {
			return nil, 0, fmt.Errorf("scan policy: %w", err)
		}
		policies = append(policies, pol)
	}
	return policies, total, nil
}

func (r *Repository) GetPolicy(ctx context.Context, id int64) (*Policy, error) {
	var pol Policy
	q := fmt.Sprintf(`SELECT %s FROM policies WHERE id = $1`, policySelectCols)
	if err := scanPolicy(r.pool.QueryRow(ctx, q, id), &pol); err != nil {
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
	scope := req.Scope
	if scope == "" {
		scope = "全部站点"
	}
	priority := req.Priority
	if priority <= 0 {
		priority = 100
	}
	builtin := false
	if req.Builtin != nil {
		builtin = *req.Builtin
	}

	var pol Policy
	q := fmt.Sprintf(`INSERT INTO policies (name, category_id, severity, action, is_enabled, description,
		scope, field, match_value, priority, builtin)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING %s`, policySelectCols)
	if err := scanPolicy(r.pool.QueryRow(ctx, q,
		req.Name, req.CategoryID, severity, action, isEnabled, req.Description,
		scope, req.Field, req.Match, priority, builtin,
	), &pol); err != nil {
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
	if req.Scope != nil {
		sets = append(sets, fmt.Sprintf("scope = $%d", argIdx))
		args = append(args, *req.Scope)
		argIdx++
	}
	if req.Field != nil {
		sets = append(sets, fmt.Sprintf("field = $%d", argIdx))
		args = append(args, *req.Field)
		argIdx++
	}
	if req.Match != nil {
		sets = append(sets, fmt.Sprintf("match_value = $%d", argIdx))
		args = append(args, *req.Match)
		argIdx++
	}
	if req.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.Builtin != nil {
		sets = append(sets, fmt.Sprintf("builtin = $%d", argIdx))
		args = append(args, *req.Builtin)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetPolicy(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE policies SET %s WHERE id = $%d RETURNING %s`,
		strings.Join(sets, ", "), argIdx, policySelectCols)
	args = append(args, id)

	var pol Policy
	if err := scanPolicy(r.pool.QueryRow(ctx, query, args...), &pol); err != nil {
		return nil, fmt.Errorf("update policy: %w", err)
	}
	return &pol, nil
}

// IncrementHits 由 agent 上报命中计数（hits += delta，last_hit_at = NOW）。
func (r *Repository) IncrementHits(ctx context.Context, id int64, delta int64) (*Policy, error) {
	if delta <= 0 {
		delta = 1
	}
	q := fmt.Sprintf(`UPDATE policies SET hits = hits + $1, last_hit_at = NOW() WHERE id = $2 RETURNING %s`,
		policySelectCols)
	var pol Policy
	if err := scanPolicy(r.pool.QueryRow(ctx, q, delta, id), &pol); err != nil {
		return nil, fmt.Errorf("increment hits: %w", err)
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
