package network

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

// --- Interfaces ---

func (r *Repository) ListInterfaces(ctx context.Context, nodeID int64) ([]Interface, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, node_id, name, COALESCE(mac_address,''), COALESCE(ip_address,''),
		COALESCE(netmask,''), COALESCE(gateway,''), mtu, status, iface_type,
		COALESCE(bond_master,''), COALESCE(bridge,''), created_at, updated_at
		FROM node_interfaces WHERE node_id = $1 ORDER BY name`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	defer rows.Close()

	var ifaces []Interface
	for rows.Next() {
		var i Interface
		if err := rows.Scan(&i.ID, &i.NodeID, &i.Name, &i.MACAddress, &i.IPAddress,
			&i.Netmask, &i.Gateway, &i.MTU, &i.Status, &i.IfaceType,
			&i.BondMaster, &i.Bridge, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan interface: %w", err)
		}
		ifaces = append(ifaces, i)
	}
	return ifaces, nil
}

func (r *Repository) CreateInterface(ctx context.Context, req CreateInterfaceRequest) (*Interface, error) {
	mtu := req.MTU
	if mtu == 0 {
		mtu = 1500
	}
	ifaceType := req.IfaceType
	if ifaceType == "" {
		ifaceType = "physical"
	}

	var i Interface
	err := r.pool.QueryRow(ctx, `INSERT INTO node_interfaces (node_id, name, mac_address, ip_address, netmask, gateway, mtu, iface_type, bond_master, bridge)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, node_id, name, COALESCE(mac_address,''), COALESCE(ip_address,''),
		COALESCE(netmask,''), COALESCE(gateway,''), mtu, status, iface_type,
		COALESCE(bond_master,''), COALESCE(bridge,''), created_at, updated_at`,
		req.NodeID, req.Name, req.MACAddress, req.IPAddress, req.Netmask, req.Gateway, mtu, ifaceType, req.BondMaster, req.Bridge).Scan(
		&i.ID, &i.NodeID, &i.Name, &i.MACAddress, &i.IPAddress,
		&i.Netmask, &i.Gateway, &i.MTU, &i.Status, &i.IfaceType,
		&i.BondMaster, &i.Bridge, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create interface: %w", err)
	}
	return &i, nil
}

func (r *Repository) UpdateInterface(ctx context.Context, id int64, req UpdateInterfaceRequest) (*Interface, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.IPAddress != nil {
		sets = append(sets, fmt.Sprintf("ip_address = $%d", argIdx))
		args = append(args, *req.IPAddress)
		argIdx++
	}
	if req.Netmask != nil {
		sets = append(sets, fmt.Sprintf("netmask = $%d", argIdx))
		args = append(args, *req.Netmask)
		argIdx++
	}
	if req.Gateway != nil {
		sets = append(sets, fmt.Sprintf("gateway = $%d", argIdx))
		args = append(args, *req.Gateway)
		argIdx++
	}
	if req.MTU != nil {
		sets = append(sets, fmt.Sprintf("mtu = $%d", argIdx))
		args = append(args, *req.MTU)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE node_interfaces SET %s WHERE id = $%d
		RETURNING id, node_id, name, COALESCE(mac_address,''), COALESCE(ip_address,''),
		COALESCE(netmask,''), COALESCE(gateway,''), mtu, status, iface_type,
		COALESCE(bond_master,''), COALESCE(bridge,''), created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var i Interface
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&i.ID, &i.NodeID, &i.Name, &i.MACAddress, &i.IPAddress,
		&i.Netmask, &i.Gateway, &i.MTU, &i.Status, &i.IfaceType,
		&i.BondMaster, &i.Bridge, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update interface: %w", err)
	}
	return &i, nil
}

func (r *Repository) DeleteInterface(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM node_interfaces WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete interface: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("interface not found")
	}
	return nil
}

// --- Bridges ---

func (r *Repository) ListBridges(ctx context.Context, nodeID int64) ([]Bridge, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, node_id, name, COALESCE(ip_address,''), COALESCE(netmask,''),
		members, stp_enabled, status, created_at, updated_at
		FROM network_bridges WHERE node_id = $1 ORDER BY name`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list bridges: %w", err)
	}
	defer rows.Close()

	var bridges []Bridge
	for rows.Next() {
		var b Bridge
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Name, &b.IPAddress, &b.Netmask,
			&b.Members, &b.STPEnabled, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge: %w", err)
		}
		bridges = append(bridges, b)
	}
	return bridges, nil
}

func (r *Repository) CreateBridge(ctx context.Context, req CreateBridgeRequest) (*Bridge, error) {
	members := req.Members
	if members == nil {
		members = []byte("[]")
	}

	var b Bridge
	err := r.pool.QueryRow(ctx, `INSERT INTO network_bridges (node_id, name, ip_address, netmask, members, stp_enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, node_id, name, COALESCE(ip_address,''), COALESCE(netmask,''),
		members, stp_enabled, status, created_at, updated_at`,
		req.NodeID, req.Name, req.IPAddress, req.Netmask, members, req.STPEnabled).Scan(
		&b.ID, &b.NodeID, &b.Name, &b.IPAddress, &b.Netmask,
		&b.Members, &b.STPEnabled, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create bridge: %w", err)
	}
	return &b, nil
}

