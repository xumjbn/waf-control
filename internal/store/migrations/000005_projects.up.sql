CREATE TABLE IF NOT EXISTS projects (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL UNIQUE,
    description VARCHAR(256),
    domain_id   VARCHAR(64) NOT NULL DEFAULT 'default',
    parent_id   BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    is_domain   BOOLEAN NOT NULL DEFAULT FALSE,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_parent ON projects(parent_id);

CREATE TABLE IF NOT EXISTS project_user_roles (
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (project_id, user_id, role_id)
);

INSERT INTO projects (name, description, domain_id, is_domain, enabled) VALUES
    ('default', '默认项目', 'default', FALSE, TRUE)
ON CONFLICT (name) DO NOTHING;
