package aggregation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SiteStats(ctx context.Context) (SiteStats, error) {
	var s SiteStats
	if err := r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN status = 'active' AND waf_enabled THEN 1 ELSE 0 END), 0) AS on_count,
			COALESCE(SUM(CASE WHEN status = 'paused' OR NOT waf_enabled THEN 1 ELSE 0 END), 0) AS off_count,
			COALESCE(SUM(CASE WHEN status = 'observe' THEN 1 ELSE 0 END), 0) AS idle_count
		FROM sites`,
	).Scan(&s.On, &s.Off, &s.Idle); err != nil {
		return s, fmt.Errorf("site stats: %w", err)
	}
	return s, nil
}

func (r *Repository) AttackRecords(ctx context.Context, limit int) ([]AttackRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT to_char(occurred_at, 'YYYY-MM-DD HH24:MI:SS'),
		       COALESCE(src_ip,''), COALESCE(dst_ip,''),
		       COALESCE(s.domain,''), COALESCE(action,''), COALESCE(attack_type,'medium')
		FROM attack_logs al
		LEFT JOIN sites s ON al.dst_port = s.listen_port
		ORDER BY occurred_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query attack logs: %w", err)
	}
	defer rows.Close()

	items := make([]AttackRecord, 0)
	for rows.Next() {
		var rec AttackRecord
		var attackType string
		if err := rows.Scan(&rec.Datetime, &rec.SrcIP, &rec.DstIP, &rec.Host, &rec.Action, &attackType); err != nil {
			return nil, fmt.Errorf("scan attack: %w", err)
		}
		rec.Severity = mapSeverity(attackType)
		items = append(items, rec)
	}
	return items, nil
}

func (r *Repository) AttackSeverity(ctx context.Context) (AttackSeverity, error) {
	var s AttackSeverity
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(attack_type,''), COUNT(*) FROM attack_logs GROUP BY attack_type`)
	if err != nil {
		return s, fmt.Errorf("severity stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v int64
		if err := rows.Scan(&k, &v); err != nil {
			return s, fmt.Errorf("scan severity: %w", err)
		}
		switch mapSeverity(k) {
		case "critical":
			s.Critical += v
		case "high":
			s.High += v
		case "medium":
			s.Medium += v
		case "low":
			s.Low += v
		default:
			s.Info += v
		}
	}
	return s, nil
}

func (r *Repository) AttackSourceTop(ctx context.Context, limit int) ([]AttackSourceTop, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(src_ip,''), COUNT(*) AS cnt
		 FROM attack_logs WHERE src_ip IS NOT NULL
		 GROUP BY src_ip ORDER BY cnt DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("attack source top: %w", err)
	}
	defer rows.Close()
	items := make([]AttackSourceTop, 0)
	for rows.Next() {
		var it AttackSourceTop
		if err := rows.Scan(&it.SrcIP, &it.Count); err != nil {
			return nil, fmt.Errorf("scan attack src ip: %w", err)
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *Repository) TopSites(ctx context.Context, limit int) ([][2]any, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.pool.Query(ctx,
		`SELECT COALESCE(s.domain,'unknown'), COUNT(al.id) AS cnt
		 FROM attack_logs al
		 LEFT JOIN sites s ON al.dst_port = s.listen_port
		 GROUP BY s.domain ORDER BY cnt DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("top sites: %w", err)
	}
	defer rows.Close()
	items := make([][2]any, 0)
	for rows.Next() {
		var host string
		var cnt int64
		if err := rows.Scan(&host, &cnt); err != nil {
			return nil, fmt.Errorf("scan top sites: %w", err)
		}
		items = append(items, [2]any{host, cnt})
	}
	return items, nil
}

func (r *Repository) AttackLogsCount(ctx context.Context) (int64, error) {
	var n int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM attack_logs`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count attack logs: %w", err)
	}
	return n, nil
}

// MetricHistory 返回 admin 监控历史曲线。entries 形如 "cpu_percent,memory_percent"。
func (r *Repository) MetricHistory(ctx context.Context, hostname, entries string) map[string][]MetricSample {
	if entries == "" {
		entries = "cpu_percent,memory_percent,disk_percent"
	}
	out := make(map[string][]MetricSample)
	names := splitCSV(entries)
	for _, name := range names {
		out[name] = r.queryMetric(ctx, hostname, name, 60)
	}
	return out
}

type MetricSample struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

func (r *Repository) queryMetric(ctx context.Context, hostname, name string, limit int) []MetricSample {
	var rows = func() ([]MetricSample, error) {
		var q string
		var args []any
		if hostname == "" {
			q = `SELECT to_char(recorded_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'), value
				FROM monitor_metrics WHERE name = $1 ORDER BY recorded_at DESC LIMIT $2`
			args = []any{name, limit}
		} else {
			q = `SELECT to_char(m.recorded_at, 'YYYY-MM-DD"T"HH24:MI:SSZ'), m.value
				FROM monitor_metrics m
				JOIN nodes n ON n.id = m.node_id
				WHERE m.name = $1 AND (n.hostname = $2 OR n.name = $2)
				ORDER BY m.recorded_at DESC LIMIT $3`
			args = []any{name, hostname, limit}
		}
		res, err := r.pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer res.Close()
		samples := make([]MetricSample, 0, limit)
		for res.Next() {
			var s MetricSample
			if err := res.Scan(&s.Timestamp, &s.Value); err != nil {
				return nil, err
			}
			samples = append(samples, s)
		}
		return samples, nil
	}
	out, err := rows()
	if err != nil {
		return []MetricSample{}
	}
	return out
}

func splitCSV(s string) []string {
	out := make([]string, 0)
	cur := ""
	for _, r := range s {
		if r == ',' || r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func mapSeverity(rawType string) string {
	switch rawType {
	case "sqli", "rce", "xxe", "lfi":
		return "critical"
	case "xss", "csrf", "ssrf":
		return "high"
	case "cc", "scan":
		return "medium"
	case "info":
		return "info"
	case "low":
		return "low"
	default:
		return "medium"
	}
}
