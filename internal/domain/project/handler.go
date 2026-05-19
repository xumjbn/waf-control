package project

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// UserLister 由 identity 包提供，返回某 role 范围内的用户（这里复用按 id 列表批量取用户的能力）。
// 仅依赖最小接口，避免与 identity 形成循环依赖。
type UserLister interface {
	GetUserByID(ctx context.Context, id int64) (any, error)
}

type Handler struct {
	repo  *Repository
	users UserLister
}

func NewHandler(repo *Repository, users UserLister) *Handler {
	return &Handler{repo: repo, users: users}
}

const defaultDomainID = "default"

type projectV3 struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	DomainID    string  `json:"domain_id"`
	ParentID    *string `json:"parent_id,omitempty"`
	IsDomain    bool    `json:"is_domain"`
	Enabled     bool    `json:"enabled"`
	Members     int     `json:"members"`
	Sites       int     `json:"sites"`
	Instances   int     `json:"instances"`
	CreatedAt   string  `json:"created_at,omitempty"`
}

type projectPayload struct {
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	ParentID    *string `json:"parent_id,omitempty"`
	IsDomain    *bool   `json:"is_domain,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

type projectWrapper struct {
	Project projectPayload `json:"project"`
}

func toV3(p *Project, members int) projectV3 {
	v := projectV3{
		ID:          strconv.FormatInt(p.ID, 10),
		Name:        p.Name,
		Description: p.Description,
		DomainID:    p.DomainID,
		IsDomain:    p.IsDomain,
		Enabled:     p.Enabled,
		Members:     members,
		Sites:       0,
		Instances:   0,
		CreatedAt:   p.CreatedAt.Format("2006-01-02"),
	}
	if p.ParentID != nil {
		s := strconv.FormatInt(*p.ParentID, 10)
		v.ParentID = &s
	}
	return v
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func parseID(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }

// List godoc
// @Summary 获取项目列表
// @Description 查询所有项目信息
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "项目列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, counts, err := h.repo.ListWithMemberCount(r.Context())
	if err != nil {
		slog.Error("list projects failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	out := make([]projectV3, 0, len(list))
	for i := range list {
		out = append(out, toV3(&list[i], counts[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

// Get godoc
// @Summary 获取项目详情
// @Description 根据ID获取单个项目详情
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 200 {object} map[string]interface{} "项目详情"
// @Failure 400 {object} map[string]string "无效的项目ID"
// @Failure 404 {object} map[string]string "项目不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}
	p, err := h.repo.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if p == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": toV3(p, 0)})
}

// Create godoc
// @Summary 创建项目
// @Description 新增一个项目
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body projectWrapper true "项目信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var wrap projectWrapper
	if err := json.NewDecoder(r.Body).Decode(&wrap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	p := payloadToModel(wrap.Project, nil)
	if p.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	if err := h.repo.Create(r.Context(), p); err != nil {
		slog.Error("create project failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create project"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"project": toV3(p, 0)})
}

// Update godoc
// @Summary 更新项目
// @Description 根据ID更新项目信息
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Param body body projectWrapper true "项目信息"
// @Success 200 {object} map[string]interface{} "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "项目不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}
	p, err := h.repo.Get(r.Context(), id)
	if err != nil || p == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	var wrap projectWrapper
	if err := json.NewDecoder(r.Body).Decode(&wrap); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	updated := payloadToModel(wrap.Project, p)
	if err := h.repo.Update(r.Context(), updated); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": toV3(updated, 0)})
}

// Delete godoc
// @Summary 删除项目
// @Description 根据ID删除项目
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 204 "删除成功"
// @Failure 400 {object} map[string]string "无效的项目ID"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// AssignUserRole godoc
// @Summary 分配用户角色
// @Description 为项目中的用户分配指定角色
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path int true "项目ID"
// @Param user_id path int true "用户ID"
// @Param role_id path int true "角色ID"
// @Success 204 "分配成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects/{project_id}/users/{user_id}/roles/{role_id} [put]
func (h *Handler) AssignUserRole(w http.ResponseWriter, r *http.Request) {
	pid, uid, rid, ok := parseTriple(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.Assign(r.Context(), pid, uid, rid); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// RevokeUserRole godoc
// @Summary 撤销用户角色
// @Description 撤销项目中用户的指定角色
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id path int true "项目ID"
// @Param user_id path int true "用户ID"
// @Param role_id path int true "角色ID"
// @Success 204 "撤销成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects/{project_id}/users/{user_id}/roles/{role_id} [delete]
func (h *Handler) RevokeUserRole(w http.ResponseWriter, r *http.Request) {
	pid, uid, rid, ok := parseTriple(r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.Revoke(r.Context(), pid, uid, rid); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ListProjectUsers godoc
// @Summary 获取项目用户列表
// @Description 返回项目下已绑定的用户列表
// @Tags 项目管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "项目ID"
// @Success 200 {object} map[string]interface{} "项目用户列表"
// @Failure 400 {object} map[string]string "无效的项目ID"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /projects/{id}/users [get]
// ListProjectUsers 返回 project 下已绑定的用户（去重）。
// 借助 UserLister 适配 identity 包的取用户接口；不可用时仅返回 user_id 列表。
func (h *Handler) ListProjectUsers(w http.ResponseWriter, r *http.Request) {
	pid, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}
	ids, err := h.repo.ListProjectUserIDs(r.Context(), pid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	users := make([]any, 0, len(ids))
	for _, uid := range ids {
		u, err := h.users.GetUserByID(r.Context(), uid)
		if err != nil || u == nil {
			users = append(users, map[string]any{"id": strconv.FormatInt(uid, 10)})
			continue
		}
		users = append(users, u)
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func parseTriple(r *http.Request) (pid, uid, rid int64, ok bool) {
	var err error
	pid, err = parseID(chi.URLParam(r, "project_id"))
	if err != nil {
		return 0, 0, 0, false
	}
	uid, err = parseID(chi.URLParam(r, "user_id"))
	if err != nil {
		return 0, 0, 0, false
	}
	rid, err = parseID(chi.URLParam(r, "role_id"))
	if err != nil {
		return 0, 0, 0, false
	}
	return pid, uid, rid, true
}

func payloadToModel(p projectPayload, prev *Project) *Project {
	out := &Project{}
	if prev != nil {
		*out = *prev
	} else {
		out.DomainID = defaultDomainID
		out.Enabled = true
	}
	if p.Name != "" {
		out.Name = p.Name
	}
	out.Description = p.Description
	if p.ParentID != nil {
		if id, err := parseID(*p.ParentID); err == nil {
			out.ParentID = &id
		}
	}
	if p.IsDomain != nil {
		out.IsDomain = *p.IsDomain
	}
	if p.Enabled != nil {
		out.Enabled = *p.Enabled
	}
	return out
}
