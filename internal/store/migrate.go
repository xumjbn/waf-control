package store

// migrate.go 是 waf-control 内置的极小 migration runner（不依赖 golang-migrate）。
//
// 设计目标：
//   1. 编译期 embed.FS 嵌入 internal/store/migrations/*.up.sql；
//   2. 启动时维护 schema_migrations(version BIGINT PK, name TEXT, applied_at TIMESTAMPTZ)；
//   3. 对已经被 EnsureSchema 跑出来的现有库 baseline：首次启动若 schema_migrations
//      为空但 users 表已存在，则把全部已知 migration 直接标记为 applied（不再重跑）。
//   4. 新机器从空库起：按版本号顺序逐个事务 apply。
//
// 各 domain 里的 EnsureSchema 仍然保留作为旁路 fallback —— 一是兼容 baseline
// 路径，二是部分增量列 ALTER 在 SQL migration 之外还有 Go 侧补丁。

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version int64
	name    string
	up      string
}

// MigrateUp 执行所有未应用的 up migration。pool 必须非 nil。
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	ms, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	applied, err := loadAppliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	// baseline：第一次启动 + 已存在 users 表 ⇒ 把全部 migration 标 applied，不重跑。
	if len(applied) == 0 {
		var hasUsers bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				 WHERE table_schema = 'public' AND table_name = 'users'
			)`).Scan(&hasUsers); err != nil {
			return fmt.Errorf("probe baseline: %w", err)
		}
		if hasUsers {
			slog.Info("migration baseline: existing schema detected, marking all migrations applied", "count", len(ms))
			for _, m := range ms {
				if _, err := pool.Exec(ctx,
					`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)
				     ON CONFLICT (version) DO NOTHING`, m.version, m.name); err != nil {
					return fmt.Errorf("baseline insert v%d: %w", m.version, err)
				}
				applied[m.version] = true
			}
			return nil
		}
	}

	for _, m := range ms {
		if applied[m.version] {
			continue
		}
		slog.Info("applying migration", "version", m.version, "name", m.name)
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin v%d: %w", m.version, err)
		}
		if _, err := tx.Exec(ctx, m.up); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply v%d %s: %w", m.version, m.name, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`,
			m.version, m.name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record v%d: %w", m.version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit v%d: %w", m.version, err)
		}
	}
	return nil
}

func loadAppliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int64]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query applied: %w", err)
	}
	defer rows.Close()
	applied := map[int64]bool{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	var ms []migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		parts := strings.SplitN(strings.TrimSuffix(name, ".up.sql"), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed migration name %q", name)
		}
		v, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse version in %q: %w", name, err)
		}
		body, err := fs.ReadFile(migrationFiles, path.Join("migrations", name))
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", name, err)
		}
		ms = append(ms, migration{version: v, name: parts[1], up: string(body)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].version < ms[j].version })
	return ms, nil
}
