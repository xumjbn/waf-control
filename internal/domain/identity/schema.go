package identity

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ensureSchemaStmts is a defensive, idempotent superset of the column
// additions in migrations 000007 + 000008. It exists because:
//
//   - The repository now SELECTs role_key / readonly / color / avatar /
//     project / last_login on hot paths (login, /me, /users).
//   - In multi-environment deploys it's easy for the code roll-out to
//     race ahead of the migration runner. When that happens
//     GetUserByUsername blows up at the SELECT and the user sees a generic
//     500 — login is then completely unrecoverable until someone notices
//     the SQL log.
//
// Running these on every startup is cheap (six ALTERs, all `IF NOT EXISTS`)
// and self-heals the drift. The "real" migrations remain in
// internal/store/migrations/ as the source of truth for fresh installs.
var ensureSchemaStmts = []string{
	`ALTER TABLE roles ADD COLUMN IF NOT EXISTS role_key VARCHAR(64)`,
	`ALTER TABLE roles ADD COLUMN IF NOT EXISTS readonly BOOLEAN NOT NULL DEFAULT FALSE`,
	`ALTER TABLE roles ADD COLUMN IF NOT EXISTS color    VARCHAR(16) NOT NULL DEFAULT '#a855f7'`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar     VARCHAR(8)`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS project    VARCHAR(64)`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login TIMESTAMPTZ`,
}

// EnsureSchema runs the ALTER-IF-NOT-EXISTS statements above. Safe to call
// multiple times; safe to call before any migration runner has executed
// (each statement is independent and idempotent).
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	for _, stmt := range ensureSchemaStmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure schema (%s): %w", stmt, err)
		}
	}
	return nil
}
