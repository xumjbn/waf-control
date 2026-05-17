package loadbalance

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

// --- VIPs ---

// ListVIPs godoc
// @Summary 获取VIP列表
// @Description 查询所有虚拟IP地址
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "VIP列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/vips [get]
func (h *Handler) ListVIPs(w http.ResponseWriter, r *http.Request) {
	vips, err := h.repo.ListVIPs(r.Context())
	if err != nil {
		slog.Error("list vips failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": vips})
}

// CreateVIP godoc
// @Summary 创建VIP
// @Description 创建新的虚拟IP地址
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateVIPRequest true "VIP信息"
// @Success 201 {object} VIP "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/vips [post]
func (h *Handler) CreateVIP(w http.ResponseWriter, r *http.Request) {
	var req CreateVIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.Address == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and address are required"})
		return
	}

	vip, err := h.repo.CreateVIP(r.Context(), req)
	if err != nil {
		slog.Error("create vip failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, vip)
}

// UpdateVIP godoc
// @Summary 更新VIP
// @Description 根据ID更新虚拟IP地址配置
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "VIP ID"
// @Param body body UpdateVIPRequest true "更新信息"
// @Success 200 {object} VIP "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/vips/{id} [put]
func (h *Handler) UpdateVIP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateVIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	vip, err := h.repo.UpdateVIP(r.Context(), id, req)
	if err != nil {
		slog.Error("update vip failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, vip)
}

// GetVIP godoc
// @Summary 获取VIP详情
// @Description 根据ID获取虚拟IP地址详细信息
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "VIP ID"
// @Success 200 {object} VIP "VIP详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "VIP不存在"
// @Router /lb/vips/{id} [get]
func (h *Handler) GetVIP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	vip, err := h.repo.GetVIP(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "vip not found"})
		return
	}
	writeJSON(w, http.StatusOK, vip)
}

