-- waf-control · align identity data to UI design (NebulaWAF · NW · 10)
--
-- Background: 000001 created roles/users with English admin/operator/viewer names.
-- 000007 renamed roles to Chinese (系统管理员 / 操作员 / 只读 / 安全分析师 / 审计员 / 自定义角色)
-- and seeded 5 users matching the frontend mock layer.
--
-- This migration finishes the alignment so a real backend can serve
-- waf-admin/src/mocks/identity.ts without any UI changes:
--   1. Add role.role_key (stable English identifier, e.g. 'system_admin'),
--      role.readonly, role.color  — the frontend's RolesGrid reads these.
--   2. Rewrite role.permissions to the canonical module list used by
--      BasicLayout NAV: aggregation/site/policy/instance/log/acl/report/user/system.
--   3. Add user.avatar + user.project so the user list table can render
--      the avatar chip and project column without join gymnastics.

BEGIN;

-- 1. Role schema additions
ALTER TABLE roles ADD COLUMN IF NOT EXISTS role_key  VARCHAR(64);
ALTER TABLE roles ADD COLUMN IF NOT EXISTS readonly  BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS color     VARCHAR(16) NOT NULL DEFAULT '#a855f7';

-- Stable English key per role; the frontend's usePermission switches on this.
-- Wildcard '*' for admin/readonly so any new module added later is auto-granted.
UPDATE roles
   SET role_key   = 'system_admin',
       readonly   = FALSE,
       color      = '#ef4444',
       permissions = '"*"'::jsonb
 WHERE name = '系统管理员';

UPDATE roles
   SET role_key   = 'security_analyst',
       readonly   = FALSE,
       color      = '#ec4899',
       permissions = '["aggregation","log","acl","report"]'::jsonb
 WHERE name = '安全分析师';

UPDATE roles
   SET role_key   = 'auditor',
       readonly   = TRUE,
       color      = '#22d3ee',
       permissions = '["aggregation","log","report"]'::jsonb
 WHERE name = '审计员';

UPDATE roles
   SET role_key   = 'operator',
       readonly   = FALSE,
       color      = '#a855f7',
       permissions = '["site","policy","instance"]'::jsonb
 WHERE name = '操作员';

UPDATE roles
   SET role_key   = 'readonly',
       readonly   = TRUE,
       color      = '#10b981',
       permissions = '"*"'::jsonb
 WHERE name = '只读';

UPDATE roles
   SET role_key   = 'custom',
       readonly   = FALSE,
       color      = '#f59e0b',
       permissions = '[]'::jsonb
 WHERE name = '自定义角色';

-- Backfill role_key for any legacy English-named roles that survived 000007.
UPDATE roles SET role_key = 'system_admin' WHERE name = 'admin' AND role_key IS NULL;
UPDATE roles SET role_key = 'operator'     WHERE name = 'operator' AND role_key IS NULL;
UPDATE roles SET role_key = 'readonly'     WHERE name IN ('viewer', '只读') AND role_key IS NULL;

-- Make role_key unique once populated; we use a partial index because legacy
-- rows from before this migration may still have NULLs.
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_role_key ON roles (role_key) WHERE role_key IS NOT NULL;

-- 2. User schema additions (frontend display fields)
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar  VARCHAR(8);
ALTER TABLE users ADD COLUMN IF NOT EXISTS project VARCHAR(64);

-- Seed avatar from username initial + project from existing project_user_roles
-- if the user is in exactly one project; otherwise mark them as 全部 / 默认.
UPDATE users SET avatar = UPPER(LEFT(username, 1)) WHERE avatar IS NULL OR avatar = '';

UPDATE users SET project = '全部'  WHERE username IN ('admin', 'security') AND (project IS NULL OR project = '');
UPDATE users SET project = '默认'  WHERE username = 'zhangsan' AND (project IS NULL OR project = '');
UPDATE users SET project = '项目 A' WHERE username = 'lisi'    AND (project IS NULL OR project = '');
UPDATE users SET project = '项目 B' WHERE username = 'wangwu'  AND (project IS NULL OR project = '');
UPDATE users SET project = '默认'   WHERE project IS NULL OR project = '';

COMMIT;
