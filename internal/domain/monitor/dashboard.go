package monitor

// dashboard.go —— NW · 01 安全总览 + NW · 02 监控大屏的真聚合端点。
//
// 之前前端 dashboard 顶部 KPI 是硬编码字符串、sparkline/趋势/热力图/TOP/类型分布
// 全是 Math.random；monitor 大屏整页 mkAttack 造数。这里把它们全部接到 attack_logs
// 真聚合：
//   GET /monitor/dashboard          一次返回 KPI(复用) + TOP 威胁源 + 攻击类型分布 + 7×24 热力图
//   GET /monitor/realtime-series    近 N 分钟 req/block/chal 三路时序 + 实时 TOP（给监控大屏轮询）
//
// 设计原则：任何子聚合失败只返回空块，不整体 500 —— dashboard 不该因为某块挂掉而全白。

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// --- 类型 ---

type TopThreatSource struct {
	SrcIP   string `json:"src_ip"`
	Country string `json:"country"`
	Count   int64  `json:"count"`
}

type AttackTypeSlice struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	Color     string `json:"color"`
	Count     int64  `json:"count"`
}

// DashboardSnapshot 是 NW · 01 一次性聚合返回。KPI 复用 KPISnapshot。
type DashboardSnapshot struct {
	KPI         *KPISnapshot       `json:"kpi"`
	TopSources  []TopThreatSource  `json:"top_sources"`
	AttackTypes []AttackTypeSlice  `json:"attack_types"`
	Heatmap     [][]int64          `json:"heatmap"` // [7][24]，行=周日..周六(0..6)，列=小时
	GeneratedAt time.Time          `json:"generated_at"`
}

// RealtimePoint 是监控大屏一条时间桶（按分钟）。
type RealtimePoint struct {
	Bucket    time.Time `json:"bucket"`
	Requests  int64     `json:"requests"`
	Blocked   int64     `json:"blocked"`
	Challenged int64    `json:"challenged"`
}

type RealtimeSeries struct {
	Points     []RealtimePoint   `json:"points"`
	TopSources []TopThreatSource `json:"top_sources"`
	GeneratedAt time.Time        `json:"generated_at"`
}

// --- Repository ---

// windowToInterval 把前端窗口选择映射到 PG interval 字面量。白名单防注入。
func windowToInterval(window string) string {
	switch window {
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	default:
		return "24 hours"
	}
}

// TopThreatSources 按 src_ip + country 聚合，取前 limit；窗口由 interval 决定。
func (r *Repository) TopThreatSources(ctx context.Context, limit int, interval string) []TopThreatSource {
	out := []TopThreatSource{}
	// interval 来自 windowToInterval 白名单，可安全拼接。
	rows, err := r.pool.Query(ctx, `
		SELECT src_ip, COALESCE(NULLIF(country,''),'未知') AS country, COUNT(*)::BIGINT AS cnt
		  FROM attack_logs
		 WHERE occurred_at >= NOW() - INTERVAL '`+interval+`'
		 GROUP BY src_ip, country
		 ORDER BY cnt DESC
		 LIMIT $1`, limit)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var t TopThreatSource
		if err := rows.Scan(&t.SrcIP, &t.Country, &t.Count); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out
}

