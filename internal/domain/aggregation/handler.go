package aggregation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/waf-control/internal/agent"
)

type Handler struct {
	repo     *Repository
	agentSvc *agent.Service
}

func NewHandler(repo *Repository, agentSvc *agent.Service) *Handler {
	return &Handler{repo: repo, agentSvc: agentSvc}
}

// === 系统监控 ===

// SystemResourceAll GET /sys_monitor/system_resource
func (h *Handler) SystemResourceAll(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Format("2006-01-02 15:04:05")
	resp := SystemMonitorAll{
		Manager: &NodeBlock{
			SystemResources: managerStub(),
			Interfaces:      []NicInterface{},
			Datetime:        now,
		},
		Instances: []map[string]NodeBlock{},
	}

	for _, ns := range h.connectedNodes() {
		block := NodeBlock{
			SystemResources: nodeResource(ns),
			Interfaces:      nodeInterfaces(ns),
			Datetime:        ns.LastSeen.Format("2006-01-02 15:04:05"),
		}
		resp.Instances = append(resp.Instances, map[string]NodeBlock{ns.Hostname: block})
	}

	writeJSON(w, http.StatusOK, resp)
}

// InstanceSystemResource GET /sys_monitor/{instance_id}/system_resource
func (h *Handler) InstanceSystemResource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "instance_id")
	for _, ns := range h.connectedNodes() {
		if ns.NodeID == id || ns.Hostname == id {
			writeJSON(w, http.StatusOK, NodeBlock{
				SystemResources: nodeResource(ns),
				Interfaces:      nodeInterfaces(ns),
				Datetime:        ns.LastSeen.Format("2006-01-02 15:04:05"),
			})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not found"})
}

// NicState GET /sys_monitor/nic_state
func (h *Handler) NicState(w http.ResponseWriter, r *http.Request) {
	resp := NicStateAll{
		Manager:   []NicInterface{},
		Instances: map[string][]NicInterface{},
	}
	for _, ns := range h.connectedNodes() {
		resp.Instances[ns.Hostname] = nodeInterfaces(ns)
	}
	writeJSON(w, http.StatusOK, resp)
}

// NetworkInterface GET /sys_monitor/interface — 复用同一份接口数据
func (h *Handler) NetworkInterface(w http.ResponseWriter, r *http.Request) {
	h.NicState(w, r)
}

// NetworkFlow GET /sys_monitor/network_flow
func (h *Handler) NetworkFlow(w http.ResponseWriter, r *http.Request) {
	flows := make(map[string]string, 0)
	for _, ns := range h.connectedNodes() {
		if ns.Resources != nil && ns.Resources.NetworkIo != "" {
			flows[ns.Hostname] = ns.Resources.NetworkIo
		}
	}
	writeJSON(w, http.StatusOK, flows)
}

// === 系统设置 ===

// RunningMode GET /system/running_mode
func (h *Handler) RunningMode(w http.ResponseWriter, r *http.Request) {
	count := len(h.connectedNodes())
	mode := "standalone"
	if count > 1 {
		mode = "physical_cluster"
	}
	writeJSON(w, http.StatusOK, RunningMode{RunningMode: mode})
}

// ManagerTime GET /system/managertime
func (h *Handler) ManagerTime(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, time.Now().Format("2006-01-02 15:04:05"))
}

// AllTime GET /system/alltime
func (h *Handler) AllTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Format("2006-01-02 15:04:05")
	resp := map[string]string{"manager": now}
	for _, ns := range h.connectedNodes() {
		resp[ns.NodeID] = ns.LastSeen.Format("2006-01-02 15:04:05")
	}
	writeJSON(w, http.StatusOK, resp)
}

// ChangeManagerTime PUT /system/changetime
func (h *Handler) ChangeManagerTime(w http.ResponseWriter, r *http.Request) {
	var req ChangeTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	slog.Info("change manager time requested", "time", req.Time)
	writeJSON(w, http.StatusOK, map[string]string{"message": "accepted (read-only in container)"})
}

// ChangeInstanceTime PUT /system/{instance_id}/changetime
func (h *Handler) ChangeInstanceTime(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "instance_id")
	var req ChangeTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	slog.Info("change instance time requested", "instance_id", id, "time", req.Time)
	writeJSON(w, http.StatusOK, map[string]string{"message": "accepted"})
}

// === 站点 / 攻击 聚合 ===

