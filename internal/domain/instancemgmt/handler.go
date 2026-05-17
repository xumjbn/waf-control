package instancemgmt

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/waf-control/internal/agent"
)

type InstanceInfo struct {
	NodeID      string  `json:"node_id"`
	Hostname    string  `json:"hostname"`
	IP          string  `json:"ip"`
	Version     string  `json:"version"`
	Status      string  `json:"status"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"memory_percent"`
	DiskPercent float64 `json:"disk_percent"`
	LastSeen    string  `json:"last_seen"`
}

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
		info := InstanceInfo{
			NodeID:   ns.NodeID,
			Hostname: ns.Hostname,
			IP:       ns.IP,
			Version:  ns.Version,
			Status:   ns.Status.String(),
			LastSeen: ns.LastSeen.Format(time.RFC3339),
		}
		if ns.Resources != nil {
			info.CPUPercent = ns.Resources.CpuPercent
			info.MemPercent = ns.Resources.MemoryPercent
			info.DiskPercent = ns.Resources.DiskPercent
		}
		instances = append(instances, info)
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
			info := InstanceInfo{
				NodeID:   ns.NodeID,
				Hostname: ns.Hostname,
				IP:       ns.IP,
				Version:  ns.Version,
				Status:   ns.Status.String(),
				LastSeen: ns.LastSeen.Format(time.RFC3339),
			}
			if ns.Resources != nil {
				info.CPUPercent = ns.Resources.CpuPercent
				info.MemPercent = ns.Resources.MemoryPercent
				info.DiskPercent = ns.Resources.DiskPercent
			}
			writeJSON(w, http.StatusOK, info)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not found"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
