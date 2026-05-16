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
	ID        int64     `json:"id"`
	Version   string    `json:"version"`
	FileName  string    `json:"file_name"`
	FileSize  int64     `json:"file_size"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateUpgradeRequest struct {
	Version  string `json:"version"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}
