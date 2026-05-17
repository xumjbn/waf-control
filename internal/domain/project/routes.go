package project

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes 在调用方已应用 AuthMiddleware 的子路由器上挂载 /projects 路由。
func RegisterRoutes(r chi.Router, pool *pgxpool.Pool, users UserLister) {
	repo := NewRepository(pool)
	h := NewHandler(repo, users)

	r.Route("/projects", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/users", h.ListProjectUsers)
		r.Put("/{project_id}/users/{user_id}/roles/{role_id}", h.AssignUserRole)
		r.Delete("/{project_id}/users/{user_id}/roles/{role_id}", h.RevokeUserRole)
	})
}
