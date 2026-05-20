-- waf-control · policies.modsec_id 与 deploy/modsec/rules.d/<cat>/<id>-*.conf 对齐
-- 用于 SyncFromDir 启动期幂等地把 modsec 规则同步成 builtin policies，
-- 让前端规则页内置规则与 agent 上 ModSecurity 实际执行的规则严格一一对应。

BEGIN;

ALTER TABLE policies ADD COLUMN IF NOT EXISTS modsec_id VARCHAR(32);

-- 唯一约束（允许 NULL —— 用户手工建的规则没有 modsec_id）
CREATE UNIQUE INDEX IF NOT EXISTS uq_policies_modsec_id
    ON policies(modsec_id) WHERE modsec_id IS NOT NULL;

COMMIT;
