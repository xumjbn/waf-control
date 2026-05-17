package deploymgmt

import "time"

type DeployType string

const (
	DeployFull   DeployType = "full"
	DeploySite   DeployType = "site"
	DeployPolicy DeployType = "policy"
)

type Deployment struct {
	ID            int64              `json:"id"`
	SiteID        int64              `json:"site_id"`
	SiteName      string             `json:"site_name"`
	SiteDomain    string             `json:"site_domain"`
	ConfigVersion string             `json:"config_version"`
	DeployType    string             `json:"deploy_type"`
	NginxConfig   string             `json:"nginx_config,omitempty"`
	ModsecConfig  string             `json:"modsec_config,omitempty"`
	TargetNodes   []TargetNode       `json:"target_nodes"`
	OperatorID    int64              `json:"operator_id"`
	OperatorName  string             `json:"operator_name"`
	NodeStatuses  []NodeDeployStatus `json:"node_statuses,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

type TargetNode struct {
	ID       int64  `json:"id"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type NodeDeployStatus struct {
	ID           int64      `json:"id"`
	DeploymentID int64      `json:"deployment_id"`
	NodeID       int64      `json:"node_id"`
	NodeHostname string     `json:"node_hostname"`
	Status       string     `json:"status"`
	Message      string     `json:"message,omitempty"`
	AppliedAt    *time.Time `json:"applied_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type DeployRequest struct {
	DeployType  string  `json:"deploy_type"`
	TargetNodes []int64 `json:"target_nodes"`
}

type DeployPreview struct {
	SiteConfig   string `json:"site_config"`
	PolicyConfig string `json:"policy_config"`
}
