package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/config"
	"github.com/waf-control/internal/domain/acl"
	"github.com/waf-control/internal/domain/device"
	"github.com/waf-control/internal/domain/ha"
	"github.com/waf-control/internal/domain/identity"
	"github.com/waf-control/internal/domain/loadbalance"
	"github.com/waf-control/internal/domain/logs"
	"github.com/waf-control/internal/domain/network"
	"github.com/waf-control/internal/domain/node"
	"github.com/waf-control/internal/domain/policy"
	"github.com/waf-control/internal/domain/site"
	"github.com/waf-control/internal/domain/system"
	"github.com/waf-control/internal/middleware"
)

type Server struct {
	cfg    *config.Config
	router chi.Router
	pool   *pgxpool.Pool
}

func New(cfg *config.Config, pool *pgxpool.Pool) *Server {
	s := &Server{
		cfg:  cfg,
		pool: pool,
	}
	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	r.Use(middleware.Recovery)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(middleware.CORS()))

	userExtractor := func(ctx context.Context) *middleware.OplogUser {
		claims := identity.GetClaimsFromContext(ctx)
		if claims == nil {
			return nil
		}
		return &middleware.OplogUser{UserID: claims.UserID, Username: claims.Username}
	}
	r.Use(middleware.OperationLog(s.pool, userExtractor))

	r.Get("/health", s.healthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		identitySvc := identity.RegisterRoutes(r, s.pool, s.cfg.Auth)

		r.Group(func(r chi.Router) {
			r.Use(identity.AuthMiddleware(identitySvc))
			device.RegisterRoutes(r, s.pool)
			node.RegisterRoutes(r, s.pool)
			network.RegisterRoutes(r, s.pool)
			site.RegisterRoutes(r, s.pool)
			policy.RegisterRoutes(r, s.pool)
			loadbalance.RegisterRoutes(r, s.pool)
			acl.RegisterRoutes(r, s.pool)
			ha.RegisterRoutes(r, s.pool)
			system.RegisterRoutes(r, s.pool)
			logs.RegisterRoutes(r, s.pool)
		})
	})

	s.router = r
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
