-- waf-control · 多租户 scope 铺到热表（attack_logs / nodes）。
-- PG 11+ 下 ADD COLUMN ... DEFAULT <常量> 是元数据操作，不重写全表，无需长锁。
-- 现有行落 project_id=1（projects 表 default 项目，见 000005）。
-- 后续 agent 上报攻击日志时可按站点归属回填真实 project_id。

ALTER TABLE attack_logs
  ADD COLUMN IF NOT EXISTS project_id BIGINT NOT NULL DEFAULT 1;
CREATE INDEX IF NOT EXISTS idx_attack_logs_project ON attack_logs(project_id);

ALTER TABLE nodes
  ADD COLUMN IF NOT EXISTS project_id BIGINT NOT NULL DEFAULT 1;
CREATE INDEX IF NOT EXISTS idx_nodes_project ON nodes(project_id);
