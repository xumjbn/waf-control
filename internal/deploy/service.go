package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/waf-control/proto/agent"
)

type ConfigSender interface {
	BroadcastConfig(nodeID string, configType pb.ConfigUpdate_ConfigType, payload []byte)
}

type NodeLister interface {
	ListByDeviceID(ctx context.Context, deviceID int64) ([]NodeBrief, error)
}

type NodeBrief struct {
	ID       int64  `json:"id"`
	DeviceID int64  `json:"device_id"`
	Hostname string `json:"hostname"`
	IPAddr   string `json:"ip_address"`
}

type Service struct {
	nodes  NodeLister
	sender ConfigSender
}

func NewService(nodes NodeLister, sender ConfigSender) *Service {
	return &Service{
		nodes:  nodes,
		sender: sender,
	}
}

func (s *Service) DeployNginx(ctx context.Context, site *SiteConfig, targets []int64) error {
	slog.Info("deploying nginx config", "domain", site.Domain, "targets", len(targets))

	cfg := generateNginx(site)
	payload := []byte(cfg)

	if len(targets) == 0 {
		s.sender.BroadcastConfig("*", pb.ConfigUpdate_FULL, payload)
		return nil
	}

	for _, deviceID := range targets {
		nodes, err := s.nodes.ListByDeviceID(ctx, deviceID)
		if err != nil {
			slog.Warn("list nodes for deploy target", "device_id", deviceID, "error", err)
			continue
		}
		for _, n := range nodes {
			s.sender.BroadcastConfig(n.Hostname, pb.ConfigUpdate_FULL, payload)
		}
	}

	return nil
}

func (s *Service) DeployModsec(ctx context.Context, policy *PolicyConfig, targets []int64) error {
	slog.Info("deploying modsec config", "policy", policy.Name, "targets", len(targets))

	cfg := generateModsec(policy)
	payload := []byte(cfg)

	if len(targets) == 0 {
		s.sender.BroadcastConfig("*", pb.ConfigUpdate_POLICY, payload)
		return nil
	}

	for _, deviceID := range targets {
		nodes, err := s.nodes.ListByDeviceID(ctx, deviceID)
		if err != nil {
			slog.Warn("list nodes for deploy target", "device_id", deviceID, "error", err)
			continue
		}
		for _, n := range nodes {
			s.sender.BroadcastConfig(n.Hostname, pb.ConfigUpdate_POLICY, payload)
		}
	}
	return nil
}

func (s *Service) DeployAll(ctx context.Context, site *SiteConfig, policy *PolicyConfig) error {
	slog.Info("deploying full configuration")

	nginxCfg := generateNginx(site)
	modsecCfg := generateModsec(policy)

	s.sender.BroadcastConfig("*", pb.ConfigUpdate_FULL, []byte(nginxCfg))
	time.Sleep(100 * time.Millisecond)
	s.sender.BroadcastConfig("*", pb.ConfigUpdate_POLICY, []byte(modsecCfg))

	return nil
}

type DeployResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Version   string `json:"version"`
	Timestamp int64  `json:"timestamp"`
}

func NewResult(success bool, message string) DeployResult {
	return DeployResult{
		Success:   success,
		Message:   message,
		Version:   fmt.Sprintf("%d", time.Now().UnixMilli()),
		Timestamp: time.Now().Unix(),
	}
}
