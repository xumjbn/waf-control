package flow

import "time"

type FlowLog struct {
	ID               int64      `json:"id"`
	SrcIP            string     `json:"src_ip"`
	DstIP            string     `json:"dst_ip"`
	SrcPort          int        `json:"src_port"`
	DstPort          int        `json:"dst_port"`
	Protocol         string     `json:"protocol"`
	BytesSent        int64      `json:"bytes_sent"`
	BytesReceived    int64      `json:"bytes_received"`
	PacketsSent      int64      `json:"packets_sent"`
	PacketsReceived  int64      `json:"packets_received"`
	Duration         *float64    `json:"duration"`
	Application      *string    `json:"application"`
	NodeID           *int64     `json:"node_id"`
	RecordedAt       time.Time  `json:"recorded_at"`
}

type FlowLogFilter struct {
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dst_ip"`
	Protocol string `json:"protocol"`
	StartAt  string `json:"start_at"`
	EndAt    string `json:"end_at"`
	OrderBy  string `json:"order_by"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type FlowStat struct {
	Item  string `json:"item"`
	Value int64  `json:"value"`
}

type SavedQuery struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"created_at"`
}

type FlowMonitorRecord struct {
	ID             int64     `json:"id"`
	NodeID         *int64    `json:"node_id"`
	TotalBytes     int64     `json:"total_bytes"`
	TotalPackets   int64     `json:"total_packets"`
	ConnCount      int       `json:"conn_count"`
	RecordedAt     time.Time `json:"recorded_at"`
}
