package node

import "time"

type Node struct {
	ID        int64     `json:"id"`
	DeviceID  *int64    `json:"device_id,omitempty"`
	Name      string    `json:"name"`
	Hostname  string    `json:"hostname,omitempty"`
	IPAddress string    `json:"ip_address"`
	Status    string    `json:"status"`
	CPUCores  *int      `json:"cpu_cores,omitempty"`
	MemoryMB  *int64    `json:"memory_mb,omitempty"`
	DiskGB    *int64    `json:"disk_gb,omitempty"`
	OSVersion string    `json:"os_version,omitempty"`
	AgentVer  string    `json:"agent_ver,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	DeviceID  *int64 `json:"device_id"`
	Name      string `json:"name"`
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip_address"`
	CPUCores  *int   `json:"cpu_cores"`
	MemoryMB  *int64 `json:"memory_mb"`
	DiskGB    *int64 `json:"disk_gb"`
	OSVersion string `json:"os_version"`
	AgentVer  string `json:"agent_ver"`
}

type UpdateRequest struct {
	DeviceID  **int64 `json:"device_id"`
	Name      *string `json:"name"`
	Hostname  *string `json:"hostname"`
	IPAddress *string `json:"ip_address"`
	Status    *string `json:"status"`
	CPUCores  *int    `json:"cpu_cores"`
	MemoryMB  *int64  `json:"memory_mb"`
	DiskGB    *int64  `json:"disk_gb"`
	OSVersion *string `json:"os_version"`
	AgentVer  *string `json:"agent_ver"`
}

type ListParams struct {
	Page     int
	PageSize int
	DeviceID *int64
	Status   string
	Search   string
}
