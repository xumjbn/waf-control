-- Rollback of 000008.

BEGIN;

DROP INDEX IF EXISTS idx_roles_role_key;

ALTER TABLE users  DROP COLUMN IF EXISTS project;
ALTER TABLE users  DROP COLUMN IF EXISTS avatar;
ALTER TABLE roles  DROP COLUMN IF EXISTS color;
ALTER TABLE roles  DROP COLUMN IF EXISTS readonly;
ALTER TABLE roles  DROP COLUMN IF EXISTS role_key;

-- Restore the broader 000007 permission shape so a re-up of 000007 + skip-008
-- produces something readable. Module names match 000007 verbatim.
UPDATE roles SET permissions = '["*"]'                                                  WHERE name = '系统管理员';
UPDATE roles SET permissions = '["sites", "policies", "instances"]'                     WHERE name = '操作员';
UPDATE roles SET permissions = '["read"]'                                               WHERE name = '只读';
UPDATE roles SET permissions = '["security_overview", "logs", "reports", "alerts"]'     WHERE name = '安全分析师';
UPDATE roles SET permissions = '["monitoring", "logs", "reports"]'                      WHERE name = '审计员';
UPDATE roles SET permissions = '[]'                                                     WHERE name = '自定义角色';

COMMIT;
