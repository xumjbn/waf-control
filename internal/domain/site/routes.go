package site

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/agent"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	registerWith(r, pool, nil)
}

func RegisterRoutesWithAgent(r chi.Router, pool *pgxpool.Pool, agentSvc *agent.Service) {
	registerWith(r, pool, agentSvc)
}

func registerWith(r chi.Router, pool *pgxpool.Pool, agentSvc *agent.Service) {
	repo := NewRepository(pool)
	h := NewHandlerWithAgent(repo, agentSvc)

	r.Route("/sites", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/devices", h.ListDevices)
		r.Post("/{id}/devices", h.BindDevice)
		r.Delete("/{id}/devices/{deviceId}", h.UnbindDevice)
	})
}
