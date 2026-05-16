package system

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	repo := NewRepository(pool)
	h := NewHandler(repo)

	r.Route("/system/settings", func(r chi.Router) {
		r.Get("/", h.ListSettings)
		r.Put("/", h.UpsertSetting)
		r.Delete("/{key}", h.DeleteSetting)
	})

	r.Route("/system/licenses", func(r chi.Router) {
		r.Get("/", h.ListLicenses)
		r.Post("/", h.CreateLicense)
		r.Post("/{id}/activate", h.ActivateLicense)
		r.Delete("/{id}", h.DeleteLicense)
	})

	r.Route("/system/upgrades", func(r chi.Router) {
		r.Get("/", h.ListUpgrades)
		r.Post("/", h.CreateUpgrade)
		r.Post("/{id}/trigger", h.TriggerUpgrade)
		r.Delete("/{id}", h.DeleteUpgrade)
	})
}
