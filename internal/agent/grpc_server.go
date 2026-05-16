package agent

import (
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
