package logs

import "time"

type AttackLog struct {
	ID         int64     `json:"id"`
	NodeID     int64     `json:"node_id"`
	SrcIP      string    `json:"src_ip"`
	DstIP      string    `json:"dst_ip"`
	SrcPort    int       `json:"src_port"`
	DstPort    int       `json:"dst_port"`
	Protocol   string    `json:"protocol"`
	AttackType string    `json:"attack_type"`
	RuleID     string    `json:"rule_id"`
	Action     string    `json:"action"`
	Payload    string    `json:"payload"`
	OccurredAt time.Time `json:"occurred_at"`
	// UI display fields（migration 000011 起）—— 由 agent 上报或 stats 富化。
	Region    string  `json:"region"`
	Country   string  `json:"country"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Site      string  `json:"site"`
	Domain    string  `json:"domain"`
	TypeLabel string  `json:"type_label"`
	TypeColor string  `json:"type_color"`
	Risk      string  `json:"risk"` // 高 / 中 / 低
	Method    string  `json:"method"`
	URI       string  `json:"uri"`
	UserAgent string  `json:"user_agent"`
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
	// 攻击日志扩展过滤（与前端 PageLogAttack 顶部筛选一致）
	Risk    string // 高/中/低
	Site    string
	Country string
	SrcIP   string
}

// IngestAttackLogRequest 是 agent / 内部模块上报一条富攻击日志的 wire 结构。
// 时间允许为零值（数据库 NOW()）。
type IngestAttackLogRequest struct {
	AttackLog
}
