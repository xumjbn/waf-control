package network

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

// --- Interfaces ---

// ListInterfaces godoc
// @Summary 获取网络接口列表
// @Description 根据节点ID获取该节点下的所有网络接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Success 200 {object} map[string]interface{} "接口列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/interfaces [get]
func (h *Handler) ListInterfaces(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	ifaces, err := h.repo.ListInterfaces(r.Context(), nodeID)
	if err != nil {
		slog.Error("list interfaces failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": ifaces})
}

// CreateInterface godoc
// @Summary 创建网络接口
// @Description 为指定节点创建新的网络接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param body body CreateInterfaceRequest true "接口信息"
// @Success 201 {object} Interface "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/interfaces [post]
func (h *Handler) CreateInterface(w http.ResponseWriter, r *http.Request) {
	var req CreateInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and name are required"})
		return
	}

	iface, err := h.repo.CreateInterface(r.Context(), req)
	if err != nil {
		slog.Error("create interface failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, iface)
}

// UpdateInterface godoc
// @Summary 更新网络接口
// @Description 根据ID更新网络接口配置
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "接口ID"
// @Param body body UpdateInterfaceRequest true "更新信息"
// @Success 200 {object} Interface "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/interfaces/{id} [put]
func (h *Handler) UpdateInterface(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateInterfaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	iface, err := h.repo.UpdateInterface(r.Context(), id, req)
	if err != nil {
		slog.Error("update interface failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, iface)
}

// DeleteInterface godoc
// @Summary 删除网络接口
// @Description 根据ID删除网络接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "接口ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "接口不存在"
// @Router /nodes/{nodeId}/interfaces/{id} [delete]
func (h *Handler) DeleteInterface(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteInterface(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "interface not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Bridges ---

// ListBridges godoc
// @Summary 获取网桥列表
// @Description 根据节点ID获取该节点下的所有网桥
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Success 200 {object} map[string]interface{} "网桥列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bridges [get]
func (h *Handler) ListBridges(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	bridges, err := h.repo.ListBridges(r.Context(), nodeID)
	if err != nil {
		slog.Error("list bridges failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": bridges})
}

// CreateBridge godoc
// @Summary 创建网桥
// @Description 为指定节点创建新的网桥
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param body body CreateBridgeRequest true "网桥信息"
// @Success 201 {object} Bridge "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bridges [post]
func (h *Handler) CreateBridge(w http.ResponseWriter, r *http.Request) {
	var req CreateBridgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and name are required"})
		return
	}

	bridge, err := h.repo.CreateBridge(r.Context(), req)
	if err != nil {
		slog.Error("create bridge failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, bridge)
}

// UpdateBridge godoc
// @Summary 更新网桥
// @Description 根据ID更新网桥配置
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "网桥ID"
// @Param body body UpdateBridgeRequest true "更新信息"
// @Success 200 {object} Bridge "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bridges/{id} [put]
func (h *Handler) UpdateBridge(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateBridgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	bridge, err := h.repo.UpdateBridge(r.Context(), id, req)
	if err != nil {
		slog.Error("update bridge failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bridge)
}

// DeleteBridge godoc
// @Summary 删除网桥
// @Description 根据ID删除网桥
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "网桥ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "网桥不存在"
// @Router /nodes/{nodeId}/bridges/{id} [delete]
func (h *Handler) DeleteBridge(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteBridge(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bridge not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Bonds ---

// ListBonds godoc
// @Summary 获取Bond列表
// @Description 根据节点ID获取该节点下的所有Bond接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Success 200 {object} map[string]interface{} "Bond列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bonds [get]
func (h *Handler) ListBonds(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	bonds, err := h.repo.ListBonds(r.Context(), nodeID)
	if err != nil {
		slog.Error("list bonds failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": bonds})
}

// CreateBond godoc
// @Summary 创建Bond接口
// @Description 为指定节点创建新的Bond接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param body body CreateBondRequest true "Bond信息"
// @Success 201 {object} Bond "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bonds [post]
func (h *Handler) CreateBond(w http.ResponseWriter, r *http.Request) {
	var req CreateBondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and name are required"})
		return
	}

	bond, err := h.repo.CreateBond(r.Context(), req)
	if err != nil {
		slog.Error("create bond failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, bond)
}

// UpdateBond godoc
// @Summary 更新Bond接口
// @Description 根据ID更新Bond接口配置
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "Bond ID"
// @Param body body UpdateBondRequest true "更新信息"
// @Success 200 {object} Bond "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bonds/{id} [put]
func (h *Handler) UpdateBond(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateBondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	bond, err := h.repo.UpdateBond(r.Context(), id, req)
	if err != nil {
		slog.Error("update bond failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bond)
}

// DeleteBond godoc
// @Summary 删除Bond接口
// @Description 根据ID删除Bond接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "Bond ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "Bond不存在"
// @Router /nodes/{nodeId}/bonds/{id} [delete]
func (h *Handler) DeleteBond(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteBond(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bond not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Routes ---

// ListRoutes godoc
// @Summary 获取路由列表
// @Description 根据节点ID获取该节点下的所有路由
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Success 200 {object} map[string]interface{} "路由列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/routes [get]
func (h *Handler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}

	routes, err := h.repo.ListRoutes(r.Context(), nodeID)
	if err != nil {
		slog.Error("list routes failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": routes})
}

// CreateRoute godoc
// @Summary 创建路由
// @Description 为指定节点创建新的路由规则
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param body body CreateRouteRequest true "路由信息"
// @Success 201 {object} Route "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/routes [post]
func (h *Handler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var req CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == 0 || req.Destination == "" || req.Gateway == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id, destination, and gateway are required"})
		return
	}

	route, err := h.repo.CreateRoute(r.Context(), req)
	if err != nil {
		slog.Error("create route failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, route)
}

// DeleteRoute godoc
// @Summary 删除路由
// @Description 根据ID删除路由规则
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "路由ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "路由不存在"
// @Router /nodes/{nodeId}/routes/{id} [delete]
func (h *Handler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteRoute(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// UpdateRoute godoc
// @Summary 更新路由
// @Description 根据ID更新路由规则配置
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "路由ID"
// @Param body body UpdateRouteRequest true "更新信息"
// @Success 200 {object} Route "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/routes/{id} [put]
func (h *Handler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	rt, err := h.repo.UpdateRoute(r.Context(), id, req)
	if err != nil {
		slog.Error("update route failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, rt)
}

// --- Interface enable/disable ---

// EnableInterface godoc
// @Summary 启用网络接口
// @Description 启用指定节点下的网络接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "接口ID"
// @Success 200 {object} Interface "启用成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/interfaces/{id}/enable [put]
func (h *Handler) EnableInterface(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	iface, err := h.repo.EnableInterface(r.Context(), nodeID, id)
	if err != nil {
		slog.Error("enable interface failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, iface)
}

// DisableInterface godoc
// @Summary 禁用网络接口
// @Description 禁用指定节点下的网络接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "接口ID"
// @Success 200 {object} Interface "禁用成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/interfaces/{id}/disable [put]
func (h *Handler) DisableInterface(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseInt(chi.URLParam(r, "nodeId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid node_id"})
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	iface, err := h.repo.DisableInterface(r.Context(), nodeID, id)
	if err != nil {
		slog.Error("disable interface failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, iface)
}

// --- Bridge/Bond slave operations ---

// AddBridgeSlave godoc
// @Summary 添加网桥从接口
// @Description 为指定网桥添加从属接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "网桥ID"
// @Param body body SlaveRequest true "从接口信息"
// @Success 200 {object} Bridge "添加成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bridges/{id}/slave [post]
func (h *Handler) AddBridgeSlave(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req SlaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	bridge, err := h.repo.AddBridgeSlave(r.Context(), id, req.Slave)
	if err != nil {
		slog.Error("add bridge slave failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bridge)
}

// DelBridgeSlave godoc
// @Summary 移除网桥从接口
// @Description 从指定网桥移除从属接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "网桥ID"
// @Param slaveId path string true "从接口ID"
// @Success 200 {object} Bridge "移除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bridges/{id}/slave/{slaveId} [delete]
func (h *Handler) DelBridgeSlave(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	slaveID := chi.URLParam(r, "slaveId")
	if slaveID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slave_id required"})
		return
	}

	bridge, err := h.repo.DelBridgeSlave(r.Context(), id, slaveID)
	if err != nil {
		slog.Error("del bridge slave failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bridge)
}

// AddBondSlave godoc
// @Summary 添加Bond从接口
// @Description 为指定Bond添加从属接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "Bond ID"
// @Param body body SlaveRequest true "从接口信息"
// @Success 200 {object} Bond "添加成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bonds/{id}/slave [post]
func (h *Handler) AddBondSlave(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req SlaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	bond, err := h.repo.AddBondSlave(r.Context(), id, req.Slave)
	if err != nil {
		slog.Error("add bond slave failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bond)
}

// DelBondSlave godoc
// @Summary 移除Bond从接口
// @Description 从指定Bond移除从属接口
// @Tags 网络管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param nodeId path int true "节点ID"
// @Param id path int true "Bond ID"
// @Param slaveId path string true "从接口ID"
// @Success 200 {object} Bond "移除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /nodes/{nodeId}/bonds/{id}/slave/{slaveId} [delete]
func (h *Handler) DelBondSlave(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	slaveID := chi.URLParam(r, "slaveId")
	if slaveID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slave_id required"})
		return
	}

	bond, err := h.repo.DelBondSlave(r.Context(), id, slaveID)
	if err != nil {
		slog.Error("del bond slave failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, bond)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
