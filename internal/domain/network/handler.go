package network

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

// --- Interfaces ---

func (h *Handler) ListInterfaces(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	ifaces, err := h.repo.ListInterfaces(r.Context(), nodeID)
	if err != nil {
		slog.Error("list interfaces failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": ifaces})
}

func (h *Handler) CreateInterface(w http.ResponseWriter, r *http.Request) {
	var req CreateInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and name are required"})
		return
	}

	iface, err := h.repo.CreateInterface(r.Context(), req)
	if err != nil {
		slog.Error("create interface failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, iface)
}

func (h *Handler) UpdateInterface(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	iface, err := h.repo.UpdateInterface(r.Context(), id, req)
	if err != nil {
		slog.Error("update interface failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, iface)
}

func (h *Handler) DeleteInterface(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteInterface(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "interface not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Bridges ---

func (h *Handler) ListBridges(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	bridges, err := h.repo.ListBridges(r.Context(), nodeID)
	if err != nil {
		slog.Error("list bridges failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": bridges})
}

func (h *Handler) CreateBridge(w http.ResponseWriter, r *http.Request) {
	var req CreateBridgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and name are required"})
		return
	}

	bridge, err := h.repo.CreateBridge(r.Context(), req)
	if err != nil {
		slog.Error("create bridge failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, bridge)
}

func (h *Handler) UpdateBridge(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateBridgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	bridge, err := h.repo.UpdateBridge(r.Context(), id, req)
	if err != nil {
		slog.Error("update bridge failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bridge)
}

func (h *Handler) DeleteBridge(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteBridge(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bridge not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Bonds ---

func (h *Handler) ListBonds(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	bonds, err := h.repo.ListBonds(r.Context(), nodeID)
	if err != nil {
		slog.Error("list bonds failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": bonds})
}

func (h *Handler) CreateBond(w http.ResponseWriter, r *http.Request) {
	var req CreateBondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and name are required"})
		return
	}

	bond, err := h.repo.CreateBond(r.Context(), req)
	if err != nil {
		slog.Error("create bond failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, bond)
}

func (h *Handler) UpdateBond(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateBondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	bond, err := h.repo.UpdateBond(r.Context(), id, req)
	if err != nil {
		slog.Error("update bond failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bond)
}

func (h *Handler) DeleteBond(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteBond(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bond not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Routes ---

func (h *Handler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	routes, err := h.repo.ListRoutes(r.Context(), nodeID)
	if err != nil {
		slog.Error("list routes failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": routes})
}

func (h *Handler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var req CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Destination == "" || req.Gateway == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id, destination, and gateway are required"})
		return
	}

	route, err := h.repo.CreateRoute(r.Context(), req)
	if err != nil {
		slog.Error("create route failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, route)
}

func (h *Handler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteRoute(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
