package logs

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// EnsureSchema 在启动时幂等补齐 NW · 05 攻击日志所需的 UI 字段（migration 000011）。
// 这样即使部署机的 migrate 流程不齐，也能自愈。详见
// internal/store/migrations/000011_attack_log_ui_fields.up.sql。
func (r *Repository) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS region      VARCHAR(64)      NOT NULL DEFAULT ''`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS country     VARCHAR(64)      NOT NULL DEFAULT ''`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS lat         DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS lng         DOUBLE PRECISION NOT NULL DEFAULT 0`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS site        VARCHAR(128)     NOT NULL DEFAULT ''`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS domain      VARCHAR(255)     NOT NULL DEFAULT ''`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS type_label  VARCHAR(64)      NOT NULL DEFAULT ''`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS type_color  VARCHAR(16)      NOT NULL DEFAULT '#8e84a3'`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS risk        VARCHAR(8)       NOT NULL DEFAULT '中'`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS method      VARCHAR(8)       NOT NULL DEFAULT 'GET'`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS uri         TEXT             NOT NULL DEFAULT ''`,
		`ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS user_agent  TEXT             NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_attack_logs_site    ON attack_logs(site)`,
		`CREATE INDEX IF NOT EXISTS idx_attack_logs_country ON attack_logs(country)`,
		`CREATE INDEX IF NOT EXISTS idx_attack_logs_risk    ON attack_logs(risk)`,
	}
	for _, s := range stmts {
		if _, err := r.pool.Exec(ctx, s); err != nil {
			return fmt.Errorf("ensure attack_logs schema (%q): %w", s, err)
		}
	}
	return nil
}

const attackSelectCols = `id, node_id, COALESCE(src_ip,''), COALESCE(dst_ip,''), src_port, dst_port,
	COALESCE(protocol,''), COALESCE(attack_type,''), COALESCE(rule_id,''), COALESCE(action,''),
	COALESCE(payload,''), occurred_at,
	COALESCE(region,''), COALESCE(country,''), COALESCE(lat,0), COALESCE(lng,0),
	COALESCE(site,''), COALESCE(domain,''), COALESCE(type_label,''), COALESCE(type_color,'#8e84a3'),
	COALESCE(risk,'中'), COALESCE(method,'GET'), COALESCE(uri,''), COALESCE(user_agent,'')`

func scanAttackLog(rs interface {
	Scan(...interface{}) error
}, l *AttackLog) error {
	return rs.Scan(
		&l.ID, &l.NodeID, &l.SrcIP, &l.DstIP, &l.SrcPort, &l.DstPort,
		&l.Protocol, &l.AttackType, &l.RuleID, &l.Action, &l.Payload, &l.OccurredAt,
		&l.Region, &l.Country, &l.Lat, &l.Lng,
		&l.Site, &l.Domain, &l.TypeLabel, &l.TypeColor,
		&l.Risk, &l.Method, &l.URI, &l.UserAgent,
	)
}

