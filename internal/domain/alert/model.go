package alert

import (
	"encoding/json"
	"time"
)

type Policy struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Metric        string    `json:"metric"`
	Operator      string    `json:"operator"`
	Threshold     float64   `json:"threshold"`
	WindowSeconds int       `json:"window_seconds"`
	Level         string    `json:"level"`
	NotifyTargets []string  `json:"notify_targets"`
	IsEnabled     bool      `json:"is_enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreatePolicyRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Metric        string   `json:"metric"`
	Operator      string   `json:"operator"`
	Threshold     float64  `json:"threshold"`
	WindowSeconds int      `json:"window_seconds"`
	Level         string   `json:"level"`
	NotifyTargets []string `json:"notify_targets"`
	IsEnabled     *bool    `json:"is_enabled"`
}

type UpdatePolicyRequest struct {
	Name          *string  `json:"name"`
	Description   *string  `json:"description"`
	Metric        *string  `json:"metric"`
	Operator      *string  `json:"operator"`
	Threshold     *float64 `json:"threshold"`
	WindowSeconds *int     `json:"window_seconds"`
	Level         *string  `json:"level"`
	NotifyTargets *[]string `json:"notify_targets"`
	IsEnabled     *bool    `json:"is_enabled"`
}

type Event struct {
	ID         int64      `json:"id"`
	PolicyID   *int64     `json:"policy_id"`
	Level      string     `json:"level"`
	Kind       string     `json:"kind"`
	Target     string     `json:"target"`
	Message    string     `json:"message"`
	Status     string     `json:"status"`
	OccurredAt time.Time  `json:"occurred_at"`
	HandledAt  *time.Time `json:"handled_at"`
	HandledBy  string     `json:"handled_by"`
}

type EventFilter struct {
	Status string
	Level  string
	Limit  int
}

type CreateEventRequest struct {
	PolicyID *int64 `json:"policy_id"`
	Level    string `json:"level"`
	Kind     string `json:"kind"`
	Target   string `json:"target"`
	Message  string `json:"message"`
}

type UpdateEventStatusRequest struct {
	Status    string `json:"status"`
	HandledBy string `json:"handled_by"`
}

type EventStats struct {
	Open   int64 `json:"open"`
	Ack    int64 `json:"ack"`
	Closed int64 `json:"closed"`
	Today  int64 `json:"today"`
}

// ChannelKind 是 alert_channels.kind 的合法集合。
// 增加新 kind 时同步 ChannelKinds() 和前端 PageAlert 渠道选项。
const (
	ChannelKindEmail     = "email"
	ChannelKindWeChat    = "wechat"
	ChannelKindDingTalk  = "dingtalk"
	ChannelKindPagerDuty = "pagerduty"
	ChannelKindWebhook   = "webhook"
	ChannelKindSMS       = "sms"
)

func ChannelKinds() []string {
	return []string{
		ChannelKindEmail, ChannelKindWeChat, ChannelKindDingTalk,
		ChannelKindPagerDuty, ChannelKindWebhook, ChannelKindSMS,
	}
}

// Channel 是单条告警渠道。Config 用于各 kind 的具体参数：
//   email     -> { "from": "...", "smtp_host": "...", ... }
//   wechat    -> { "corp_id": "...", "agent_id": "...", "secret": "..." }
//   pagerduty -> { "service_key": "..." }
//   webhook   -> { "url": "...", "headers": {...}, "secret": "..." }
//   sms       -> { "provider": "aliyun"|"twilio", "key": "...", "tpl": "..." }
type Channel struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Target      string          `json:"target"`
	Description string          `json:"description"`
	Severity    string          `json:"severity"`
	Config      json.RawMessage `json:"config,omitempty"`
	IsEnabled   bool            `json:"is_enabled"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CreateChannelRequest struct {
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Target      string          `json:"target"`
	Description string          `json:"description"`
	Severity    string          `json:"severity"`
	Config      json.RawMessage `json:"config"`
	IsEnabled   *bool           `json:"is_enabled"`
}

type UpdateChannelRequest struct {
	Name        *string         `json:"name"`
	Kind        *string         `json:"kind"`
	Target      *string         `json:"target"`
	Description *string         `json:"description"`
	Severity    *string         `json:"severity"`
	Config      json.RawMessage `json:"config"`
	IsEnabled   *bool           `json:"is_enabled"`
}
