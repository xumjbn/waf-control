package ha

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// GetConfig godoc
// @Summary 获取高可用配置
// @Description 查询当前高可用集群的配置信息
// @Tags 高可用
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Config "高可用配置详情"
// @Failure 404 {object} map[string]string "配置未找到"
// @Router /ha/config [get]
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.repo.Get(r.Context())
	if err != nil {
		slog.Error("get ha config failed", "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ha config not found"})
		return
	}
	writeJSON(w, http.StatusOK, config)
}

// UpsertConfig godoc
// @Summary 创建或更新高可用配置
// @Description 创建或更新高可用集群的配置，包括模式、虚拟IP、优先级等参数
// @Tags 高可用
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpsertConfigRequest true "高可用配置参数"
// @Success 200 {object} Config "配置更新成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /ha/config [put]
func (h *Handler) UpsertConfig(w http.ResponseWriter, r *http.Request) {
	var req UpsertConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	config, err := h.repo.Upsert(r.Context(), req)
	if err != nil {
		slog.Error("upsert ha config failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
