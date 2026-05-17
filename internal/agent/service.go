package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/waf-control/proto/agent"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	pb.UnimplementedAgentServiceServer
	pool    *pgxpool.Pool
	mu      sync.RWMutex
	nodes   map[string]*NodeState
	configs chan configEvent
}

type NodeState struct {
	NodeID    string
	Hostname  string
	IP        string
	Version   string
	LastSeen  time.Time
	Status    pb.NodeStatus_State
	Resources *pb.ResourceUsage
}

type configEvent struct {
	nodeID string
	update *pb.ConfigUpdate
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		nodes:   make(map[string]*NodeState),
		configs: make(chan configEvent, 256),
	}
}

func (s *Service) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	slog.Info("agent register", "node_id", req.NodeId, "hostname", req.Hostname, "ip", req.IpAddress)

	s.mu.Lock()
	s.nodes[req.NodeId] = &NodeState{
		NodeID:   req.NodeId,
		Hostname: req.Hostname,
		IP:       req.IpAddress,
		Version:  req.Version,
		LastSeen: time.Now(),
		Status:   pb.NodeStatus_HEALTHY,
	}
	s.mu.Unlock()

	_, err := s.pool.Exec(ctx, `INSERT INTO heartbeats (node_id, status, cpu_percent, memory_percent, disk_percent)
		VALUES ((SELECT id FROM nodes WHERE hostname = $1 LIMIT 1), 'healthy', 0, 0, 0)
		ON CONFLICT DO NOTHING`, req.Hostname)
	if err != nil {
		slog.Warn("register heartbeat insert failed", "error", err)
	}

	return &pb.RegisterResponse{
		Accepted:            true,
		Message:             "registered",
		AssignedId:          req.NodeId,
		HeartbeatIntervalSec: 10,
	}, nil
}

func (s *Service) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.mu.Lock()
	if ns, ok := s.nodes[req.NodeId]; ok {
		ns.LastSeen = time.Now()
		ns.Status = req.Status.GetState()
		ns.Resources = req.Resources
	}
	s.mu.Unlock()

	if req.Resources != nil {
		_, err := s.pool.Exec(ctx, `INSERT INTO heartbeats (node_id, status, cpu_percent, memory_percent, disk_percent)
			VALUES ((SELECT id FROM nodes WHERE hostname = $1 LIMIT 1), $2, $3, $4, $5)`,
			req.NodeId, req.Status.GetState().String(), req.Resources.CpuPercent,
			req.Resources.MemoryPercent, req.Resources.DiskPercent)
		if err != nil {
			slog.Warn("heartbeat insert failed", "error", err)
		}
	}

	return &pb.HeartbeatResponse{Ack: true}, nil
}

func (s *Service) PushConfig(req *pb.ConfigRequest, stream pb.AgentService_PushConfigServer) error {
	slog.Info("config stream opened", "node_id", req.NodeId)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case evt := <-s.configs:
			if evt.nodeID == req.NodeId || evt.nodeID == "*" {
				if err := stream.Send(evt.update); err != nil {
					return fmt.Errorf("send config update: %w", err)
				}
			}
		}
	}
}

func (s *Service) ReportMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	slog.Debug("metrics received", "node_id", req.NodeId, "count", len(req.Metrics))
	return &pb.MetricsResponse{Ack: true}, nil
}

func (s *Service) ReportLogs(stream pb.AgentService_ReportLogsServer) error {
	var count int64
	for {
		entry, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.LogResponse{ReceivedCount: count, Ack: true})
		}
		if err != nil {
			return err
		}

		if err := s.persistLog(stream.Context(), entry); err != nil {
			slog.Warn("persist log failed", "type", entry.Type, "error", err)
		}
		count++
	}
}

func (s *Service) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	slog.Info("execute command", "node_id", req.NodeId, "type", req.Type, "command_id", req.CommandId)

	return &pb.CommandResponse{
		CommandId: req.CommandId,
		Success:   true,
		Message:   "command accepted",
	}, nil
}

// BroadcastConfig pushes a config update to all connected agents or a specific node.
func (s *Service) BroadcastConfig(nodeID string, configType pb.ConfigUpdate_ConfigType, payload []byte) {
	s.configs <- configEvent{
		nodeID: nodeID,
		update: &pb.ConfigUpdate{
			Version:   fmt.Sprintf("%d", time.Now().UnixMilli()),
			Type:      configType,
			Payload:   payload,
			Timestamp: timestamppb.Now(),
		},
	}
}

func (s *Service) persistLog(ctx context.Context, entry *pb.LogEntry) error {
	switch entry.Type {
	case pb.LogEntry_ATTACK:
		_, err := s.pool.Exec(ctx, `INSERT INTO attack_logs (node_id, src_ip, action, occurred_at)
			VALUES ((SELECT id FROM nodes WHERE hostname = $1 LIMIT 1), '', 'block', $2)`,
			entry.NodeId, entry.OccurredAt.AsTime())
		return err
	case pb.LogEntry_ANTIVIRUS:
		_, err := s.pool.Exec(ctx, `INSERT INTO antivirus_logs (node_id, file_name, virus_name, action, occurred_at)
			VALUES ((SELECT id FROM nodes WHERE hostname = $1 LIMIT 1), '', '', 'block', $2)`,
			entry.NodeId, entry.OccurredAt.AsTime())
		return err
	case pb.LogEntry_ANTITAMPER:
		_, err := s.pool.Exec(ctx, `INSERT INTO antitamper_logs (node_id, file_path, change_type, action, occurred_at)
			VALUES ((SELECT id FROM nodes WHERE hostname = $1 LIMIT 1), '', '', 'block', $2)`,
			entry.NodeId, entry.OccurredAt.AsTime())
		return err
	default:
		return nil
	}
}

// GetConnectedNodes returns a snapshot of currently connected nodes.
func (s *Service) GetConnectedNodes() []NodeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes := make([]NodeState, 0, len(s.nodes))
	for _, ns := range s.nodes {
		nodes = append(nodes, *ns)
	}
	return nodes
}

// ReportDeployResult handles deploy result reported back from an agent.
func (s *Service) ReportDeployResult(ctx context.Context, req *pb.DeployResult) (*pb.DeployResultResponse, error) {
	slog.Info("deploy result", "node", req.NodeId, "version", req.Version, "success", req.Success, "msg", req.Message)

	if req.Success {
		_, err := s.pool.Exec(ctx,
			`UPDATE deployment_node_status SET status = 'success', message = $1, applied_at = $2
			 WHERE node_hostname = $3 AND status = 'pending'`,
			req.Message, req.AppliedAt.AsTime(), req.NodeId)
		if err != nil {
			slog.Warn("update deploy status failed", "error", err)
		}
	} else {
		_, err := s.pool.Exec(ctx,
			`UPDATE deployment_node_status SET status = 'failed', message = $1
			 WHERE node_hostname = $2 AND status = 'pending'`,
			req.Message, req.NodeId)
		if err != nil {
			slog.Warn("update deploy status failed", "error", err)
		}
	}

	return &pb.DeployResultResponse{Ack: true}, nil
}
