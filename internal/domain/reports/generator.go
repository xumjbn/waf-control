package reports

// generator.go —— 真实报表生成：把 attack_logs 在时间窗口内聚合成 CSV 落库，
// 供下载端点直接服务。替代之前 DownloadReport 返回 JSON snapshot 的占位实现。
//
// 调用路径：
//   - cron 调度器 OnReportDue → GenerateTiming（每日滚动近 24h）
//   - 手动 RunReport → Generate（按 kind 即时生成一份产物）
//   - DownloadReport → 取 report_outputs 最近一行；无则即时生成
//
// CSV 安全：cells 做 RFC 4180 转义 + Excel formula-injection 防御（= + - @ 前补 '）。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Generator struct {
	pool *pgxpool.Pool
}

func NewGenerator(pool *pgxpool.Pool) *Generator {
	return &Generator{pool: pool}
}

// EnsureSchema 幂等补建 report_outputs（与 migration 000026 等价）。
func (g *Generator) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS report_outputs (
			id BIGSERIAL PRIMARY KEY,
			report_kind VARCHAR(16) NOT NULL,
			report_id BIGINT NOT NULL,
			filename VARCHAR(256) NOT NULL,
			content TEXT NOT NULL,
			row_count INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_report_outputs_lookup
			ON report_outputs(report_kind, report_id, created_at DESC)`,
	}
	for _, s := range stmts {
		if _, err := g.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure report_outputs (%q): %w", s, err)
		}
	}
	return nil
}

// ReportOutput 是一份已生成的报表产物。
type ReportOutput struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"report_kind"`
	ReportID  int64     `json:"report_id"`
	Filename  string    `json:"filename"`
	Content   string    `json:"-"`
	RowCount  int       `json:"row_count"`
	CreatedAt time.Time `json:"created_at"`
}

// GenerateTiming 给某定时报表生成一份近 24h 的攻击日志 CSV 产物（cron 调度调用）。
func (g *Generator) GenerateTiming(ctx context.Context, reportID int64) error {
	_, err := g.Generate(ctx, "timing", reportID)
	return err
}

// Generate 按 kind/id 即时生成一份产物并落库，返回产物元信息。
// 当前各 kind 统一产出"近 24h 攻击日志明细" CSV —— 真实数据，
// 不同 kind 的差异化模板可后续按 filters 扩展。
func (g *Generator) Generate(ctx context.Context, kind string, reportID int64) (*ReportOutput, error) {
	end := time.Now()
	start := end.Add(-24 * time.Hour)
	csv, rowCount, err := g.buildAttackCSV(ctx, start, end)
	if err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("report-%s-%d-%s.csv", kind, reportID, end.Format("20060102-1504"))

	out := &ReportOutput{Kind: kind, ReportID: reportID, Filename: filename, RowCount: rowCount}
	if err := g.pool.QueryRow(ctx, `
		INSERT INTO report_outputs (report_kind, report_id, filename, content, row_count)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id, created_at`,
		kind, reportID, filename, csv, rowCount).Scan(&out.ID, &out.CreatedAt); err != nil {
		return nil, fmt.Errorf("store report output: %w", err)
	}
	return out, nil
}

// LatestOutput 取某报表最近一次产物。无则返回 nil, nil。
func (g *Generator) LatestOutput(ctx context.Context, kind string, reportID int64) (*ReportOutput, error) {
	var o ReportOutput
	err := g.pool.QueryRow(ctx, `
		SELECT id, report_kind, report_id, filename, content, row_count, created_at
		  FROM report_outputs
		 WHERE report_kind = $1 AND report_id = $2
		 ORDER BY created_at DESC LIMIT 1`, kind, reportID).
		Scan(&o.ID, &o.Kind, &o.ReportID, &o.Filename, &o.Content, &o.RowCount, &o.CreatedAt)
	if err != nil {
		return nil, nil // 无产物视为 nil，由调用方决定是否即时生成
	}
	return &o, nil
}

// buildAttackCSV 把 [start,end] 内的攻击日志聚合成 CSV 文本，返回内容 + 行数。
func (g *Generator) buildAttackCSV(ctx context.Context, start, end time.Time) (string, int, error) {
	rows, err := g.pool.Query(ctx, `
		SELECT occurred_at, src_ip, COALESCE(country,''), COALESCE(site,''),
		       COALESCE(attack_type,''), COALESCE(method,''), COALESCE(uri,''),
		       COALESCE(rule_id,''), COALESCE(action,''), COALESCE(risk,'')
		  FROM attack_logs
		 WHERE occurred_at >= $1 AND occurred_at <= $2
		 ORDER BY occurred_at DESC
		 LIMIT 10000`, start, end)
	if err != nil {
		return "", 0, fmt.Errorf("query attack logs: %w", err)
	}
	defer rows.Close()

	var b strings.Builder
	// UTF-8 BOM（EF BB BF），让 Excel 正确识别中文。用字节写避免源码内嵌 BOM。
	b.WriteByte(0xEF)
	b.WriteByte(0xBB)
	b.WriteByte(0xBF)
	header := []string{"时间", "来源 IP", "国家", "站点", "攻击类型", "方法", "URI", "规则 ID", "处置", "风险"}
	b.WriteString(csvRowLine(header))
	b.WriteByte('\n')

	count := 0
	for rows.Next() {
		var occurredAt time.Time
		var srcIP, country, site, atype, method, uri, ruleID, action, risk string
		if err := rows.Scan(&occurredAt, &srcIP, &country, &site, &atype,
			&method, &uri, &ruleID, &action, &risk); err != nil {
			continue
		}
		line := csvRowLine([]string{
			occurredAt.Format("2006-01-02 15:04:05"),
			srcIP, country, site, atype, method, uri, ruleID, action, risk,
		})
		b.WriteString(line)
		b.WriteByte('\n')
		count++
	}
	return b.String(), count, nil
}

// --- CSV 安全工具（与前端 utils/csv.ts 等价）---

func csvRowLine(cells []string) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = csvCell(c)
	}
	return strings.Join(parts, ",")
}

// csvCell 做 RFC 4180 转义 + Excel formula-injection 防御。
func csvCell(s string) string {
	if len(s) > 0 {
		switch s[0] {
		case '=', '+', '-', '@', '\t', '\r':
			s = "'" + s
		}
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
