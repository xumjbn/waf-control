package flow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const defaultPageSize = 20

func (r *Repository) ListFlowLogs(ctx context.Context, filter FlowLogFilter) ([]FlowLog, int, error) {
	var conds []string
	var args []interface{}
	argIdx := 1

	if filter.SrcIP != "" {
		conds = append(conds, fmt.Sprintf("src_ip = $%d", argIdx))
		args = append(args, filter.SrcIP)
		argIdx++
	}
	if filter.DstIP != "" {
		conds = append(conds, fmt.Sprintf("dst_ip = $%d", argIdx))
		args = append(args, filter.DstIP)
		argIdx++
	}
	if filter.Protocol != "" {
		conds = append(conds, fmt.Sprintf("protocol = $%d", argIdx))
		args = append(args, filter.Protocol)
		argIdx++
	}
	if filter.StartAt != "" {
		t, err := time.Parse(time.RFC3339, filter.StartAt)
		if err == nil {
			conds = append(conds, fmt.Sprintf("recorded_at >= $%d", argIdx))
			args = append(args, t)
			argIdx++
		}
	}
	if filter.EndAt != "" {
		t, err := time.Parse(time.RFC3339, filter.EndAt)
		if err == nil {
			conds = append(conds, fmt.Sprintf("recorded_at <= $%d", argIdx))
			args = append(args, t)
			argIdx++
		}
	}

	whereClause := ""
	if len(conds) > 0 {
		whereClause = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM flow_logs" + whereClause
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count flow logs: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	offset := (page - 1) * pageSize

	orderClause := "ORDER BY recorded_at DESC"
	if filter.OrderBy == "bytes" {
		orderClause = "ORDER BY (bytes_sent + bytes_received) DESC"
	}

	query := fmt.Sprintf(`SELECT id, src_ip, dst_ip, src_port, dst_port, protocol,
		bytes_sent, bytes_received, packets_sent, packets_received,
		duration, application, node_id, recorded_at
		FROM flow_logs%s %s LIMIT $%d OFFSET $%d`,
		whereClause, orderClause, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list flow logs: %w", err)
	}
	defer rows.Close()

	var items []FlowLog
	for rows.Next() {
		var fl FlowLog
		if err := rows.Scan(&fl.ID, &fl.SrcIP, &fl.DstIP, &fl.SrcPort, &fl.DstPort,
			&fl.Protocol, &fl.BytesSent, &fl.BytesReceived, &fl.PacketsSent, &fl.PacketsReceived,
			&fl.Duration, &fl.Application, &fl.NodeID, &fl.RecordedAt); err != nil {
			return nil, 0, fmt.Errorf("scan flow log: %w", err)
		}
		items = append(items, fl)
	}
	return items, total, nil
}

func (r *Repository) GetFlowLog(ctx context.Context, id int64) (*FlowLog, error) {
	var fl FlowLog
	err := r.pool.QueryRow(ctx, `SELECT id, src_ip, dst_ip, src_port, dst_port, protocol,
		bytes_sent, bytes_received, packets_sent, packets_received,
		duration, application, node_id, recorded_at
		FROM flow_logs WHERE id = $1`, id).Scan(
		&fl.ID, &fl.SrcIP, &fl.DstIP, &fl.SrcPort, &fl.DstPort,
		&fl.Protocol, &fl.BytesSent, &fl.BytesReceived, &fl.PacketsSent, &fl.PacketsReceived,
		&fl.Duration, &fl.Application, &fl.NodeID, &fl.RecordedAt)
	if err != nil {
		return nil, fmt.Errorf("get flow log %d: %w", id, err)
	}
	return &fl, nil
}

// --- Flow Statistics ---

func (r *Repository) TopSrcIPs(ctx context.Context) ([]FlowStat, error) {
	rows, err := r.pool.Query(ctx, `SELECT src_ip, COUNT(*) as cnt
		FROM flow_logs GROUP BY src_ip ORDER BY cnt DESC LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("top flow src IPs: %w", err)
	}
	defer rows.Close()

	var items []FlowStat
	for rows.Next() {
		var fs FlowStat
		if err := rows.Scan(&fs.Item, &fs.Value); err != nil {
			return nil, fmt.Errorf("scan flow stat: %w", err)
		}
		items = append(items, fs)
	}
	return items, nil
}

func (r *Repository) TopDstIPs(ctx context.Context) ([]FlowStat, error) {
	rows, err := r.pool.Query(ctx, `SELECT dst_ip, COUNT(*) as cnt
		FROM flow_logs GROUP BY dst_ip ORDER BY cnt DESC LIMIT 10`)
	if err != nil {
		return nil, fmt.Errorf("top flow dst IPs: %w", err)
	}
	defer rows.Close()

	var items []FlowStat
	for rows.Next() {
		var fs FlowStat
		if err := rows.Scan(&fs.Item, &fs.Value); err != nil {
			return nil, fmt.Errorf("scan flow stat: %w", err)
		}
		items = append(items, fs)
	}
	return items, nil
}

func (r *Repository) ProtocolDistribution(ctx context.Context) ([]FlowStat, error) {
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(protocol,'unknown'), COUNT(*) as cnt
		FROM flow_logs GROUP BY protocol ORDER BY cnt DESC`)
	if err != nil {
		return nil, fmt.Errorf("protocol distribution: %w", err)
	}
	defer rows.Close()

	var items []FlowStat
	for rows.Next() {
		var fs FlowStat
		if err := rows.Scan(&fs.Item, &fs.Value); err != nil {
			return nil, fmt.Errorf("scan protocol stat: %w", err)
		}
		items = append(items, fs)
	}
	return items, nil
}

// --- Saved Queries ---

func (r *Repository) ListSavedQueries(ctx context.Context) ([]SavedQuery, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, query, created_at
		FROM flow_saved_queries ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list saved queries: %w", err)
	}
	defer rows.Close()

	var items []SavedQuery
	for rows.Next() {
		var sq SavedQuery
		if err := rows.Scan(&sq.ID, &sq.Name, &sq.Query, &sq.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan saved query: %w", err)
		}
		items = append(items, sq)
	}
	return items, nil
}

func (r *Repository) CreateSavedQuery(ctx context.Context, sq SavedQuery) (*SavedQuery, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO flow_saved_queries (name, query)
		VALUES ($1, $2) RETURNING id, created_at`,
		sq.Name, sq.Query).Scan(&sq.ID, &sq.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create saved query: %w", err)
	}
	return &sq, nil
}

func (r *Repository) DeleteSavedQuery(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM flow_saved_queries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete saved query %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("saved query %d not found", id)
	}
	return nil
}

// --- Flow Monitor ---

func (r *Repository) MonitorRecords(ctx context.Context) ([]FlowMonitorRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, node_id, total_bytes, total_packets, conn_count, recorded_at
		FROM flow_monitor_records ORDER BY recorded_at DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("list monitor records: %w", err)
	}
	defer rows.Close()

	var items []FlowMonitorRecord
	for rows.Next() {
		var fm FlowMonitorRecord
		if err := rows.Scan(&fm.ID, &fm.NodeID, &fm.TotalBytes, &fm.TotalPackets, &fm.ConnCount, &fm.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan monitor record: %w", err)
		}
		items = append(items, fm)
	}
	return items, nil
}
