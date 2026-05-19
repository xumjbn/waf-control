-- Add last_login column to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login TIMESTAMPTZ;

-- Update existing roles to match frontend UI names
UPDATE roles SET name = '系统管理员', description = '全部资源 + 全部操作', permissions = '["*"]'
WHERE name = 'admin';

UPDATE roles SET name = '操作员', description = '业务管理 + 规则配置', permissions = '["sites", "policies", "instances"]'
WHERE name = 'operator';

UPDATE roles SET name = '只读', description = '查看权限', permissions = '["read"]'
WHERE name = 'viewer';

-- Insert new roles
INSERT INTO roles (name, description, permissions) VALUES
    ('安全分析师', '查看 + 审计 + 告警处置', '["security_overview", "logs", "reports", "alerts"]'),
    ('审计员', '只读 + 报表导出', '["monitoring", "logs", "reports"]'),
    ('自定义角色', '按需组合', '[]')
ON CONFLICT (name) DO NOTHING;

-- Insert seed users (password: user123)
INSERT INTO users (username, password, email, real_name, is_active, last_login) VALUES
    ('zhangsan', '$2a$10$belziPMcaEnl4cLD21ogZOaGEbNqT9IhiL/h4OqDmAzIqgSHpwKmS', 'zhangsan@example.com', '张三', TRUE, '2026-05-17 14:22:00+08'),
    ('lisi', '$2a$10$belziPMcaEnl4cLD21ogZOaGEbNqT9IhiL/h4OqDmAzIqgSHpwKmS', 'lisi@example.com', '李四', FALSE, '2026-05-16 09:15:00+08'),
    ('wangwu', '$2a$10$belziPMcaEnl4cLD21ogZOaGEbNqT9IhiL/h4OqDmAzIqgSHpwKmS', 'wangwu@example.com', '王五', TRUE, '2026-05-15 18:42:00+08'),
    ('security', '$2a$10$belziPMcaEnl4cLD21ogZOaGEbNqT9IhiL/h4OqDmAzIqgSHpwKmS', 'sec-team@example.com', '安全团队', TRUE, '2026-05-17 11:08:00+08')
ON CONFLICT (username) DO NOTHING;

-- Update admin user
UPDATE users SET email = 'admin@cloudwall.local', last_login = '2026-05-17 15:30:00+08'
WHERE username = 'admin';

-- Assign user roles
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r WHERE u.username = 'admin' AND r.name = '系统管理员'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r WHERE u.username = 'zhangsan' AND r.name = '审计员'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r WHERE u.username = 'lisi' AND r.name = '操作员'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r WHERE u.username = 'wangwu' AND r.name = '操作员'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r WHERE u.username = 'security' AND r.name = '安全分析师'
ON CONFLICT DO NOTHING;

-- Update default project
UPDATE projects SET name = '默认项目', description = '系统默认 · 全局可见'
WHERE name = 'default';

-- Insert additional projects
INSERT INTO projects (name, description, domain_id, is_domain, enabled, created_at) VALUES
    ('项目 A — 主业务', '官网 + API', 'default', FALSE, TRUE, '2026-03-10 00:00:00+08'),
    ('项目 B — 支付', '独立合规环境', 'default', FALSE, TRUE, '2026-05-01 00:00:00+08')
ON CONFLICT (name) DO NOTHING;

-- Assign users to projects (project_user_roles)
-- zhangsan → 默认项目
INSERT INTO project_user_roles (project_id, user_id, role_id)
SELECT p.id, u.id, r.id FROM projects p, users u, roles r
WHERE p.name = '默认项目' AND u.username = 'zhangsan' AND r.name = '审计员'
ON CONFLICT DO NOTHING;

-- lisi → 项目 A
INSERT INTO project_user_roles (project_id, user_id, role_id)
SELECT p.id, u.id, r.id FROM projects p, users u, roles r
WHERE p.name = '项目 A — 主业务' AND u.username = 'lisi' AND r.name = '操作员'
ON CONFLICT DO NOTHING;

-- wangwu → 项目 B
INSERT INTO project_user_roles (project_id, user_id, role_id)
SELECT p.id, u.id, r.id FROM projects p, users u, roles r
WHERE p.name = '项目 B — 支付' AND u.username = 'wangwu' AND r.name = '操作员'
ON CONFLICT DO NOTHING;

-- admin + security → all projects (全部)
INSERT INTO project_user_roles (project_id, user_id, role_id)
SELECT p.id, u.id, r.id FROM projects p, users u, roles r
WHERE u.username = 'admin' AND r.name = '系统管理员'
ON CONFLICT DO NOTHING;

INSERT INTO project_user_roles (project_id, user_id, role_id)
SELECT p.id, u.id, r.id FROM projects p, users u, roles r
WHERE u.username = 'security' AND r.name = '安全分析师'
ON CONFLICT DO NOTHING;
