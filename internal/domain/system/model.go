package system

import "time"

type Setting struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpsertSettingRequest struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Category string `json:"category"`
}

type License struct {
	ID          int64     `json:"id"`
	LicenseKey  string    `json:"license_key"`
	ProductName string    `json:"product_name"`
	MaxNodes    int       `json:"max_nodes"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateLicenseRequest struct {
	LicenseKey  string `json:"license_key"`
	ProductName string `json:"product_name"`
	MaxNodes    int    `json:"max_nodes"`
	ExpiresAt   string `json:"expires_at"`
}

type Upgrade struct {
	ID              int64      `json:"id"`
	Version         string     `json:"version"`
	Type            string     `json:"type"`            // patch / minor / major / security
	Channel         string     `json:"channel"`         // stable / beta / dev
	FileName        string     `json:"file_name"`
	FileSize        int64      `json:"file_size"`
	Checksum        string     `json:"checksum"`
	DownloadURL     string     `json:"download_url"`
	Notes           string     `json:"notes"`
	ChangesSummary  string     `json:"changes_summary"`
	Status          string     `json:"status"` // pending / in_progress / completed / failed
	IsCurrent       bool       `json:"is_current"`
	IsLatest        bool       `json:"is_latest"`
	ReleasedAt      *time.Time `json:"released_at,omitempty"`
	AppliedAt       *time.Time `json:"applied_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CreateUpgradeRequest struct {
	Version         string `json:"version"`
	Type            string `json:"type"`
	Channel         string `json:"channel"`
	FileName        string `json:"file_name"`
	FileSize        int64  `json:"file_size"`
	Checksum        string `json:"checksum"`
	DownloadURL     string `json:"download_url"`
	Notes           string `json:"notes"`
	ChangesSummary  string `json:"changes_summary"`
	ReleasedAt      string `json:"released_at"` // RFC3339
	IsLatest        bool   `json:"is_latest"`
}
