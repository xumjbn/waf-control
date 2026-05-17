package instancemgmt

import (
	"github.com/go-chi/chi/v5"
	"github.com/waf-control/internal/agent"
)

func RegisterRoutes(r chi.Router, agentSvc *agent.Service) {
	h := NewHandler(agentSvc)

	r.Get("/instances", h.ListInstances)
	r.Get("/instances/detail", h.GetInstance)
}
