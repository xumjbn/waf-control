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

func (r *Repository) ListAttackLogs(ctx context.Context, q LogQuery) ([]AttackLog, int64, error) {
	where, args := buildWhere(q)

	var total int64
	countQuery := "SELECT COUNT(*) FROM attack_logs" + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count attack logs: %w", err)
	}

	offset, limit := paginate(q)
	dataQuery := fmt.Sprintf(`SELECT id, node_id, COALESCE(src_ip,''), COALESCE(dst_ip,''), src_port, dst_port,
		COALESCE(protocol,''), COALESCE(attack_type,''), COALESCE(rule_id,''), COALESCE(action,''),
		COALESCE(payload,''), occurred_at
		FROM attack_logs%s ORDER BY occurred_at DESC LIMIT %d OFFSET %d`, where, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list attack logs: %w", err)
	}
	defer rows.Close()

	var logs []AttackLog
	for rows.Next() {
		var l AttackLog
		if err := rows.Scan(&l.ID, &l.NodeID, &l.SrcIP, &l.DstIP, &l.SrcPort, &l.DstPort,
			&l.Protocol, &l.AttackType, &l.RuleID, &l.Action, &l.Payload, &l.OccurredAt); err != nil {
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
	err := r.pool.QueryRow(ctx, `SELECT id, node_id, COALESCE(src_ip,''), COALESCE(dst_ip,''), src_port, dst_port,
		COALESCE(protocol,''), COALESCE(attack_type,''), COALESCE(rule_id,''), COALESCE(action,''),
		COALESCE(payload,''), occurred_at FROM attack_logs WHERE id = $1`, id).Scan(
		&l.ID, &l.NodeID, &l.SrcIP, &l.DstIP, &l.SrcPort, &l.DstPort,
		&l.Protocol, &l.AttackType, &l.RuleID, &l.Action, &l.Payload, &l.OccurredAt)
	if err != nil {
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
