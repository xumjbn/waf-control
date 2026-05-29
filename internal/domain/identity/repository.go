package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// userSelectColumns is shared by every users SELECT below. The trailing
// avatar / project / last_login columns were added in migration 000008
// (and last_login in 000007); they are nullable so a row that pre-dates the
// migration scans cleanly into the *string / *time.Time fields.
const userSelectColumns = `id, username, password, email, real_name, is_active,
	avatar, project, last_login, created_at, updated_at`

// scanUser populates a User from a single row that selected userSelectColumns
// in order. Kept as a helper so query call sites stay short.
func scanUser(row pgx.Row, u *User) error {
	return row.Scan(&u.ID, &u.Username, &u.Password, &u.Email, &u.RealName,
		&u.IsActive, &u.Avatar, &u.Project, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt)
}

// --- Users ---

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectColumns+` FROM users WHERE username = $1`, username)

	var u User
	if err := scanUser(row, &u); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectColumns+` FROM users WHERE id = $1`, id)

	var u User
	if err := scanUser(row, &u); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *Repository) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userSelectColumns+` FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := scanUser(rows, &u); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

// ListUsersWithRoles 一次取所有 user + 全部 user_role 关联，handler 侧 group。
// 取代 handler 里 `for user { GetUserRoles(uid) }` 的 N+1 模式。
// 两条 SQL 共 O(U)+O(UR)，无论 user 数量多少都恒定开销。
func (r *Repository) ListUsersWithRoles(ctx context.Context) ([]User, error) {
	users, err := r.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return users, nil
	}
	// 单条 JOIN 把 (user_id, role) 全部一次拉回。无 user_id 过滤 —— 用户量级
	// 现实里 < 几百，全表 scan 配主键 join 比 IN(...) 更省 plan。
	rows, err := r.pool.Query(ctx, `
		SELECT ur.user_id,
		       r.id, r.name, COALESCE(r.role_key,'') AS role_key,
		       r.description, r.permissions,
		       COALESCE(r.readonly,false) AS readonly,
		       COALESCE(r.color,'') AS color,
		       r.created_at, r.updated_at
		  FROM user_roles ur
		  JOIN roles r ON r.id = ur.role_id
		 ORDER BY ur.user_id, r.id`)
	if err != nil {
		return nil, fmt.Errorf("list user_roles: %w", err)
	}
	defer rows.Close()
	byUser := map[int64][]Role{}
	for rows.Next() {
		var uid int64
		var role Role
		var permsJSON []byte
		if err := rows.Scan(&uid, &role.ID, &role.Name, &role.RoleKey,
			&role.Description, &permsJSON, &role.Readonly, &role.Color,
			&role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user_role: %w", err)
		}
		if err := unmarshalPermissions(permsJSON, &role.Permissions); err != nil {
			return nil, err
		}
		byUser[uid] = append(byUser[uid], role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range users {
		users[i].Roles = byUser[users[i].ID]
	}
	return users, nil
}

type UserEnriched struct {
	User
	RoleName    *string
	ProjectName *string
}

func (r *Repository) ListUsersEnriched(ctx context.Context) ([]UserEnriched, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.username, u.password, u.email, u.real_name, u.is_active,
		       u.last_login, u.created_at, u.updated_at,
		       (SELECT ro.name FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ur.user_id = u.id LIMIT 1) AS role_name,
		       (SELECT p.name FROM project_user_roles pur JOIN projects p ON p.id = pur.project_id WHERE pur.user_id = u.id LIMIT 1) AS project_name
		FROM users u ORDER BY u.id`)
	if err != nil {
		return nil, fmt.Errorf("list users enriched: %w", err)
	}
	defer rows.Close()

	var users []UserEnriched
	for rows.Next() {
		var ue UserEnriched
		if err := rows.Scan(&ue.ID, &ue.Username, &ue.Password, &ue.Email, &ue.RealName, &ue.IsActive,
			&ue.LastLogin, &ue.CreatedAt, &ue.UpdatedAt, &ue.RoleName, &ue.ProjectName); err != nil {
			return nil, fmt.Errorf("scan enriched user: %w", err)
		}
		users = append(users, ue)
	}
	return users, nil
}

