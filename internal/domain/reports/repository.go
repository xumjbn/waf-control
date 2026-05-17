package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Custom Reports ---

func (r *Repository) ListCustom(ctx context.Context) ([]CustomReport, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, filters, schedule, created_at, updated_at
		FROM report_custom ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list custom reports: %w", err)
	}
	defer rows.Close()

	var items []CustomReport
	for rows.Next() {
		var cr CustomReport
		if err := rows.Scan(&cr.ID, &cr.Name, &cr.Description, &cr.Filters, &cr.Schedule, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan custom report: %w", err)
		}
		items = append(items, cr)
	}
	return items, nil
}

func (r *Repository) GetCustom(ctx context.Context, id int64) (*CustomReport, error) {
	var cr CustomReport
	err := r.pool.QueryRow(ctx, `SELECT id, name, description, filters, schedule, created_at, updated_at
		FROM report_custom WHERE id = $1`, id).Scan(
		&cr.ID, &cr.Name, &cr.Description, &cr.Filters, &cr.Schedule, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get custom report %d: %w", id, err)
	}
	return &cr, nil
}

func (r *Repository) CreateCustom(ctx context.Context, cr CustomReport) (*CustomReport, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO report_custom (name, description, filters, schedule)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		cr.Name, cr.Description, cr.Filters, cr.Schedule).
		Scan(&cr.ID, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create custom report: %w", err)
	}
	return &cr, nil
}

func (r *Repository) UpdateCustom(ctx context.Context, cr CustomReport) (*CustomReport, error) {
	err := r.pool.QueryRow(ctx, `UPDATE report_custom
		SET name = $1, description = $2, filters = $3, schedule = $4, updated_at = NOW()
		WHERE id = $5 RETURNING id, name, description, filters, schedule, created_at, updated_at`,
		cr.Name, cr.Description, cr.Filters, cr.Schedule, cr.ID).
		Scan(&cr.ID, &cr.Name, &cr.Description, &cr.Filters, &cr.Schedule, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update custom report %d: %w", cr.ID, err)
	}
	return &cr, nil
}

func (r *Repository) DeleteCustom(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM report_custom WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete custom report %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("custom report %d not found", id)
	}
	return nil
}

func (r *Repository) CustomReportData(ctx context.Context, id int64) (*ReportData, error) {
	_, err := r.GetCustom(ctx, id)
	if err != nil {
		return nil, err
	}

	return &ReportData{
		Columns: []string{"timestamp", "metric", "value"},
		Rows:    [][]interface{}{},
	}, nil
}

// --- Combined Reports ---

func (r *Repository) ListCombined(ctx context.Context) ([]CombinedReport, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, created_at, updated_at
		FROM report_combined ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list combined reports: %w", err)
	}
	defer rows.Close()

	var items []CombinedReport
	for rows.Next() {
		var cr CombinedReport
		if err := rows.Scan(&cr.ID, &cr.Name, &cr.Description, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan combined report: %w", err)
		}
		items = append(items, cr)
	}
	return items, nil
}

func (r *Repository) GetCombined(ctx context.Context, id int64) (*CombinedReport, error) {
	var cr CombinedReport
	err := r.pool.QueryRow(ctx, `SELECT id, name, description, created_at, updated_at
		FROM report_combined WHERE id = $1`, id).Scan(
		&cr.ID, &cr.Name, &cr.Description, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get combined report %d: %w", id, err)
	}
	return &cr, nil
}

