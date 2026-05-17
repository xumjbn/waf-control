package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Attack Monitor ---

func (r *Repository) TopSites(ctx context.Context) ([]TopSite, error) {
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(s.id,0), COALESCE(s.name,'unknown'), COUNT(al.id) as cnt
		FROM attack_logs al
		LEFT JOIN sites s ON al.dst_port = s.listen_port
		GROUP BY s.id, s.name ORDER BY cnt DESC LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("top sites: %w", err)
	}
	defer rows.Close()

	var items []TopSite
	for rows.Next() {
		var t TopSite
		if err := rows.Scan(&t.SiteID, &t.SiteName, &t.Count); err != nil {
			return nil, fmt.Errorf("scan top site: %w", err)
		}
		items = append(items, t)
	}
	return items, nil
}

func (r *Repository) TopSrcIPs(ctx context.Context) ([]TopSrcIP, error) {
	rows, err := r.pool.Query(ctx, `SELECT src_ip, COUNT(*) as cnt
		FROM attack_logs
		GROUP BY src_ip ORDER BY cnt DESC LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("top src IPs: %w", err)
	}
	defer rows.Close()

	var items []TopSrcIP
	for rows.Next() {
		var t TopSrcIP
		if err := rows.Scan(&t.SrcIP, &t.Count); err != nil {
			return nil, fmt.Errorf("scan top src ip: %w", err)
		}
		items = append(items, t)
	}
	return items, nil
}

func (r *Repository) SeverityStats(ctx context.Context) ([]SeverityStat, error) {
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(action,'unknown') as severity, COUNT(*) as cnt
		FROM attack_logs
		GROUP BY action ORDER BY cnt DESC`)
	if err != nil {
		return nil, fmt.Errorf("severity stats: %w", err)
	}
	defer rows.Close()

	var items []SeverityStat
	for rows.Next() {
		var s SeverityStat
		if err := rows.Scan(&s.Severity, &s.Count); err != nil {
			return nil, fmt.Errorf("scan severity: %w", err)
		}
		items = append(items, s)
	}
	return items, nil
}

// --- System Monitor ---

func (r *Repository) ListMetrics(ctx context.Context) ([]Metric, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, value, COALESCE(unit,''), node_id, recorded_at
		FROM monitor_metrics ORDER BY recorded_at DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("list metrics: %w", err)
	}
	defer rows.Close()

	var items []Metric
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.ID, &m.Name, &m.Value, &m.Unit, &m.NodeID, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		items = append(items, m)
	}
	return items, nil
}

func (r *Repository) ListMetricSpecs(ctx context.Context) ([]MetricSpec, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, COALESCE(description,''), COALESCE(unit,'')
		FROM monitor_metric_specs ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list metric specs: %w", err)
	}
	defer rows.Close()

	var items []MetricSpec
	for rows.Next() {
		var ms MetricSpec
		if err := rows.Scan(&ms.ID, &ms.Name, &ms.Description, &ms.Unit); err != nil {
			return nil, fmt.Errorf("scan metric spec: %w", err)
		}
		items = append(items, ms)
	}
	return items, nil
}

func (r *Repository) GetMetricSpec(ctx context.Context, id int64) (*MetricSpec, error) {
	var ms MetricSpec
	err := r.pool.QueryRow(ctx, `SELECT id, name, COALESCE(description,''), COALESCE(unit,'')
		FROM monitor_metric_specs WHERE id = $1`, id).Scan(
		&ms.ID, &ms.Name, &ms.Description, &ms.Unit)
	if err != nil {
		return nil, fmt.Errorf("get metric spec: %w", err)
	}
	return &ms, nil
}

func (r *Repository) QueryRealtime(ctx context.Context, req RealtimeQuery) ([]Metric, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, value, COALESCE(unit,''), node_id, recorded_at
		FROM monitor_metrics
		WHERE node_id = $1 AND name = $2
		ORDER BY recorded_at DESC LIMIT 1`, req.NodeID, req.Metric)
	if err != nil {
		return nil, fmt.Errorf("query realtime: %w", err)
	}
	defer rows.Close()

	var items []Metric
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.ID, &m.Name, &m.Value, &m.Unit, &m.NodeID, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan realtime metric: %w", err)
		}
		items = append(items, m)
	}
	return items, nil
}

func (r *Repository) QueryHistory(ctx context.Context, req HistoryQuery) ([]Metric, error) {
	var start, end time.Time
	var err error

	if req.StartTime != "" {
		start, err = time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			start = time.Now().Add(-24 * time.Hour)
		}
	} else {
		start = time.Now().Add(-24 * time.Hour)
	}

	if req.EndTime != "" {
		end, err = time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			end = time.Now()
		}
	} else {
		end = time.Now()
	}

	rows, err := r.pool.Query(ctx, `SELECT id, name, value, COALESCE(unit,''), node_id, recorded_at
		FROM monitor_metrics
		WHERE node_id = $1 AND name = $2 AND recorded_at >= $3 AND recorded_at <= $4
		ORDER BY recorded_at ASC`, req.NodeID, req.Metric, start, end)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var items []Metric
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.ID, &m.Name, &m.Value, &m.Unit, &m.NodeID, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan history metric: %w", err)
		}
		items = append(items, m)
	}
	return items, nil
}
