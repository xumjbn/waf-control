package policy

import (
	"encoding/json"
	"time"
)

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

type Policy struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	CategoryID  *int64    `json:"category_id,omitempty"`
	Severity    string    `json:"severity"`
	Action      string    `json:"action"`
	IsEnabled   bool      `json:"is_enabled"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Rule struct {
	ID        int64     `json:"id"`
	PolicyID  int64     `json:"policy_id"`
	RuleType  string    `json:"rule_type"`
	Field     string    `json:"field"`
	Operator  string    `json:"operator"`
	Value     string    `json:"value"`
	Logic     string    `json:"logic"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type ChangeHistory struct {
	ID        int64           `json:"id"`
	PolicyID  int64           `json:"policy_id"`
	UserID    *int64          `json:"user_id,omitempty"`
	Username  string          `json:"username"`
	Action    string          `json:"action"`
	OldValue  json.RawMessage `json:"old_value,omitempty"`
	NewValue  json.RawMessage `json:"new_value,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Request types

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
}

type CreatePolicyRequest struct {
	Name        string `json:"name"`
	CategoryID  *int64 `json:"category_id"`
	Severity    string `json:"severity"`
	Action      string `json:"action"`
	IsEnabled   *bool  `json:"is_enabled"`
	Description string `json:"description"`
}

type UpdatePolicyRequest struct {
	Name        *string `json:"name"`
	CategoryID  *int64  `json:"category_id"`
	Severity    *string `json:"severity"`
	Action      *string `json:"action"`
	IsEnabled   *bool   `json:"is_enabled"`
	Description *string `json:"description"`
}

type CreateRuleRequest struct {
	PolicyID  int64  `json:"policy_id"`
	RuleType  string `json:"rule_type"`
	Field     string `json:"field"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
	Logic     string `json:"logic"`
	SortOrder int    `json:"sort_order"`
}

type ListPolicyParams struct {
	Page       int
	PageSize   int
	CategoryID *int64
	Severity   string
	Action     string
	IsEnabled  *bool
	Search     string
}
