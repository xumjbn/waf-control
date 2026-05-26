package instancemgmt

// metrics_trend.go —— 实例时序指标查询
//
// 数据来源：control 在每次 gRPC heartbeat 落地时已经写入两张表：
//   · heartbeats(node_id, status, cpu_percent, memory_percent, disk_percent, reported_at)
//     由 service.go:persistHeartbeat 写入
//   · monitor_metrics(name, value, unit, node_id, recorded_at)
//     由 service.go:persistMetrics 按 metric name 拆分写入（cpu_percent / memory_percent /
//     disk_percent / net_connections / requests_per_second / ...）
//
// 端点：GET /api/v1/instances/{nodeId}/metrics-trend?hours=24&metric=requests_per_second
// 返回：{ metric, points: [{t: RFC3339, v: float64}], buckets, hours }
// 点数：按 hours 自动选 bucket（24h=每小时 / 7d=每 4 小时 / 1h=每分钟），
// 用 PG 的 date_trunc/time_bucket 聚合平均值。

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/agent"
)

type TrendPoint struct {
	T string  `json:"t"` // RFC3339 时间戳
	V float64 `json:"v"`
}

type TrendResponse struct {
	NodeID  string       `json:"node_id"`
	Metric  string       `json:"metric"`
	Hours   int          `json:"hours"`
	Bucket  string       `json:"bucket"` // 'minute' / 'hour' / '4 hours'
	Points  []TrendPoint `json:"points"`
}

type TrendHandler struct {
	pool     *pgxpool.Pool
	agentSvc *agent.Service
}

func NewTrendHandler(pool *pgxpool.Pool, agentSvc *agent.Service) *TrendHandler {
	return &TrendHandler{pool: pool, agentSvc: agentSvc}
}

// 已知 metric → PG 表 + 列。优先 heartbeats（cpu/mem/disk），其他走 monitor_metrics by name。
var heartbeatMetrics = map[string]string{
	"cpu_percent":    "cpu_percent",
	"memory_percent": "memory_percent",
	"disk_percent":   "disk_percent",
}

func (h *TrendHandler) lookupDBNodeID(ctx context.Context, nodeID string) (int64, error) {
	// 先看 agent service 在线节点
	for _, ns := range h.agentSvc.GetConnectedNodes() {
		if ns.NodeID == nodeID || ns.Hostname == nodeID {
			return ns.DBNodeID, nil
		}
	}
	// 离线节点：从 nodes 表按 hostname/node_id 列查
	var id int64
	err := h.pool.QueryRow(ctx,
		`SELECT id FROM nodes WHERE hostname = $1 OR node_id = $1 LIMIT 1`, nodeID,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("node %q not found", nodeID)
		}
		return 0, err
	}
	return id, nil
}

// 选 bucket：hours <= 2 → 1 分钟；hours <= 24 → 1 小时；hours <= 48 → 2 小时；其他 → 4 小时
func pickBucket(hours int) (interval string, label string) {
	switch {
	case hours <= 2:
		return "1 minute", "minute"
	case hours <= 24:
		return "1 hour", "hour"
	case hours <= 48:
		return "2 hours", "2 hours"
	default:
		return "4 hours", "4 hours"
	}
}

func (h *TrendHandler) GetTrend(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeId")
	if nodeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id required"})
		return
	}

	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 {
		hours = 24
	}
	if hours > 720 {
		hours = 720 // 最多 30 天，防止扫表
	}

	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "requests_per_second"
	}

	dbID, err := h.lookupDBNodeID(r.Context(), nodeID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	interval, bucketLabel := pickBucket(hours)
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	var (
		points []TrendPoint
		qerr   error
	)
	if col, ok := heartbeatMetrics[metric]; ok {
		// heartbeats 表更稠密（每次 heartbeat 一行）
		points, qerr = h.queryHeartbeats(r.Context(), dbID, col, since, interval)
	} else {
		// 其他 metric 从 monitor_metrics by name 取
		points, qerr = h.queryMetrics(r.Context(), dbID, metric, since, interval)
	}
	if qerr != nil {
		slog.Error("metrics trend query", "node_id", nodeID, "metric", metric, "err", qerr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": qerr.Error()})
		return
	}

	writeJSON(w, http.StatusOK, TrendResponse{
		NodeID: nodeID,
		Metric: metric,
		Hours:  hours,
		Bucket: bucketLabel,
		Points: points,
	})
}

func (h *TrendHandler) queryHeartbeats(
	ctx context.Context,
	dbNodeID int64,
	col string,
	since time.Time,
	interval string,
) ([]TrendPoint, error) {
	// 用 to_timestamp(floor(...)) 做 bucket：按 interval 截断到时间桶，再 AVG。
	// 避免使用 timescaledb 的 time_bucket（项目用纯 PG）。
	q := fmt.Sprintf(`
		SELECT
			to_timestamp(floor(extract(epoch FROM reported_at) / extract(epoch FROM interval '%s'))
				* extract(epoch FROM interval '%s')) AS bucket,
			AVG(COALESCE(%s, 0))::float8 AS avg_value
		FROM heartbeats
		WHERE node_id = $1 AND reported_at >= $2
		GROUP BY bucket
		ORDER BY bucket ASC`, interval, interval, col)

	rows, err := h.pool.Query(ctx, q, dbNodeID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TrendPoint, 0, 64)
	for rows.Next() {
		var t time.Time
		var v float64
		if err := rows.Scan(&t, &v); err != nil {
			return nil, err
		}
		out = append(out, TrendPoint{T: t.UTC().Format(time.RFC3339), V: v})
	}
	return out, rows.Err()
}

func (h *TrendHandler) queryMetrics(
	ctx context.Context,
	dbNodeID int64,
	name string,
	since time.Time,
	interval string,
) ([]TrendPoint, error) {
	q := fmt.Sprintf(`
		SELECT
			to_timestamp(floor(extract(epoch FROM recorded_at) / extract(epoch FROM interval '%s'))
				* extract(epoch FROM interval '%s')) AS bucket,
			AVG(value)::float8 AS avg_value
		FROM monitor_metrics
		WHERE node_id = $1 AND name = $2 AND recorded_at >= $3
		GROUP BY bucket
		ORDER BY bucket ASC`, interval, interval)

	rows, err := h.pool.Query(ctx, q, dbNodeID, name, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TrendPoint, 0, 64)
	for rows.Next() {
		var t time.Time
		var v float64
		if err := rows.Scan(&t, &v); err != nil {
			return nil, err
		}
		out = append(out, TrendPoint{T: t.UTC().Format(time.RFC3339), V: v})
	}
	return out, rows.Err()
}
