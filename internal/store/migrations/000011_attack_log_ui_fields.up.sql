-- waf-control · attack log full UI fields for NW · 05
--
-- 前端 mocks/nebula.ts AttackEvent 列：t, ip, region, country, lat, lng, site,
-- domain, type, typeLabel, typeColor, risk, action, method, uri, payload,
-- ruleId, ua。当前 AttackLog 只有 src_ip / attack_type / rule_id / action /
-- payload / occurred_at，补齐其余列。

BEGIN;

ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS region      VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS country     VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS lat         DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS lng         DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS site        VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS domain      VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS type_label  VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS type_color  VARCHAR(16) NOT NULL DEFAULT '#8e84a3';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS risk        VARCHAR(8)  NOT NULL DEFAULT '中';   -- 高/中/低
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS method      VARCHAR(8)  NOT NULL DEFAULT 'GET';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS uri         TEXT NOT NULL DEFAULT '';
ALTER TABLE attack_logs ADD COLUMN IF NOT EXISTS user_agent  TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_attack_logs_site ON attack_logs(site);
CREATE INDEX IF NOT EXISTS idx_attack_logs_country ON attack_logs(country);
CREATE INDEX IF NOT EXISTS idx_attack_logs_risk ON attack_logs(risk);

COMMIT;
