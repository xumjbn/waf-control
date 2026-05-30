package monitor

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/monitor", func(r chi.Router) {
		r.Get("/kpi", h.KPI)                       // NW · 02 仪表盘 KPI snapshot
		r.Get("/dashboard", h.Dashboard)           // NW · 01 总览：KPI + TOP + 类型分布 + 热力图
		r.Get("/realtime-series", h.RealtimeSeries) // NW · 02 大屏：近 N 分钟 req/block/chal 时序
		r.Get("/cluster-resources", h.ClusterResourcesHandler) // NW · 02 集群 CPU/内存/带宽/RPS 水位
		r.Get("/metric", h.ListMetrics)
		r.Get("/metricspec", h.ListMetricSpecs)
		r.Get("/metricspec/{id}", h.GetMetricSpec)
		r.Put("/realtime", h.QueryRealtime)
		r.Put("/history", h.QueryHistory)
	})
}
