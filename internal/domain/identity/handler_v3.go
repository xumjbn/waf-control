package identity

// OpenStack Keystone v3 风格的适配层，把内部 User/Role 结构以 `{ user: ... }`
// `{ role: ... }` 包裹的形式暴露给 waf-admin 前端。
//
// 前端路径 /users、/roles 由 server.go 中以别名形式挂载到本 handler。

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const defaultDomainID = "default"

type userV3 struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Email       *string `json:"email,omitempty"`
	Enabled     bool    `json:"enabled"`
	DomainID    string  `json:"domain_id"`
	Description *string `json:"description,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
	Role        string  `json:"role,omitempty"`
	Project     string  `json:"project,omitempty"`
	LastLogin   string  `json:"last_login,omitempty"`
}

type roleV3 struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	DomainID    string `json:"domain_id"`
	Permissions string `json:"permissions,omitempty"`
	UserCount   int    `json:"user_count"`
}

type userPayload struct {
	Name        string  `json:"name"`
	Password    string  `json:"password,omitempty"`
	Email       *string `json:"email,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	Description *string `json:"description,omitempty"`
}

type rolePayload struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type userWrapper struct {
	User userPayload `json:"user"`
}

type roleWrapper struct {
	Role rolePayload `json:"role"`
}

func toUserV3(u *User) userV3 {
	if u == nil {
		return userV3{}
	}
	v := userV3{
		ID:          strconv.FormatInt(u.ID, 10),
		Name:        u.Username,
		Email:       u.Email,
		Enabled:     u.IsActive,
		DomainID:    defaultDomainID,
		Description: u.RealName,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if u.LastLogin != nil {
		v.LastLogin = u.LastLogin.Format("2006-01-02 15:04")
	}
	return v
}

func toUserV3Enriched(ue *UserEnriched) userV3 {
	v := toUserV3(&ue.User)
	if ue.RoleName != nil {
		v.Role = *ue.RoleName
	}
	if ue.ProjectName != nil {
		v.Project = *ue.ProjectName
	} else if v.Role == "系统管理员" || v.Role == "安全分析师" {
		v.Project = "全部"
	}
	return v
}

func toRoleV3(r *Role) roleV3 {
	if r == nil {
		return roleV3{}
	}
	return roleV3{
		ID:          strconv.FormatInt(r.ID, 10),
		Name:        r.Name,
		Description: r.Description,
		DomainID:    defaultDomainID,
	}
}

func toRoleV3WithCount(r *Role, count int) roleV3 {
	v := toRoleV3(r)
	v.UserCount = count
	return v
}

func parseID(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }

// === Users (v3 wrap) ===

func (h *Handler) ListUsersV3(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.repo.ListUsersEnriched(r.Context())
	if err != nil {
		slog.Error("v3 list users failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	out := make([]userV3, 0, len(users))
	for i := range users {
		out = append(out, toUserV3Enriched(&users[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (h *Handler) GetUserV3(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	u, err := h.svc.repo.GetUserByID(r.Context(), id)
	if err != nil {
		slog.Error("v3 get user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if u == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": toUserV3(u)})
}

func (h *Handler) CreateUserV3(w http.ResponseWriter, r *http.Request) {
	var wrap userWrapper
	if err := json.NewDecoder(r.Body).Decode(&wrap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	p := wrap.User
	if p.Name == "" || p.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and password required"})
		return
	}
	hashed, err := HashPassword(p.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	enabled := true
	if p.Enabled != nil {
		enabled = *p.Enabled
	}
	u := &User{
		Username: p.Name,
		Password: hashed,
		Email:    p.Email,
		RealName: p.Description,
		IsActive: enabled,
	}
	if err := h.svc.repo.CreateUser(r.Context(), u); err != nil {
		slog.Error("v3 create user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": toUserV3(u)})
}

func (h *Handler) UpdateUserV3(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	u, err := h.svc.repo.GetUserByID(r.Context(), id)
	if err != nil || u == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	var wrap userWrapper
	if err := json.NewDecoder(r.Body).Decode(&wrap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	p := wrap.User
	if p.Email != nil {
		u.Email = p.Email
	}
	if p.Description != nil {
		u.RealName = p.Description
	}
	if p.Enabled != nil {
		u.IsActive = *p.Enabled
	}
	if p.Password != "" {
		hashed, hErr := HashPassword(p.Password)
		if hErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		u.Password = hashed
	}
	if err := h.svc.repo.UpdateUser(r.Context(), u); err != nil {
		slog.Error("v3 update user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": toUserV3(u)})
}

func (h *Handler) DeleteUserV3(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	if err := h.svc.repo.DeleteUser(r.Context(), id); err != nil {
		slog.Error("v3 delete user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) ListUserRolesV3(w http.ResponseWriter, r *http.Request) {
	uid, err := parseID(chi.URLParam(r, "user_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	roles, err := h.svc.repo.GetUserRoles(r.Context(), uid)
	if err != nil {
		slog.Error("v3 list user roles failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	out := make([]roleV3, 0, len(roles))
	for i := range roles {
		out = append(out, toRoleV3(&roles[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": out})
}

// === Roles (v3 wrap) ===

func (h *Handler) ListRolesV3(w http.ResponseWriter, r *http.Request) {
	roles, counts, err := h.svc.repo.ListRolesWithCount(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	out := make([]roleV3, 0, len(roles))
	for i := range roles {
		out = append(out, toRoleV3WithCount(&roles[i], counts[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": out})
}

func (h *Handler) GetRoleV3(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}
	role, err := h.svc.repo.GetRoleByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if role == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "role not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"role": toRoleV3(role)})
}

func (h *Handler) CreateRoleV3(w http.ResponseWriter, r *http.Request) {
	var wrap roleWrapper
	if err := json.NewDecoder(r.Body).Decode(&wrap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if wrap.Role.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	role := &Role{
		Name:        wrap.Role.Name,
		Description: wrap.Role.Description,
		Permissions: []string{"read"},
	}
	if err := h.svc.repo.CreateRole(r.Context(), role); err != nil {
		slog.Error("v3 create role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create role"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"role": toRoleV3(role)})
}

func (h *Handler) UpdateRoleV3(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}
	role, err := h.svc.repo.GetRoleByID(r.Context(), id)
	if err != nil || role == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "role not found"})
		return
	}
	var wrap roleWrapper
	if err := json.NewDecoder(r.Body).Decode(&wrap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if wrap.Role.Name != "" {
		role.Name = wrap.Role.Name
	}
	role.Description = wrap.Role.Description
	if err := h.svc.repo.UpdateRole(r.Context(), role); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"role": toRoleV3(role)})
}

func (h *Handler) DeleteRoleV3(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}
	if err := h.svc.repo.DeleteRole(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// === User-Role assignment (v3) ===

func (h *Handler) AssignUserRoleV3(w http.ResponseWriter, r *http.Request) {
	roleID, err := parseID(chi.URLParam(r, "role_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}
	userID, err := parseID(chi.URLParam(r, "user_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	if err := h.svc.repo.AssignUserRole(r.Context(), userID, roleID); err != nil {
		slog.Error("v3 assign user role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) RevokeUserRoleV3(w http.ResponseWriter, r *http.Request) {
	roleID, err := parseID(chi.URLParam(r, "role_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role id"})
		return
	}
	userID, err := parseID(chi.URLParam(r, "user_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}
	if err := h.svc.repo.RevokeUserRole(r.Context(), userID, roleID); err != nil {
		slog.Error("v3 revoke user role failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
