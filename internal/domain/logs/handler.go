package logs

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/waf-control/internal/domain/acl"
)

type Handler struct {
	repo *Repository
	acl  *acl.Repository // 可空：用于 IP 封禁 / 加入白名单
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// NewHandlerWithACL 注入 acl 仓库，从而启用 /logs/attack/{id}/ban
// 与 /logs/attack/{id}/whitelist 这两类来自前端 PageLogAttack 的按钮。
func NewHandlerWithACL(repo *Repository, aclRepo *acl.Repository) *Handler {
	return &Handler{repo: repo, acl: aclRepo}
}

// ListAttackLogs godoc
// @Summary 获取攻击日志列表
// @Description 分页查询攻击日志，支持按节点、时间范围筛选
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param node_id query int false "节点ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "攻击日志列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/attack [get]
func (h *Handler) ListAttackLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	logs, total, err := h.repo.ListAttackLogs(r.Context(), q)
	if err != nil {
		slog.Error("list attack logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": logs, "total": total, "page": q.Page, "page_size": q.PageSize})
}

// ListAntivirusLogs godoc
// @Summary 获取防病毒日志列表
// @Description 分页查询防病毒日志，支持按节点、时间范围筛选
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param node_id query int false "节点ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "防病毒日志列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/antivirus [get]
func (h *Handler) ListAntivirusLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	logs, total, err := h.repo.ListAntivirusLogs(r.Context(), q)
	if err != nil {
		slog.Error("list antivirus logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": logs, "total": total, "page": q.Page, "page_size": q.PageSize})
}

// ListAntitamperLogs godoc
// @Summary 获取防篡改日志列表
// @Description 分页查询防篡改日志，支持按节点、时间范围筛选
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param node_id query int false "节点ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "防篡改日志列表"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/antitamper [get]
func (h *Handler) ListAntitamperLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	logs, total, err := h.repo.ListAntitamperLogs(r.Context(), q)
	if err != nil {
		slog.Error("list antitamper logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": logs, "total": total, "page": q.Page, "page_size": q.PageSize})
}

// --- Attack log sub-endpoints ---

// GetAttackLog godoc
// @Summary 获取攻击日志详情
// @Description 根据ID获取单条攻击日志详情
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "攻击日志ID"
// @Success 200 {object} map[string]interface{} "攻击日志详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "攻击日志不存在"
// @Router /logs/attack/{id} [get]
func (h *Handler) GetAttackLog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	logEntry, err := h.repo.GetAttackLog(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "attack log not found"})
		return
	}
	writeJSON(w, http.StatusOK, logEntry)
}

// CountAttackLogs godoc
// @Summary 统计攻击日志数量
// @Description 根据筛选条件统计攻击日志总数
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param node_id query int false "节点ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} map[string]int64 "攻击日志总数"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/attack/count [get]
func (h *Handler) CountAttackLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	total, err := h.repo.CountAttackLogs(r.Context(), q)
	if err != nil {
		slog.Error("count attack logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"total": total})
}

// ClearAttackLogs godoc
// @Summary 清空攻击日志
// @Description 删除所有攻击日志记录
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "清空成功"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/attack [delete]
func (h *Handler) ClearAttackLogs(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.ClearAttackLogs(r.Context()); err != nil {
		slog.Error("clear attack logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cleared"})
}

// IngestAttackLog 接受 agent 或内部模块上报的富攻击日志。
// POST /logs/attack
func (h *Handler) IngestAttackLog(w http.ResponseWriter, r *http.Request) {
	var req IngestAttackLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	id, err := h.repo.IngestAttackLog(r.Context(), &req.AttackLog)
	if err != nil {
		slog.Error("ingest attack log failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

// BanIP 把目标日志的 src_ip 写入 acl 黑名单（direction=in、action=deny）。
// POST /logs/attack/{id}/ban
func (h *Handler) BanIP(w http.ResponseWriter, r *http.Request) {
	if h.acl == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "acl repo unavailable"})
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	log, err := h.repo.GetAttackLog(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "attack log not found"})
		return
	}
	if log.SrcIP == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "log has no src_ip"})
		return
	}
	rule, err := h.acl.Create(r.Context(), acl.CreateRuleRequest{
		Name:        fmt.Sprintf("封禁 %s (来自日志 #%d)", log.SrcIP, log.ID),
		Description: fmt.Sprintf("自 attack_logs#%d 自动生成；攻击类型 %s", log.ID, log.AttackType),
		Direction:   "in",
		Action:      "deny",
		Protocol:    "any",
		SrcIP:       log.SrcIP,
		Priority:    100,
	})
	if err != nil {
		slog.Error("ban ip failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

// WhitelistIP 把 src_ip 写入 acl 白名单（direction=in、action=allow、最高优先级）。
// POST /logs/attack/{id}/whitelist
func (h *Handler) WhitelistIP(w http.ResponseWriter, r *http.Request) {
	if h.acl == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "acl repo unavailable"})
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	log, err := h.repo.GetAttackLog(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "attack log not found"})
		return
	}
	if log.SrcIP == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "log has no src_ip"})
		return
	}
	rule, err := h.acl.Create(r.Context(), acl.CreateRuleRequest{
		Name:        fmt.Sprintf("白名单 %s (来自日志 #%d)", log.SrcIP, log.ID),
		Description: fmt.Sprintf("自 attack_logs#%d 自动生成", log.ID),
		Direction:   "in",
		Action:      "allow",
		Protocol:    "any",
		SrcIP:       log.SrcIP,
		Priority:    1, // 越小越优先
	})
	if err != nil {
		slog.Error("whitelist ip failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

// RelatedEvents 返回与目标日志 src_ip 相同的近期攻击日志。
// GET /logs/attack/{id}/related?limit=50
func (h *Handler) RelatedEvents(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	log, err := h.repo.GetAttackLog(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "attack log not found"})
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 && n <= 500 {
			limit = n
		}
	}
	q := LogQuery{SrcIP: log.SrcIP, Page: 1, PageSize: limit}
	logs, total, err := h.repo.ListAttackLogs(r.Context(), q)
	if err != nil {
		slog.Error("related events failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":   logs,
		"total":  total,
		"src_ip": log.SrcIP,
	})
}

