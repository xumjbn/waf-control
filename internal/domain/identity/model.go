package identity

import "time"

type User struct {
	ID        int64      `json:"id"`
	Username  string     `json:"username"`
	Password  string     `json:"-"`
	Email     *string    `json:"email,omitempty"`
	RealName  *string    `json:"real_name,omitempty"`
	IsActive  bool       `json:"is_active"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Roles     []Role     `json:"roles,omitempty"`
}

type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
