package aggregation

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waf-control/internal/agent"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool, agentSvc *agent.Service) {
	repo := NewRepository(pool)
	h := NewHandler(repo, agentSvc)

	r.Route("/sys_monitor", func(r chi.Router) {
		r.Get("/system_resource", h.SystemResourceAll)
		r.Get("/{instance_id}/system_resource", h.InstanceSystemResource)
		r.Get("/nic_state", h.NicState)
		r.Get("/interface", h.NetworkInterface)
		r.Get("/network_flow", h.NetworkFlow)
		r.Get("/history/manager", h.HistoryManager)
		r.Get("/history/instance/{instance_id}", h.HistoryInstance)
	})

	r.Route("/system", func(r chi.Router) {
		r.Get("/running_mode", h.RunningMode)
		r.Get("/managertime", h.ManagerTime)
		r.Get("/alltime", h.AllTime)
		r.Put("/changetime", h.ChangeManagerTime)
		r.Put("/{instance_id}/changetime", h.ChangeInstanceTime)
	})

	r.Get("/site_stats", h.SiteStats)
	r.Get("/attack_stats/statistic_info", h.AttackStatistic)
	r.Get("/attack_logs", h.AttackLogs)
	r.Get("/attack_logs_num", h.AttackLogsCount)

	r.Get("/statistic_trend/top_sites_info", h.TopSites)
	r.Get("/monitor/attack/severity", h.AttackSeverity)
	r.Get("/monitor/attack/src-ips-top", h.AttackSourceTop)
	r.Get("/monitor/flow/top10app", h.FlowTop10App)
	r.Get("/monitor/flow/top10ip", h.FlowTop10IP)
	r.Get("/monitor/flow/top10app/{app}", h.FlowAppDetail)
	r.Get("/monitor/flow/top10ip/{ip}", h.FlowIPDetail)
	r.Get("/monitor/realtime", h.RealtimeMetricGet)
	r.Get("/monitor/history", h.HistoryMetricGet)
	r.Get("/service-statistics", h.ServiceStatistics)
}
