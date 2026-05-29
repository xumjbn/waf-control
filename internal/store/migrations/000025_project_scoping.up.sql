-- waf-control · 多租户最小可用：给主要资源表加 project_id 列，关联 projects 表。
-- 现有行（迁移前已存在数据）落到 id=1 即 000005 的默认 'default' 项目。
-- 后续 ScopeMiddleware 按 project_user_roles 把请求限定到用户可访问的项目集合。
--
-- 暂未铺到 logs / monitor_metrics / heartbeats —— 它们是 hot path 巨表，
-- 改列需停服。先把 sites / policies / alert_policies / system_upgrades 落地，
-- 后续在 logs 上加 project_id 时另做一个 backfill migration。

ALTER TABLE sites
  ADD COLUMN IF NOT EXISTS project_id BIGINT NOT NULL DEFAULT 1
  REFERENCES projects(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_sites_project ON sites(project_id);

ALTER TABLE policies
  ADD COLUMN IF NOT EXISTS project_id BIGINT NOT NULL DEFAULT 1
  REFERENCES projects(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_policies_project ON policies(project_id);

ALTER TABLE alert_policies
  ADD COLUMN IF NOT EXISTS project_id BIGINT NOT NULL DEFAULT 1
  REFERENCES projects(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_alert_policies_project ON alert_policies(project_id);

ALTER TABLE system_upgrades
  ADD COLUMN IF NOT EXISTS project_id BIGINT NOT NULL DEFAULT 1
  REFERENCES projects(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_system_upgrades_project ON system_upgrades(project_id);
