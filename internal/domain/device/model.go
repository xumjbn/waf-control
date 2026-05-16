package device

import "time"

type Device struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	SerialNo    string    `json:"serial_no,omitempty"`
	Model       string    `json:"model,omitempty"`
	Status      string    `json:"status"`
	IPAddress   string    `json:"ip_address,omitempty"`
	Version     string    `json:"version,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name        string `json:"name"`
	SerialNo    string `json:"serial_no"`
	Model       string `json:"model"`
	IPAddress   string `json:"ip_address"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type UpdateRequest struct {
	Name        *string `json:"name"`
	SerialNo    *string `json:"serial_no"`
	Model       *string `json:"model"`
	Status      *string `json:"status"`
	IPAddress   *string `json:"ip_address"`
	Version     *string `json:"version"`
	Description *string `json:"description"`
}

type ListParams struct {
	Page     int
	PageSize int
	Status   string
	Search   string
}
