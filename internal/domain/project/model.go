package project

import "time"

type Project struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	DomainID    string    `json:"domain_id"`
	ParentID    *int64    `json:"parent_id,omitempty"`
	IsDomain    bool      `json:"is_domain"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Assignment struct {
	ProjectID int64
	UserID    int64
	RoleID    int64
}
