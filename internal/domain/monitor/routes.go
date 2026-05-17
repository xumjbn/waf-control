package monitor

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Get("/statistic_trend/top_sites_info", h.TopSites)

	r.Route("/monitor", func(r chi.Router) {
		r.Get("/attack/src-ips-top", h.TopSrcIPs)
		r.Get("/attack/severity", h.SeverityStats)

		r.Get("/metric", h.ListMetrics)
		r.Get("/metricspec", h.ListMetricSpecs)
		r.Get("/metricspec/{id}", h.GetMetricSpec)
		r.Put("/realtime", h.QueryRealtime)
		r.Put("/history", h.QueryHistory)
	})
}
