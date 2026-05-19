package alert

import "time"

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

type Channel struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Target    string    `json:"target"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateChannelRequest struct {
	Name      *string `json:"name"`
	Kind      *string `json:"kind"`
	Target    *string `json:"target"`
	IsEnabled *bool   `json:"is_enabled"`
}
