package acl

import "time"

type Rule struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Direction   string    `json:"direction"`
	Action      string    `json:"action"`
	Protocol    string    `json:"protocol"`
	SrcIP       string    `json:"src_ip"`
	SrcPort     *int      `json:"src_port"`
	DstIP       string    `json:"dst_ip"`
	DstPort     *int      `json:"dst_port"`
	Priority    int       `json:"priority"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateRuleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Direction   string `json:"direction"`
	Action      string `json:"action"`
	Protocol    string `json:"protocol"`
	SrcIP       string `json:"src_ip"`
	SrcPort     *int   `json:"src_port"`
	DstIP       string `json:"dst_ip"`
	DstPort     *int   `json:"dst_port"`
	Priority    int    `json:"priority"`
}

type UpdateRuleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Direction   *string `json:"direction"`
	Action      *string `json:"action"`
	Protocol    *string `json:"protocol"`
	SrcIP       *string `json:"src_ip"`
	SrcPort     *int    `json:"src_port"`
	DstIP       *string `json:"dst_ip"`
	DstPort     *int    `json:"dst_port"`
	Priority    *int    `json:"priority"`
	IsEnabled   *bool   `json:"is_enabled"`
}
