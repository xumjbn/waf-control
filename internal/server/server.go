package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/waf-control/internal/agent"
	"github.com/waf-control/internal/config"
	"github.com/waf-control/internal/domain/acl"
	"github.com/waf-control/internal/domain/alert"
	"github.com/waf-control/internal/domain/deploymgmt"
	"github.com/waf-control/internal/domain/device"
	"github.com/waf-control/internal/domain/instancemgmt"
	"github.com/waf-control/internal/domain/ha"
	"github.com/waf-control/internal/domain/identity"
	"github.com/waf-control/internal/domain/loadbalance"
	"github.com/waf-control/internal/domain/flow"
	"github.com/waf-control/internal/domain/logs"
	"github.com/waf-control/internal/domain/monitor"
	"github.com/waf-control/internal/domain/network"
	"github.com/waf-control/internal/domain/node"
	"github.com/waf-control/internal/domain/operate"
	"github.com/waf-control/internal/domain/policy"
	"github.com/waf-control/internal/domain/project"
	"github.com/waf-control/internal/domain/reports"
	"github.com/waf-control/internal/domain/security"
	"github.com/waf-control/internal/domain/site"
	"github.com/waf-control/internal/domain/system"
	"github.com/waf-control/internal/middleware"
)

type Server struct {
	cfg      *config.Config
	router   chi.Router
	pool     *pgxpool.Pool
	agentSrv *agent.Server
}

func New(cfg *config.Config, pool *pgxpool.Pool, agentSrv *agent.Server) *Server {
	s := &Server{
		cfg:      cfg,
		pool:     pool,
		agentSrv: agentSrv,
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
	// 全局请求体上限 1MiB —— 防止恶意大 body 耗内存。业务负载远小于此。
	r.Use(middleware.MaxBody(middleware.DefaultMaxBodyBytes))

	userExtractor := func(ctx context.Context) *middleware.OplogUser {
		claims := identity.GetClaimsFromContext(ctx)
		if claims == nil {
			return nil
		}
		return &middleware.OplogUser{UserID: claims.UserID, Username: claims.Username}
	}
	r.Use(middleware.OperationLog(s.pool, userExtractor))

	r.Get("/health", s.healthCheck)

	// Swagger API 文档
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/api/v1", func(r chi.Router) {
		identitySvc := identity.RegisterRoutes(r, s.pool, s.cfg.Auth)

		r.Group(func(r chi.Router) {
			r.Use(identity.AuthMiddleware(identitySvc))
			// ScopeMiddleware 必须紧跟 Auth：从 project_user_roles 取当前用户能访问的
			// project_id 集合，放进 ctx，让 sites/policies 这类 repository 按租户隔离。
			r.Use(identity.ScopeMiddleware(s.pool))
			device.RegisterRoutes(r, s.pool)
			flow.RegisterRoutes(r, s.pool)
			monitor.RegisterRoutes(r, s.pool)
			network.RegisterRoutes(r, s.pool)
			node.RegisterRoutes(r, s.pool)
			site.RegisterRoutes(r, s.pool)
			policy.RegisterRoutes(r, s.pool)
			loadbalance.RegisterRoutes(r, s.pool)
			acl.RegisterRoutes(r, s.pool)
			alert.RegisterRoutes(r, s.pool)
			ha.RegisterRoutes(r, s.pool)
			operate.RegisterRoutes(r, s.pool)
			reports.RegisterRoutes(r, s.pool)
			security.RegisterRoutes(r, s.pool)
			system.RegisterRoutes(r, s.pool)
			// logs：注入 acl 仓库，启用 /logs/attack/{id}/ban、whitelist 联动
			logs.RegisterRoutesWithACL(r, s.pool, acl.NewRepository(s.pool))

			// Keystone v3 风格的别名路由 (/users, /roles, /projects)
			// 适配 waf-admin 前端 user 模块（OpenStack 风格 wrap 请求/响应）
			identity.RegisterV3Aliases(r, identitySvc)
			project.RegisterRoutes(r, s.pool, identitySvc)

			s.registerDeployMgmtRoutes(r)
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

func (s *Server) registerDeployMgmtRoutes(r chi.Router) {
	if s.agentSrv == nil || s.pool == nil {
		return
	}
	siteRepo := site.NewRepository(s.pool)
	deploymgmt.RegisterRoutes(r, s.pool, siteRepo, s.agentSrv.Service())
	instancemgmt.RegisterRoutesWithDB(r, s.agentSrv.Service(), s.pool)
}
