package policy

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

// --- Categories ---

// ListCategories godoc
// @Summary 获取策略分类列表
// @Description 查询所有策略分类
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "分类列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policy-categories [get]
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.repo.ListCategories(r.Context())
	if err != nil {
		slog.Error("list categories failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": cats})
}

// CreateCategory godoc
// @Summary 创建策略分类
// @Description 创建新的策略分类
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateCategoryRequest true "分类信息"
// @Success 201 {object} Category "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policy-categories [post]
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	cat, err := h.repo.CreateCategory(r.Context(), req)
	if err != nil {
		slog.Error("create category failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, cat)
}

// UpdateCategory godoc
// @Summary 更新策略分类
// @Description 根据ID更新策略分类
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "分类ID"
// @Param body body UpdateCategoryRequest true "更新信息"
// @Success 200 {object} Category "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policy-categories/{id} [put]
func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cat, err := h.repo.UpdateCategory(r.Context(), id, req)
	if err != nil {
		slog.Error("update category failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, cat)
}

// DeleteCategory godoc
// @Summary 删除策略分类
// @Description 根据ID删除策略分类
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "分类ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "分类不存在"
// @Router /policy-categories/{id} [delete]
func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteCategory(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Policies ---

// ListPolicies godoc
// @Summary 获取策略列表
// @Description 分页查询策略列表，支持按分类、严重级别、动作、启用状态和关键词筛选
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param category_id query int false "分类ID"
// @Param severity query string false "严重级别"
// @Param action query string false "动作"
// @Param is_enabled query bool false "是否启用"
// @Param search query string false "搜索关键词"
// @Success 200 {object} map[string]interface{} "策略列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policies [get]
func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	params := ListPolicyParams{
		Page:     page,
		PageSize: pageSize,
		Severity: r.URL.Query().Get("severity"),
		Action:   r.URL.Query().Get("action"),
		Search:   r.URL.Query().Get("search"),
	}

	if cid := r.URL.Query().Get("category_id"); cid != "" {
		if v, err := strconv.ParseInt(cid, 10, 64); err == nil {
			params.CategoryID = &v
		}
	}
	if en := r.URL.Query().Get("is_enabled"); en != "" {
		v := en == "true"
		params.IsEnabled = &v
	}

	policies, total, err := h.repo.ListPolicies(r.Context(), params)
	if err != nil {
		slog.Error("list policies failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  policies,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetPolicy godoc
// @Summary 获取策略详情
// @Description 根据ID获取策略详细信息
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "策略ID"
// @Success 200 {object} Policy "策略详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "策略不存在"
// @Router /policies/{id} [get]
func (h *Handler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	pol, err := h.repo.GetPolicy(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "policy not found"})
		return
	}

	writeJSON(w, http.StatusOK, pol)
}

// CreatePolicy godoc
// @Summary 创建策略
// @Description 创建新的安全策略
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreatePolicyRequest true "策略信息"
// @Success 201 {object} Policy "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policies [post]
func (h *Handler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	pol, err := h.repo.CreatePolicy(r.Context(), req)
	if err != nil {
		slog.Error("create policy failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, pol)
}

// UpdatePolicy godoc
// @Summary 更新策略
// @Description 根据ID更新策略信息
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "策略ID"
// @Param body body UpdatePolicyRequest true "更新信息"
// @Success 200 {object} Policy "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policies/{id} [put]
func (h *Handler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	pol, err := h.repo.UpdatePolicy(r.Context(), id, req)
	if err != nil {
		slog.Error("update policy failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, pol)
}

// DeletePolicy godoc
// @Summary 删除策略
// @Description 根据ID删除策略
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "策略ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "策略不存在"
// @Router /policies/{id} [delete]
func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeletePolicy(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "policy not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Rules ---

// ListRules godoc
// @Summary 获取策略规则列表
// @Description 根据策略ID获取该策略下的所有规则
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "策略ID"
// @Success 200 {object} map[string]interface{} "规则列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policies/{id}/rules [get]
func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	policyID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid policy id"})
		return
	}

	rules, err := h.repo.ListRules(r.Context(), policyID)
	if err != nil {
		slog.Error("list rules failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": rules})
}

// CreateRule godoc
// @Summary 创建策略规则
// @Description 为指定策略创建新的匹配规则
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "策略ID"
// @Param body body CreateRuleRequest true "规则信息"
// @Success 201 {object} Rule "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policies/{id}/rules [post]
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	policyID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid policy id"})
		return
	}

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.PolicyID = policyID

	if req.RuleType == "" || req.Field == "" || req.Operator == "" || req.Value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rule_type, field, operator, and value are required"})
		return
	}

	rule, err := h.repo.CreateRule(r.Context(), req)
	if err != nil {
		slog.Error("create rule failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

// DeleteRule godoc
// @Summary 删除策略规则
// @Description 根据规则ID删除策略下的指定规则
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "策略ID"
// @Param ruleId path int true "规则ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "规则不存在"
// @Router /policies/{id}/rules/{ruleId} [delete]
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "ruleId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rule id"})
		return
	}

	if err := h.repo.DeleteRule(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- History ---

// ListHistory godoc
// @Summary 获取策略变更历史
// @Description 根据策略ID获取变更历史记录
// @Tags 策略管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "策略ID"
// @Param limit query int false "返回数量限制"
// @Success 200 {object} map[string]interface{} "变更历史列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /policies/{id}/history [get]
func (h *Handler) ListHistory(w http.ResponseWriter, r *http.Request) {
	policyID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid policy id"})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	history, err := h.repo.ListHistory(r.Context(), policyID, limit)
	if err != nil {
		slog.Error("list history failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": history})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
