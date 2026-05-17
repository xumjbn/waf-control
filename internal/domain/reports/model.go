package reports

import "time"

type CustomReport struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Metrics     []string   `json:"metrics"`
	Filters     *string    `json:"filters"`
	Schedule    *string    `json:"schedule"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CombinedReport struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	ReportIDs   []int64    `json:"report_ids"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TimingReport struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Metric      string     `json:"metric"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     time.Time  `json:"end_time"`
	Interval    string     `json:"interval"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ManualReport struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Content     *string    `json:"content"`
	Format      string     `json:"format"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ReportData struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}
