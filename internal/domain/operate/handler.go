package operate

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

// List godoc
// @Summary 获取操作日志列表
// @Description 分页查询操作日志，支持按用户名、请求方法、路径、状态码和时间范围筛选
// @Tags 运维操作
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param username query string false "用户名"
// @Param method query string false "请求方法"
// @Param path query string false "请求路径"
// @Param min_code query int false "最小状态码"
// @Param max_code query int false "最大状态码"
// @Param start_at query string false "开始时间"
// @Param end_at query string false "结束时间"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "操作日志列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /operation-logs [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := OperationLogFilter{
		Username: q.Get("username"),
		Method:   q.Get("method"),
		Path:     q.Get("path"),
		StartAt:  q.Get("start_at"),
		EndAt:    q.Get("end_at"),
	}
	if mc, err := strconv.Atoi(q.Get("min_code")); err == nil {
		filter.MinCode = mc
	}
	if mc, err := strconv.Atoi(q.Get("max_code")); err == nil {
		filter.MaxCode = mc
	}
	if p, err := strconv.Atoi(q.Get("page")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(q.Get("page_size")); err == nil {
		filter.PageSize = ps
	}

	items, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		slog.Error("list operation logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  items,
		"total": total,
	})
}

// Get godoc
// @Summary 获取操作日志详情
// @Description 根据ID获取单条操作日志详情
// @Tags 运维操作
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "操作日志ID"
// @Success 200 {object} map[string]interface{} "操作日志详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "操作日志不存在"
// @Router /operation-logs/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	item, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "operation log not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
