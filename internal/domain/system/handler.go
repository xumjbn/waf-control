package system

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

// --- Settings ---

func (h *Handler) ListSettings(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	settings, err := h.repo.ListSettings(r.Context(), category)
	if err != nil {
		slog.Error("list settings failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": settings})
}

func (h *Handler) UpsertSetting(w http.ResponseWriter, r *http.Request) {
	var req UpsertSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}

	setting, err := h.repo.UpsertSetting(r.Context(), req)
	if err != nil {
		slog.Error("upsert setting failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, setting)
}

func (h *Handler) DeleteSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if err := h.repo.DeleteSetting(r.Context(), key); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "setting not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Licenses ---

func (h *Handler) ListLicenses(w http.ResponseWriter, r *http.Request) {
	licenses, err := h.repo.ListLicenses(r.Context())
	if err != nil {
		slog.Error("list licenses failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": licenses})
}

func (h *Handler) CreateLicense(w http.ResponseWriter, r *http.Request) {
	var req CreateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.LicenseKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "license_key is required"})
		return
	}

	license, err := h.repo.CreateLicense(r.Context(), req)
	if err != nil {
		slog.Error("create license failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, license)
}

func (h *Handler) ActivateLicense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.ActivateLicense(r.Context(), id); err != nil {
		slog.Error("activate license failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "activated"})
}

func (h *Handler) DeleteLicense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteLicense(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Upgrades ---

func (h *Handler) ListUpgrades(w http.ResponseWriter, r *http.Request) {
	upgrades, err := h.repo.ListUpgrades(r.Context())
	if err != nil {
		slog.Error("list upgrades failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": upgrades})
}

func (h *Handler) CreateUpgrade(w http.ResponseWriter, r *http.Request) {
	var req CreateUpgradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "version is required"})
		return
	}

	upgrade, err := h.repo.CreateUpgrade(r.Context(), req)
	if err != nil {
		slog.Error("create upgrade failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, upgrade)
}

func (h *Handler) TriggerUpgrade(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.TriggerUpgrade(r.Context(), id); err != nil {
		slog.Error("trigger upgrade failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "upgrade triggered"})
}

func (h *Handler) DeleteUpgrade(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteUpgrade(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "upgrade not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
