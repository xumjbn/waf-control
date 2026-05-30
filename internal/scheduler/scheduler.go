package scheduler

// scheduler.go —— waf-control 的后台定时调度器。
//
// 之前 report_timing.cron / system_upgrades 自动跟进字段都存了，但没有任何东西
// 消费它们 —— 所有"定时"实际不会触发。这里补上一个每分钟 tick 的 goroutine：
//   - 扫 report_timing 启用行，cron 到点 → 触发报表生成回调 + 回写 last_run_at/next_run_at
//
// 设计：
//   - 用 OnReportDue 回调解耦真实生成逻辑（reports 包注入），避免 import 环。
//   - 幂等：用 last_run_at 防同一分钟重复触发（进程重启也安全）。
//   - 单实例假设：多副本部署需要分布式锁（advisory lock），见 docs/api-conventions 待办。

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Scheduler struct {
	pool *pgxpool.Pool
	// OnReportDue 在某个定时报表到点时被调用（kind 固定 "timing"）。
	// 由 main.go 注入 reports 包的真实生成函数；nil 时仅回写时间戳。
	OnReportDue func(ctx context.Context, reportID int64)
}

func New(pool *pgxpool.Pool) *Scheduler {
	return &Scheduler{pool: pool}
}

// Start 阻塞运行调度循环，直到 ctx 取消。通常 `go sched.Start(ctx)`。
// 对齐到整分钟边界后每分钟 tick。
func (s *Scheduler) Start(ctx context.Context) {
	// 先睡到下一个整分钟，让 tick 落在 :00 秒附近。
	now := time.Now()
	next := now.Truncate(time.Minute).Add(time.Minute)
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(next)):
	}

	s.tick(ctx) // 立即跑一次对齐后的 tick
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	slog.Info("scheduler started", "interval", "1m")
	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("scheduler tick panicked", "recover", r)
		}
	}()
	s.runDueTimingReports(ctx)
}

// runDueTimingReports 扫启用的定时报表，cron 命中且本分钟未跑过 → 触发。
func (s *Scheduler) runDueTimingReports(ctx context.Context) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(cron,'0 0 * * *'), last_run_at
		  FROM report_timing
		 WHERE COALESCE(is_enabled, true) = true`)
	if err != nil {
		// 表不存在（schema 未初始化）等 → 静默跳过，下个 tick 再来。
		return
	}
	type due struct {
		id   int64
		cron string
	}
	var dues []due
	now := time.Now()
	thisMinute := now.Truncate(time.Minute)
	for rows.Next() {
		var id int64
		var cronExpr string
		var lastRun *time.Time
		if err := rows.Scan(&id, &cronExpr, &lastRun); err != nil {
			continue
		}
		// 本分钟已跑过 → 跳过（防重启 / 多 tick 重复）。
		if lastRun != nil && !lastRun.Before(thisMinute) {
			continue
		}
		sched, err := Parse(cronExpr)
		if err != nil {
			slog.Warn("invalid cron in report_timing", "id", id, "cron", cronExpr, "error", err)
			continue
		}
		if sched.Matches(now) {
			dues = append(dues, due{id: id, cron: cronExpr})
		}
	}
	rows.Close()

	for _, d := range dues {
		nextRun := time.Time{}
		if sched, err := Parse(d.cron); err == nil {
			nextRun = sched.Next(now)
		}
		// 先回写时间戳（幂等保护），再触发生成。
		if _, err := s.pool.Exec(ctx, `
			UPDATE report_timing
			   SET last_run_at = NOW(),
			       next_run_at = $2,
			       updated_at  = NOW()
			 WHERE id = $1`, d.id, nullableTime(nextRun)); err != nil {
			slog.Error("scheduler bump report_timing failed", "id", d.id, "error", err)
			continue
		}
		slog.Info("scheduled report due", "id", d.id, "cron", d.cron)
		if s.OnReportDue != nil {
			// 用 background ctx，避免被 tick 的 ctx 误取消（生成可能略久）。
			go s.OnReportDue(context.Background(), d.id)
		}
	}
}

func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
