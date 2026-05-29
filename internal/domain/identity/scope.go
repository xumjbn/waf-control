package identity

// scope.go —— 多租户最小可用：项目（project）级别的查询范围控制。
//
// 设计：
//  1. 用户登录后 JWT 仅含 user_id；项目归属随时可变（用户被加/移出项目），
//     所以不把 project_ids 塞进 JWT，而是在 AuthMiddleware 之后的
//     ScopeMiddleware 里现 DB 查 project_user_roles 装进 context。
//  2. admin 同义角色 → ProjectScope.IsAdmin=true → 不做 project 过滤（看全部）。
//  3. handler / repository 通过 ScopeFromContext 取得 *ProjectScope：
//       - 若 IsAdmin → 不加 WHERE
//       - 否则查询追加 WHERE project_id = ANY($scope.IDs)
//  4. 创建资源时若 ctx scope 非空，project_id 取 scope.IDs[0]（默认归到用户的
//     第一个项目），admin 不指定时落到 1（default 项目）。
//
// 注意：本期仅给 sites / policies / alert_policies / system_upgrades 加了
// project_id 列（migration 000025）。logs / monitor_metrics 等热表暂未铺，
// 触发查询的 handler 应明确说明"未做 project 过滤"。

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectScope 表达"本次请求的 project 范围"。零值（IDs=nil, IsAdmin=false）
// 表示"未配范围"，按"无权限"看 —— 防御性默认。
type ProjectScope struct {
	IsAdmin bool    // 全部项目可见
	IDs     []int64 // IsAdmin=false 时生效；空数组 ⇒ 看不到任何资源
}

type scopeCtxKey struct{}

func NewContextWithScope(ctx context.Context, scope *ProjectScope) context.Context {
	return context.WithValue(ctx, scopeCtxKey{}, scope)
}

// ScopeFromContext 取出当前请求的 scope。无 scope 返回 nil 让调用方决定降级策略。
func ScopeFromContext(ctx context.Context) *ProjectScope {
	v, _ := ctx.Value(scopeCtxKey{}).(*ProjectScope)
	return v
}

// AllowsProject 判断目标 project 是否在 scope 内。nil scope → false（拒绝）。
func (s *ProjectScope) AllowsProject(projectID int64) bool {
	if s == nil {
		return false
	}
	if s.IsAdmin {
		return true
	}
	for _, id := range s.IDs {
		if id == projectID {
			return true
		}
	}
	return false
}

// DefaultProjectID 返回新建资源时该用的 project_id。admin/空 scope 落到 1。
func (s *ProjectScope) DefaultProjectID() int64 {
	if s == nil || s.IsAdmin || len(s.IDs) == 0 {
		return 1
	}
	return s.IDs[0]
}

// ScopeMiddleware 必须挂在 AuthMiddleware 之后（依赖 claims）。
// 查 project_user_roles 把 user 能访问的 project_id 列表装进 ctx。
// admin 同义角色直接 IsAdmin=true。
//
// 失败时（DB unreachable / 用户没分配项目）：保留 nil scope。下游 handler 需
// 区分"不做过滤（admin）" / "ID 集合" / "nil（拒绝/降级）"。
func ScopeMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				next.ServeHTTP(w, r)
				return
			}
			scope := &ProjectScope{}
			if hasAdminRole(claims.Roles) {
				scope.IsAdmin = true
			} else {
				ids, err := loadUserProjectIDs(r.Context(), pool, claims.UserID)
				if err == nil {
					scope.IDs = ids
				}
			}
			ctx := NewContextWithScope(r.Context(), scope)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// 与 requireRole 里的 adminSynonyms 一致，避免角色 key/名称漂移导致漏判。
func hasAdminRole(roles []string) bool {
	syn := map[string]struct{}{
		"admin":         {},
		"system_admin":  {},
		"service_admin": {},
		"superadmin":    {},
		"系统管理员":         {},
	}
	for _, r := range roles {
		if _, ok := syn[r]; ok {
			return true
		}
	}
	return false
}

func loadUserProjectIDs(ctx context.Context, pool *pgxpool.Pool, userID int64) ([]int64, error) {
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT project_id FROM project_user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
