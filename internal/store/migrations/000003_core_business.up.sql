-- devices (设备/实例)
CREATE TABLE IF NOT EXISTS devices (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL UNIQUE,
    serial_no   VARCHAR(64),
    model       VARCHAR(64),
    status      VARCHAR(16) NOT NULL DEFAULT 'offline',
    ip_address  VARCHAR(45),
    version     VARCHAR(32),
    description VARCHAR(256),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- nodes (节点)
CREATE TABLE IF NOT EXISTS nodes (
    id          BIGSERIAL PRIMARY KEY,
    device_id   BIGINT REFERENCES devices(id) ON DELETE SET NULL,
    name        VARCHAR(128) NOT NULL,
    hostname    VARCHAR(256),
    ip_address  VARCHAR(45) NOT NULL,
    status      VARCHAR(16) NOT NULL DEFAULT 'offline',
    cpu_cores   INTEGER,
    memory_mb   BIGINT,
    disk_gb     BIGINT,
    os_version  VARCHAR(64),
    agent_ver   VARCHAR(32),
    last_seen   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_nodes_device_id ON nodes(device_id);

-- node_interfaces (网络接口)
CREATE TABLE IF NOT EXISTS node_interfaces (
    id          BIGSERIAL PRIMARY KEY,
    node_id     BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    name        VARCHAR(32) NOT NULL,
    mac_address VARCHAR(17),
    ip_address  VARCHAR(45),
    netmask     VARCHAR(45),
    gateway     VARCHAR(45),
    mtu         INTEGER DEFAULT 1500,
    status      VARCHAR(16) NOT NULL DEFAULT 'down',
    iface_type  VARCHAR(16) NOT NULL DEFAULT 'physical',
    bond_master VARCHAR(32),
    bridge      VARCHAR(32),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(node_id, name)
);

-- network_bridges (网桥)
CREATE TABLE IF NOT EXISTS network_bridges (
    id          BIGSERIAL PRIMARY KEY,
    node_id     BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    name        VARCHAR(32) NOT NULL,
    ip_address  VARCHAR(45),
    netmask     VARCHAR(45),
    members     JSONB NOT NULL DEFAULT '[]',
    stp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    status      VARCHAR(16) NOT NULL DEFAULT 'down',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(node_id, name)
);

-- network_bonds (bond)
CREATE TABLE IF NOT EXISTS network_bonds (
    id          BIGSERIAL PRIMARY KEY,
    node_id     BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    name        VARCHAR(32) NOT NULL,
    mode        VARCHAR(16) NOT NULL DEFAULT 'active-backup',
    ip_address  VARCHAR(45),
    netmask     VARCHAR(45),
    members     JSONB NOT NULL DEFAULT '[]',
    status      VARCHAR(16) NOT NULL DEFAULT 'down',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(node_id, name)
);

-- network_routes (路由)
CREATE TABLE IF NOT EXISTS network_routes (
    id          BIGSERIAL PRIMARY KEY,
    node_id     BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    destination VARCHAR(45) NOT NULL,
    netmask     VARCHAR(45) NOT NULL,
    gateway     VARCHAR(45) NOT NULL,
    interface   VARCHAR(32),
    metric      INTEGER DEFAULT 100,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_routes_node_id ON network_routes(node_id);

-- sites (防护站点)
CREATE TABLE IF NOT EXISTS sites (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(256) NOT NULL,
    domain      VARCHAR(256) NOT NULL,
    listen_port INTEGER NOT NULL DEFAULT 80,
    ssl_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ssl_cert    TEXT,
    ssl_key     TEXT,
    upstream    JSONB NOT NULL DEFAULT '[]',
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
    waf_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description VARCHAR(512),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sites_domain ON sites(domain);

-- protect_assoc (站点-设备关联)
CREATE TABLE IF NOT EXISTS protect_assoc (
    id        BIGSERIAL PRIMARY KEY,
    site_id   BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    UNIQUE(site_id, device_id)
);

-- policy_categories (策略分类)
CREATE TABLE IF NOT EXISTS policy_categories (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL UNIQUE,
    description VARCHAR(256),
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- policies (WAF策略)
CREATE TABLE IF NOT EXISTS policies (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(128) NOT NULL,
    category_id   BIGINT REFERENCES policy_categories(id) ON DELETE SET NULL,
    severity      VARCHAR(16) NOT NULL DEFAULT 'medium',
    action        VARCHAR(16) NOT NULL DEFAULT 'block',
    is_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    description   VARCHAR(512),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policies_category ON policies(category_id);

-- policy_rules (策略规则)
CREATE TABLE IF NOT EXISTS policy_rules (
    id          BIGSERIAL PRIMARY KEY,
    policy_id   BIGINT NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    rule_type   VARCHAR(32) NOT NULL,
    field       VARCHAR(64) NOT NULL,
    operator    VARCHAR(16) NOT NULL,
    value       TEXT NOT NULL,
    logic       VARCHAR(4) NOT NULL DEFAULT 'AND',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policy_rules_policy ON policy_rules(policy_id);

-- policy_change_history (策略变更历史)
CREATE TABLE IF NOT EXISTS policy_change_history (
    id          BIGSERIAL PRIMARY KEY,
    policy_id   BIGINT NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL,
    action      VARCHAR(16) NOT NULL,
    old_value   JSONB,
    new_value   JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policy_history_policy ON policy_change_history(policy_id);
CREATE INDEX idx_policy_history_time ON policy_change_history(created_at DESC);

-- seed policy categories
INSERT INTO policy_categories (name, description, sort_order) VALUES
    ('SQL注入', 'SQL Injection 防护规则', 1),
    ('XSS', '跨站脚本攻击防护规则', 2),
    ('命令注入', 'OS Command Injection 防护规则', 3),
    ('路径遍历', 'Path Traversal 防护规则', 4),
    ('文件包含', 'File Inclusion 防护规则', 5),
    ('信息泄露', 'Information Disclosure 防护规则', 6),
    ('自定义规则', '用户自定义规则', 99)
ON CONFLICT (name) DO NOTHING;
