package flow

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

// ListFlowLogs godoc
// @Summary 查询流量日志列表
// @Description 根据筛选条件分页查询流量日志，支持按源IP、目标IP、协议、时间范围过滤
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param src_ip query string false "源IP地址"
// @Param dst_ip query string false "目标IP地址"
// @Param protocol query string false "协议类型"
// @Param start_at query string false "开始时间"
// @Param end_at query string false "结束时间"
// @Param order_by query string false "排序字段"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} map[string]interface{} "流量日志列表及总数"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /flow-logs [get]
func (h *Handler) ListFlowLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := FlowLogFilter{
		SrcIP:    q.Get("src_ip"),
		DstIP:    q.Get("dst_ip"),
		Protocol: q.Get("protocol"),
		StartAt:  q.Get("start_at"),
		EndAt:    q.Get("end_at"),
		OrderBy:  q.Get("order_by"),
	}
	if p, err := strconv.Atoi(q.Get("page")); err == nil {
		filter.Page = p
	}
	if ps, err := strconv.Atoi(q.Get("page_size")); err == nil {
		filter.PageSize = ps
	}

	items, total, err := h.repo.ListFlowLogs(r.Context(), filter)
	if err != nil {
		slog.Error("list flow logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  items,
		"total": total,
	})
}

// GetFlowLog godoc
// @Summary 获取单条流量日志
// @Description 根据ID获取流量日志详情
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "流量日志ID"
// @Success 200 {object} FlowLog "流量日志详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "未找到"
// @Router /flow-logs/{id} [get]
func (h *Handler) GetFlowLog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	item, err := h.repo.GetFlowLog(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "flow log not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// TopSrcIPs godoc
// @Summary 获取流量源IP排行
// @Description 查询流量日志中源IP地址的Top排行
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "源IP排行数据"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /flow-logs/top-src-ips [get]
func (h *Handler) TopSrcIPs(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.TopSrcIPs(r.Context())
	if err != nil {
		slog.Error("top flow src IPs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// TopDstIPs godoc
// @Summary 获取流量目标IP排行
// @Description 查询流量日志中目标IP地址的Top排行
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "目标IP排行数据"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /flow-logs/top-dst-ips [get]
func (h *Handler) TopDstIPs(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.TopDstIPs(r.Context())
	if err != nil {
		slog.Error("top flow dst IPs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// ProtocolDistribution godoc
// @Summary 获取协议分布统计
// @Description 查询流量日志中各协议的分布情况
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "协议分布数据"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /flow-logs/protocols [get]
func (h *Handler) ProtocolDistribution(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ProtocolDistribution(r.Context())
	if err != nil {
		slog.Error("protocol distribution failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// ListSavedQueries godoc
// @Summary 获取保存的查询列表
// @Description 查询所有已保存的流量查询条件
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "保存的查询列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /flow-saved-queries [get]
func (h *Handler) ListSavedQueries(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListSavedQueries(r.Context())
	if err != nil {
		slog.Error("list saved queries failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// CreateSavedQuery godoc
// @Summary 创建保存的查询
// @Description 保存一个流量查询条件，便于后续快速复用
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body SavedQuery true "查询条件（name和query必填）"
// @Success 201 {object} SavedQuery "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /flow-saved-queries [post]
func (h *Handler) CreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	var sq SavedQuery
	if err := json.NewDecoder(r.Body).Decode(&sq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if sq.Name == "" || sq.Query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and query are required"})
		return
	}

	result, err := h.repo.CreateSavedQuery(r.Context(), sq)
	if err != nil {
		slog.Error("create saved query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// DeleteSavedQuery godoc
// @Summary 删除保存的查询
// @Description 根据ID删除已保存的流量查询条件
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "保存的查询ID"
// @Success 204 "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "未找到"
// @Router /flow-saved-queries/{id} [delete]
func (h *Handler) DeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteSavedQuery(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MonitorRecords godoc
// @Summary 获取流量监控记录
// @Description 查询流量监控的汇总记录数据
// @Tags 流量监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "流量监控记录列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /flow-monitor/records [get]
func (h *Handler) MonitorRecords(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.MonitorRecords(r.Context())
	if err != nil {
		slog.Error("list monitor records failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
