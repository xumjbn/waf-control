package instancemgmt

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// RestartInstance accepts POST /instances/{hostname}/restart and records the
// intent. Actual restart is delivered via gRPC by the deploy pipeline; this
// endpoint exists so the SPA's『重启』action stops being a no-op alert.
//
// The handler is intentionally idempotent and returns 202 — the agent is
// expected to act asynchronously and report status back through the regular
// heartbeat / status feed.
func (h *Handler) RestartInstance(w http.ResponseWriter, r *http.Request) {
	hostname := r.URL.Query().Get("hostname")
	if hostname == "" {
		// Body-based form: { "hostname": "..." }
		var body struct {
			Hostname string `json:"hostname"`
			Reason   string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			hostname = body.Hostname
		}
	}
	if hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname required"})
		return
	}

	// Look up the node in the agent service so we don't accept restarts for
	// hosts we don't know about.
	nodes := h.agentSvc.GetConnectedNodes()
	known := false
	for _, ns := range nodes {
		if ns.Hostname == hostname {
			known = true
			break
		}
	}
	if !known {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not connected"})
		return
	}

	slog.Info("instance restart requested", "hostname", hostname)
	// 通过 PushConfig 下行流真下发 restart_service 命令到 agent。
	// agent 收到后优雅退出，由容器/systemd 重启 —— 在下次心跳前后状态自然反映。
	if err := h.agentSvc.SendCommandToHost(hostname, "restart_service", "operator requested restart"); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "无法下发重启命令：" + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"hostname": hostname,
		"status":   "dispatched",
		"message":  "重启命令已下发，agent 将在收到后重启",
	})
}

// RegisterNode accepts POST /instances/register-intent — records a
// pre-registration intent. Real agents register themselves over gRPC at
// startup, so this endpoint is informational; it lets the SPA show "已通知"
// after the『新增节点』flow rather than failing.
func (h *Handler) RegisterNodeIntent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Hostname    string `json:"hostname"`
		IP          string `json:"ip"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname required"})
		return
	}
	slog.Info("node register intent",
		"hostname", body.Hostname, "ip", body.IP, "desc", body.Description)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"hostname": body.Hostname,
		"status":   "intent-recorded",
		"message":  "agent must run installer/register itself; intent is informational",
	})
}
