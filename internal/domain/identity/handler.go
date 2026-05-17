package identity

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Login godoc
// @Summary 用户登录
// @Description 使用用户名密码进行身份认证，返回 JWT 令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body LoginRequest true "登录请求"
// @Success 200 {object} TokenPair "令牌对"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "认证失败"
// @Router /identity/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	pair, err := h.svc.Authenticate(r.Context(), req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials, ErrUserDisabled:
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		default:
			slog.Error("login failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

// Logout godoc
// @Summary 用户登出
// @Description 撤销刷新令牌，使当前会话失效
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body object{refresh_token=string} true "登出请求，包含 refresh_token"
// @Success 200 {object} map[string]string "登出成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refresh_token required"})
		return
	}

	if err := h.svc.Logout(r.Context(), body.RefreshToken); err != nil {
		slog.Error("logout failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// Me godoc
// @Summary 获取当前用户信息
// @Description 根据当前认证令牌获取登录用户的详细信息及角色
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} User "当前用户信息"
// @Failure 401 {object} map[string]string "未认证"
// @Failure 404 {object} map[string]string "用户不存在"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	user, err := h.svc.GetUserWithRoles(r.Context(), claims.UserID)
	if err != nil {
		slog.Error("get me failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// ListUsers godoc
// @Summary 获取用户列表
// @Description 获取系统中所有用户及其角色信息，需要管理员权限
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Success 200 {array} User "用户列表"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/users [get]
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.repo.ListUsers(r.Context())
	if err != nil {
		slog.Error("list users failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	for i := range users {
		roles, _ := h.svc.repo.GetUserRoles(r.Context(), users[i].ID)
		users[i].Roles = roles
	}

	writeJSON(w, http.StatusOK, users)
}

// CreateUser godoc
// @Summary 创建用户
// @Description 创建新用户并可选分配角色，需要管理员权限
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateUserRequest true "创建用户请求"
// @Success 201 {object} User "创建成功的用户"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/users [post]
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	hashed, err := HashPassword(req.Password)
	if err != nil {
		slog.Error("hash password failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	user := &User{
		Username: req.Username,
		Password: hashed,
		Email:    &req.Email,
		RealName: &req.RealName,
		IsActive: true,
	}

	if err := h.svc.repo.CreateUser(r.Context(), user); err != nil {
		slog.Error("create user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}

	if len(req.RoleIDs) > 0 {
		if err := h.svc.repo.SetUserRoles(r.Context(), user.ID, req.RoleIDs); err != nil {
			slog.Error("set user roles failed", "error", err)
		}
	}

	writeJSON(w, http.StatusCreated, user)
}

// GetUser godoc
// @Summary 获取用户详情
// @Description 根据用户 ID 获取用户详细信息及角色，需要管理员权限
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户 ID"
// @Success 200 {object} User "用户详情"
// @Failure 400 {object} map[string]string "无效的用户 ID"
// @Failure 404 {object} map[string]string "用户不存在"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/users/{id} [get]
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	user, err := h.svc.GetUserWithRoles(r.Context(), id)
	if err != nil {
		slog.Error("get user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// UpdateUser godoc
// @Summary 更新用户
// @Description 根据用户 ID 更新用户信息（邮箱、姓名、状态、密码、角色），需要管理员权限
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户 ID"
// @Param body body UpdateUserRequest true "更新用户请求"
// @Success 200 {object} User "更新后的用户"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "用户不存在"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/users/{id} [put]
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	user, err := h.svc.repo.GetUserByID(r.Context(), id)
	if err != nil {
		slog.Error("get user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email != nil {
		user.Email = req.Email
	}
	if req.RealName != nil {
		user.RealName = req.RealName
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Password != nil && *req.Password != "" {
		hashed, err := HashPassword(*req.Password)
		if err != nil {
			slog.Error("hash password failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		user.Password = hashed
	}

	if err := h.svc.repo.UpdateUser(r.Context(), user); err != nil {
		slog.Error("update user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if req.RoleIDs != nil {
		if err := h.svc.repo.SetUserRoles(r.Context(), user.ID, req.RoleIDs); err != nil {
			slog.Error("set user roles failed", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, user)
}

// DeleteUser godoc
// @Summary 删除用户
// @Description 根据用户 ID 删除用户，需要管理员权限
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户 ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "无效的用户 ID"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/users/{id} [delete]
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	if err := h.svc.repo.DeleteUser(r.Context(), id); err != nil {
		slog.Error("delete user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

// ListRoles godoc
// @Summary 获取角色列表
// @Description 获取系统中所有角色信息，需要管理员权限
// @Tags 角色管理
// @Produce json
// @Security BearerAuth
// @Success 200 {array} Role "角色列表"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/roles [get]
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.repo.ListRoles(r.Context())
	if err != nil {
		slog.Error("list roles failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, roles)
}

// CreateRole godoc
// @Summary 创建角色
// @Description 创建新角色并设置权限列表，需要管理员权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRoleRequest true "创建角色请求"
// @Success 201 {object} Role "创建成功的角色"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/roles [post]
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}

	role := &Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	}
	if role.Permissions == nil {
		role.Permissions = []string{}
	}

	if err := h.svc.repo.CreateRole(r.Context(), role); err != nil {
		slog.Error("create role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create role"})
		return
	}

	writeJSON(w, http.StatusCreated, role)
}

// GetRole godoc
// @Summary 获取角色详情
// @Description 根据角色 ID 获取角色详细信息，需要管理员权限
// @Tags 角色管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色 ID"
// @Success 200 {object} Role "角色详情"
// @Failure 400 {object} map[string]string "无效的角色 ID"
// @Failure 404 {object} map[string]string "角色不存在"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/roles/{id} [get]
func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}

	role, err := h.svc.repo.GetRoleByID(r.Context(), id)
	if err != nil {
		slog.Error("get role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if role == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "role not found"})
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// UpdateRole godoc
// @Summary 更新角色
// @Description 根据角色 ID 更新角色描述和权限列表，需要管理员权限
// @Tags 角色管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色 ID"
// @Param body body UpdateRoleRequest true "更新角色请求"
// @Success 200 {object} Role "更新后的角色"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "角色不存在"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/roles/{id} [put]
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}

	role, err := h.svc.repo.GetRoleByID(r.Context(), id)
	if err != nil {
		slog.Error("get role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if role == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "role not found"})
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Permissions != nil {
		role.Permissions = req.Permissions
	}

	if err := h.svc.repo.UpdateRole(r.Context(), role); err != nil {
		slog.Error("update role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// DeleteRole godoc
// @Summary 删除角色
// @Description 根据角色 ID 删除角色，需要管理员权限
// @Tags 角色管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "角色 ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "无效的角色 ID"
// @Failure 500 {object} map[string]string "服务器内部错误"
// @Router /identity/roles/{id} [delete]
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}

	if err := h.svc.repo.DeleteRole(r.Context(), id); err != nil {
		slog.Error("delete role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "role deleted"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
