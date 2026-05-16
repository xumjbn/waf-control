package identity

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Route("/identity", func(r chi.Router) {
		r.Post("/login", loginHandler)
		r.Post("/logout", logoutHandler)
		r.Get("/me", meHandler)

		r.Route("/users", func(r chi.Router) {
			r.Get("/", listUsersHandler)
			r.Post("/", createUserHandler)
			r.Get("/{id}", getUserHandler)
			r.Put("/{id}", updateUserHandler)
			r.Delete("/{id}", deleteUserHandler)
		})

		r.Route("/roles", func(r chi.Router) {
			r.Get("/", listRolesHandler)
			r.Post("/", createRoleHandler)
			r.Get("/{id}", getRoleHandler)
			r.Put("/{id}", updateRoleHandler)
			r.Delete("/{id}", deleteRoleHandler)
		})
	})
}

func placeholder(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

var (
	loginHandler       = placeholder
	logoutHandler      = placeholder
	meHandler          = placeholder
	listUsersHandler   = placeholder
	createUserHandler  = placeholder
	getUserHandler     = placeholder
	updateUserHandler  = placeholder
	deleteUserHandler  = placeholder
	listRolesHandler   = placeholder
	createRoleHandler  = placeholder
	getRoleHandler     = placeholder
	updateRoleHandler  = placeholder
	deleteRoleHandler  = placeholder
)
