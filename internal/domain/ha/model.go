package ha

import "time"

type Config struct {
	ID            int64     `json:"id"`
	Mode          string    `json:"mode"`
	VirtualIP     string    `json:"virtual_ip"`
	Priority      int       `json:"priority"`
	Interface     string    `json:"interface"`
	PeerAddress   string    `json:"peer_address"`
	IsEnabled     bool      `json:"is_enabled"`
	HeartbeatSec  int       `json:"heartbeat_sec"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UpsertConfigRequest struct {
	Mode         string `json:"mode"`
	VirtualIP    string `json:"virtual_ip"`
	Priority     int    `json:"priority"`
	Interface    string `json:"interface"`
	PeerAddress  string `json:"peer_address"`
	IsEnabled    *bool  `json:"is_enabled"`
	HeartbeatSec int    `json:"heartbeat_sec"`
}
