package security

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/security_management/auth_hosts", func(r chi.Router) {
		r.Get("/", h.ListAuthHosts)
		r.Post("/", h.CreateAuthHost)
		r.Get("/{id}", h.GetAuthHost)
		r.Put("/{id}", h.UpdateAuthHost)
		r.Delete("/{id}", h.DeleteAuthHost)
	})

	r.Get("/security_management/auth_host_cfg", h.GetAuthHostConfig)
	r.Put("/security_management/auth_host_cfg", h.UpdateAuthHostConfig)

	r.Get("/system/query_pswd_status/{userId}", h.GetPasswordStatus)
}
