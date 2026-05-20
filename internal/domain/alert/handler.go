package alert

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.repo.ListPolicies(r.Context())
	if err != nil {
		slog.Error("list alert policies failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": policies})
}

func (h *Handler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.Metric == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and metric are required"})
		return
	}
	policy, err := h.repo.CreatePolicy(r.Context(), req)
	if err != nil {
		slog.Error("create alert policy failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, policy)
}

func (h *Handler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	policy, err := h.repo.UpdatePolicy(r.Context(), id, req)
	if err != nil {
		slog.Error("update alert policy failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.DeletePolicy(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert policy not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 100
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	filter := EventFilter{
		Status: q.Get("status"),
		Level:  q.Get("level"),
		Limit:  limit,
	}
	events, err := h.repo.ListEvents(r.Context(), filter)
	if err != nil {
		slog.Error("list alert events failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": events})
}

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Kind == "" || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kind and message are required"})
		return
	}
	ev, err := h.repo.CreateEvent(r.Context(), req)
	if err != nil {
		slog.Error("create alert event failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, ev)
}

func (h *Handler) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req UpdateEventStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Status == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status is required"})
		return
	}
	ev, err := h.repo.UpdateEventStatus(r.Context(), id, req.Status, req.HandledBy)
	if err != nil {
		slog.Error("update alert event status failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HandledBy string `json:"handled_by"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	n, err := h.repo.MarkAllRead(r.Context(), req.HandledBy)
	if err != nil {
		slog.Error("mark all alert events read failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"updated": n})
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.EventStats(r.Context())
	if err != nil {
		slog.Error("alert event stats failed", "error", err)
		writeJSON(w, http.StatusOK, EventStats{})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) ListChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.repo.ListChannels(r.Context())
	if err != nil {
		slog.Error("list alert channels failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": channels, "kinds": ChannelKinds()})
}

func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.Kind == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and kind are required"})
		return
	}
	ch, err := h.repo.CreateChannel(r.Context(), req)
	if err != nil {
		slog.Error("create alert channel failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

func (h *Handler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteChannel(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "channel not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// TestChannel 发送一条 INFO 级别测试事件到该 channel（占位实现）。
// POST /alert/channels/{id}/test
func (h *Handler) TestChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ev, err := h.repo.TestChannel(r.Context(), id)
	if err != nil {
		slog.Error("test alert channel failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusAccepted, ev)
}

func (h *Handler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	channel, err := h.repo.UpdateChannel(r.Context(), id, req)
	if err != nil {
		slog.Error("update alert channel failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, channel)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
