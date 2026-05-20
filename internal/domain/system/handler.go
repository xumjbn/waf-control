package system

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

// --- Settings ---

// ListSettings godoc
// @Summary 获取系统设置列表
// @Description 查询系统设置，支持按分类筛选
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category query string false "设置分类"
// @Success 200 {object} map[string]interface{} "系统设置列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /system/settings [get]
func (h *Handler) ListSettings(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	settings, err := h.repo.ListSettings(r.Context(), category)
	if err != nil {
		slog.Error("list settings failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": settings})
}

// UpsertSetting godoc
// @Summary 创建或更新系统设置
// @Description 根据key创建或更新系统设置项
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpsertSettingRequest true "设置信息"
// @Success 200 {object} map[string]interface{} "设置详情"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /system/settings [put]
func (h *Handler) UpsertSetting(w http.ResponseWriter, r *http.Request) {
	var req UpsertSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key is required"})
		return
	}

	setting, err := h.repo.UpsertSetting(r.Context(), req)
	if err != nil {
		slog.Error("upsert setting failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, setting)
}

// DeleteSetting godoc
// @Summary 删除系统设置
// @Description 根据key删除系统设置项
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "设置key"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 404 {object} map[string]string "设置不存在"
// @Router /system/settings/{key} [delete]
func (h *Handler) DeleteSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if err := h.repo.DeleteSetting(r.Context(), key); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "setting not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Licenses ---

// ListLicenses godoc
// @Summary 获取许可证列表
// @Description 查询所有许可证信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "许可证列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /system/licenses [get]
func (h *Handler) ListLicenses(w http.ResponseWriter, r *http.Request) {
	licenses, err := h.repo.ListLicenses(r.Context())
	if err != nil {
		slog.Error("list licenses failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	// 顺手补 status 字段（前端展示用）。
	out := make([]map[string]any, 0, len(licenses))
	for _, l := range licenses {
		out = append(out, licenseToMap(l))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": out})
}

// CurrentLicense  GET /system/license
// 返回当前激活的 license 详情（含 status 推断），NW · 09 系统页"当前授权"卡片消费。
func (h *Handler) CurrentLicense(w http.ResponseWriter, r *http.Request) {
	l, err := h.repo.CurrentLicense(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no active license"})
		return
	}
	writeJSON(w, http.StatusOK, licenseToMap(*l))
}

func licenseToMap(l License) map[string]any {
	return map[string]any{
		"id":            l.ID,
		"license_key":   l.LicenseKey,
		"product_name":  l.ProductName,
		"edition":       l.Edition,
		"customer":      l.Customer,
		"contact_email": l.ContactEmail,
		"max_nodes":     l.MaxNodes,
		"issued_at":     l.IssuedAt,
		"expires_at":    l.ExpiresAt,
		"grace_until":   l.GraceUntil,
		"is_active":     l.IsActive,
		"status":        l.Status(),
		"created_at":    l.CreatedAt,
	}
}

// CreateLicense godoc
// @Summary 创建许可证
// @Description 新增一个许可证记录
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateLicenseRequest true "许可证信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /system/licenses [post]
func (h *Handler) CreateLicense(w http.ResponseWriter, r *http.Request) {
	var req CreateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.LicenseKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "license_key is required"})
		return
	}

	license, err := h.repo.CreateLicense(r.Context(), req)
	if err != nil {
		slog.Error("create license failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, license)
}

// ActivateLicense godoc
// @Summary 激活许可证
// @Description 根据ID激活指定许可证
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "许可证ID"
// @Success 200 {object} map[string]string "激活成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "许可证不存在"
// @Router /system/licenses/{id}/activate [post]
func (h *Handler) ActivateLicense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.ActivateLicense(r.Context(), id); err != nil {
		slog.Error("activate license failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "activated"})
}

// DeleteLicense godoc
// @Summary 删除许可证
// @Description 根据ID删除许可证
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "许可证ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "许可证不存在"
// @Router /system/licenses/{id} [delete]
func (h *Handler) DeleteLicense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteLicense(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// --- Upgrades ---

// ListUpgrades godoc
// @Summary 获取升级包列表
// @Description 查询所有升级包信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "升级包列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /system/upgrades [get]
func (h *Handler) ListUpgrades(w http.ResponseWriter, r *http.Request) {
	upgrades, err := h.repo.ListUpgrades(r.Context())
	if err != nil {
		slog.Error("list upgrades failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": upgrades})
}

// CheckUpgrade 一次返回"当前版本 + 最新可用版本"，前端 PageUpgrade 顶部消费。
// GET /system/upgrades/check
func (h *Handler) CheckUpgrade(w http.ResponseWriter, r *http.Request) {
	current, _ := h.repo.CurrentUpgrade(r.Context())
	latest, _ := h.repo.LatestUpgrade(r.Context())
	out := map[string]any{
		"current":             current,
		"latest":              latest,
		"upgrade_available":   current != nil && latest != nil && current.Version != latest.Version,
	}
	writeJSON(w, http.StatusOK, out)
}

// ApplyUpgrade POST /system/upgrades/{id}/apply
// 触发安装完成（real worker 会异步执行，这里做"标记完成"）。
func (h *Handler) ApplyUpgrade(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.repo.MarkApplied(r.Context(), id); err != nil {
		slog.Error("apply upgrade failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "applied"})
}

// CreateUpgrade godoc
// @Summary 创建升级包
// @Description 新增一个升级包记录
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateUpgradeRequest true "升级包信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /system/upgrades [post]
func (h *Handler) CreateUpgrade(w http.ResponseWriter, r *http.Request) {
	var req CreateUpgradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "version is required"})
		return
	}

	upgrade, err := h.repo.CreateUpgrade(r.Context(), req)
	if err != nil {
		slog.Error("create upgrade failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, upgrade)
}

// TriggerUpgrade godoc
// @Summary 触发升级
// @Description 根据ID触发系统升级操作
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "升级包ID"
// @Success 200 {object} map[string]string "触发成功"
// @Failure 400 {object} map[string]string "触发失败"
// @Router /system/upgrades/{id}/trigger [post]
func (h *Handler) TriggerUpgrade(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.TriggerUpgrade(r.Context(), id); err != nil {
		slog.Error("trigger upgrade failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "upgrade triggered"})
}

// DeleteUpgrade godoc
// @Summary 删除升级包
// @Description 根据ID删除升级包
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "升级包ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "升级包不存在"
// @Router /system/upgrades/{id} [delete]
func (h *Handler) DeleteUpgrade(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.repo.DeleteUpgrade(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "upgrade not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
