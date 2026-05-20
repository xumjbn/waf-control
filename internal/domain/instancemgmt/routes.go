package instancemgmt

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/agent"
)

// RegisterRoutes — backwards-compatible signature (used by older call
// sites in server.go). Mounts agent-driven instance routes only.
func RegisterRoutes(r chi.Router, agentSvc *agent.Service) {
	h := NewHandler(agentSvc)
	r.Get("/instances", h.ListInstances)
	r.Get("/instances/detail", h.GetInstance)
	r.Post("/instances/restart", h.RestartInstance)
	r.Post("/instances/register-intent", h.RegisterNodeIntent)
}

// RegisterRoutesWithDB additionally mounts /clusters CRUD (needs Postgres).
// Server bootstrap should call this when pool is available.
func RegisterRoutesWithDB(r chi.Router, agentSvc *agent.Service, pool *pgxpool.Pool) {
	RegisterRoutes(r, agentSvc)
	if pool == nil {
		return
	}
	ch := NewClusterHandler(NewClusterStore(pool))
	r.Route("/clusters", func(r chi.Router) {
		r.Get("/", ch.List)
		r.Post("/", ch.Create)
		r.Get("/{id}", ch.Get)
		r.Put("/{id}", ch.Update)
		r.Delete("/{id}", ch.Delete)
		r.Put("/{id}/nodes/{nodeId}", ch.AssignNode)
		r.Delete("/{id}/nodes/{nodeId}", ch.RemoveNode)
	})
}
