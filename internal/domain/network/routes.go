package network

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/nodes/{nodeId}/interfaces", func(r chi.Router) {
		r.Get("/", h.ListInterfaces)
		r.Post("/", h.CreateInterface)
		r.Put("/{id}", h.UpdateInterface)
		r.Put("/{id}/enable", h.EnableInterface)
		r.Put("/{id}/disable", h.DisableInterface)
		r.Delete("/{id}", h.DeleteInterface)
	})

	r.Route("/nodes/{nodeId}/bridges", func(r chi.Router) {
		r.Get("/", h.ListBridges)
		r.Post("/", h.CreateBridge)
		r.Put("/{id}", h.UpdateBridge)
		r.Post("/{id}/slave", h.AddBridgeSlave)
		r.Delete("/{id}/slave/{slaveId}", h.DelBridgeSlave)
		r.Delete("/{id}", h.DeleteBridge)
	})

	r.Route("/nodes/{nodeId}/bonds", func(r chi.Router) {
		r.Get("/", h.ListBonds)
		r.Post("/", h.CreateBond)
		r.Put("/{id}", h.UpdateBond)
		r.Post("/{id}/slave", h.AddBondSlave)
		r.Delete("/{id}/slave/{slaveId}", h.DelBondSlave)
		r.Delete("/{id}", h.DeleteBond)
	})

	r.Route("/nodes/{nodeId}/routes", func(r chi.Router) {
		r.Get("/", h.ListRoutes)
		r.Post("/", h.CreateRoute)
		r.Put("/{id}", h.UpdateRoute)
		r.Delete("/{id}", h.DeleteRoute)
	})
}
