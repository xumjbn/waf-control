package device

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
// @Summary 获取设备列表
// @Description 分页获取设备列表，支持按状态和关键词筛选
// @Tags 设备管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码，默认 1"
// @Param page_size query int false "每页数量，默认 20，最大 100"
// @Param status query string false "设备状态筛选"
// @Param search query string false "搜索关键词"
// @Success 200 {object} map[string]interface{} "设备列表（含 data、total、page、size）"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /devices [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	params := ListParams{
		Page:     page,
		PageSize: pageSize,
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
	}

	devices, total, err := h.repo.List(r.Context(), params)
	if err != nil {
		slog.Error("list devices failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  devices,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// Get godoc
// @Summary 获取设备详情
// @Description 根据设备 ID 获取设备详细信息
// @Tags 设备管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "设备 ID"
// @Success 200 {object} Device "设备详情"
// @Failure 400 {object} map[string]string "无效的设备 ID"
// @Failure 404 {object} map[string]string "设备不存在"
// @Router /devices/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	d, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
		return
	}

	writeJSON(w, http.StatusOK, d)
}

// Create godoc
// @Summary 创建设备
// @Description 创建新的 WAF 设备记录
// @Tags 设备管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRequest true "创建设备请求"
// @Success 201 {object} Device "创建成功的设备"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /devices [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	d, err := h.repo.Create(r.Context(), req)
	if err != nil {
		slog.Error("create device failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, d)
}

// Update godoc
// @Summary 更新设备
// @Description 根据设备 ID 更新设备信息
// @Tags 设备管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "设备 ID"
// @Param body body UpdateRequest true "更新设备请求"
// @Success 200 {object} Device "更新后的设备"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /devices/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	d, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		slog.Error("update device failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, d)
}

// Delete godoc
// @Summary 删除设备
// @Description 根据设备 ID 删除设备记录
// @Tags 设备管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "设备 ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "无效的设备 ID"
// @Failure 404 {object} map[string]string "设备不存在"
// @Router /devices/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
