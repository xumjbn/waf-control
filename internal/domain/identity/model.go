package identity

import (
	"encoding/json"
	"time"
)

type User struct {
	ID        int64      `json:"id"`
	Username  string     `json:"username"`
	Password  string     `json:"-"`
	Email     *string    `json:"email,omitempty"`
	RealName  *string    `json:"real_name,omitempty"`
	IsActive  bool       `json:"is_active"`
	// Avatar / Project / LastLogin are surfaced for the NW · 10 user list UI.
	// They are not load-bearing for auth; nil-safe defaults are applied at the
	// handler layer when missing.
	Avatar    *string    `json:"avatar,omitempty"`
	Project   *string    `json:"project,omitempty"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Roles     []Role     `json:"roles,omitempty"`
}

// Role is the internal record. Wire serialisation goes through RoleDTO
// (see dto.go) so legacy callers reading role.Permissions keep working
// while the frontend gets the modules-shaped payload it expects.
//
// New fields added in migration 000008:
//   - RoleKey   stable English identifier, e.g. "system_admin"
//   - Readonly  scope is view-only
//   - Color     accent colour for the RolesGrid card
type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`                  // Chinese display ("系统管理员")
	RoleKey     string    `json:"role_key,omitempty"`    // English key ("system_admin")
	Description string    `json:"description,omitempty"`
	Permissions []string  `json:"permissions"`           // module list; "[*]" or ["*"] == wildcard
	Readonly    bool      `json:"readonly"`
	Color       string    `json:"color,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IsWildcard reports whether the role grants every module. We accept three
// equivalent encodings to be compatible with all migration versions:
//   - permissions stored as the JSON scalar "*"   (post-000008)
//   - permissions stored as the array ["*"]      (post-000007)
//   - permissions stored as ["read"] + Readonly  (000001 legacy, when role.Name == "只读")
func (r Role) IsWildcard() bool {
	if len(r.Permissions) == 1 && r.Permissions[0] == "*" {
		return true
	}
	return r.RoleKey == "system_admin" || r.RoleKey == "readonly"
}

// Modules is the wire-format scope union: "*" (wildcard) or a list of module
// names. Used by RoleDTO and downstream JSON encoding.
type Modules struct {
	Wildcard bool
	List     []string
}

func (m Modules) MarshalJSON() ([]byte, error) {
	if m.Wildcard {
		return []byte(`"*"`), nil
	}
	if m.List == nil {
		return []byte(`[]`), nil
	}
	return json.Marshal(m.List)
}

func (m *Modules) UnmarshalJSON(data []byte) error {
	if len(data) >= 3 && data[0] == '"' && data[len(data)-1] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "*" {
			m.Wildcard = true
			m.List = nil
			return nil
		}
		m.Wildcard = false
		m.List = []string{s}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	m.Wildcard = false
	m.List = list
	return nil
}

// Contains reports whether the module is in scope.
func (m Modules) Contains(module string) bool {
	if m.Wildcard {
		return true
	}
	for _, x := range m.List {
		if x == module {
			return true
		}
	}
	return false
}

type Token struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	TokenType string    `json:"token_type"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Email    string   `json:"email,omitempty"`
	RealName string   `json:"real_name,omitempty"`
	RoleIDs  []int64  `json:"role_ids,omitempty"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	RealName *string `json:"real_name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
	Password *string `json:"password,omitempty"`
	RoleIDs  []int64 `json:"role_ids,omitempty"`
}

type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
}

type UpdateRoleRequest struct {
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
