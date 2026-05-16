package loadbalance

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

// --- VIPs ---

func (h *Handler) ListVIPs(w http.ResponseWriter, r *http.Request) {
	vips, err := h.repo.ListVIPs(r.Context())
	if err != nil {
		slog.Error("list vips failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": vips})
}

func (h *Handler) CreateVIP(w http.ResponseWriter, r *http.Request) {
	var req CreateVIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.Address == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and address are required"})
		return
	}

	vip, err := h.repo.CreateVIP(r.Context(), req)
	if err != nil {
		slog.Error("create vip failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, vip)
}

func (h *Handler) UpdateVIP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateVIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	vip, err := h.repo.UpdateVIP(r.Context(), id, req)
	if err != nil {
		slog.Error("update vip failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, vip)
}

func (h *Handler) DeleteVIP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteVIP(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "vip not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Pools ---

func (h *Handler) ListPools(w http.ResponseWriter, r *http.Request) {
	pools, err := h.repo.ListPools(r.Context())
	if err != nil {
		slog.Error("list pools failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": pools})
}

func (h *Handler) CreatePool(w http.ResponseWriter, r *http.Request) {
	var req CreatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	pool, err := h.repo.CreatePool(r.Context(), req)
	if err != nil {
		slog.Error("create pool failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, pool)
}

func (h *Handler) UpdatePool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	pool, err := h.repo.UpdatePool(r.Context(), id, req)
	if err != nil {
		slog.Error("update pool failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, pool)
}

func (h *Handler) DeletePool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeletePool(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "pool not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Members ---

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	members, err := h.repo.ListMembers(r.Context(), poolID)
	if err != nil {
		slog.Error("list members failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": members})
}

func (h *Handler) CreateMember(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	var req CreateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.PoolID = poolID

	if req.Address == "" || req.ProtocolPort <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "address and protocol_port are required"})
		return
	}

	member, err := h.repo.CreateMember(r.Context(), req)
	if err != nil {
		slog.Error("create member failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

func (h *Handler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "memberId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}

	if err := h.repo.DeleteMember(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Health Monitors ---

func (h *Handler) GetHealthMonitor(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	hm, err := h.repo.GetHealthMonitor(r.Context(), poolID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "health monitor not found"})
		return
	}
	writeJSON(w, http.StatusOK, hm)
}

func (h *Handler) CreateHealthMonitor(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	var req CreateHealthMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.PoolID = poolID

	hm, err := h.repo.CreateHealthMonitor(r.Context(), req)
	if err != nil {
		slog.Error("create health monitor failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, hm)
}

func (h *Handler) DeleteHealthMonitor(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	if err := h.repo.DeleteHealthMonitor(r.Context(), poolID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "health monitor not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
