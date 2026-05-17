package monitor

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

// --- Attack Monitor ---

// TopSites godoc
// @Summary 获取攻击目标站点排行
// @Description 查询受攻击最多的站点Top排行
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "站点排行数据"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /statistic_trend/top_sites_info [get]
func (h *Handler) TopSites(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.TopSites(r.Context())
	if err != nil {
		slog.Error("top sites failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// TopSrcIPs godoc
// @Summary 获取攻击源IP排行
// @Description 查询攻击来源IP地址的Top排行
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "攻击源IP排行数据"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /monitor/attack/src-ips-top [get]
func (h *Handler) TopSrcIPs(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.TopSrcIPs(r.Context())
	if err != nil {
		slog.Error("top src IPs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// SeverityStats godoc
// @Summary 获取攻击严重程度统计
// @Description 查询各严重程度等级的攻击事件数量统计
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "严重程度统计数据"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /monitor/attack/severity [get]
func (h *Handler) SeverityStats(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.SeverityStats(r.Context())
	if err != nil {
		slog.Error("severity stats failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// --- System Monitor ---

// ListMetrics godoc
// @Summary 获取监控指标列表
// @Description 查询所有系统监控指标的当前值
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "监控指标列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /monitor/metric [get]
func (h *Handler) ListMetrics(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListMetrics(r.Context())
	if err != nil {
		slog.Error("list metrics failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// ListMetricSpecs godoc
// @Summary 获取监控指标规格列表
// @Description 查询所有可用的监控指标规格定义
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "指标规格列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /monitor/metricspec [get]
func (h *Handler) ListMetricSpecs(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListMetricSpecs(r.Context())
	if err != nil {
		slog.Error("list metric specs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// GetMetricSpec godoc
// @Summary 获取单个监控指标规格
// @Description 根据ID获取监控指标规格的详细定义
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "指标规格ID"
// @Success 200 {object} MetricSpec "指标规格详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "未找到"
// @Router /monitor/metricspec/{id} [get]
func (h *Handler) GetMetricSpec(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	spec, err := h.repo.GetMetricSpec(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "metric spec not found"})
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

// QueryRealtime godoc
// @Summary 查询实时监控数据
// @Description 根据节点ID和指标名称查询实时监控数据
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body RealtimeQuery true "实时查询参数（node_id和metric必填）"
// @Success 200 {object} map[string]interface{} "实时监控数据"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /monitor/realtime [put]
func (h *Handler) QueryRealtime(w http.ResponseWriter, r *http.Request) {
	var req RealtimeQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID <= 0 || req.Metric == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and metric are required"})
		return
	}

	items, err := h.repo.QueryRealtime(r.Context(), req)
	if err != nil {
		slog.Error("query realtime failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// QueryHistory godoc
// @Summary 查询历史监控数据
// @Description 根据节点ID、指标名称和时间范围查询历史监控数据
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body HistoryQuery true "历史查询参数（node_id和metric必填）"
// @Success 200 {object} map[string]interface{} "历史监控数据"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /monitor/history [put]
func (h *Handler) QueryHistory(w http.ResponseWriter, r *http.Request) {
	var req HistoryQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID <= 0 || req.Metric == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and metric are required"})
		return
	}

	items, err := h.repo.QueryHistory(r.Context(), req)
	if err != nil {
		slog.Error("query history failed", "error", err)
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