func (r *Repository) ListAttackLogs(ctx context.Context, q LogQuery) ([]AttackLog, int64, error) {
	where, args := buildWhere(q)

	var total int64
	countQuery := "SELECT COUNT(*) FROM attack_logs" + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count attack logs: %w", err)
	}

	offset, limit := paginate(q)
	dataQuery := fmt.Sprintf(`SELECT %s FROM attack_logs%s ORDER BY occurred_at DESC LIMIT %d OFFSET %d`,
		attackSelectCols, where, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list attack logs: %w", err)
	}
	defer rows.Close()

	var logs []AttackLog
	for rows.Next() {
		var l AttackLog
		if err := scanAttackLog(rows, &l); err != nil {
			return nil, 0, fmt.Errorf("scan attack log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (r *Repository) ListAntivirusLogs(ctx context.Context, q LogQuery) ([]AntivirusLog, int64, error) {
	where, args := buildWhere(q)

	var total int64
	countQuery := "SELECT COUNT(*) FROM antivirus_logs" + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count antivirus logs: %w", err)
	}

	offset, limit := paginate(q)
	dataQuery := fmt.Sprintf(`SELECT id, node_id, COALESCE(file_name,''), COALESCE(virus_name,''),
		COALESCE(file_path,''), COALESCE(action,''), COALESCE(src_ip,''), occurred_at
		FROM antivirus_logs%s ORDER BY occurred_at DESC LIMIT %d OFFSET %d`, where, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list antivirus logs: %w", err)
	}
	defer rows.Close()

	var logs []AntivirusLog
	for rows.Next() {
		var l AntivirusLog
		if err := rows.Scan(&l.ID, &l.NodeID, &l.FileName, &l.VirusName, &l.FilePath, &l.Action, &l.SrcIP, &l.OccurredAt); err != nil {
			return nil, 0, fmt.Errorf("scan antivirus log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (r *Repository) ListAntitamperLogs(ctx context.Context, q LogQuery) ([]AntitamperLog, int64, error) {
	where, args := buildWhere(q)

	var total int64
	countQuery := "SELECT COUNT(*) FROM antitamper_logs" + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count antitamper logs: %w", err)
	}

	offset, limit := paginate(q)
	dataQuery := fmt.Sprintf(`SELECT id, node_id, COALESCE(file_path,''), COALESCE(change_type,''),
		COALESCE(action,''), COALESCE(detail,''), occurred_at
		FROM antitamper_logs%s ORDER BY occurred_at DESC LIMIT %d OFFSET %d`, where, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list antitamper logs: %w", err)
	}
	defer rows.Close()

	var logs []AntitamperLog
	for rows.Next() {
		var l AntitamperLog
		if err := rows.Scan(&l.ID, &l.NodeID, &l.FilePath, &l.ChangeType, &l.Action, &l.Detail, &l.OccurredAt); err != nil {
			return nil, 0, fmt.Errorf("scan antitamper log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

// --- Attack log sub-endpoints ---

func (r *Repository) GetAttackLog(ctx context.Context, id int64) (*AttackLog, error) {
	var l AttackLog
	q := fmt.Sprintf(`SELECT %s FROM attack_logs WHERE id = $1`, attackSelectCols)
	if err := scanAttackLog(r.pool.QueryRow(ctx, q, id), &l); err != nil {
		return nil, fmt.Errorf("get attack log: %w", err)
	}
	return &l, nil
}

func (r *Repository) CountAttackLogs(ctx context.Context, q LogQuery) (int64, error) {
	where, args := buildWhere(q)
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM attack_logs"+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count attack logs: %w", err)
	}
	return total, nil
}

func (r *Repository) ClearAttackLogs(ctx context.Context) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM attack_logs")
	if err != nil {
		return fmt.Errorf("clear attack logs: %w", err)
	}
	_ = tag.RowsAffected()
	return nil
}

// IngestAttackLog 由 agent / 内部模块上报一条富攻击日志。
// 不要求 ID 由调用方提供，会返回数据库分配的 ID。
func (r *Repository) IngestAttackLog(ctx context.Context, l *AttackLog) (int64, error) {
	if l.TypeColor == "" {
		l.TypeColor = "#8e84a3"
	}
	if l.Risk == "" {
		l.Risk = "中"
	}
	if l.Method == "" {
		l.Method = "GET"
	}
	var id int64
	err := r.pool.QueryRow(ctx, `INSERT INTO attack_logs (
		node_id, src_ip, dst_ip, src_port, dst_port, protocol, attack_type, rule_id, action, payload, occurred_at,
		region, country, lat, lng, site, domain, type_label, type_color, risk, method, uri, user_agent
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, COALESCE($11, NOW()),
	          $12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
	RETURNING id`,
		l.NodeID, l.SrcIP, l.DstIP, l.SrcPort, l.DstPort, l.Protocol, l.AttackType, l.RuleID, l.Action, l.Payload, l.OccurredAt,
		l.Region, l.Country, l.Lat, l.Lng, l.Site, l.Domain, l.TypeLabel, l.TypeColor, l.Risk, l.Method, l.URI, l.UserAgent,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("ingest attack log: %w", err)
	}
	l.ID = id
	return id, nil
}

// --- Antivirus log sub-endpoints ---

func (r *Repository) GetAntivirusLog(ctx context.Context, id int64) (*AntivirusLog, error) {
	var l AntivirusLog
	err := r.pool.QueryRow(ctx, `SELECT id, node_id, COALESCE(file_name,''), COALESCE(virus_name,''),
		COALESCE(file_path,''), COALESCE(action,''), COALESCE(src_ip,''), occurred_at
		FROM antivirus_logs WHERE id = $1`, id).Scan(
		&l.ID, &l.NodeID, &l.FileName, &l.VirusName, &l.FilePath, &l.Action, &l.SrcIP, &l.OccurredAt)
	if err != nil {
		return nil, fmt.Errorf("get antivirus log: %w", err)
	}
	return &l, nil
}

func (r *Repository) CountAntivirusLogs(ctx context.Context, q LogQuery) (int64, error) {
	where, args := buildWhere(q)
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM antivirus_logs"+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count antivirus logs: %w", err)
	}
	return total, nil
}

func (r *Repository) ClearAntivirusLogs(ctx context.Context) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM antivirus_logs")
	if err != nil {
		return fmt.Errorf("clear antivirus logs: %w", err)
	}
	_ = tag.RowsAffected()
	return nil
}

// --- Antitamper log sub-endpoints ---

func (r *Repository) GetAntitamperLog(ctx context.Context, id int64) (*AntitamperLog, error) {
	var l AntitamperLog
	err := r.pool.QueryRow(ctx, `SELECT id, node_id, COALESCE(file_path,''), COALESCE(change_type,''),
		COALESCE(action,''), COALESCE(detail,''), occurred_at
		FROM antitamper_logs WHERE id = $1`, id).Scan(
		&l.ID, &l.NodeID, &l.FilePath, &l.ChangeType, &l.Action, &l.Detail, &l.OccurredAt)
	if err != nil {
		return nil, fmt.Errorf("get antitamper log: %w", err)
	}
	return &l, nil
}

func (r *Repository) CountAntitamperLogs(ctx context.Context, q LogQuery) (int64, error) {
	where, args := buildWhere(q)
	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM antitamper_logs"+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count antitamper logs: %w", err)
	}
	return total, nil
}

