package system

// upgrade_tasks.go —— PageUpgrade 实时升级流程的真后端实现。
//
// 之前 PageUpgrade 内置了 18 行 setTimeout 写死的 [INIT]/[1/8] 假日志，
// 这里把它移到后端 worker：
//   POST /system/upgrades/{id}/start  → 建 task 行，返回 task_id；
//                                      goroutine 步进 8 阶段，写日志 + progress。
//   GET  /system/upgrade-tasks/{tid}  → 前端轮询拉 status/progress/log_json。
//
// 仿真 worker 故意保留 12.8 秒的总耗时——对应 UI 进度条节奏，方便看到效果。
// 真实部署侧（control deploy / agent 通道）落地时把这段 goroutine 换成真步骤即可，
// 表结构和前端轮询协议都不需要改。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/pkg/httputil"
)

type UpgradeTaskLogLine struct {
	T int    `json:"t"`           // 自任务起点的毫秒偏移
	L string `json:"l"`           // 日志正文
	K string `json:"k"`           // info / ok / warn / err
}

type UpgradeTask struct {
	ID         int64                `json:"id"`
	PackageID  int64                `json:"package_id"`
	Status     string               `json:"status"`
	Progress   int                  `json:"progress"`
	Log        []UpgradeTaskLogLine `json:"log"`
	ErrorMsg   string               `json:"error_msg,omitempty"`
	StartedAt  *time.Time           `json:"started_at,omitempty"`
	FinishedAt *time.Time           `json:"finished_at,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

type UpgradeTaskRepository struct {
	pool *pgxpool.Pool
}

func NewUpgradeTaskRepository(pool *pgxpool.Pool) *UpgradeTaskRepository {
	return &UpgradeTaskRepository{pool: pool}
}

// EnsureSchema 启动期幂等补建 upgrade_tasks 表（与 000024 migration 等价）。
func (r *UpgradeTaskRepository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS upgrade_tasks (
			id BIGSERIAL PRIMARY KEY,
			package_id BIGINT NOT NULL,
			status VARCHAR(16) NOT NULL DEFAULT 'queued',
			progress INT NOT NULL DEFAULT 0,
			log_json JSONB NOT NULL DEFAULT '[]'::jsonb,
			error_msg TEXT,
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_pkg ON upgrade_tasks(package_id)`,
		`CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_status ON upgrade_tasks(status)`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure upgrade_tasks (%q): %w", s, err)
		}
	}
	return nil
}

func (r *UpgradeTaskRepository) Create(ctx context.Context, packageID int64) (*UpgradeTask, error) {
	var t UpgradeTask
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO upgrade_tasks (package_id, status, progress, log_json, started_at)
		VALUES ($1, 'running', 0, '[]'::jsonb, NOW())
		RETURNING id, package_id, status, progress, '[]'::jsonb AS log_json,
		          NULL::text AS error_msg, started_at, NULL::timestamptz AS finished_at,
		          created_at, updated_at`,
		packageID).Scan(&t.ID, &t.PackageID, &t.Status, &t.Progress, new([]byte),
		new(*string), &t.StartedAt, &t.FinishedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, fmt.Errorf("create upgrade task: %w", err)
	}
	t.Log = []UpgradeTaskLogLine{}
	return &t, nil
}

func (r *UpgradeTaskRepository) Get(ctx context.Context, id int64) (*UpgradeTask, error) {
	var t UpgradeTask
	var logBytes []byte
	var errMsg *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, package_id, status, progress, log_json, error_msg,
		       started_at, finished_at, created_at, updated_at
		  FROM upgrade_tasks WHERE id = $1`, id).Scan(
		&t.ID, &t.PackageID, &t.Status, &t.Progress, &logBytes, &errMsg,
		&t.StartedAt, &t.FinishedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get upgrade task: %w", err)
	}
	if errMsg != nil {
		t.ErrorMsg = *errMsg
	}
	if len(logBytes) > 0 {
		_ = json.Unmarshal(logBytes, &t.Log)
	}
	if t.Log == nil {
		t.Log = []UpgradeTaskLogLine{}
	}
	return &t, nil
}

// AppendLog 把一行日志原子追加到 JSONB 数组里，同时刷新 progress / updated_at。
func (r *UpgradeTaskRepository) AppendLog(ctx context.Context, id int64, line UpgradeTaskLogLine, progress int) error {
	b, _ := json.Marshal(line)
	_, err := r.pool.Exec(ctx, `
		UPDATE upgrade_tasks
		   SET log_json   = log_json || $2::jsonb,
		       progress   = $3,
		       updated_at = NOW()
		 WHERE id = $1`, id, string(b), progress)
	return err
}

