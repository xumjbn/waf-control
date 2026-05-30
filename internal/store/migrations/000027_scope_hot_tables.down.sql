DROP INDEX IF EXISTS idx_nodes_project;
DROP INDEX IF EXISTS idx_attack_logs_project;
ALTER TABLE nodes DROP COLUMN IF EXISTS project_id;
ALTER TABLE attack_logs DROP COLUMN IF EXISTS project_id;