func (r *Repository) ClearAntitamperLogs(ctx context.Context) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM antitamper_logs")
	if err != nil {
		return fmt.Errorf("clear antitamper logs: %w", err)
	}
	_ = tag.RowsAffected()
	return nil
}

func buildWhere(q LogQuery) (string, []interface{}) {
	var conds []string
	var args []interface{}
	argIdx := 1

	if q.NodeID > 0 {
		conds = append(conds, fmt.Sprintf("node_id = $%d", argIdx))
		args = append(args, q.NodeID)
		argIdx++
	}
	if q.StartTime != "" {
		conds = append(conds, fmt.Sprintf("occurred_at >= $%d", argIdx))
		args = append(args, q.StartTime)
		argIdx++
	}
	if q.EndTime != "" {
		conds = append(conds, fmt.Sprintf("occurred_at <= $%d", argIdx))
		args = append(args, q.EndTime)
		argIdx++
	}
	if q.Risk != "" {
		conds = append(conds, fmt.Sprintf("risk = $%d", argIdx))
		args = append(args, q.Risk)
		argIdx++
	}
	if q.Site != "" {
		conds = append(conds, fmt.Sprintf("site = $%d", argIdx))
		args = append(args, q.Site)
		argIdx++
	}
	if q.Country != "" {
		conds = append(conds, fmt.Sprintf("country = $%d", argIdx))
		args = append(args, q.Country)
		argIdx++
	}
	if q.SrcIP != "" {
		conds = append(conds, fmt.Sprintf("src_ip = $%d", argIdx))
		args = append(args, q.SrcIP)
		argIdx++
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func paginate(q LogQuery) (int, int) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size <= 0 || size > 100 {
		size = 20
	}
	return (page - 1) * size, size
}
