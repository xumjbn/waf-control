package ha

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Get(ctx context.Context) (*Config, error) {
	var c Config
	err := r.pool.QueryRow(ctx, `SELECT id, mode, COALESCE(virtual_ip,''), priority,
		COALESCE(interface_name,''), COALESCE(peer_address,''), is_enabled, heartbeat_interval_sec,
		created_at, updated_at FROM ha_config LIMIT 1`).Scan(
		&c.ID, &c.Mode, &c.VirtualIP, &c.Priority,
		&c.Interface, &c.PeerAddress, &c.IsEnabled, &c.HeartbeatSec,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get ha config: %w", err)
	}
	return &c, nil
}

func (r *Repository) Upsert(ctx context.Context, req UpsertConfigRequest) (*Config, error) {
	mode := req.Mode
	if mode == "" {
		mode = "active-standby"
	}
	priority := req.Priority
	if priority <= 0 {
		priority = 100
	}
	heartbeat := req.HeartbeatSec
	if heartbeat <= 0 {
		heartbeat = 5
	}
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	var c Config
	err := r.pool.QueryRow(ctx, `INSERT INTO ha_config (id, mode, virtual_ip, priority, interface_name, peer_address, is_enabled, heartbeat_interval_sec)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			mode = EXCLUDED.mode,
			virtual_ip = EXCLUDED.virtual_ip,
			priority = EXCLUDED.priority,
			interface_name = EXCLUDED.interface_name,
			peer_address = EXCLUDED.peer_address,
			is_enabled = EXCLUDED.is_enabled,
			heartbeat_interval_sec = EXCLUDED.heartbeat_interval_sec,
			updated_at = NOW()
		RETURNING id, mode, COALESCE(virtual_ip,''), priority,
			COALESCE(interface_name,''), COALESCE(peer_address,''), is_enabled, heartbeat_interval_sec,
			created_at, updated_at`,
		mode, req.VirtualIP, priority, req.Interface, req.PeerAddress, isEnabled, heartbeat).Scan(
		&c.ID, &c.Mode, &c.VirtualIP, &c.Priority,
		&c.Interface, &c.PeerAddress, &c.IsEnabled, &c.HeartbeatSec,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert ha config: %w", err)
	}
	return &c, nil
}
