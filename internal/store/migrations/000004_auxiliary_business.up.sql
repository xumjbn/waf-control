-- Load Balancing: VIPs
CREATE TABLE IF NOT EXISTS lb_vips (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    description     VARCHAR(256),
    address         VARCHAR(45) NOT NULL,
    protocol        VARCHAR(16) NOT NULL DEFAULT 'HTTP',
    protocol_port   INTEGER NOT NULL DEFAULT 80,
    pool_id         BIGINT,
    connection_limit INTEGER DEFAULT -1,
    session_persistence BOOLEAN NOT NULL DEFAULT FALSE,
    admin_state_up  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Load Balancing: Pools
CREATE TABLE IF NOT EXISTS lb_pools (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    description     VARCHAR(256),
    protocol        VARCHAR(16) NOT NULL DEFAULT 'HTTP',
    lb_method       VARCHAR(32) NOT NULL DEFAULT 'ROUND_ROBIN',
    admin_state_up  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE lb_vips ADD CONSTRAINT fk_vip_pool FOREIGN KEY (pool_id) REFERENCES lb_pools(id) ON DELETE SET NULL;

-- Load Balancing: Members
CREATE TABLE IF NOT EXISTS lb_members (
    id              BIGSERIAL PRIMARY KEY,
    pool_id         BIGINT NOT NULL REFERENCES lb_pools(id) ON DELETE CASCADE,
    address         VARCHAR(45) NOT NULL,
    protocol_port   INTEGER NOT NULL,
    weight          INTEGER NOT NULL DEFAULT 1,
    admin_state_up  BOOLEAN NOT NULL DEFAULT TRUE,
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lb_members_pool ON lb_members(pool_id);

-- Load Balancing: Health Monitors
CREATE TABLE IF NOT EXISTS lb_health_monitors (
    id              BIGSERIAL PRIMARY KEY,
    pool_id         BIGINT NOT NULL REFERENCES lb_pools(id) ON DELETE CASCADE,
    type            VARCHAR(16) NOT NULL DEFAULT 'HTTP',
    delay           INTEGER NOT NULL DEFAULT 5,
    timeout         INTEGER NOT NULL DEFAULT 3,
    max_retries     INTEGER NOT NULL DEFAULT 3,
    http_method     VARCHAR(8) DEFAULT 'GET',
    url_path        VARCHAR(256) DEFAULT '/',
    expected_codes  VARCHAR(64) DEFAULT '200',
    admin_state_up  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_lb_hm_pool ON lb_health_monitors(pool_id);

-- ACL Rules
CREATE TABLE IF NOT EXISTS acl_rules (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    description     VARCHAR(256),
    direction       VARCHAR(8) NOT NULL DEFAULT 'inbound',
    action          VARCHAR(8) NOT NULL DEFAULT 'deny',
    protocol        VARCHAR(8),
    src_ip          VARCHAR(45),
    src_port        INTEGER,
    dst_ip          VARCHAR(45),
    dst_port        INTEGER,
    priority        INTEGER NOT NULL DEFAULT 100,
    is_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_acl_priority ON acl_rules(priority);

-- HA Configuration (singleton via id=1 ON CONFLICT)
CREATE TABLE IF NOT EXISTS ha_config (
    id                    BIGSERIAL PRIMARY KEY,
    mode                  VARCHAR(16) NOT NULL DEFAULT 'active-standby',
    virtual_ip            VARCHAR(45),
    priority              INTEGER DEFAULT 100,
    interface_name        VARCHAR(32),
    peer_address          VARCHAR(45),
    is_enabled            BOOLEAN NOT NULL DEFAULT TRUE,
    heartbeat_interval_sec INTEGER DEFAULT 5,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- System Settings (KV store with id)
CREATE TABLE IF NOT EXISTS system_settings (
    id          BIGSERIAL PRIMARY KEY,
    key         VARCHAR(128) NOT NULL UNIQUE,
    value       TEXT NOT NULL,
    category    VARCHAR(64),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_settings (key, value, category) VALUES
    ('system.name', 'CloudWall WAF', 'general'),
    ('system.language', 'zh-CN', 'general'),
    ('system.timezone', 'Asia/Shanghai', 'general'),
    ('log.retain_days', '90', 'log'),
    ('log.max_backup_count', '10', 'log')
ON CONFLICT (key) DO NOTHING;

-- Licenses
CREATE TABLE IF NOT EXISTS licenses (
    id              BIGSERIAL PRIMARY KEY,
    license_key     TEXT NOT NULL,
    product_name    VARCHAR(128),
    max_nodes       INTEGER DEFAULT 1,
    expires_at      TIMESTAMPTZ,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- System Upgrades
CREATE TABLE IF NOT EXISTS system_upgrades (
    id              BIGSERIAL PRIMARY KEY,
    version         VARCHAR(32) NOT NULL,
    file_name       VARCHAR(256) NOT NULL,
    file_size       BIGINT,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Attack Logs
CREATE TABLE IF NOT EXISTS attack_logs (
    id              BIGSERIAL PRIMARY KEY,
    node_id         BIGINT,
    src_ip          VARCHAR(45),
    dst_ip          VARCHAR(45),
    src_port        INTEGER,
    dst_port        INTEGER,
    protocol        VARCHAR(16),
    attack_type     VARCHAR(64),
    rule_id         VARCHAR(64),
    action          VARCHAR(16),
    payload         TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_attack_logs_time ON attack_logs(occurred_at DESC);
CREATE INDEX idx_attack_logs_node ON attack_logs(node_id, occurred_at DESC);

-- Antivirus Logs
CREATE TABLE IF NOT EXISTS antivirus_logs (
    id              BIGSERIAL PRIMARY KEY,
    node_id         BIGINT,
    file_name       VARCHAR(256),
    virus_name      VARCHAR(128),
    file_path       VARCHAR(512),
    action          VARCHAR(16),
    src_ip          VARCHAR(45),
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_antivirus_logs_time ON antivirus_logs(occurred_at DESC);

-- Anti-tamper Logs
CREATE TABLE IF NOT EXISTS antitamper_logs (
    id              BIGSERIAL PRIMARY KEY,
    node_id         BIGINT,
    file_path       VARCHAR(512),
    change_type     VARCHAR(32),
    action          VARCHAR(16),
    detail          TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_antitamper_logs_time ON antitamper_logs(occurred_at DESC);

-- Heartbeats
CREATE TABLE IF NOT EXISTS heartbeats (
    id              BIGSERIAL PRIMARY KEY,
    node_id         BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    status          VARCHAR(16) NOT NULL DEFAULT 'healthy',
    cpu_percent     REAL,
    memory_percent  REAL,
    disk_percent    REAL,
    reported_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_heartbeats_node_time ON heartbeats(node_id, reported_at DESC);
