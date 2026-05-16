package ha

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.repo.Get(r.Context())
	if err != nil {
		slog.Error("get ha config failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ha config not found"})
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (h *Handler) UpsertConfig(w http.ResponseWriter, r *http.Request) {
	var req UpsertConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	config, err := h.repo.Upsert(r.Context(), req)
	if err != nil {
		slog.Error("upsert ha config failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