func (r *Repository) ListRolesWithCount(ctx context.Context) ([]Role, []int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.name, COALESCE(r.role_key,'') AS role_key, r.description,
		       r.permissions, COALESCE(r.readonly,false) AS readonly,
		       COALESCE(r.color,'') AS color, r.created_at, r.updated_at,
		       (SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = r.id) AS user_count
		FROM roles r ORDER BY r.id`)
	if err != nil {
		return nil, nil, fmt.Errorf("list roles with count: %w", err)
	}
	defer rows.Close()

	var roles []Role
	var counts []int
	for rows.Next() {
		var role Role
		var permsJSON []byte
		var count int
		if err := rows.Scan(&role.ID, &role.Name, &role.RoleKey, &role.Description,
			&permsJSON, &role.Readonly, &role.Color, &role.CreatedAt, &role.UpdatedAt, &count); err != nil {
			return nil, nil, fmt.Errorf("scan role with count: %w", err)
		}
		_ = unmarshalPermissions(permsJSON, &role.Permissions)
		roles = append(roles, role)
		counts = append(counts, count)
	}
	return roles, counts, nil
}

func (r *Repository) CreateUser(ctx context.Context, u *User) error {
	// avatar / project / last_login are post-000007+000008 columns; EnsureSchema
	// guarantees they exist. Avatar defaults to the username's first rune so the
	// frontend list never has to render an empty chip.
	avatar := u.Avatar
	if avatar == nil || *avatar == "" {
		fallback := ""
		if len(u.Username) > 0 {
			fallback = strings.ToUpper(string([]rune(u.Username)[0]))
		}
		avatar = &fallback
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, password, email, real_name, is_active, avatar, project)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		u.Username, u.Password, u.Email, u.RealName, u.IsActive, avatar, u.Project,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	u.Avatar = avatar
	return nil
}

func (r *Repository) UpdateUser(ctx context.Context, u *User) error {
	// avatar / project are post-000008 columns; EnsureSchema makes them safe.
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		   SET email=$1, real_name=$2, is_active=$3, password=$4,
		       avatar=$5, project=$6, updated_at=NOW()
		 WHERE id=$7`,
		u.Email, u.RealName, u.IsActive, u.Password, u.Avatar, u.Project, u.ID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *Repository) DeleteUser(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// --- Roles ---

// roleSelectColumns is the canonical projection used by every role read.
// COALESCE keeps pre-000008 rows scannable into Go's non-pointer fields.
const roleSelectColumns = `id, name, COALESCE(role_key,'') AS role_key,
	description, permissions, COALESCE(readonly,false) AS readonly,
	COALESCE(color,'') AS color, created_at, updated_at`

func scanRole(row pgx.Row, role *Role) error {
	var permsJSON []byte
	if err := row.Scan(&role.ID, &role.Name, &role.RoleKey, &role.Description,
		&permsJSON, &role.Readonly, &role.Color, &role.CreatedAt, &role.UpdatedAt); err != nil {
		return err
	}
	return unmarshalPermissions(permsJSON, &role.Permissions)
}

// unmarshalPermissions decodes the JSONB permissions column, accepting
// either the scalar string "*" (post-000008 wildcard) or a JSON array.
// Scalar "*" becomes the single-element list ["*"] so the in-memory Role
// stays uniform; Role.IsWildcard() collapses it back to the wire wildcard.
func unmarshalPermissions(raw []byte, out *[]string) error {
	if len(raw) == 0 {
		*out = nil
		return nil
	}
	// Try array first (the historical shape).
	if err := json.Unmarshal(raw, out); err == nil {
		return nil
	}
	// Fall back to scalar string.
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("unmarshal permissions: %w", err)
	}
	*out = []string{s}
	return nil
}

func (r *Repository) GetRoleByID(ctx context.Context, id int64) (*Role, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+roleSelectColumns+` FROM roles WHERE id = $1`, id)

	var role Role
	if err := scanRole(row, &role); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get role by id: %w", err)
	}
	return &role, nil
}

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+roleSelectColumns+` FROM roles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := scanRole(rows, &role); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	permsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO roles (name, description, permissions)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`,
		role.Name, role.Description, permsJSON,
	).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}
	return nil
}

func (r *Repository) UpdateRole(ctx context.Context, role *Role) error {
	permsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}
	// name / role_key / readonly / color are post-000008 columns; the
	// startup EnsureSchema guarantees they exist by the time this runs.
	_, err = r.pool.Exec(ctx, `
		UPDATE roles
		   SET name        = $1,
		       role_key    = NULLIF($2, ''),
		       description = $3,
		       permissions = $4,
		       readonly    = $5,
		       color       = $6,
		       updated_at  = NOW()
		 WHERE id = $7`,
		role.Name, role.RoleKey, role.Description, permsJSON,
		role.Readonly, role.Color, role.ID)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	return nil
}

func (r *Repository) DeleteRole(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

// --- User Roles ---

func (r *Repository) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.name, COALESCE(r.role_key,'') AS role_key, r.description,
		       r.permissions, COALESCE(r.readonly,false) AS readonly,
		       COALESCE(r.color,'') AS color, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := scanRole(rows, &role); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *Repository) AssignUserRole(ctx context.Context, userID, roleID int64) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, roleID)
	if err != nil {
		return fmt.Errorf("assign user role: %w", err)
	}
	return nil
}

func (r *Repository) RevokeUserRole(ctx context.Context, userID, roleID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`,
		userID, roleID)
	if err != nil {
		return fmt.Errorf("revoke user role: %w", err)
	}
	return nil
}

func (r *Repository) ListUsersByRoleID(ctx context.Context, roleID int64) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.username, u.email, u.real_name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_id = $1
		ORDER BY u.id`, roleID)
	if err != nil {
		return nil, fmt.Errorf("list users by role: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.RealName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *Repository) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear user roles: %w", err)
	}

	for _, roleID := range roleIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, roleID); err != nil {
			return fmt.Errorf("assign role %d: %w", roleID, err)
		}
	}

	return tx.Commit(ctx)
}

// --- Tokens ---

func (r *Repository) SaveToken(ctx context.Context, t *Token) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tokens (user_id, token_type, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		t.UserID, t.TokenType, t.TokenHash, t.ExpiresAt,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

func (r *Repository) RevokeTokenByHash(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE tokens SET revoked = TRUE WHERE token_hash = $1`, hash)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	return nil
}

func (r *Repository) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE tokens SET revoked = TRUE WHERE user_id = $1 AND revoked = FALSE`, userID)
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}

func (r *Repository) IsTokenRevoked(ctx context.Context, hash string) (bool, error) {
	var revoked bool
	err := r.pool.QueryRow(ctx, `
		SELECT revoked FROM tokens WHERE token_hash = $1`, hash).Scan(&revoked)
	if err != nil {
		if err == pgx.ErrNoRows {
			return true, nil
		}
		return false, fmt.Errorf("check token revoked: %w", err)
	}
	return revoked, nil
}

func (r *Repository) CleanExpiredTokens(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tokens WHERE expires_at < $1`, time.Now())
	if err != nil {
		return fmt.Errorf("clean expired tokens: %w", err)
	}
	return nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