// AttackTypeDistribution 按 attack_type 聚合，带 type_label/type_color；窗口由 interval 决定。
func (r *Repository) AttackTypeDistribution(ctx context.Context, limit int, interval string) []AttackTypeSlice {
	out := []AttackTypeSlice{}
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(attack_type,''),'unknown') AS atype,
		       COALESCE(NULLIF(MAX(type_label),''), COALESCE(NULLIF(attack_type,''),'未分类')) AS label,
		       COALESCE(NULLIF(MAX(type_color),''),'#8e84a3') AS color,
		       COUNT(*)::BIGINT AS cnt
		  FROM attack_logs
		 WHERE occurred_at >= NOW() - INTERVAL '`+interval+`'
		 GROUP BY atype
		 ORDER BY cnt DESC
		 LIMIT $1`, limit)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var s AttackTypeSlice
		if err := rows.Scan(&s.Type, &s.Label, &s.Color, &s.Count); err != nil {
			continue
		}
		out = append(out, s)
	}
	return out
}

// WeeklyHeatmap 近 7 天按 (dow, hour) 聚合成 7×24 矩阵。dow: 0=周日..6=周六。
func (r *Repository) WeeklyHeatmap(ctx context.Context) [][]int64 {
	grid := make([][]int64, 7)
	for i := range grid {
		grid[i] = make([]int64, 24)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(DOW  FROM occurred_at)::INT AS dow,
		       EXTRACT(HOUR FROM occurred_at)::INT AS hour,
		       COUNT(*)::BIGINT AS cnt
		  FROM attack_logs
		 WHERE occurred_at >= NOW() - INTERVAL '7 days'
		 GROUP BY dow, hour`)
	if err != nil {
		return grid
	}
	defer rows.Close()
	for rows.Next() {
		var dow, hour int
		var cnt int64
		if err := rows.Scan(&dow, &hour, &cnt); err != nil {
			continue
		}
		if dow >= 0 && dow < 7 && hour >= 0 && hour < 24 {
			grid[dow][hour] = cnt
		}
	}
	return grid
}

// RealtimeBuckets 近 minutes 分钟按分钟分桶，区分 blocked / challenged / 总请求。
// blocked: action 含 block/deny；challenged: action 含 chal/captcha；requests: 全部攻击事件
// （真实总请求需 agent 上报 metrics，此处用攻击事件量作为活动强度的近似，缺指标时不为空）。
func (r *Repository) RealtimeBuckets(ctx context.Context, minutes int) []RealtimePoint {
	out := []RealtimePoint{}
	rows, err := r.pool.Query(ctx, `
		SELECT date_trunc('minute', occurred_at) AS bucket,
		       COUNT(*)::BIGINT AS total,
		       COUNT(*) FILTER (WHERE action ILIKE '%block%' OR action ILIKE '%deny%')::BIGINT AS blocked,
		       COUNT(*) FILTER (WHERE action ILIKE '%chal%' OR action ILIKE '%captcha%')::BIGINT AS challenged
		  FROM attack_logs
		 WHERE occurred_at >= NOW() - make_interval(mins => $1)
		 GROUP BY bucket
		 ORDER BY bucket`, minutes)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var p RealtimePoint
		if err := rows.Scan(&p.Bucket, &p.Requests, &p.Blocked, &p.Challenged); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

// --- Handler ---

// Dashboard GET /monitor/dashboard?window=24h|7d|30d
// window 影响 TOP 威胁源 / 攻击类型分布的时间区间；KPI sparkline 固定 24h（语义如此），
// 热力图固定近 7 天（周×时辰本就是周维度）。
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	interval := windowToInterval(r.URL.Query().Get("window"))
	kpi, err := h.repo.KPISnapshot(ctx)
	if err != nil {
		// KPI 挂了也别整页 500：给一个空 KPI，其余块照常聚合。
		kpi = &KPISnapshot{
			SparkBlocked:   make([]int64, 24),
			SparkRequests:  make([]int64, 24),
			SparkLatency:   make([]float64, 24),
			SparkBlockRate: make([]float64, 24),
			SparkAlerts:    make([]int64, 24),
			GeneratedAt:    time.Now(),
		}
	}
	snap := DashboardSnapshot{
		KPI:         kpi,
		TopSources:  h.repo.TopThreatSources(ctx, 6, interval),
		AttackTypes: h.repo.AttackTypeDistribution(ctx, 8, interval),
		Heatmap:     h.repo.WeeklyHeatmap(ctx),
		GeneratedAt: time.Now(),
	}
	writeJSON(w, http.StatusOK, snap)
}

// RealtimeSeries GET /monitor/realtime-series?minutes=60
func (h *Handler) RealtimeSeries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	minutes := 60
	if v := r.URL.Query().Get("minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 360 {
			minutes = n
		}
	}
	out := RealtimeSeries{
		Points:      h.repo.RealtimeBuckets(ctx, minutes),
		TopSources:  h.repo.TopThreatSources(ctx, 5, "24 hours"),
		GeneratedAt: time.Now(),
	}
	writeJSON(w, http.StatusOK, out)
}
