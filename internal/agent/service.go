package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
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
	DBNodeID  int64
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

	dbID, err := s.upsertNode(ctx, req)
	if err != nil {
		// 不能写库 → 拒绝注册，让 agent 端走重连退避；之前『静默 dbID=0 + 继续注册』
		// 会导致后续 heartbeat 走 lookupNodeID 兜底分支或产生孤儿 heartbeats 行。
		slog.Error("upsert node failed", "error", err, "node_id", req.NodeId)
		return &pb.RegisterResponse{
			Accepted: false,
			Message:  "control database unavailable",
		}, nil
	}

	s.mu.Lock()
	s.nodes[req.NodeId] = &NodeState{
		NodeID:   req.NodeId,
		Hostname: req.Hostname,
		IP:       req.IpAddress,
		Version:  req.Version,
		LastSeen: time.Now(),
		Status:   pb.NodeStatus_HEALTHY,
		DBNodeID: dbID,
	}
	s.mu.Unlock()

	if dbID > 0 {
		_, err := s.pool.Exec(ctx, `INSERT INTO heartbeats (node_id, status, cpu_percent, memory_percent, disk_percent)
			VALUES ($1, 'healthy', 0, 0, 0)`, dbID)
		if err != nil {
			slog.Warn("register heartbeat insert failed", "error", err)
		}
	}

	return &pb.RegisterResponse{
		Accepted:             true,
		Message:              "registered",
		AssignedId:           req.NodeId,
		HeartbeatIntervalSec: 10,
	}, nil
}

func (s *Service) upsertNode(ctx context.Context, req *pb.RegisterRequest) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		UPDATE nodes
		SET ip_address = $2,
		    status = 'online',
		    agent_ver = COALESCE(NULLIF($3,''), agent_ver),
		    last_seen = NOW(),
		    updated_at = NOW()
		WHERE hostname = $1 OR name = $1
		RETURNING id`, req.Hostname, req.IpAddress, req.Version).Scan(&id)
	if err == nil && id > 0 {
		return id, nil
	}

	err = s.pool.QueryRow(ctx, `
		INSERT INTO nodes (name, hostname, ip_address, status, agent_ver, last_seen)
		VALUES ($1, $1, $2, 'online', $3, NOW())
		RETURNING id`, req.Hostname, req.IpAddress, req.Version).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert node: %w", err)
	}
	return id, nil
}

func (s *Service) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.mu.Lock()
	ns, ok := s.nodes[req.NodeId]
	if ok {
		ns.LastSeen = time.Now()
		ns.Status = req.Status.GetState()
		ns.Resources = req.Resources
	}
	s.mu.Unlock()

	if !ok || ns.DBNodeID == 0 {
		dbID, err := s.lookupNodeID(ctx, req.NodeId)
		if err != nil {
			slog.Warn("heartbeat lookup node failed", "node_id", req.NodeId, "error", err)
		} else {
			s.mu.Lock()
			if ns, ok = s.nodes[req.NodeId]; ok {
				ns.DBNodeID = dbID
			}
			s.mu.Unlock()
		}
	}

	if req.Resources != nil && ns != nil && ns.DBNodeID > 0 {
		if err := s.persistHeartbeat(ctx, ns.DBNodeID, req); err != nil {
			slog.Warn("heartbeat insert failed", "error", err)
		}
		s.persistMetrics(ctx, ns.DBNodeID, req)
	}

	if ns != nil && ns.DBNodeID > 0 {
		_, err := s.pool.Exec(ctx, `UPDATE nodes SET last_seen = NOW(), status = 'online', updated_at = NOW() WHERE id = $1`, ns.DBNodeID)
		if err != nil {
			slog.Debug("update node last_seen failed", "error", err)
		}
	}

	return &pb.HeartbeatResponse{Ack: true}, nil
}

func (s *Service) lookupNodeID(ctx context.Context, hostname string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM nodes WHERE hostname = $1 OR name = $1 LIMIT 1`, hostname,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("lookup node id: %w", err)
	}
	return id, nil
}

func (s *Service) persistHeartbeat(ctx context.Context, dbNodeID int64, req *pb.HeartbeatRequest) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO heartbeats (node_id, status, cpu_percent, memory_percent, disk_percent)
		VALUES ($1, $2, $3, $4, $5)`,
		dbNodeID, mapHeartbeatStatus(req.Status.GetState()), req.Resources.CpuPercent,
		req.Resources.MemoryPercent, req.Resources.DiskPercent)
	return err
}

func mapHeartbeatStatus(state pb.NodeStatus_State) string {
	switch state {
	case pb.NodeStatus_HEALTHY:
		return "healthy"
	case pb.NodeStatus_DEGRADED:
		return "degraded"
	case pb.NodeStatus_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

func (s *Service) persistMetrics(ctx context.Context, dbNodeID int64, req *pb.HeartbeatRequest) {
	r := req.Resources
	samples := []struct {
		Name  string
		Value float64
		Unit  string
	}{
		{"cpu_percent", r.CpuPercent, "%"},
		{"memory_percent", r.MemoryPercent, "%"},
		{"disk_percent", r.DiskPercent, "%"},
		{"net_connections", float64(r.NetConnections), "count"},
		{"requests_per_second", float64(r.RequestsPerSecond), "rps"},
		{"memory_total_bytes", float64(r.MemoryTotalBytes), "bytes"},
		{"disk_total_bytes", float64(r.DiskTotalBytes), "bytes"},
	}
	for _, m := range samples {
		_, err := s.pool.Exec(ctx,
			`INSERT INTO monitor_metrics (name, value, unit, node_id, recorded_at)
			 VALUES ($1, $2, $3, $4, NOW())`,
			m.Name, m.Value, m.Unit, dbNodeID)
		if err != nil {
			slog.Debug("persist metric failed", "metric", m.Name, "error", err)
		}
	}
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
		fields := parseLogPayload(entry.Payload)
		_, err := s.pool.Exec(ctx, `INSERT INTO attack_logs (node_id, src_ip, dst_ip, attack_type, rule_id, action, payload, occurred_at)
			VALUES ((SELECT id FROM nodes WHERE hostname = $1 LIMIT 1), $2, $3, $4, $5, $6, $7, $8)`,
			entry.NodeId,
			fields["src_ip"],
			fields["host"],
			fields["attack_type"],
			fields["rule_id"],
			"block",
			fields["msg"],
			entry.OccurredAt.AsTime())
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

func parseLogPayload(payload []byte) map[string]string {
	out := make(map[string]string)
	for _, kv := range strings.Split(string(payload), "|") {
		if i := strings.IndexByte(kv, '='); i > 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
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
