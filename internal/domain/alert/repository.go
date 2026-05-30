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

// EnsureSchema 启动时幂等补齐 alert_channels NW · 06 UI 字段（migration 000013）。
func (r *Repository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE alert_channels ADD COLUMN IF NOT EXISTS config      JSONB        NOT NULL DEFAULT '{}'::jsonb`,
		`ALTER TABLE alert_channels ADD COLUMN IF NOT EXISTS description VARCHAR(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE alert_channels ADD COLUMN IF NOT EXISTS severity    VARCHAR(16)  NOT NULL DEFAULT 'warn'`,
		`CREATE INDEX IF NOT EXISTS idx_alert_channels_kind ON alert_channels(kind)`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure alert_channels schema (%q): %w", s, err)
		}
	}
	return nil
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

// HourlyBucket 是告警分布的单个小时桶。监控大屏「近 24h 告警分布」消费。
type HourlyBucket struct {
	Hour     int   `json:"hour"`     // 0-23（本地小时）
	Total    int64 `json:"total"`
	Critical int64 `json:"critical"`
	Warning  int64 `json:"warning"`
	Info     int64 `json:"info"`
}

// HourlyStats 返回近 24 小时按小时分桶的告警计数（缺失桶填 0），按时间正序。
func (r *Repository) HourlyStats(ctx context.Context) ([]HourlyBucket, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(HOUR FROM occurred_at)::INT AS hour,
		       COUNT(*)::BIGINT AS total,
		       COUNT(*) FILTER (WHERE level = 'critical')::BIGINT,
		       COUNT(*) FILTER (WHERE level = 'warning' OR level = 'warn')::BIGINT,
		       COUNT(*) FILTER (WHERE level = 'info')::BIGINT
		  FROM alert_events
		 WHERE occurred_at >= NOW() - INTERVAL '24 hours'
		 GROUP BY hour`)
	if err != nil {
		return nil, fmt.Errorf("alert hourly stats: %w", err)
	}
	defer rows.Close()
	byHour := map[int]HourlyBucket{}
	for rows.Next() {
		var b HourlyBucket
		if err := rows.Scan(&b.Hour, &b.Total, &b.Critical, &b.Warning, &b.Info); err != nil {
			return nil, fmt.Errorf("scan hourly bucket: %w", err)
		}
		byHour[b.Hour] = b
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 输出 24 桶，缺失填 0。
	out := make([]HourlyBucket, 24)
	for h := 0; h < 24; h++ {
		if b, ok := byHour[h]; ok {
			out[h] = b
		} else {
			out[h] = HourlyBucket{Hour: h}
		}
	}
	return out, nil
}

const channelCols = `id, name, kind, COALESCE(target,''), COALESCE(description,''),
	COALESCE(severity,'warn'), COALESCE(config,'{}'::jsonb), is_enabled, created_at, updated_at`

func scanChannel(s rowScanner, c *Channel) error {
	return s.Scan(&c.ID, &c.Name, &c.Kind, &c.Target, &c.Description, &c.Severity,
		&c.Config, &c.IsEnabled, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) ListChannels(ctx context.Context) ([]Channel, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+channelCols+` FROM alert_channels ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list alert channels: %w", err)
	}
	defer rows.Close()
	out := make([]Channel, 0)
	for rows.Next() {
		var c Channel
		if err := scanChannel(rows, &c); err != nil {
			return nil, fmt.Errorf("scan alert channel: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *Repository) GetChannel(ctx context.Context, id int64) (*Channel, error) {
	var c Channel
	if err := scanChannel(r.pool.QueryRow(ctx, `SELECT `+channelCols+` FROM alert_channels WHERE id = $1`, id), &c); err != nil {
		return nil, fmt.Errorf("get alert channel: %w", err)
	}
	return &c, nil
}

func (r *Repository) CreateChannel(ctx context.Context, req CreateChannelRequest) (*Channel, error) {
	kind := req.Kind
	if kind == "" {
		kind = ChannelKindWebhook
	}
	severity := req.Severity
	if severity == "" {
		severity = "warn"
	}
	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}
	cfg := req.Config
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	query := `INSERT INTO alert_channels (name, kind, target, description, severity, config, is_enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING ` + channelCols
	var c Channel
	if err := scanChannel(r.pool.QueryRow(ctx, query, req.Name, kind, req.Target, req.Description, severity, cfg, enabled), &c); err != nil {
		return nil, fmt.Errorf("create alert channel: %w", err)
	}
	return &c, nil
}

func (r *Repository) UpdateChannel(ctx context.Context, id int64, req UpdateChannelRequest) (*Channel, error) {
	sets := make([]string, 0, 8)
	args := make([]any, 0, 9)
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
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *req.Description)
		idx++
	}
	if req.Severity != nil {
		sets = append(sets, fmt.Sprintf("severity = $%d", idx))
		args = append(args, *req.Severity)
		idx++
	}
	if len(req.Config) > 0 {
		sets = append(sets, fmt.Sprintf("config = $%d", idx))
		args = append(args, req.Config)
		idx++
	}
	if req.IsEnabled != nil {
		sets = append(sets, fmt.Sprintf("is_enabled = $%d", idx))
		args = append(args, *req.IsEnabled)
		idx++
	}
	if len(sets) == 0 {
		return r.GetChannel(ctx, id)
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)
	query := fmt.Sprintf(`UPDATE alert_channels SET %s WHERE id = $%d RETURNING %s`,
		strings.Join(sets, ", "), idx, channelCols)
	var c Channel
	if err := scanChannel(r.pool.QueryRow(ctx, query, args...), &c); err != nil {
		return nil, fmt.Errorf("update alert channel: %w", err)
	}
	return &c, nil
}

func (r *Repository) DeleteChannel(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM alert_channels WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete alert channel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("channel not found")
	}
	return nil
}

// RecordTestEvent 写一条 INFO 级 alert_events 审计记录（投递结果 status 写进 message）。
// 真实投递由 alert.Send 完成；这里只留痕。
func (r *Repository) RecordTestEvent(ctx context.Context, ch *Channel, result string) (*Event, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO alert_events
		(policy_id, level, kind, target, message, status, occurred_at)
		VALUES (NULL, 'info', $1, $2, $3, 'open', NOW())
		RETURNING `+eventCols,
		ch.Kind, ch.Target, fmt.Sprintf("【测试】渠道 %s —— %s", ch.Name, result))
	ev, err := scanEvent(row)
	if err != nil {
		return nil, fmt.Errorf("record test event: %w", err)
	}
	return &ev, nil
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
