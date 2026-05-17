package flow

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/flow-logs", func(r chi.Router) {
		r.Get("/", h.ListFlowLogs)
		r.Get("/{id}", h.GetFlowLog)
		r.Get("/top-src-ips", h.TopSrcIPs)
		r.Get("/top-dst-ips", h.TopDstIPs)
		r.Get("/protocols", h.ProtocolDistribution)
	})

	r.Route("/flow-saved-queries", func(r chi.Router) {
		r.Get("/", h.ListSavedQueries)
		r.Post("/", h.CreateSavedQuery)
		r.Delete("/{id}", h.DeleteSavedQuery)
	})

	r.Route("/flow-monitor", func(r chi.Router) {
		r.Get("/records", h.MonitorRecords)
	})
}
