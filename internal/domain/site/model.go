package site

import (
	"encoding/json"
	"time"
)

type Site struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Domain      string          `json:"domain"`
	ListenPort  int             `json:"listen_port"`
	SSLEnabled  bool            `json:"ssl_enabled"`
	SSLCert     string          `json:"ssl_cert,omitempty"`
	SSLKey      string          `json:"ssl_key,omitempty"`
	Upstream    json.RawMessage `json:"upstream"`
	Status      string          `json:"status"`
	WAFEnabled  bool            `json:"waf_enabled"`
	Description string          `json:"description,omitempty"`
	// Runtime/binding cache for NW · 03 列表展示。
	// 监控管道（stats job）写回；UI 列表直接读，免 N+1 跨表 join。
	RPS              float64    `json:"rps"`
	BlockedRate      float64    `json:"blocked_rate"`
	InstanceLabel    string     `json:"instance_label"`
	MetricsUpdatedAt *time.Time `json:"metrics_updated_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// UpdateMetricsRequest is the body of PUT /sites/{id}/metrics — pushed by the
// stats pipeline. UI never sends this directly.
type UpdateMetricsRequest struct {
	RPS           float64 `json:"rps"`
	BlockedRate   float64 `json:"blocked_rate"`
	InstanceLabel string  `json:"instance_label,omitempty"`
}

type CreateRequest struct {
	Name        string          `json:"name"`
	Domain      string          `json:"domain"`
	ListenPort  int             `json:"listen_port"`
	SSLEnabled  bool            `json:"ssl_enabled"`
	SSLCert     string          `json:"ssl_cert"`
	SSLKey      string          `json:"ssl_key"`
	Upstream    json.RawMessage `json:"upstream"`
	WAFEnabled  *bool           `json:"waf_enabled"`
	Description string          `json:"description"`
}

type UpdateRequest struct {
	Name        *string          `json:"name"`
	Domain      *string          `json:"domain"`
	ListenPort  *int             `json:"listen_port"`
	SSLEnabled  *bool            `json:"ssl_enabled"`
	SSLCert     *string          `json:"ssl_cert"`
	SSLKey      *string          `json:"ssl_key"`
	Upstream    *json.RawMessage `json:"upstream"`
	Status      *string          `json:"status"`
	WAFEnabled  *bool            `json:"waf_enabled"`
	Description *string          `json:"description"`
}

type ListParams struct {
	Page     int
	PageSize int
	Status   string
	Search   string
}

type ProtectAssoc struct {
	ID       int64 `json:"id"`
	SiteID   int64 `json:"site_id"`
	DeviceID int64 `json:"device_id"`
}
