package operate

import "time"

type OperationLog struct {
	ID           int64     `json:"id"`
	UserID       *int64    `json:"user_id"`
	Username     string    `json:"username"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	DurationMs   int       `json:"duration_ms"`
	ClientIP     *string   `json:"client_ip"`
	RequestBody  *string   `json:"request_body,omitempty"`
	ResponseBody *string   `json:"response_body,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type OperationLogFilter struct {
	Username string `json:"username"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	MinCode  int    `json:"min_code"`
	MaxCode  int    `json:"max_code"`
	StartAt  string `json:"start_at"`
	EndAt    string `json:"end_at"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
