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
		r.Get("/{id}", h.GetVIP)
		r.Put("/{id}", h.UpdateVIP)
		r.Delete("/{id}", h.DeleteVIP)
	})

	r.Route("/lb/pools", func(r chi.Router) {
		r.Get("/", h.ListPools)
		r.Post("/", h.CreatePool)
		r.Get("/{id}", h.GetPool)
		r.Put("/{id}", h.UpdatePool)
		r.Delete("/{id}", h.DeletePool)
		r.Get("/{poolId}/members", h.ListMembers)
		r.Post("/{poolId}/members", h.CreateMember)
		r.Get("/{poolId}/members/{memberId}", h.GetMember)
		r.Put("/{poolId}/members/{memberId}", h.UpdateMember)
		r.Delete("/{poolId}/members/{memberId}", h.DeleteMember)
		r.Get("/{poolId}/health-monitor", h.GetHealthMonitor)
		r.Post("/{poolId}/health-monitor", h.CreateHealthMonitor)
		r.Put("/{poolId}/health-monitor", h.UpdateHealthMonitor)
		r.Delete("/{poolId}/health-monitor", h.DeleteHealthMonitor)
	})
}
