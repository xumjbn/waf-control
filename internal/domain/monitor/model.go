package monitor

import "time"

// Attack monitor
type TopSite struct {
	SiteID   int64  `json:"site_id"`
	SiteName string `json:"site_name"`
	Count    int64  `json:"count"`
}

type TopSrcIP struct {
	SrcIP string `json:"src_ip"`
	Count int64  `json:"count"`
}

type SeverityStat struct {
	Severity string `json:"severity"`
	Count    int64  `json:"count"`
}

// System monitor
type Metric struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	NodeID    *int64    `json:"node_id"`
	RecordedAt time.Time `json:"recorded_at"`
}

type MetricSpec struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Unit        string `json:"unit"`
}

type RealtimeQuery struct {
	NodeID int64  `json:"node_id"`
	Metric string `json:"metric"`
}

type HistoryQuery struct {
	NodeID    int64  `json:"node_id"`
	Metric    string `json:"metric"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Interval  string `json:"interval"`
}
