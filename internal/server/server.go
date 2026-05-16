package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-backend/internal/config"
	"github.com/waf-backend/internal/domain/identity"
	"github.com/waf-backend/internal/middleware"
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

	r.Get("/health", s.healthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		identity.RegisterRoutes(r)
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