func (r *Repository) UpdateBridge(ctx context.Context, id int64, req UpdateBridgeRequest) (*Bridge, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.IPAddress != nil {
		sets = append(sets, fmt.Sprintf("ip_address = $%d", argIdx))
		args = append(args, *req.IPAddress)
		argIdx++
	}
	if req.Netmask != nil {
		sets = append(sets, fmt.Sprintf("netmask = $%d", argIdx))
		args = append(args, *req.Netmask)
		argIdx++
	}
	if req.Members != nil {
		sets = append(sets, fmt.Sprintf("members = $%d", argIdx))
		args = append(args, *req.Members)
		argIdx++
	}
	if req.STPEnabled != nil {
		sets = append(sets, fmt.Sprintf("stp_enabled = $%d", argIdx))
		args = append(args, *req.STPEnabled)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE network_bridges SET %s WHERE id = $%d
		RETURNING id, node_id, name, COALESCE(ip_address,''), COALESCE(netmask,''),
		members, stp_enabled, status, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var b Bridge
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&b.ID, &b.NodeID, &b.Name, &b.IPAddress, &b.Netmask,
		&b.Members, &b.STPEnabled, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update bridge: %w", err)
	}
	return &b, nil
}

func (r *Repository) DeleteBridge(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM network_bridges WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete bridge: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("bridge not found")
	}
	return nil
}

// --- Bonds ---

func (r *Repository) ListBonds(ctx context.Context, nodeID int64) ([]Bond, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, node_id, name, mode, COALESCE(ip_address,''), COALESCE(netmask,''),
		members, status, created_at, updated_at
		FROM network_bonds WHERE node_id = $1 ORDER BY name`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list bonds: %w", err)
	}
	defer rows.Close()

	var bonds []Bond
	for rows.Next() {
		var b Bond
		if err := rows.Scan(&b.ID, &b.NodeID, &b.Name, &b.Mode, &b.IPAddress, &b.Netmask,
			&b.Members, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan bond: %w", err)
		}
		bonds = append(bonds, b)
	}
	return bonds, nil
}

func (r *Repository) CreateBond(ctx context.Context, req CreateBondRequest) (*Bond, error) {
	members := req.Members
	if members == nil {
		members = []byte("[]")
	}
	mode := req.Mode
	if mode == "" {
		mode = "active-backup"
	}

	var b Bond
	err := r.pool.QueryRow(ctx, `INSERT INTO network_bonds (node_id, name, mode, ip_address, netmask, members)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, node_id, name, mode, COALESCE(ip_address,''), COALESCE(netmask,''),
		members, status, created_at, updated_at`,
		req.NodeID, req.Name, mode, req.IPAddress, req.Netmask, members).Scan(
		&b.ID, &b.NodeID, &b.Name, &b.Mode, &b.IPAddress, &b.Netmask,
		&b.Members, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create bond: %w", err)
	}
	return &b, nil
}

func (r *Repository) UpdateBond(ctx context.Context, id int64, req UpdateBondRequest) (*Bond, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Mode != nil {
		sets = append(sets, fmt.Sprintf("mode = $%d", argIdx))
		args = append(args, *req.Mode)
		argIdx++
	}
	if req.IPAddress != nil {
		sets = append(sets, fmt.Sprintf("ip_address = $%d", argIdx))
		args = append(args, *req.IPAddress)
		argIdx++
	}
	if req.Netmask != nil {
		sets = append(sets, fmt.Sprintf("netmask = $%d", argIdx))
		args = append(args, *req.Netmask)
		argIdx++
	}
	if req.Members != nil {
		sets = append(sets, fmt.Sprintf("members = $%d", argIdx))
		args = append(args, *req.Members)
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	sets = append(sets, "updated_at = NOW()")
	query := fmt.Sprintf(`UPDATE network_bonds SET %s WHERE id = $%d
		RETURNING id, node_id, name, mode, COALESCE(ip_address,''), COALESCE(netmask,''),
		members, status, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var b Bond
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&b.ID, &b.NodeID, &b.Name, &b.Mode, &b.IPAddress, &b.Netmask,
		&b.Members, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update bond: %w", err)
	}
	return &b, nil
}

func (r *Repository) DeleteBond(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM network_bonds WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete bond: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("bond not found")
	}
	return nil
}

// --- Routes ---

func (r *Repository) ListRoutes(ctx context.Context, nodeID int64) ([]Route, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, node_id, destination, netmask, gateway,
		COALESCE(interface,''), metric, created_at
		FROM network_routes WHERE node_id = $1 ORDER BY metric, destination`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var rt Route
		if err := rows.Scan(&rt.ID, &rt.NodeID, &rt.Destination, &rt.Netmask, &rt.Gateway,
			&rt.Interface, &rt.Metric, &rt.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan route: %w", err)
		}
		routes = append(routes, rt)
	}
	return routes, nil
}

func (r *Repository) CreateRoute(ctx context.Context, req CreateRouteRequest) (*Route, error) {
	metric := req.Metric
	if metric == 0 {
		metric = 100
	}

	var rt Route
	err := r.pool.QueryRow(ctx, `INSERT INTO network_routes (node_id, destination, netmask, gateway, interface, metric)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, node_id, destination, netmask, gateway, COALESCE(interface,''), metric, created_at`,
		req.NodeID, req.Destination, req.Netmask, req.Gateway, req.Interface, metric).Scan(
		&rt.ID, &rt.NodeID, &rt.Destination, &rt.Netmask, &rt.Gateway,
		&rt.Interface, &rt.Metric, &rt.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create route: %w", err)
	}
	return &rt, nil
}

func (r *Repository) DeleteRoute(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM network_routes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete route: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("route not found")
	}
	return nil
}