// SiteStats GET /site_stats
func (h *Handler) SiteStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.SiteStats(r.Context())
	if err != nil {
		slog.Error("site stats failed", "error", err)
		writeJSON(w, http.StatusOK, SiteStats{})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// AttackStatistic GET /attack_stats/statistic_info
func (h *Handler) AttackStatistic(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.AttackRecords(r.Context(), 200)
	if err != nil {
		slog.Error("attack records failed", "error", err)
		writeJSON(w, http.StatusOK, AttackStats{StatisticInfo: []AttackRecord{}})
		return
	}
	writeJSON(w, http.StatusOK, AttackStats{StatisticInfo: items})
}

// AttackSeverity GET /monitor/attack/severity (admin getAttackSeverity 单对象响应)
func (h *Handler) AttackSeverity(w http.ResponseWriter, r *http.Request) {
	s, err := h.repo.AttackSeverity(r.Context())
	if err != nil {
		slog.Error("attack severity failed", "error", err)
		writeJSON(w, http.StatusOK, AttackSeverity{})
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// AttackSourceTop GET /monitor/attack/src-ips-top (admin 期望直接数组)
func (h *Handler) AttackSourceTop(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.AttackSourceTop(r.Context(), 10)
	if err != nil {
		slog.Error("attack source top failed", "error", err)
		writeJSON(w, http.StatusOK, []AttackSourceTop{})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// TopSites GET /statistic_trend/top_sites_info (admin 期望直接数组 [host, count])
func (h *Handler) TopSites(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.TopSites(r.Context(), 10)
	if err != nil {
		slog.Error("top sites failed", "error", err)
		writeJSON(w, http.StatusOK, [][2]any{})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// AttackLogs GET /attack_logs
func (h *Handler) AttackLogs(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.AttackRecords(r.Context(), 500)
	if err != nil {
		slog.Error("attack logs failed", "error", err)
		writeJSON(w, http.StatusOK, []AttackRecord{})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// AttackLogsCount GET /attack_logs_num
func (h *Handler) AttackLogsCount(w http.ResponseWriter, r *http.Request) {
	n, err := h.repo.AttackLogsCount(r.Context())
	if err != nil {
		slog.Error("attack logs count failed", "error", err)
		writeJSON(w, http.StatusOK, map[string]int64{"count": 0})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"count": n})
}

// === 流量 / 服务 ===

// FlowTop10App GET /monitor/flow/top10app
func (h *Handler) FlowTop10App(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []FlowTopItem{})
}

// FlowTop10IP GET /monitor/flow/top10ip
func (h *Handler) FlowTop10IP(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.AttackSourceTop(r.Context(), 10)
	if err != nil {
		writeJSON(w, http.StatusOK, []FlowTopItem{})
		return
	}
	resp := make([]FlowTopItem, 0, len(items))
	for _, it := range items {
		resp = append(resp, FlowTopItem{Name: it.SrcIP, Value: it.Count})
	}
	writeJSON(w, http.StatusOK, resp)
}

// ServiceStatistics GET /service-statistics
func (h *Handler) ServiceStatistics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"services": []any{}})
}

// HistoryManager GET /sys_monitor/history/manager
func (h *Handler) HistoryManager(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.repo.MetricHistory(r.Context(), "", r.URL.Query().Get("entries")))
}

// HistoryInstance GET /sys_monitor/history/instance/{instance_id}
func (h *Handler) HistoryInstance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "instance_id")
	writeJSON(w, http.StatusOK, h.repo.MetricHistory(r.Context(), id, r.URL.Query().Get("entries")))
}

// FlowAppDetail GET /monitor/flow/top10app/{app}
func (h *Handler) FlowAppDetail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"name": chi.URLParam(r, "app"), "series": []any{}})
}

// FlowIPDetail GET /monitor/flow/top10ip/{ip}
func (h *Handler) FlowIPDetail(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ip": chi.URLParam(r, "ip"), "series": []any{}})
}

// RealtimeMetricGet GET /monitor/realtime — admin getRealtimeMetric 走 GET（monitor 包的 PUT 仍保留）
func (h *Handler) RealtimeMetricGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

// HistoryMetricGet GET /monitor/history
func (h *Handler) HistoryMetricGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

// === 共享辅助 ===

func (h *Handler) connectedNodes() []agent.NodeState {
	if h.agentSvc == nil {
		return nil
	}
	return h.agentSvc.GetConnectedNodes()
}

func nodeResource(ns agent.NodeState) SystemResource {
	if ns.Resources == nil {
		return SystemResource{}
	}
	return SystemResource{
		CPUPercent:     ns.Resources.CpuPercent,
		MemoryPercent:  ns.Resources.MemoryPercent,
		MemoryTotal:    ns.Resources.MemoryTotalBytes,
		DiskPercent:    ns.Resources.DiskPercent,
		DiskTotal:      ns.Resources.DiskTotalBytes,
		NetConnections: ns.Resources.NetConnections,
		NetworkIO:      ns.Resources.NetworkIo,
	}
}

func nodeInterfaces(ns agent.NodeState) []NicInterface {
	if ns.Resources == nil {
		return []NicInterface{}
	}
	out := make([]NicInterface, 0, len(ns.Resources.Interfaces))
	for _, iface := range ns.Resources.Interfaces {
		ni := NicInterface{Name: iface.Name, State: iface.State}
		if iface.Ip != "" {
			ni.Addresses = []NicAddress{{IP: iface.Ip}}
		}
		out = append(out, ni)
	}
	return out
}

// managerStub 控制面机本身没采集 agent，给个静态零值，便于前端不报错。
func managerStub() SystemResource {
	return SystemResource{
		CPUPercent:    0,
		MemoryPercent: 0,
		DiskPercent:   0,
		NetworkIO:     "↓0B/s ↑0B/s",
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
