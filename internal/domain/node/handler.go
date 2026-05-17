package node

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
// @Summary 获取节点列表
// @Description 分页查询节点列表，支持按设备ID、状态和关键字筛选
// @Tags 节点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param device_id query int false "设备 ID 筛选"
// @Param status query string false "状态筛选"
// @Param search query string false "搜索关键字"
// @Success 200 {object} map[string]interface{} "节点列表，包含 data、total、page、size 字段"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes [get]
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

	if did := r.URL.Query().Get("device_id"); did != "" {
		if v, err := strconv.ParseInt(did, 10, 64); err == nil {
			params.DeviceID = &v
		}
	}

	nodes, total, err := h.repo.List(r.Context(), params)
	if err != nil {
		slog.Error("list nodes failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  nodes,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// Get godoc
// @Summary 获取节点详情
// @Description 根据节点 ID 获取节点详细信息
// @Tags 节点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "节点 ID"
// @Success 200 {object} Node "节点详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "节点不存在"
// @Router /nodes/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	n, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}

	writeJSON(w, http.StatusOK, n)
}

// Create godoc
// @Summary 创建节点
// @Description 创建新的 WAF 节点，需提供节点名称和 IP 地址
// @Tags 节点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRequest true "节点创建参数"
// @Success 201 {object} Node "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.IPAddress == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and ip_address are required"})
		return
	}

	n, err := h.repo.Create(r.Context(), req)
	if err != nil {
		slog.Error("create node failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, n)
}

// Update godoc
// @Summary 更新节点
// @Description 根据节点 ID 更新节点信息，支持部分字段更新
// @Tags 节点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "节点 ID"
// @Param body body UpdateRequest true "节点更新参数"
// @Success 200 {object} Node "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{id} [put]
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

	n, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		slog.Error("update node failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, n)
}

// Delete godoc
// @Summary 删除节点
// @Description 根据节点 ID 删除节点
// @Tags 节点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "节点 ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "节点不存在"
// @Router /nodes/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
