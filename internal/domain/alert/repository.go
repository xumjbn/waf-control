package alert

import (
	"context"
	"encoding/json"
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

const policyCols = `id, name, COALESCE(description,''), metric, operator, threshold,
	window_seconds, level, notify_targets, is_enabled, created_at, updated_at`

func (r *Repository) ListPolicies(ctx context.Context) ([]Policy, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+policyCols+` FROM alert_policies ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list alert policies: %w", err)
	}
	defer rows.Close()

	out := make([]Policy, 0)
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *Repository) GetPolicy(ctx context.Context, id int64) (*Policy, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+policyCols+` FROM alert_policies WHERE id = $1`, id)
	p, err := scanPolicy(row)
	if err != nil {
		return nil, fmt.Errorf("get alert policy: %w", err)
	}
	return &p, nil
}

func (r *Repository) CreatePolicy(ctx context.Context, req CreatePolicyRequest) (*Policy, error) {
	level := req.Level
	if level == "" {
		level = "warn"
	}
	op := req.Operator
	if op == "" {
		op = ">"
	}
	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}
	targets, err := json.Marshal(nonNilTargets(req.NotifyTargets))
	if err != nil {
		return nil, fmt.Errorf("marshal targets: %w", err)
	}
	row := r.pool.QueryRow(ctx, `INSERT INTO alert_policies
		(name, description, metric, operator, threshold, window_seconds, level, notify_targets, is_enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING `+policyCols,
		req.Name, req.Description, req.Metric, op, req.Threshold,
		req.WindowSeconds, level, targets, enabled)
	p, err := scanPolicy(row)
	if err != nil {
		return nil, fmt.Errorf("create alert policy: %w", err)
	}
	return &p, nil
}

func (r *Repository) UpdatePolicy(ctx context.Context, id int64, req UpdatePolicyRequest) (*Policy, error) {
	sets := make([]string, 0, 9)
	args := make([]any, 0, 10)
	idx := 1
	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *req.Description)
		idx++
	}
	if req.Metric != nil {
		sets = append(sets, fmt.Sprintf("metric = $%d", idx))
		args = append(args, *req.Metric)
		idx++
	}
	if req.Operator != nil {
		sets = append(sets, fmt.Sprintf("operator = $%d", idx))
		args = append(args, *req.Operator)
		idx++
	}
	if req.Threshold != nil {
		sets = append(sets, fmt.Sprintf("threshold = $%d", idx))
		args = append(args, *req.Threshold)
		idx++
	}
	if req.WindowSeconds != nil {
		sets = append(sets, fmt.Sprintf("window_seconds = $%d", idx))
		args = append(args, *req.WindowSeconds)
		idx++
	}
	if req.Level != nil {
		sets = append(sets, fmt.Sprintf("level = $%d", idx))
		args = append(args, *req.Level)
		idx++
	}
	if req.NotifyTargets != nil {
		raw, err := json.Marshal(nonNilTargets(*req.NotifyTargets))
		if err != nil {
			return nil, fmt.Errorf("marshal targets: %w", err)
		}
		sets = append(sets, fmt.Sprintf("notify_targets = $%d", idx))
		args = append(args, raw)
		idx++
	}
	if req.IsEnabled != nil {
		sets = append(sets, fmt.Sprintf("is_enabled = $%d", idx))
		args = append(args, *req.IsEnabled)
		idx++
	}
	if len(sets) == 0 {
		return r.GetPolicy(ctx, id)
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)
	query := fmt.Sprintf(`UPDATE alert_policies SET %s WHERE id = $%d RETURNING %s`,
		strings.Join(sets, ", "), idx, policyCols)
	row := r.pool.QueryRow(ctx, query, args...)
	p, err := scanPolicy(row)
	if err != nil {
		return nil, fmt.Errorf("update alert policy: %w", err)
	}
	return &p, nil
}

func (r *Repository) DeletePolicy(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM alert_policies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete alert policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("alert policy not found")
	}
	return nil
}

const eventCols = `id, policy_id, level, kind, target, message, status, occurred_at, handled_at,
	COALESCE(handled_by,'')`

func (r *Repository) ListEvents(ctx context.Context, f EventFilter) ([]Event, error) {
	conds := make([]string, 0, 2)
	args := make([]any, 0, 3)
	idx := 1
	if f.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.Level != "" {
		conds = append(conds, fmt.Sprintf("level = $%d", idx))
		args = append(args, f.Level)
		idx++
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit)
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	q := fmt.Sprintf(`SELECT %s FROM alert_events %s ORDER BY occurred_at DESC LIMIT $%d`,
		eventCols, where, idx)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list alert events: %w", err)
	}
	defer rows.Close()
	out := make([]Event, 0)
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

func (r *Repository) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
	level := req.Level
	if level == "" {
		level = "warn"
	}
	row := r.pool.QueryRow(ctx, `INSERT INTO alert_events
		(policy_id, level, kind, target, message, status, occurred_at)
		VALUES ($1,$2,$3,$4,$5,'open',NOW())
		RETURNING `+eventCols,
		req.PolicyID, level, req.Kind, req.Target, req.Message)
	ev, err := scanEvent(row)
	if err != nil {
		return nil, fmt.Errorf("create alert event: %w", err)
	}
	return &ev, nil
}

func (r *Repository) UpdateEventStatus(ctx context.Context, id int64, status, handledBy string) (*Event, error) {
	if status != "open" && status != "ack" && status != "closed" {
		return nil, fmt.Errorf("invalid status: %s", status)
	}
	var handledAt any
	if status == "open" {
		handledAt = nil
		handledBy = ""
	} else {
		handledAt = time.Now()
	}
	row := r.pool.QueryRow(ctx, `UPDATE alert_events
		SET status = $1, handled_at = $2, handled_by = $3
		WHERE id = $4 RETURNING `+eventCols,
		status, handledAt, handledBy, id)
	ev, err := scanEvent(row)
	if err != nil {
		return nil, fmt.Errorf("update alert event: %w", err)
	}
	return &ev, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, handledBy string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE alert_events
		SET status = 'ack', handled_at = NOW(), handled_by = $1
		WHERE status = 'open'`, handledBy)
	if err != nil {
		return 0, fmt.Errorf("mark all alert events read: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *Repository) EventStats(ctx context.Context) (EventStats, error) {
	var s EventStats
	if err := r.pool.QueryRow(ctx, `SELECT
		COALESCE(SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status = 'ack' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN occurred_at >= date_trunc('day', NOW()) THEN 1 ELSE 0 END),0)
		FROM alert_events`).Scan(&s.Open, &s.Ack, &s.Closed, &s.Today); err != nil {
		return s, fmt.Errorf("alert event stats: %w", err)
	}
	return s, nil
}

const channelCols = `id, name, kind, COALESCE(target,''), is_enabled, created_at, updated_at`

func (r *Repository) ListChannels(ctx context.Context) ([]Channel, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+channelCols+` FROM alert_channels ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list alert channels: %w", err)
	}
	defer rows.Close()
	out := make([]Channel, 0)
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Name, &c.Kind, &c.Target, &c.IsEnabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan alert channel: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *Repository) UpdateChannel(ctx context.Context, id int64, req UpdateChannelRequest) (*Channel, error) {
	sets := make([]string, 0, 4)
	args := make([]any, 0, 5)
	idx := 1
	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Kind != nil {
		sets = append(sets, fmt.Sprintf("kind = $%d", idx))
		args = append(args, *req.Kind)
		idx++
	}
	if req.Target != nil {
		sets = append(sets, fmt.Sprintf("target = $%d", idx))
		args = append(args, *req.Target)
		idx++
	}
	if req.IsEnabled != nil {
		sets = append(sets, fmt.Sprintf("is_enabled = $%d", idx))
		args = append(args, *req.IsEnabled)
		idx++
	}
	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)
	query := fmt.Sprintf(`UPDATE alert_channels SET %s WHERE id = $%d RETURNING %s`,
		strings.Join(sets, ", "), idx, channelCols)
	row := r.pool.QueryRow(ctx, query, args...)
	var c Channel
	if err := row.Scan(&c.ID, &c.Name, &c.Kind, &c.Target, &c.IsEnabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("update alert channel: %w", err)
	}
	return &c, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPolicy(s rowScanner) (Policy, error) {
	var p Policy
	var raw []byte
	if err := s.Scan(&p.ID, &p.Name, &p.Description, &p.Metric, &p.Operator, &p.Threshold,
		&p.WindowSeconds, &p.Level, &raw, &p.IsEnabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return p, fmt.Errorf("scan alert policy: %w", err)
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p.NotifyTargets); err != nil {
			p.NotifyTargets = []string{}
		}
	} else {
		p.NotifyTargets = []string{}
	}
	if p.NotifyTargets == nil {
		p.NotifyTargets = []string{}
	}
	return p, nil
}

func scanEvent(s rowScanner) (Event, error) {
	var ev Event
	if err := s.Scan(&ev.ID, &ev.PolicyID, &ev.Level, &ev.Kind, &ev.Target, &ev.Message,
		&ev.Status, &ev.OccurredAt, &ev.HandledAt, &ev.HandledBy); err != nil {
		return ev, fmt.Errorf("scan alert event: %w", err)
	}
	return ev, nil
}

func nonNilTargets(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