// DeleteVIP godoc
// @Summary 删除VIP
// @Description 根据ID删除虚拟IP地址
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "VIP ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "VIP不存在"
// @Router /lb/vips/{id} [delete]
func (h *Handler) DeleteVIP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteVIP(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "vip not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Pools ---

// ListPools godoc
// @Summary 获取资源池列表
// @Description 查询所有负载均衡资源池
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "资源池列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools [get]
func (h *Handler) ListPools(w http.ResponseWriter, r *http.Request) {
	pools, err := h.repo.ListPools(r.Context())
	if err != nil {
		slog.Error("list pools failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": pools})
}

// CreatePool godoc
// @Summary 创建资源池
// @Description 创建新的负载均衡资源池
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreatePoolRequest true "资源池信息"
// @Success 201 {object} Pool "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools [post]
func (h *Handler) CreatePool(w http.ResponseWriter, r *http.Request) {
	var req CreatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	pool, err := h.repo.CreatePool(r.Context(), req)
	if err != nil {
		slog.Error("create pool failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, pool)
}

// UpdatePool godoc
// @Summary 更新资源池
// @Description 根据ID更新负载均衡资源池配置
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源池ID"
// @Param body body UpdatePoolRequest true "更新信息"
// @Success 200 {object} Pool "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools/{id} [put]
func (h *Handler) UpdatePool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	pool, err := h.repo.UpdatePool(r.Context(), id, req)
	if err != nil {
		slog.Error("update pool failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, pool)
}

// GetPool godoc
// @Summary 获取资源池详情
// @Description 根据ID获取负载均衡资源池详细信息
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源池ID"
// @Success 200 {object} Pool "资源池详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "资源池不存在"
// @Router /lb/pools/{id} [get]
func (h *Handler) GetPool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	pool, err := h.repo.GetPool(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "pool not found"})
		return
	}
	writeJSON(w, http.StatusOK, pool)
}

// DeletePool godoc
// @Summary 删除资源池
// @Description 根据ID删除负载均衡资源池
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源池ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "资源池不存在"
// @Router /lb/pools/{id} [delete]
func (h *Handler) DeletePool(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeletePool(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "pool not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Members ---

// ListMembers godoc
// @Summary 获取资源池成员列表
// @Description 根据资源池ID获取该池下的所有成员
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Success 200 {object} map[string]interface{} "成员列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools/{poolId}/members [get]
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	members, err := h.repo.ListMembers(r.Context(), poolID)
	if err != nil {
		slog.Error("list members failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": members})
}

// CreateMember godoc
// @Summary 创建资源池成员
// @Description 为指定资源池添加新的后端成员
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Param body body CreateMemberRequest true "成员信息"
// @Success 201 {object} Member "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools/{poolId}/members [post]
func (h *Handler) CreateMember(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	var req CreateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.PoolID = poolID

	if req.Address == "" || req.ProtocolPort <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "address and protocol_port are required"})
		return
	}

	member, err := h.repo.CreateMember(r.Context(), req)
	if err != nil {
		slog.Error("create member failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

// GetMember godoc
// @Summary 获取资源池成员详情
// @Description 根据成员ID获取成员详细信息
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Param memberId path int true "成员ID"
// @Success 200 {object} Member "成员详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "成员不存在"
// @Router /lb/pools/{poolId}/members/{memberId} [get]
func (h *Handler) GetMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "memberId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}

	member, err := h.repo.GetMember(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
		return
	}
	writeJSON(w, http.StatusOK, member)
}

// UpdateMember godoc
// @Summary 更新资源池成员
// @Description 根据成员ID更新成员配置
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Param memberId path int true "成员ID"
// @Param body body UpdateMemberRequest true "更新信息"
// @Success 200 {object} Member "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools/{poolId}/members/{memberId} [put]
func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "memberId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}

	var req UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	member, err := h.repo.UpdateMember(r.Context(), id, req)
	if err != nil {
		slog.Error("update member failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, member)
}

// DeleteMember godoc
// @Summary 删除资源池成员
// @Description 根据成员ID删除资源池成员
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Param memberId path int true "成员ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "成员不存在"
// @Router /lb/pools/{poolId}/members/{memberId} [delete]
func (h *Handler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "memberId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}

	if err := h.repo.DeleteMember(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "member not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Health Monitors ---

// GetHealthMonitor godoc
// @Summary 获取健康检查配置
// @Description 根据资源池ID获取健康检查监控配置
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Success 200 {object} HealthMonitor "健康检查配置"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "健康检查不存在"
// @Router /lb/pools/{poolId}/health-monitor [get]
func (h *Handler) GetHealthMonitor(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	hm, err := h.repo.GetHealthMonitor(r.Context(), poolID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "health monitor not found"})
		return
	}
	writeJSON(w, http.StatusOK, hm)
}

// CreateHealthMonitor godoc
// @Summary 创建健康检查配置
// @Description 为指定资源池创建健康检查监控配置
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Param body body CreateHealthMonitorRequest true "健康检查配置信息"
// @Success 201 {object} HealthMonitor "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools/{poolId}/health-monitor [post]
func (h *Handler) CreateHealthMonitor(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	var req CreateHealthMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	req.PoolID = poolID

	hm, err := h.repo.CreateHealthMonitor(r.Context(), req)
	if err != nil {
		slog.Error("create health monitor failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, hm)
}

// UpdateHealthMonitor godoc
// @Summary 更新健康检查配置
// @Description 根据资源池ID更新健康检查监控配置
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Param body body UpdateHealthMonitorRequest true "更新信息"
// @Success 200 {object} HealthMonitor "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /lb/pools/{poolId}/health-monitor [put]
func (h *Handler) UpdateHealthMonitor(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	var req UpdateHealthMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	hm, err := h.repo.UpdateHealthMonitor(r.Context(), poolID, req)
	if err != nil {
		slog.Error("update health monitor failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, hm)
}

// DeleteHealthMonitor godoc
// @Summary 删除健康检查配置
// @Description 根据资源池ID删除健康检查监控配置
// @Tags 负载均衡
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param poolId path int true "资源池ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "健康检查不存在"
// @Router /lb/pools/{poolId}/health-monitor [delete]
func (h *Handler) DeleteHealthMonitor(w http.ResponseWriter, r *http.Request) {
	poolID, err := strconv.ParseInt(chi.URLParam(r, "poolId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool id"})
		return
	}

	if err := h.repo.DeleteHealthMonitor(r.Context(), poolID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "health monitor not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
