package deploy

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes adds deploy routes to the router.
// The caller must provide a ConfigSender (e.g., *agent.Service) and a NodeLister.
func RegisterRoutes(r chi.Router, nodes NodeLister, sender ConfigSender) {
	svc := NewService(nodes, sender)
	h := NewHandler(svc)

	r.Route("/deploy", func(r chi.Router) {
		r.Post("/nginx", h.DeployNginx)
		r.Post("/modsec", h.DeployModsec)
		r.Post("/all", h.DeployAll)
	})
}
