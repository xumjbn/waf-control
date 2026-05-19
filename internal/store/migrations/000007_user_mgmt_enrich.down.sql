-- Remove project assignments for seed users
DELETE FROM project_user_roles WHERE user_id IN (
    SELECT id FROM users WHERE username IN ('zhangsan', 'lisi', 'wangwu', 'security')
);

-- Remove role assignments for seed users
DELETE FROM user_roles WHERE user_id IN (
    SELECT id FROM users WHERE username IN ('zhangsan', 'lisi', 'wangwu', 'security')
);

-- Remove seed users
DELETE FROM users WHERE username IN ('zhangsan', 'lisi', 'wangwu', 'security');

-- Remove added projects
DELETE FROM projects WHERE name IN ('项目 A — 主业务', '项目 B — 支付');

-- Restore default project name
UPDATE projects SET name = 'default', description = '默认项目' WHERE name = '默认项目';

-- Remove new roles
DELETE FROM roles WHERE name IN ('安全分析师', '审计员', '自定义角色');

-- Restore original role names
UPDATE roles SET name = 'admin', description = '系统管理员', permissions = '["*"]'
WHERE name = '系统管理员';

UPDATE roles SET name = 'operator', description = '操作员', permissions = '["read", "write"]'
WHERE name = '操作员';

UPDATE roles SET name = 'viewer', description = '只读用户', permissions = '["read"]'
WHERE name = '只读';

-- Remove last_login column
ALTER TABLE users DROP COLUMN IF EXISTS last_login;
