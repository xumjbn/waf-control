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

	r.Post("/sites/{id}/deploy", h.DeploySite)
	r.Get("/sites/{id}/deployments", h.ListDeployments)
	r.Post("/sites/{id}/preview", h.PreviewConfig)
}
