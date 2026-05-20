package monitor

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// KPISnapshot 是 NW · 02 仪表盘顶部 5 张 KPI 卡的聚合返回。
// 字段命名采用前端蛇形，前端 dashboard 适配层一对一映射。
type KPISnapshot struct {
	BlockedToday       int64     `json:"blocked_today"`        // 今日拦截攻击
	TotalRequestsToday int64     `json:"total_requests_today"` // 今日总请求（mock 估算）
	AvgLatencyMs       float64   `json:"avg_latency_ms"`       // 平均响应延迟
	BlockedRatePct     float64   `json:"blocked_rate_pct"`     // 拦截/挑战率（百分比）
	ActiveHighAlerts   int64     `json:"active_high_alerts"`   // 活跃高危告警

	SparkBlocked   []int64   `json:"spark_blocked"`     // 24 小时拦截趋势
	SparkRequests  []int64   `json:"spark_requests"`    // 24 小时请求趋势
	SparkLatency   []float64 `json:"spark_latency"`     // 24 小时延迟趋势
	SparkBlockRate []float64 `json:"spark_block_rate"`  // 24 小时拦截率趋势
	SparkAlerts    []int64   `json:"spark_alerts"`      // 24 小时告警趋势

	GeneratedAt time.Time `json:"generated_at"`
}

// KPISnapshot 由 attack_logs 与 alert_events 聚合得到，metrics 表（rps / latency）
// 由 agent 上报后落地于此处的 SparkXxx 字段。
func (r *Repository) KPISnapshot(ctx context.Context) (*KPISnapshot, error) {
	out := &KPISnapshot{GeneratedAt: time.Now()}

	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM attack_logs WHERE occurred_at >= NOW() - INTERVAL '1 day'`,
	).Scan(&out.BlockedToday); err != nil {
		return nil, fmt.Errorf("kpi blocked: %w", err)
	}

	// 总请求量：metrics 表里的 total_requests 累积；缺失时按 blocked / 2.98% 反推（与 mock 一致）。
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(value),0)::BIGINT FROM metrics
		WHERE name = 'total_requests' AND recorded_at >= NOW() - INTERVAL '1 day'`).
		Scan(&out.TotalRequestsToday); err != nil {
		// 表/列不存在视为 0
		out.TotalRequestsToday = 0
	}
	if out.TotalRequestsToday == 0 && out.BlockedToday > 0 {
		out.TotalRequestsToday = int64(float64(out.BlockedToday) / 0.0298)
	}

	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(AVG(value),0) FROM metrics
		WHERE name = 'latency_ms' AND recorded_at >= NOW() - INTERVAL '1 day'`).
		Scan(&out.AvgLatencyMs); err != nil {
		out.AvgLatencyMs = 0
	}

	if out.TotalRequestsToday > 0 {
		out.BlockedRatePct = float64(out.BlockedToday) / float64(out.TotalRequestsToday) * 100
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM alert_events WHERE level = 'critical' AND status = 'open'`,
	).Scan(&out.ActiveHighAlerts); err != nil {
		return nil, fmt.Errorf("kpi alerts: %w", err)
	}

	// 24 小时趋势 spark：按小时分桶，缺失桶填 0
	out.SparkBlocked = r.bucketHourlyCount(ctx, "attack_logs", "occurred_at", "")
	out.SparkAlerts = r.bucketHourlyCount(ctx, "alert_events", "occurred_at", "level = 'critical'")
	out.SparkRequests = r.bucketHourlyMetric(ctx, "total_requests", true)
	out.SparkLatency = r.bucketHourlyMetricFloat(ctx, "latency_ms", false)
	out.SparkBlockRate = r.computeBlockRateSeries(out.SparkBlocked, out.SparkRequests)

	return out, nil
}

func (r *Repository) bucketHourlyCount(ctx context.Context, table, tsCol, extraCond string) []int64 {
	cond := ""
	if extraCond != "" {
		cond = " AND " + extraCond
	}
	q := fmt.Sprintf(`SELECT
		date_trunc('hour', %s) AS bucket, COUNT(*)::BIGINT
		FROM %s
		WHERE %s >= NOW() - INTERVAL '24 hours'%s
		GROUP BY bucket ORDER BY bucket`, tsCol, table, tsCol, cond)
	return r.runBucketCount(ctx, q)
}

func (r *Repository) bucketHourlyMetric(ctx context.Context, name string, asSum bool) []int64 {
	agg := "AVG(value)"
	if asSum {
		agg = "SUM(value)"
	}
	q := fmt.Sprintf(`SELECT
		date_trunc('hour', recorded_at) AS bucket, COALESCE(%s,0)::BIGINT
		FROM metrics WHERE name = $1 AND recorded_at >= NOW() - INTERVAL '24 hours'
		GROUP BY bucket ORDER BY bucket`, agg)
	return r.runBucketCountWith(ctx, q, name)
}

func (r *Repository) bucketHourlyMetricFloat(ctx context.Context, name string, asSum bool) []float64 {
	agg := "AVG(value)"
	if asSum {
		agg = "SUM(value)"
	}
	q := fmt.Sprintf(`SELECT
		date_trunc('hour', recorded_at) AS bucket, COALESCE(%s,0)
		FROM metrics WHERE name = $1 AND recorded_at >= NOW() - INTERVAL '24 hours'
		GROUP BY bucket ORDER BY bucket`, agg)
	out := make([]float64, 24)
	rows, err := r.pool.Query(ctx, q, name)
	if err != nil {
		return out
	}
	defer rows.Close()
	buckets := map[int]float64{}
	for rows.Next() {
		var t time.Time
		var v float64
		if err := rows.Scan(&t, &v); err != nil {
			continue
		}
		offset := 23 - int(time.Since(t).Hours())
		if offset >= 0 && offset < 24 {
			buckets[offset] = v
		}
	}
	for i := 0; i < 24; i++ {
		out[i] = buckets[i]
	}
	return out
}

func (r *Repository) runBucketCount(ctx context.Context, q string) []int64 {
	out := make([]int64, 24)
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return out
	}
	defer rows.Close()
	buckets := map[int]int64{}
	for rows.Next() {
		var t time.Time
		var n int64
		if err := rows.Scan(&t, &n); err != nil {
			continue
		}
		offset := 23 - int(time.Since(t).Hours())
		if offset >= 0 && offset < 24 {
			buckets[offset] = n
		}
	}
	for i := 0; i < 24; i++ {
		out[i] = buckets[i]
	}
	return out
}

func (r *Repository) runBucketCountWith(ctx context.Context, q string, args ...any) []int64 {
	out := make([]int64, 24)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	buckets := map[int]int64{}
	for rows.Next() {
		var t time.Time
		var n int64
		if err := rows.Scan(&t, &n); err != nil {
			continue
		}
		offset := 23 - int(time.Since(t).Hours())
		if offset >= 0 && offset < 24 {
			buckets[offset] = n
		}
	}
	for i := 0; i < 24; i++ {
		out[i] = buckets[i]
	}
	return out
}

func (r *Repository) computeBlockRateSeries(blocked, total []int64) []float64 {
	out := make([]float64, len(blocked))
	for i := range blocked {
		t := int64(0)
		if i < len(total) {
			t = total[i]
		}
		if t > 0 {
			out[i] = float64(blocked[i]) / float64(t) * 100
		}
	}
	return out
}

// KPI handler. GET /monitor/kpi
func (h *Handler) KPI(w http.ResponseWriter, r *http.Request) {
	kpi, err := h.repo.KPISnapshot(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, kpi)
}
