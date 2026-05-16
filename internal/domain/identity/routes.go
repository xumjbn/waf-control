package identity

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/config"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool, cfg config.AuthConfig) *Service {
	repo := NewRepository(pool)
	svc := NewService(repo, cfg)
	h := NewHandler(svc)

	r.Route("/identity", func(r chi.Router) {
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware(svc))
			r.Get("/me", h.Me)

			r.Route("/users", func(r chi.Router) {
				r.Use(requireRole("admin"))
				r.Get("/", h.ListUsers)
				r.Post("/", h.CreateUser)
				r.Get("/{id}", h.GetUser)
				r.Put("/{id}", h.UpdateUser)
				r.Delete("/{id}", h.DeleteUser)
			})

			r.Route("/roles", func(r chi.Router) {
				r.Use(requireRole("admin"))
				r.Get("/", h.ListRoles)
				r.Post("/", h.CreateRole)
				r.Get("/{id}", h.GetRole)
				r.Put("/{id}", h.UpdateRole)
				r.Delete("/{id}", h.DeleteRole)
			})
		})
	})

	return svc
}
