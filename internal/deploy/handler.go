package deploy

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type deployRequest struct {
	Site    *SiteConfig   `json:"site,omitempty"`
	Policy  *PolicyConfig `json:"policy,omitempty"`
	Devices []int64       `json:"devices,omitempty"`
}

func (h *Handler) DeployNginx(w http.ResponseWriter, r *http.Request) {
	var req deployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, NewResult(false, "invalid request body"))
		return
	}
	if req.Site == nil || req.Site.Domain == "" {
		writeJSON(w, http.StatusBadRequest, NewResult(false, "site.domain is required"))
		return
	}

	if err := h.svc.DeployNginx(r.Context(), req.Site, req.Devices); err != nil {
		slog.Error("deploy nginx failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, NewResult(false, "deploy failed"))
		return
	}

	writeJSON(w, http.StatusOK, NewResult(true, "nginx config deployed"))
}

func (h *Handler) DeployModsec(w http.ResponseWriter, r *http.Request) {
	var req deployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, NewResult(false, "invalid request body"))
		return
	}
	if req.Policy == nil || req.Policy.Name == "" {
		writeJSON(w, http.StatusBadRequest, NewResult(false, "policy.name is required"))
		return
	}

	if err := h.svc.DeployModsec(r.Context(), req.Policy, req.Devices); err != nil {
		slog.Error("deploy modsec failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, NewResult(false, "deploy failed"))
		return
	}

	writeJSON(w, http.StatusOK, NewResult(true, "modsecurity config deployed"))
}

func (h *Handler) DeployAll(w http.ResponseWriter, r *http.Request) {
	var req deployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, NewResult(false, "invalid request body"))
		return
	}
	if req.Site == nil || req.Site.Domain == "" {
		writeJSON(w, http.StatusBadRequest, NewResult(false, "site.domain is required"))
		return
	}
	if req.Policy == nil || req.Policy.Name == "" {
		writeJSON(w, http.StatusBadRequest, NewResult(false, "policy.name is required"))
		return
	}

	if err := h.svc.DeployAll(r.Context(), req.Site, req.Policy); err != nil {
		slog.Error("deploy all failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, NewResult(false, "full deploy failed"))
		return
	}

	writeJSON(w, http.StatusOK, NewResult(true, "full configuration deployed"))
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
