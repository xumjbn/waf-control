package security

import "time"

type AuthHost struct {
	ID          int64     `json:"id"`
	Host        string    `json:"host"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuthHostConfig struct {
	ID              int64     `json:"id"`
	Enabled         bool      `json:"enabled"`
	MaxAttempts     int       `json:"max_attempts"`
	LockoutDuration int       `json:"lockout_duration"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PasswordStatus struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Expired   bool   `json:"expired"`
	LastReset string `json:"last_reset"`
}

type CreateAuthHostRequest struct {
	Host        string `json:"host"`
	Description string `json:"description"`
}

type UpdateAuthHostRequest struct {
	Host        *string `json:"host"`
	Description *string `json:"description"`
}

type UpdateAuthHostConfigRequest struct {
	Enabled         *bool `json:"enabled"`
	MaxAttempts     *int  `json:"max_attempts"`
	LockoutDuration *int  `json:"lockout_duration"`
}
