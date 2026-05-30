package reports

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
	gen  *Generator
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo, gen: NewGenerator(repo.pool)}
}

// --- Custom Reports ---

// ListCustom godoc
// @Summary 获取自定义报表列表
// @Description 查询所有自定义报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "自定义报表列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/custom [get]
func (h *Handler) ListCustom(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListCustom(r.Context())
	if err != nil {
		slog.Error("list custom reports failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// GetCustom godoc
// @Summary 获取自定义报表详情
// @Description 根据ID获取单个自定义报表详情
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "自定义报表ID"
// @Success 200 {object} map[string]interface{} "自定义报表详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/custom/{id} [get]
func (h *Handler) GetCustom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	item, err := h.repo.GetCustom(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "custom report not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// CreateCustom godoc
// @Summary 创建自定义报表
// @Description 新增一个自定义报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CustomReport true "自定义报表信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/custom [post]
func (h *Handler) CreateCustom(w http.ResponseWriter, r *http.Request) {
	var cr CustomReport
	if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if cr.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	result, err := h.repo.CreateCustom(r.Context(), cr)
	if err != nil {
		slog.Error("create custom report failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// UpdateCustom godoc
// @Summary 更新自定义报表
// @Description 根据ID更新自定义报表信息
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "自定义报表ID"
// @Param body body CustomReport true "自定义报表信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/custom/{id} [put]
func (h *Handler) UpdateCustom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var cr CustomReport
	if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	cr.ID = id
	result, err := h.repo.UpdateCustom(r.Context(), cr)
	if err != nil {
		slog.Error("update custom report failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// DeleteCustom godoc
// @Summary 删除自定义报表
// @Description 根据ID删除自定义报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "自定义报表ID"
// @Success 204 "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/custom/{id} [delete]
func (h *Handler) DeleteCustom(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteCustom(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CustomReportData godoc
// @Summary 获取自定义报表数据
// @Description 根据ID获取自定义报表的统计数据
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "自定义报表ID"
// @Success 200 {object} map[string]interface{} "报表数据"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/custom/{id}/data [get]
func (h *Handler) CustomReportData(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	data, err := h.repo.CustomReportData(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// --- Combined Reports ---

// ListCombined godoc
// @Summary 获取综合报表列表
// @Description 查询所有综合报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "综合报表列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/combined [get]
func (h *Handler) ListCombined(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListCombined(r.Context())
	if err != nil {
		slog.Error("list combined reports failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// GetCombined godoc
// @Summary 获取综合报表详情
// @Description 根据ID获取单个综合报表详情
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "综合报表ID"
// @Success 200 {object} map[string]interface{} "综合报表详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/combined/{id} [get]
func (h *Handler) GetCombined(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	item, err := h.repo.GetCombined(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "combined report not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// CreateCombined godoc
// @Summary 创建综合报表
// @Description 新增一个综合报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CombinedReport true "综合报表信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/combined [post]
func (h *Handler) CreateCombined(w http.ResponseWriter, r *http.Request) {
	var cr CombinedReport
	if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if cr.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	result, err := h.repo.CreateCombined(r.Context(), cr)
	if err != nil {
		slog.Error("create combined report failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// UpdateCombined godoc
// @Summary 更新综合报表
// @Description 根据ID更新综合报表信息
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "综合报表ID"
// @Param body body CombinedReport true "综合报表信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/combined/{id} [put]
func (h *Handler) UpdateCombined(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var cr CombinedReport
	if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	cr.ID = id
	result, err := h.repo.UpdateCombined(r.Context(), cr)
	if err != nil {
		slog.Error("update combined report failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// DeleteCombined godoc
// @Summary 删除综合报表
// @Description 根据ID删除综合报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "综合报表ID"
// @Success 204 "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/combined/{id} [delete]
func (h *Handler) DeleteCombined(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteCombined(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CombinedReportData godoc
// @Summary 获取综合报表数据
// @Description 根据ID获取综合报表的统计数据
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "综合报表ID"
// @Success 200 {object} map[string]interface{} "报表数据"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/combined/{id}/data [get]
func (h *Handler) CombinedReportData(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	data, err := h.repo.CombinedReportData(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// --- Timing Reports ---

// ListTiming godoc
// @Summary 获取定时报表列表
// @Description 查询所有定时报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "定时报表列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/timing [get]
func (h *Handler) ListTiming(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListTiming(r.Context())
	if err != nil {
		slog.Error("list timing reports failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// GetTiming godoc
// @Summary 获取定时报表详情
// @Description 根据ID获取单个定时报表详情
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "定时报表ID"
// @Success 200 {object} map[string]interface{} "定时报表详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/timing/{id} [get]
func (h *Handler) GetTiming(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	item, err := h.repo.GetTiming(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "timing report not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// CreateTiming godoc
// @Summary 创建定时报表
// @Description 新增一个定时报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body TimingReport true "定时报表信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/timing [post]
func (h *Handler) CreateTiming(w http.ResponseWriter, r *http.Request) {
	var tr TimingReport
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if tr.Name == "" || tr.Metric == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and metric are required"})
		return
	}
	result, err := h.repo.CreateTiming(r.Context(), tr)
	if err != nil {
		slog.Error("create timing report failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// UpdateTiming godoc
// @Summary 更新定时报表
// @Description 根据ID更新定时报表信息
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "定时报表ID"
// @Param body body TimingReport true "定时报表信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/timing/{id} [put]
func (h *Handler) UpdateTiming(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var tr TimingReport
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	tr.ID = id
	result, err := h.repo.UpdateTiming(r.Context(), tr)
	if err != nil {
		slog.Error("update timing report failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// DeleteTiming godoc
// @Summary 删除定时报表
// @Description 根据ID删除定时报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "定时报表ID"
// @Success 204 "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/timing/{id} [delete]
func (h *Handler) DeleteTiming(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteTiming(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TimingReportData godoc
// @Summary 获取定时报表数据
// @Description 根据ID获取定时报表的统计数据
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "定时报表ID"
// @Success 200 {object} map[string]interface{} "报表数据"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/timing/{id}/data [get]
func (h *Handler) TimingReportData(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	data, err := h.repo.TimingReportData(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// --- Manual Reports ---

// ListManual godoc
// @Summary 获取手动报表列表
// @Description 查询所有手动报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "手动报表列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/manual [get]
func (h *Handler) ListManual(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListManual(r.Context())
	if err != nil {
		slog.Error("list manual reports failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// GetManual godoc
// @Summary 获取手动报表详情
// @Description 根据ID获取单个手动报表详情
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "手动报表ID"
// @Success 200 {object} map[string]interface{} "手动报表详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/manual/{id} [get]
func (h *Handler) GetManual(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	item, err := h.repo.GetManual(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "manual report not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// CreateManual godoc
// @Summary 创建手动报表
// @Description 新增一个手动报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ManualReport true "手动报表信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /reports/manual [post]
func (h *Handler) CreateManual(w http.ResponseWriter, r *http.Request) {
	var mr ManualReport
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if mr.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	result, err := h.repo.CreateManual(r.Context(), mr)
	if err != nil {
		slog.Error("create manual report failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// DeleteManual godoc
// @Summary 删除手动报表
// @Description 根据ID删除手动报表
// @Tags 报表管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "手动报表ID"
// @Success 204 "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "报表不存在"
// @Router /reports/manual/{id} [delete]
func (h *Handler) DeleteManual(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteManual(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
