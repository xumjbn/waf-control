package identity

import (
	"net/http"
	"strings"
)

func AuthMiddleware(svc *Service) func(http.Handler) http.Handler {
	return authMiddleware(svc)
}

func authMiddleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			claims, err := svc.ParseAccessToken(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := NewContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// admin 同义词：identity migrations 之后 admin 这条角色的 key 是
			// system_admin / role.Name 可能是 '系统管理员' / 老库可能仍写 'admin'。
			// 任何一条命中都放行。
			adminSynonyms := map[string]struct{}{
				"admin":         {},
				"system_admin":  {},
				"service_admin": {},
				"superadmin":    {},
				"系统管理员":         {},
			}
			for _, required := range roles {
				for _, userRole := range claims.Roles {
					if _, ok := adminSynonyms[userRole]; ok {
						next.ServeHTTP(w, r)
						return
					}
					if userRole == required {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}
