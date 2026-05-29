DROP INDEX IF EXISTS idx_system_upgrades_project;
DROP INDEX IF EXISTS idx_alert_policies_project;
DROP INDEX IF EXISTS idx_policies_project;
DROP INDEX IF EXISTS idx_sites_project;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS project_id;
ALTER TABLE alert_policies DROP COLUMN IF EXISTS project_id;
ALTER TABLE policies DROP COLUMN IF EXISTS project_id;
ALTER TABLE sites DROP COLUMN IF EXISTS project_id;