// --- Antivirus log sub-endpoints ---

// GetAntivirusLog godoc
// @Summary 获取防病毒日志详情
// @Description 根据ID获取单条防病毒日志详情
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "防病毒日志ID"
// @Success 200 {object} map[string]interface{} "防病毒日志详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "防病毒日志不存在"
// @Router /logs/antivirus/{id} [get]
func (h *Handler) GetAntivirusLog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	logEntry, err := h.repo.GetAntivirusLog(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "antivirus log not found"})
		return
	}
	writeJSON(w, http.StatusOK, logEntry)
}

// CountAntivirusLogs godoc
// @Summary 统计防病毒日志数量
// @Description 根据筛选条件统计防病毒日志总数
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param node_id query int false "节点ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} map[string]int64 "防病毒日志总数"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/antivirus/count [get]
func (h *Handler) CountAntivirusLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	total, err := h.repo.CountAntivirusLogs(r.Context(), q)
	if err != nil {
		slog.Error("count antivirus logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"total": total})
}

// ClearAntivirusLogs godoc
// @Summary 清空防病毒日志
// @Description 删除所有防病毒日志记录
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "清空成功"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/antivirus [delete]
func (h *Handler) ClearAntivirusLogs(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.ClearAntivirusLogs(r.Context()); err != nil {
		slog.Error("clear antivirus logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cleared"})
}

// --- Antitamper log sub-endpoints ---

// GetAntitamperLog godoc
// @Summary 获取防篡改日志详情
// @Description 根据ID获取单条防篡改日志详情
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "防篡改日志ID"
// @Success 200 {object} map[string]interface{} "防篡改日志详情"
// @Failure 400 {object} map[string]string "无效的ID"
// @Failure 404 {object} map[string]string "防篡改日志不存在"
// @Router /logs/antitamper/{id} [get]
func (h *Handler) GetAntitamperLog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	logEntry, err := h.repo.GetAntitamperLog(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "antitamper log not found"})
		return
	}
	writeJSON(w, http.StatusOK, logEntry)
}

// CountAntitamperLogs godoc
// @Summary 统计防篡改日志数量
// @Description 根据筛选条件统计防篡改日志总数
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param node_id query int false "节点ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} map[string]int64 "防篡改日志总数"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/antitamper/count [get]
func (h *Handler) CountAntitamperLogs(w http.ResponseWriter, r *http.Request) {
	q := parseLogQuery(r)
	total, err := h.repo.CountAntitamperLogs(r.Context(), q)
	if err != nil {
		slog.Error("count antitamper logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"total": total})
}

// ClearAntitamperLogs godoc
// @Summary 清空防篡改日志
// @Description 删除所有防篡改日志记录
// @Tags 日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "清空成功"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /logs/antitamper [delete]
func (h *Handler) ClearAntitamperLogs(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.ClearAntitamperLogs(r.Context()); err != nil {
		slog.Error("clear antitamper logs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "cleared"})
}

func parseLogQuery(r *http.Request) LogQuery {
	q := LogQuery{
		StartTime: r.URL.Query().Get("start_time"),
		EndTime:   r.URL.Query().Get("end_time"),
		Risk:      r.URL.Query().Get("risk"),
		Site:      r.URL.Query().Get("site"),
		Country:   r.URL.Query().Get("country"),
		SrcIP:     r.URL.Query().Get("src_ip"),
	}
	if v := r.URL.Query().Get("node_id"); v != "" {
		q.NodeID, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := r.URL.Query().Get("page"); v != "" {
		q.Page, _ = strconv.Atoi(v)
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		q.PageSize, _ = strconv.Atoi(v)
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	return q
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
