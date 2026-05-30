package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/waf-control/proto/agent"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	service    *Service
	port       int
}

func NewServer(pool *pgxpool.Pool, port int) *Server {
	svc := NewService(pool)
	// 兜底 nodes.engine 列（migration 000028）—— baseline 路径下迁移被标记跳过，
	// 这里幂等补列，避免 Register upsertNode 因缺列报 42703。
	if pool != nil {
		if _, err := pool.Exec(context.Background(),
			`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS engine VARCHAR(16) NOT NULL DEFAULT 'nginx'`); err != nil {
			slog.Warn("ensure nodes.engine column failed", "error", err)
		}
	}
	grpcSrv := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcSrv, svc)

	return &Server{
		grpcServer: grpcSrv,
		service:    svc,
		port:       port,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}
	slog.Info("gRPC agent server started", "port", s.port)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *Server) Service() *Service {
	return s.service
}
