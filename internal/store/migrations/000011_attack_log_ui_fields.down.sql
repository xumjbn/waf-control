BEGIN;
DROP INDEX IF EXISTS idx_attack_logs_risk;
DROP INDEX IF EXISTS idx_attack_logs_country;
DROP INDEX IF EXISTS idx_attack_logs_site;

ALTER TABLE attack_logs DROP COLUMN IF EXISTS user_agent;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS uri;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS method;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS risk;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS type_color;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS type_label;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS domain;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS site;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS lng;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS lat;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS country;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS region;
COMMIT;
