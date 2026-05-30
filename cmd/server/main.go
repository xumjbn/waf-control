package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/waf-control/internal/agent"
	"github.com/waf-control/internal/config"
	"github.com/waf-control/internal/domain/reports"
	"github.com/waf-control/internal/scheduler"
	"github.com/waf-control/internal/server"
	"github.com/waf-control/internal/store"

	_ "github.com/waf-control/docs"
)

// @title WAF Control API
// @version 1.0
// @description WAF 防火墙管理系统控制面 API 接口文档
// @termsOfService http://swagger.io/terms/

// @contact.name API 支持
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer {token} 格式的 JWT 令牌

func main() {
	configPath := flag.String("config", "configs/config.toml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setupLogger(cfg.Log)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := store.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		slog.Warn("database not available, running without DB", "error", err)
		pool = nil
	}
	if pool != nil {
		defer pool.Close()
		// 启动期跑 SQL migration（带 baseline 兜底，对已用 EnsureSchema 出来的旧库不会重跑）。
		// 失败必须 fail-closed —— schema 不一致继续启动只会让 handler 后续 500 反复刷错。
		if err := store.MigrateUp(ctx, pool); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
	}

	var grpcSrv *agent.Server
	if pool != nil && cfg.Server.GRPCPort > 0 {
		grpcSrv = agent.NewServer(pool, cfg.Server.GRPCPort)
		go func() {
			if err := grpcSrv.Start(); err != nil {
				slog.Error("grpc server failed", "error", err)
			}
		}()
	}

	// 后台定时调度器：每分钟扫 report_timing.cron，到点触发报表生成 + 回写时间戳。
	// schedCtx 独立于启动用的 10s ctx，随进程退出取消。
	var schedCancel context.CancelFunc
	if pool != nil {
		var schedCtx context.Context
		schedCtx, schedCancel = context.WithCancel(context.Background())
		sched := scheduler.New(pool)
		gen := reports.NewGenerator(pool)
		sched.OnReportDue = func(c context.Context, reportID int64) {
			if err := gen.GenerateTiming(c, reportID); err != nil {
				slog.Error("scheduled report generation failed", "id", reportID, "error", err)
			}
		}
		go sched.Start(schedCtx)
	}

	srv := server.New(cfg, pool, grpcSrv)

	httpServer := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", cfg.Server.Addr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	if schedCancel != nil {
		schedCancel()
	}
	if grpcSrv != nil {
		grpcSrv.Stop()
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}

func setupLogger(cfg config.LogConfig) {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
