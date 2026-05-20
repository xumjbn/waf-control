package system

import "time"

type Setting struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpsertSettingRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type License struct {
	ID           int64      `json:"id"`
	LicenseKey   string     `json:"license_key"`
	ProductName  string     `json:"product_name"`
	Edition      string     `json:"edition"`       // community / enterprise / oem
	Customer     string     `json:"customer"`      // 客户名 / 单位
	ContactEmail string     `json:"contact_email"`
	MaxNodes     int        `json:"max_nodes"`
	IssuedAt     time.Time  `json:"issued_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	GraceUntil   *time.Time `json:"grace_until,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Status 根据时间与启用状态推断"激活 / 试用 / 宽限期 / 已过期 / 未激活"。
func (l License) Status() string {
	if !l.IsActive {
		return "inactive"
	}
	now := time.Now()
	if l.ExpiresAt.After(now) {
		return "active"
	}
	if l.GraceUntil != nil && l.GraceUntil.After(now) {
		return "grace"
	}
	return "expired"
}

type CreateLicenseRequest struct {
	LicenseKey   string `json:"license_key"`
	ProductName  string `json:"product_name"`
	Edition      string `json:"edition"`
	Customer     string `json:"customer"`
	ContactEmail string `json:"contact_email"`
	MaxNodes     int    `json:"max_nodes"`
	ExpiresAt    string `json:"expires_at"`
	GraceUntil   string `json:"grace_until"`
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
