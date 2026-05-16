package logs

import "time"

type AttackLog struct {
	ID          int64     `json:"id"`
	NodeID      int64     `json:"node_id"`
	SrcIP       string    `json:"src_ip"`
	DstIP       string    `json:"dst_ip"`
	SrcPort     int       `json:"src_port"`
	DstPort     int       `json:"dst_port"`
	Protocol    string    `json:"protocol"`
	AttackType  string    `json:"attack_type"`
	RuleID      string    `json:"rule_id"`
	Action      string    `json:"action"`
	Payload     string    `json:"payload"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type AntivirusLog struct {
	ID         int64     `json:"id"`
	NodeID     int64     `json:"node_id"`
	FileName   string    `json:"file_name"`
	VirusName  string    `json:"virus_name"`
	FilePath   string    `json:"file_path"`
	Action     string    `json:"action"`
	SrcIP      string    `json:"src_ip"`
	OccurredAt time.Time `json:"occurred_at"`
}

type AntitamperLog struct {
	ID         int64     `json:"id"`
	NodeID     int64     `json:"node_id"`
	FilePath   string    `json:"file_path"`
	ChangeType string    `json:"change_type"`
	Action     string    `json:"action"`
	Detail     string    `json:"detail"`
	OccurredAt time.Time `json:"occurred_at"`
}

type LogQuery struct {
	NodeID    int64
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}
