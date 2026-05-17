package security

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

// --- Auth Hosts ---

// ListAuthHosts godoc
// @Summary 获取授权主机列表
// @Description 查询所有授权主机信息
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "授权主机列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /security_management/auth_hosts [get]
func (h *Handler) ListAuthHosts(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListAuthHosts(r.Context())
	if err != nil {
		slog.Error("list auth hosts failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": items})
}

// CreateAuthHost godoc
// @Summary 创建授权主机
// @Description 新增一个授权主机记录
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateAuthHostRequest true "授权主机信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /security_management/auth_hosts [post]
func (h *Handler) CreateAuthHost(w http.ResponseWriter, r *http.Request) {
	var req CreateAuthHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Host == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host is required"})
		return
	}

	item, err := h.repo.CreateAuthHost(r.Context(), req)
	if err != nil {
		slog.Error("create auth host failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// GetAuthHost godoc
// @Summary 获取授权主机详情
// @Description 根据ID获取单个授权主机详情
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "授权主机ID"
// @Success 200 {object} map[string]interface{} "授权主机详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "授权主机不存在"
// @Router /security_management/auth_hosts/{id} [get]
func (h *Handler) GetAuthHost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	item, err := h.repo.GetAuthHost(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "auth host not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// UpdateAuthHost godoc
// @Summary 更新授权主机
// @Description 根据ID更新授权主机信息
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "授权主机ID"
// @Param body body UpdateAuthHostRequest true "更新的授权主机信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /security_management/auth_hosts/{id} [put]
func (h *Handler) UpdateAuthHost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateAuthHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	item, err := h.repo.UpdateAuthHost(r.Context(), id, req)
	if err != nil {
		slog.Error("update auth host failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// DeleteAuthHost godoc
// @Summary 删除授权主机
// @Description 根据ID删除授权主机
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "授权主机ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "授权主机不存在"
// @Router /security_management/auth_hosts/{id} [delete]
func (h *Handler) DeleteAuthHost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteAuthHost(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "auth host not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Auth Host Config ---

// GetAuthHostConfig godoc
// @Summary 获取授权主机配置
// @Description 获取当前授权主机全局配置
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "授权主机配置"
// @Failure 404 {object} map[string]string "配置不存在"
// @Router /security_management/auth_host_cfg [get]
func (h *Handler) GetAuthHostConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.repo.GetAuthHostConfig(r.Context())
	if err != nil {
		slog.Error("get auth host config failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "config not found"})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// UpdateAuthHostConfig godoc
// @Summary 更新授权主机配置
// @Description 更新或创建授权主机全局配置
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateAuthHostConfigRequest true "授权主机配置信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /security_management/auth_host_cfg [put]
func (h *Handler) UpdateAuthHostConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateAuthHostConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := h.repo.UpsertAuthHostConfig(r.Context(), req)
	if err != nil {
		slog.Error("update auth host config failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// --- Password Status ---

// GetPasswordStatus godoc
// @Summary 获取用户密码状态
// @Description 根据用户ID查询密码状态信息
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "用户ID"
// @Success 200 {object} map[string]interface{} "密码状态信息"
// @Failure 400 {object} map[string]string "无效的用户ID"
// @Failure 404 {object} map[string]string "用户不存在"
// @Router /system/query_pswd_status/{userId} [get]
func (h *Handler) GetPasswordStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	ps, err := h.repo.GetPasswordStatus(r.Context(), userID)
	if err != nil {
		slog.Error("get password status failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
