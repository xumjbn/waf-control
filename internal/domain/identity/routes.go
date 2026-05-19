package identity

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/config"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool, cfg config.AuthConfig) *Service {
	// Self-heal schema drift: when a deploy lands the new code before its
	// matching migrations have run, GetUserByUsername would crash at the
	// SELECT (role_key/avatar/project missing) and every login returns 500.
	// EnsureSchema makes the ALTERs idempotent and runs them on each boot,
	// so a fresh code roll-out always lands on a compatible DB even if the
	// migration runner is late.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := EnsureSchema(ctx, pool); err != nil {
		slog.Warn("identity ensure schema failed; continuing", "error", err)
	}

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

// RegisterV3Aliases 挂载 OpenStack Keystone v3 风格的别名路由，供前端 user 页面调用。
// 路径：/users、/roles、/users/{id}/roles、/roles/{rid}/users/{uid}。
// 所有路由要求 admin 角色，因此调用方必须在已应用了 AuthMiddleware 的子路由器中调用。
func RegisterV3Aliases(r chi.Router, svc *Service) {
	h := NewHandler(svc)
	r.Group(func(r chi.Router) {
		r.Use(requireRole("admin"))

		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.ListUsersV3)
			r.Post("/", h.CreateUserV3)
			r.Get("/{id}", h.GetUserV3)
			r.Put("/{id}", h.UpdateUserV3)
			r.Delete("/{id}", h.DeleteUserV3)
			r.Get("/{user_id}/roles", h.ListUserRolesV3)
		})

		r.Route("/roles", func(r chi.Router) {
			r.Get("/", h.ListRolesV3)
			r.Post("/", h.CreateRoleV3)
			r.Get("/{id}", h.GetRoleV3)
			r.Put("/{id}", h.UpdateRoleV3)
			r.Delete("/{id}", h.DeleteRoleV3)
			r.Put("/{role_id}/users/{user_id}", h.AssignUserRoleV3)
			r.Delete("/{role_id}/users/{user_id}", h.RevokeUserRoleV3)
		})
	})
}
