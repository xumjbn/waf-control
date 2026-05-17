package loadbalance

import "time"

type VIP struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Address            string    `json:"address"`
	Protocol           string    `json:"protocol"`
	ProtocolPort       int       `json:"protocol_port"`
	PoolID             *int64    `json:"pool_id"`
	ConnectionLimit    int       `json:"connection_limit"`
	SessionPersistence bool      `json:"session_persistence"`
	AdminStateUp       bool      `json:"admin_state_up"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Pool struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Protocol     string    `json:"protocol"`
	LBMethod     string    `json:"lb_method"`
	AdminStateUp bool      `json:"admin_state_up"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Member struct {
	ID           int64     `json:"id"`
	PoolID       int64     `json:"pool_id"`
	Address      string    `json:"address"`
	ProtocolPort int       `json:"protocol_port"`
	Weight       int       `json:"weight"`
	AdminStateUp bool      `json:"admin_state_up"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type HealthMonitor struct {
	ID            int64     `json:"id"`
	PoolID        int64     `json:"pool_id"`
	Type          string    `json:"type"`
	Delay         int       `json:"delay"`
	Timeout       int       `json:"timeout"`
	MaxRetries    int       `json:"max_retries"`
	HTTPMethod    string    `json:"http_method"`
	URLPath       string    `json:"url_path"`
	ExpectedCodes string    `json:"expected_codes"`
	AdminStateUp  bool      `json:"admin_state_up"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Requests

type CreateVIPRequest struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	Address            string `json:"address"`
	Protocol           string `json:"protocol"`
	ProtocolPort       int    `json:"protocol_port"`
	PoolID             *int64 `json:"pool_id"`
	ConnectionLimit    int    `json:"connection_limit"`
	SessionPersistence bool   `json:"session_persistence"`
}

type UpdateVIPRequest struct {
	Name               *string `json:"name"`
	Description        *string `json:"description"`
	Address            *string `json:"address"`
	Protocol           *string `json:"protocol"`
	ProtocolPort       *int    `json:"protocol_port"`
	PoolID             *int64  `json:"pool_id"`
	ConnectionLimit    *int    `json:"connection_limit"`
	SessionPersistence *bool   `json:"session_persistence"`
	AdminStateUp       *bool   `json:"admin_state_up"`
}

type CreatePoolRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Protocol    string `json:"protocol"`
	LBMethod    string `json:"lb_method"`
}

type UpdatePoolRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Protocol     *string `json:"protocol"`
	LBMethod     *string `json:"lb_method"`
	AdminStateUp *bool   `json:"admin_state_up"`
}

type CreateMemberRequest struct {
	PoolID       int64  `json:"pool_id"`
	Address      string `json:"address"`
	ProtocolPort int    `json:"protocol_port"`
	Weight       int    `json:"weight"`
}

type UpdateMemberRequest struct {
	Address      *string `json:"address"`
	ProtocolPort *int    `json:"protocol_port"`
	Weight       *int    `json:"weight"`
	AdminStateUp *bool   `json:"admin_state_up"`
}

type UpdateHealthMonitorRequest struct {
	Type          *string `json:"type"`
	Delay         *int    `json:"delay"`
	Timeout       *int    `json:"timeout"`
	MaxRetries    *int    `json:"max_retries"`
	HTTPMethod    *string `json:"http_method"`
	URLPath       *string `json:"url_path"`
	ExpectedCodes *string `json:"expected_codes"`
	AdminStateUp  *bool   `json:"admin_state_up"`
}

type CreateHealthMonitorRequest struct {
	PoolID        int64  `json:"pool_id"`
	Type          string `json:"type"`
	Delay         int    `json:"delay"`
	Timeout       int    `json:"timeout"`
	MaxRetries    int    `json:"max_retries"`
	HTTPMethod    string `json:"http_method"`
	URLPath       string `json:"url_path"`
	ExpectedCodes string `json:"expected_codes"`
}