func (r *Repository) CreateCombined(ctx context.Context, cr CombinedReport) (*CombinedReport, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO report_combined (name, description)
		VALUES ($1, $2) RETURNING id, created_at, updated_at`,
		cr.Name, cr.Description).Scan(&cr.ID, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create combined report: %w", err)
	}
	return &cr, nil
}

func (r *Repository) UpdateCombined(ctx context.Context, cr CombinedReport) (*CombinedReport, error) {
	err := r.pool.QueryRow(ctx, `UPDATE report_combined
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3 RETURNING id, name, description, created_at, updated_at`,
		cr.Name, cr.Description, cr.ID).
		Scan(&cr.ID, &cr.Name, &cr.Description, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update combined report %d: %w", cr.ID, err)
	}
	return &cr, nil
}

func (r *Repository) DeleteCombined(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM report_combined WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete combined report %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("combined report %d not found", id)
	}
	return nil
}

func (r *Repository) CombinedReportData(ctx context.Context, id int64) (*ReportData, error) {
	_, err := r.GetCombined(ctx, id)
	if err != nil {
		return nil, err
	}

	return &ReportData{
		Columns: []string{"report_id", "timestamp", "metric", "value"},
		Rows:    [][]interface{}{},
	}, nil
}

// --- Timing Reports ---

func (r *Repository) ListTiming(ctx context.Context) ([]TimingReport, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, metric, start_time, end_time, interval, created_at, updated_at
		FROM report_timing ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list timing reports: %w", err)
	}
	defer rows.Close()

	var items []TimingReport
	for rows.Next() {
		var tr TimingReport
		if err := rows.Scan(&tr.ID, &tr.Name, &tr.Description, &tr.Metric,
			&tr.StartTime, &tr.EndTime, &tr.Interval, &tr.CreatedAt, &tr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan timing report: %w", err)
		}
		items = append(items, tr)
	}
	return items, nil
}

func (r *Repository) GetTiming(ctx context.Context, id int64) (*TimingReport, error) {
	var tr TimingReport
	err := r.pool.QueryRow(ctx, `SELECT id, name, description, metric, start_time, end_time, interval, created_at, updated_at
		FROM report_timing WHERE id = $1`, id).Scan(
		&tr.ID, &tr.Name, &tr.Description, &tr.Metric,
		&tr.StartTime, &tr.EndTime, &tr.Interval, &tr.CreatedAt, &tr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get timing report %d: %w", id, err)
	}
	return &tr, nil
}

func (r *Repository) CreateTiming(ctx context.Context, tr TimingReport) (*TimingReport, error) {
	if tr.StartTime.IsZero() {
		tr.StartTime = time.Now().Add(-24 * time.Hour)
	}
	if tr.EndTime.IsZero() {
		tr.EndTime = time.Now()
	}

	err := r.pool.QueryRow(ctx, `INSERT INTO report_timing (name, description, metric, start_time, end_time, interval)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`,
		tr.Name, tr.Description, tr.Metric, tr.StartTime, tr.EndTime, tr.Interval).
		Scan(&tr.ID, &tr.CreatedAt, &tr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create timing report: %w", err)
	}
	return &tr, nil
}

func (r *Repository) UpdateTiming(ctx context.Context, tr TimingReport) (*TimingReport, error) {
	err := r.pool.QueryRow(ctx, `UPDATE report_timing
		SET name = $1, description = $2, metric = $3, start_time = $4, end_time = $5, interval = $6, updated_at = NOW()
		WHERE id = $7 RETURNING id, name, description, metric, start_time, end_time, interval, created_at, updated_at`,
		tr.Name, tr.Description, tr.Metric, tr.StartTime, tr.EndTime, tr.Interval, tr.ID).
		Scan(&tr.ID, &tr.Name, &tr.Description, &tr.Metric,
			&tr.StartTime, &tr.EndTime, &tr.Interval, &tr.CreatedAt, &tr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update timing report %d: %w", tr.ID, err)
	}
	return &tr, nil
}

func (r *Repository) DeleteTiming(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM report_timing WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete timing report %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("timing report %d not found", id)
	}
	return nil
}

func (r *Repository) TimingReportData(ctx context.Context, id int64) (*ReportData, error) {
	tr, err := r.GetTiming(ctx, id)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `SELECT recorded_at, value
		FROM monitor_metrics
		WHERE name = $1 AND recorded_at >= $2 AND recorded_at <= $3
		ORDER BY recorded_at ASC`, tr.Metric, tr.StartTime, tr.EndTime)
	if err != nil {
		return nil, fmt.Errorf("query timing data: %w", err)
	}
	defer rows.Close()

	data := &ReportData{
		Columns: []string{"timestamp", tr.Metric},
	}
	for rows.Next() {
		var t time.Time
		var v float64
		if err := rows.Scan(&t, &v); err != nil {
			return nil, fmt.Errorf("scan timing data: %w", err)
		}
		data.Rows = append(data.Rows, []interface{}{t.Format(time.RFC3339), v})
	}
	return data, nil
}

// --- Manual Reports ---

func (r *Repository) ListManual(ctx context.Context) ([]ManualReport, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, content, format, created_at
		FROM report_manual ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list manual reports: %w", err)
	}
	defer rows.Close()

	var items []ManualReport
	for rows.Next() {
		var mr ManualReport
		if err := rows.Scan(&mr.ID, &mr.Name, &mr.Description, &mr.Content, &mr.Format, &mr.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan manual report: %w", err)
		}
		items = append(items, mr)
	}
	return items, nil
}

func (r *Repository) GetManual(ctx context.Context, id int64) (*ManualReport, error) {
	var mr ManualReport
	err := r.pool.QueryRow(ctx, `SELECT id, name, description, content, format, created_at
		FROM report_manual WHERE id = $1`, id).Scan(
		&mr.ID, &mr.Name, &mr.Description, &mr.Content, &mr.Format, &mr.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get manual report %d: %w", id, err)
	}
	return &mr, nil
}

func (r *Repository) CreateManual(ctx context.Context, mr ManualReport) (*ManualReport, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO report_manual (name, description, content, format)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		mr.Name, mr.Description, mr.Content, mr.Format).Scan(&mr.ID, &mr.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create manual report: %w", err)
	}
	return &mr, nil
}

func (r *Repository) DeleteManual(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM report_manual WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete manual report %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("manual report %d not found", id)
	}
	return nil
}
