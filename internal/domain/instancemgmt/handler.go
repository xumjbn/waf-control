package instancemgmt

import (
	"encoding/json"
	"net/http"
	"time"

	pb "github.com/waf-control/proto/agent"

	"github.com/waf-control/internal/agent"
)

type InstanceInfo struct {
	NodeID         string  `json:"node_id"`
	ID             string  `json:"id"`
	Hostname       string  `json:"hostname"`
	Name           string  `json:"name"`
	IP             string  `json:"ip"`
	Version        string  `json:"version"`
	Status         string  `json:"status"`
	CPUPercent     float64 `json:"cpu_percent"`
	MemPercent     float64 `json:"memory_percent"`
	DiskPercent    float64 `json:"disk_percent"`
	NetConnections int64   `json:"net_connections"`
	RPS            int64   `json:"rps"`
	NetworkIO      string  `json:"network_io"`
	MemoryTotal    int64   `json:"memory_total"`
	DiskTotal      int64   `json:"disk_total"`
	Engine         string  `json:"engine"` // nginx / openresty / safeline
	LastSeen       string  `json:"last_seen"`
}

const offlineThreshold = 30 * time.Second

type Handler struct {
	agentSvc *agent.Service
}

func NewHandler(agentSvc *agent.Service) *Handler {
	return &Handler{agentSvc: agentSvc}
}

// ListInstances 列出所有在线实例
func (h *Handler) ListInstances(w http.ResponseWriter, r *http.Request) {
	nodes := h.agentSvc.GetConnectedNodes()

	instances := make([]InstanceInfo, 0, len(nodes))
	for _, ns := range nodes {
		instances = append(instances, buildInstanceInfo(ns))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"instances": instances})
}

// GetInstance 获取单个实例详情
func (h *Handler) GetInstance(w http.ResponseWriter, r *http.Request) {
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname is required"})
		return
	}

	nodes := h.agentSvc.GetConnectedNodes()
	for _, ns := range nodes {
		if ns.Hostname == hostname {
			writeJSON(w, http.StatusOK, buildInstanceInfo(ns))
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not found"})
}

func buildInstanceInfo(ns agent.NodeState) InstanceInfo {
	engine := ns.Engine
	if engine == "" {
		engine = "nginx"
	}
	info := InstanceInfo{
		NodeID:   ns.NodeID,
		ID:       ns.NodeID,
		Hostname: ns.Hostname,
		Name:     ns.Hostname,
		IP:       ns.IP,
		Version:  ns.Version,
		Engine:   engine,
		Status:   mapStatus(ns),
		LastSeen: ns.LastSeen.Format(time.RFC3339),
	}
	if ns.Resources != nil {
		info.CPUPercent = ns.Resources.CpuPercent
		info.MemPercent = ns.Resources.MemoryPercent
		info.DiskPercent = ns.Resources.DiskPercent
		info.NetConnections = ns.Resources.NetConnections
		info.RPS = ns.Resources.RequestsPerSecond
		info.NetworkIO = ns.Resources.NetworkIo
		info.MemoryTotal = ns.Resources.MemoryTotalBytes
		info.DiskTotal = ns.Resources.DiskTotalBytes
	}
	return info
}

// mapStatus maps internal NodeStatus to admin-facing strings.
// admin 期望 connected/busy/disconnected (PageInstances 适配器约束)。
func mapStatus(ns agent.NodeState) string {
	if time.Since(ns.LastSeen) > offlineThreshold {
		return "disconnected"
	}
	switch ns.Status {
	case pb.NodeStatus_HEALTHY:
		return "connected"
	case pb.NodeStatus_DEGRADED:
		return "busy"
	case pb.NodeStatus_ERROR, pb.NodeStatus_UNKNOWN:
		return "disconnected"
	default:
		return "connected"
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
