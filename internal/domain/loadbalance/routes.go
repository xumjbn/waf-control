package loadbalance

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/lb/vips", func(r chi.Router) {
		r.Get("/", h.ListVIPs)
		r.Post("/", h.CreateVIP)
		r.Put("/{id}", h.UpdateVIP)
		r.Delete("/{id}", h.DeleteVIP)
	})

	r.Route("/lb/pools", func(r chi.Router) {
		r.Get("/", h.ListPools)
		r.Post("/", h.CreatePool)
		r.Put("/{id}", h.UpdatePool)
		r.Delete("/{id}", h.DeletePool)
		r.Get("/{poolId}/members", h.ListMembers)
		r.Post("/{poolId}/members", h.CreateMember)
		r.Delete("/{poolId}/members/{memberId}", h.DeleteMember)
		r.Get("/{poolId}/health-monitor", h.GetHealthMonitor)
		r.Post("/{poolId}/health-monitor", h.CreateHealthMonitor)
		r.Delete("/{poolId}/health-monitor", h.DeleteHealthMonitor)
	})
}
