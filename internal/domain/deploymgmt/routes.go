package deploymgmt

import (
	"github.com/go-chi/chi/v5"

	"github.com/waf-control/internal/agent"
	"github.com/waf-control/internal/domain/site"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool, siteRepo *site.Repository, agentSvc *agent.Service) {
	repo := NewRepository(pool)
	h := NewHandler(repo, siteRepo, agentSvc)

	r.Get("/deployments/{id}", h.GetDeployment)

	r.Route("/sites/{id}", func(r chi.Router) {
		r.Post("/deploy", h.DeploySite)
		r.Get("/deployments", h.ListDeployments)
		r.Post("/preview", h.PreviewConfig)
	})
}
