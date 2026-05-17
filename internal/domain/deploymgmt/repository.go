package deploymgmt

import (
	"context"
	"encoding/json"
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

func (r *Repository) Create(ctx context.Context, d *Deployment) error {
	nodesJSON, _ := json.Marshal(d.TargetNodes)
	return r.pool.QueryRow(ctx,
		`INSERT INTO deployments (site_id, site_name, site_domain, config_version, deploy_type,
		 nginx_config, modsec_config, target_nodes, operator_id, operator_name)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at`,
		d.SiteID, d.SiteName, d.SiteDomain, d.ConfigVersion, d.DeployType,
		d.NginxConfig, d.ModsecConfig, nodesJSON, d.OperatorID, d.OperatorName,
	).Scan(&d.ID, &d.CreatedAt)
}

func (r *Repository) ListBySite(ctx context.Context, siteID int64, page, pageSize int) ([]Deployment, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM deployments WHERE site_id = $1`, siteID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deployments: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, site_id, site_name, site_domain, config_version, deploy_type,
		 operator_id, operator_name, created_at
		 FROM deployments WHERE site_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		siteID, pageSize, (page-1)*pageSize,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list deployments: %w", err)
	}
	defer rows.Close()

	var items []Deployment
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.SiteID, &d.SiteName, &d.SiteDomain,
			&d.ConfigVersion, &d.DeployType, &d.OperatorID, &d.OperatorName,
			&d.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan deployment: %w", err)
		}
		items = append(items, d)
	}
	return items, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*Deployment, error) {
	d := &Deployment{}
	var nodesJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, site_id, site_name, site_domain, config_version, deploy_type,
		 nginx_config, modsec_config, target_nodes, operator_id, operator_name, created_at
		 FROM deployments WHERE id = $1`, id,
	).Scan(&d.ID, &d.SiteID, &d.SiteName, &d.SiteDomain, &d.ConfigVersion,
		&d.DeployType, &d.NginxConfig, &d.ModsecConfig, &nodesJSON,
		&d.OperatorID, &d.OperatorName, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get deployment: %w", err)
	}
	json.Unmarshal(nodesJSON, &d.TargetNodes)

	// Load node statuses
	rows, err := r.pool.Query(ctx,
		`SELECT id, deployment_id, node_id, node_hostname, status, message, applied_at, created_at
		 FROM deployment_node_status WHERE deployment_id = $1 ORDER BY id`, id)
	if err != nil {
		return d, nil // statuses are optional
	}
	defer rows.Close()

	for rows.Next() {
		var ns NodeDeployStatus
		if err := rows.Scan(&ns.ID, &ns.DeploymentID, &ns.NodeID, &ns.NodeHostname,
			&ns.Status, &ns.Message, &ns.AppliedAt, &ns.CreatedAt); err != nil {
			continue
		}
		d.NodeStatuses = append(d.NodeStatuses, ns)
	}
	return d, nil
}

func (r *Repository) CreateNodeStatuses(ctx context.Context, deploymentID int64, nodes []TargetNode) error {
	for _, n := range nodes {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO deployment_node_status (deployment_id, node_id, node_hostname, status)
			 VALUES ($1,$2,$3,'pending')
			 ON CONFLICT (deployment_id, node_id) DO NOTHING`,
			deploymentID, n.ID, n.Hostname)
		if err != nil {
			return fmt.Errorf("create node status for %s: %w", n.Hostname, err)
		}
	}
	return nil
}

func (r *Repository) UpdateNodeStatus(ctx context.Context, deploymentID int64, nodeHostname, status, message string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE deployment_node_status SET status = $1, message = $2, applied_at = $3
		 WHERE deployment_id = $4 AND node_hostname = $5`,
		status, message, time.Now(), deploymentID, nodeHostname)
	return err
}
