package logs

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) ListAttackLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	logs, total, err := h.repo.ListAttackLogs(r.Context(), q)
	if err != nil {
		slog.Error("list attack logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": logs, "total": total, "page": q.Page, "page_size": q.PageSize})
}

func (h *Handler) ListAntivirusLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	logs, total, err := h.repo.ListAntivirusLogs(r.Context(), q)
	if err != nil {
		slog.Error("list antivirus logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": logs, "total": total, "page": q.Page, "page_size": q.PageSize})
}

func (h *Handler) ListAntitamperLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	logs, total, err := h.repo.ListAntitamperLogs(r.Context(), q)
	if err != nil {
		slog.Error("list antitamper logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": logs, "total": total, "page": q.Page, "page_size": q.PageSize})
}

func parseLogQuery(r *http.Request) LogQuery {
	q := LogQuery{
		StartTime: r.URL.Query().Get("start_time"),
		EndTime:   r.URL.Query().Get("end_time"),
	}
	if v := r.URL.Query().Get("node_id"); v != "" {
		q.NodeID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := r.URL.Query().Get("page"); v != "" {
		q.Page, _ = strconv.Atoi(v)
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		q.PageSize, _ = strconv.Atoi(v)
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	return q
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
