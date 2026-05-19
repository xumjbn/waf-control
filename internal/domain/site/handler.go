package site

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/waf-control/internal/agent"
	"github.com/waf-control/internal/deploy"
	pb "github.com/waf-control/proto/agent"
)

type Handler struct {
	repo     *Repository
	agentSvc *agent.Service
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func NewHandlerWithAgent(repo *Repository, agentSvc *agent.Service) *Handler {
	return &Handler{repo: repo, agentSvc: agentSvc}
}

// List godoc
// @Summary 获取站点列表
// @Description 分页查询站点列表，支持按状态和关键字筛选
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "状态筛选"
// @Param search query string false "搜索关键字"
// @Success 200 {object} map[string]interface{} "站点列表，包含 data、total、page、size 字段"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /sites [get]
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

	sites, total, err := h.repo.List(r.Context(), params)
	if err != nil {
		slog.Error("list sites failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  sites,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// Get godoc
// @Summary 获取站点详情
// @Description 根据站点 ID 获取站点详细信息
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "站点 ID"
// @Success 200 {object} Site "站点详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "站点不存在"
// @Router /sites/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	s, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return
	}

	writeJSON(w, http.StatusOK, s)
}

// Create godoc
// @Summary 创建站点
// @Description 创建新的防护站点，需提供站点名称和域名
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRequest true "站点创建参数"
// @Success 201 {object} Site "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /sites [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and domain are required"})
		return
	}

	s, err := h.repo.Create(r.Context(), req)
	if err != nil {
		slog.Error("create site failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	h.broadcastSite(s)
	writeJSON(w, http.StatusCreated, s)
}

// Update godoc
// @Summary 更新站点
// @Description 根据站点 ID 更新站点信息，支持部分字段更新
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "站点 ID"
// @Param body body UpdateRequest true "站点更新参数"
// @Success 200 {object} Site "更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /sites/{id} [put]
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

	s, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		slog.Error("update site failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	h.broadcastSite(s)
	writeJSON(w, http.StatusOK, s)
}

// Delete godoc
// @Summary 删除站点
// @Description 根据站点 ID 删除站点
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "站点 ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "站点不存在"
// @Router /sites/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	s, _ := h.repo.GetByID(r.Context(), id)

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return
	}

	h.broadcastDelete(s)
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// ListDevices godoc
// @Summary 获取站点绑定的设备列表
// @Description 根据站点 ID 获取该站点绑定的所有设备 ID 列表
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "站点 ID"
// @Success 200 {object} map[string]interface{} "设备 ID 列表"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /sites/{id}/devices [get]
func (h *Handler) ListDevices(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	ids, err := h.repo.ListDevices(r.Context(), siteID)
	if err != nil {
		slog.Error("list site devices failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"device_ids": ids})
}

// BindDevice godoc
// @Summary 绑定设备到站点
// @Description 将指定设备绑定到站点，建立站点与设备的关联关系
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "站点 ID"
// @Param body body object true "绑定参数" example({"device_id": 1})
// @Success 200 {object} map[string]string "绑定成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /sites/{id}/devices [post]
func (h *Handler) BindDevice(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}

	var body struct {
		DeviceID int64 `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DeviceID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "device_id is required"})
		return
	}

	if err := h.repo.BindDevice(r.Context(), siteID, body.DeviceID); err != nil {
		slog.Error("bind device failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "bound"})
}

// UnbindDevice godoc
// @Summary 解绑站点设备
// @Description 解除站点与指定设备的绑定关系
// @Tags 站点管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "站点 ID"
// @Param deviceId path int true "设备 ID"
// @Success 200 {object} map[string]string "解绑成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /sites/{id}/devices/{deviceId} [delete]
func (h *Handler) UnbindDevice(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}

	deviceID, err := strconv.ParseInt(chi.URLParam(r, "deviceId"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid device id"})
		return
	}

	if err := h.repo.UnbindDevice(r.Context(), siteID, deviceID); err != nil {
		slog.Error("unbind device failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "unbound"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) broadcastSite(s *Site) {
	if h.agentSvc == nil || s == nil {
		return
	}
	var upstreams []string
	if s.Upstream != nil {
		_ = json.Unmarshal(s.Upstream, &upstreams)
	}
	siteCfg := &deploy.SiteConfig{
		Domain:     s.Domain,
		Protocol:   "http",
		Upstreams:  upstreams,
		WAFEnabled: s.WAFEnabled,
	}
	if s.SSLEnabled {
		siteCfg.Protocol = "https"
		siteCfg.SSLName = s.Name
	}
	nginxConfig := deploy.GenerateNginxPublic(siteCfg)
	nodes := h.agentSvc.GetConnectedNodes()
	for _, ns := range nodes {
		h.agentSvc.BroadcastConfig(ns.Hostname, pb.ConfigUpdate_SITE, []byte(nginxConfig))
	}
	slog.Info("broadcast site config", "site_id", s.ID, "domain", s.Domain, "agents", len(nodes))
}

func (h *Handler) broadcastDelete(s *Site) {
	if h.agentSvc == nil || s == nil {
		return
	}
	payload := []byte("# delete:" + s.Domain + "\n")
	for _, ns := range h.agentSvc.GetConnectedNodes() {
		h.agentSvc.BroadcastConfig(ns.Hostname, pb.ConfigUpdate_SITE, payload)
	}
}