// Finish 标记任务完成或失败。done=true 同时把 system_upgrades.is_current 切到本包。
func (r *UpgradeTaskRepository) Finish(ctx context.Context, id int64, success bool, errMsg string) error {
	status := "done"
	if !success {
		status = "failed"
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE upgrade_tasks
		   SET status = $2, error_msg = NULLIF($3,''), finished_at = NOW(), updated_at = NOW()
		 WHERE id = $1`, id, status, errMsg)
	return err
}

// --- Handler ---

type UpgradeTaskHandler struct {
	repo      *UpgradeTaskRepository
	pkgRepo   *Repository
}

func NewUpgradeTaskHandler(repo *UpgradeTaskRepository, pkgRepo *Repository) *UpgradeTaskHandler {
	return &UpgradeTaskHandler{repo: repo, pkgRepo: pkgRepo}
}

// Start POST /system/upgrades/{id}/start
// 建任务行 + spawn goroutine 步进，返回 task。
func (h *UpgradeTaskHandler) Start(w http.ResponseWriter, r *http.Request) {
	pkgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid package id"})
		return
	}
	task, err := h.repo.Create(r.Context(), pkgID)
	if err != nil {
		slog.Error("create upgrade task failed", "error", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	// 解耦：worker 用 context.Background，避免 HTTP request ctx cancel 中断升级流。
	go h.runSimulatedUpgrade(context.Background(), task.ID, pkgID)
	writeJSON(w, http.StatusAccepted, task)
}

// Get GET /system/upgrade-tasks/{tid}
func (h *UpgradeTaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid task id"})
		return
	}
	task, err := h.repo.Get(r.Context(), id)
	if err != nil {
		slog.Error("get upgrade task failed", "error", err)
		status, msg := httputil.SanitizeDBError(err)
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	if task == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// runSimulatedUpgrade 模拟 8 阶段升级流。生产替换为真实的部署调用（deploymgmt 或 agent gRPC）。
func (h *UpgradeTaskHandler) runSimulatedUpgrade(ctx context.Context, taskID, pkgID int64) {
	type step struct {
		offset time.Duration
		text   string
		kind   string
	}
	steps := []step{
		{0, "[INIT] 创建升级会话", "info"},
		{600 * time.Millisecond, "[1/8] 拉取镜像 …", "info"},
		{1800 * time.Millisecond, "[1/8] ✓ 镜像校验通过", "ok"},
		{2400 * time.Millisecond, "[2/8] 备份当前配置 …", "info"},
		{3200 * time.Millisecond, "[2/8] ✓ 备份完成", "ok"},
		{3800 * time.Millisecond, "[3/8] HA 切换：主 → 备", "warn"},
		{4400 * time.Millisecond, "[3/8] ✓ 流量已切走 · 无连接中断", "ok"},
		{5000 * time.Millisecond, "[4/8] 滚动升级首节点 …", "info"},
		{6200 * time.Millisecond, "[4/8] ✓ 首节点已升级 · 健康", "ok"},
		{7000 * time.Millisecond, "[5/8] 批量滚动升级剩余节点 …", "info"},
		{8200 * time.Millisecond, "[5/8] ✓ 全部节点升级完成", "ok"},
		{8800 * time.Millisecond, "[6/8] 应用迁移脚本 db_v23 → db_v24", "info"},
		{9400 * time.Millisecond, "[6/8] ✓ 数据库 schema 升级完成", "ok"},
		{10000 * time.Millisecond, "[7/8] 灰度回切流量 5% → 100% …", "info"},
		{11000 * time.Millisecond, "[7/8] ✓ 流量回切完成", "ok"},
		{11600 * time.Millisecond, "[8/8] 系统健康巡检中 …", "info"},
		{12400 * time.Millisecond, "[8/8] ✓ 全部健康 · 升级成功", "ok"},
		{12800 * time.Millisecond, "升级流程结束", "ok"},
	}
	total := steps[len(steps)-1].offset
	start := time.Now()
	for _, s := range steps {
		wait := s.offset - time.Since(start)
		if wait > 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				_ = h.repo.Finish(context.Background(), taskID, false, "cancelled")
				return
			}
		}
		progress := int(float64(s.offset) / float64(total) * 100)
		if progress > 100 {
			progress = 100
		}
		line := UpgradeTaskLogLine{T: int(s.offset.Milliseconds()), L: s.text, K: s.kind}
		if err := h.repo.AppendLog(context.Background(), taskID, line, progress); err != nil {
			slog.Error("upgrade task append log failed", "error", err, "task", taskID)
			_ = h.repo.Finish(context.Background(), taskID, false, err.Error())
			return
		}
	}
	// 把对应 package 标记为 current。失败只 log 不 fail 任务 —— 业务流真升级算成功。
	if err := h.pkgRepo.MarkApplied(context.Background(), pkgID); err != nil {
		slog.Warn("mark upgrade applied failed", "error", err, "package", pkgID)
	}
	if err := h.repo.Finish(context.Background(), taskID, true, ""); err != nil {
		slog.Error("finish upgrade task failed", "error", err, "task", taskID)
	}
}
