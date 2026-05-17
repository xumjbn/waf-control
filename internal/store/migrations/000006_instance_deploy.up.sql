-- 000006_instance_deploy.up.sql
-- 部署记录与实例管理

CREATE TABLE IF NOT EXISTS deployments (
    id              BIGSERIAL PRIMARY KEY,
    site_id         BIGINT NOT NULL,
    site_name       VARCHAR(128) NOT NULL DEFAULT '',
    site_domain     VARCHAR(256) NOT NULL DEFAULT '',
    config_version  VARCHAR(64) NOT NULL DEFAULT '',
    deploy_type     VARCHAR(32) NOT NULL DEFAULT 'full',
    nginx_config    TEXT NOT NULL DEFAULT '',
    modsec_config   TEXT NOT NULL DEFAULT '',
    target_nodes    JSONB NOT NULL DEFAULT '[]',
    operator_id     BIGINT NOT NULL DEFAULT 0,
    operator_name   VARCHAR(64) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deployment_node_status (
    id              BIGSERIAL PRIMARY KEY,
    deployment_id   BIGINT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    node_id         BIGINT NOT NULL,
    node_hostname   VARCHAR(128) NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    message         TEXT NOT NULL DEFAULT '',
    applied_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(deployment_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_deployments_site_id ON deployments(site_id);
CREATE INDEX IF NOT EXISTS idx_deployments_created_at ON deployments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dns_deployment_id ON deployment_node_status(deployment_id);
CREATE INDEX IF NOT EXISTS idx_dns_status ON deployment_node_status(status);
