package deploymgmt

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/waf-control/internal/agent"
	"github.com/waf-control/internal/deploy"
	"github.com/waf-control/internal/domain/identity"
	"github.com/waf-control/internal/domain/site"
)

type Handler struct {
	repo      *Repository
	siteRepo  *site.Repository
	agentSvc  *agent.Service
}

func NewHandler(repo *Repository, siteRepo *site.Repository, agentSvc *agent.Service) *Handler {
	return &Handler{repo: repo, siteRepo: siteRepo, agentSvc: agentSvc}
}

// DeploySite 对站点执行部署
func (h *Handler) DeploySite(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.DeployType == "" {
		req.DeployType = "full"
	}

	// Get site info
	s, err := h.siteRepo.GetByID(r.Context(), siteID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return
	}

	// Get operator info
	operatorName := "unknown"
	var operatorID int64
	if claims := identity.GetClaimsFromContext(r.Context()); claims != nil {
		operatorID = claims.UserID
		operatorName = claims.Username
	}

	// Parse upstreams
	var upstreams []string
	if s.Upstream != nil {
		json.Unmarshal(s.Upstream, &upstreams)
	}

	// Generate configs
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
	modsecConfig := deploy.GenerateModsecPublic(&deploy.PolicyConfig{
		Name: s.Name,
		Mode: "on",
	})

	// Resolve target nodes
	targetNodes := h.resolveTargetNodes(req.TargetNodes)

	// Create deployment record
	version := strconv.FormatInt(time.Now().UnixMilli(), 10)
	dep := &Deployment{
		SiteID:        siteID,
		SiteName:      s.Name,
		SiteDomain:    s.Domain,
		ConfigVersion: version,
		DeployType:    req.DeployType,
		NginxConfig:   nginxConfig,
		ModsecConfig:  modsecConfig,
		TargetNodes:   targetNodes,
		OperatorID:    operatorID,
		OperatorName:  operatorName,
	}

	if err := h.repo.Create(r.Context(), dep); err != nil {
		slog.Error("create deployment record failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Create node status records
	if err := h.repo.CreateNodeStatuses(r.Context(), dep.ID, targetNodes); err != nil {
		slog.Warn("create node statuses failed", "error", err)
	}

	// Broadcast configs via gRPC
	for _, node := range targetNodes {
		h.agentSvc.BroadcastConfig(node.Hostname, 0, []byte(nginxConfig))
		time.Sleep(50 * time.Millisecond)
		if req.DeployType == "full" || req.DeployType == "policy" {
			h.agentSvc.BroadcastConfig(node.Hostname, 1, []byte(modsecConfig))
			time.Sleep(50 * time.Millisecond)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id":  dep.ID,
		"config_version": version,
		"status":         "deploying",
		"node_count":     len(targetNodes),
	})
}

// ListDeployments 查询站点的部署历史
func (h *Handler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	deployments, total, err := h.repo.ListBySite(r.Context(), siteID, page, pageSize)
	if err != nil {
		slog.Error("list deployments failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  deployments,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetDeployment 查询单次部署详情
func (h *Handler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid deployment id"})
		return
	}

	dep, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"deployment": dep})
}

// PreviewConfig 预览站点生成的配置
func (h *Handler) PreviewConfig(w http.ResponseWriter, r *http.Request) {
	siteID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid site id"})
		return
	}

	s, err := h.siteRepo.GetByID(r.Context(), siteID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "site not found"})
		return
	}

	var upstreams []string
	if s.Upstream != nil {
		json.Unmarshal(s.Upstream, &upstreams)
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

	preview := DeployPreview{
		SiteConfig:   deploy.GenerateNginxPublic(siteCfg),
		PolicyConfig: deploy.GenerateModsecPublic(&deploy.PolicyConfig{Name: s.Name, Mode: "on"}),
	}

	writeJSON(w, http.StatusOK, preview)
}

func (h *Handler) resolveTargetNodes(ids []int64) []TargetNode {
	connected := h.agentSvc.GetConnectedNodes()

	// If specific nodes requested, filter
	if len(ids) > 0 {
		idSet := make(map[int64]bool, len(ids))
		for _, id := range ids {
			idSet[id] = true
		}
		var nodes []TargetNode
		for _, ns := range connected {
			// Parse node ID from hostname
			hash := hashHostname(ns.Hostname)
			if idSet[hash] {
				nodes = append(nodes, TargetNode{ID: hash, Hostname: ns.Hostname, IP: ns.IP})
			}
		}
		return nodes
	}

	// Default: all connected nodes
	nodes := make([]TargetNode, 0, len(connected))
	for _, ns := range connected {
		nodes = append(nodes, TargetNode{ID: hashHostname(ns.Hostname), Hostname: ns.Hostname, IP: ns.IP})
	}
	return nodes
}

func hashHostname(s string) int64 {
	var h int64
	for i := 0; i < len(s); i++ {
		h = h*31 + int64(s[i])
	}
	if h < 0 {
		h = -h
	}
	return h
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
