-- NMAP: Port Scan Tasks
CREATE TABLE IF NOT EXISTS portscans (
    id              BIGSERIAL PRIMARY KEY,
    target          VARCHAR(255) NOT NULL,
    ports           VARCHAR(128) NOT NULL DEFAULT '1-1024',
    scan_type       VARCHAR(32) NOT NULL DEFAULT 'tcp_syn',
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    result          JSONB,
    error_message   TEXT,
    node_id         BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portscans_status ON portscans(status);

-- Security: Auth Hosts
CREATE TABLE IF NOT EXISTS auth_hosts (
    id              BIGSERIAL PRIMARY KEY,
    address         VARCHAR(255) NOT NULL,
    port            INTEGER NOT NULL DEFAULT 6379,
    host_type       VARCHAR(32) NOT NULL DEFAULT 'redis',
    username        VARCHAR(128),
    password_enc    TEXT,
    is_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    description     VARCHAR(256),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Security: Auth Host Config (singleton row via id)
CREATE TABLE IF NOT EXISTS auth_host_config (
    id              BIGSERIAL PRIMARY KEY,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    max_attempts    INTEGER NOT NULL DEFAULT 5,
    lockout_duration INTEGER NOT NULL DEFAULT 300,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO auth_host_config (id, enabled, max_attempts, lockout_duration)
VALUES (1, TRUE, 5, 300)
ON CONFLICT (id) DO NOTHING;

-- Monitor: Metric Specs
CREATE TABLE IF NOT EXISTS monitor_metric_specs (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL UNIQUE,
    description     VARCHAR(256),
    unit            VARCHAR(32)
);

-- Monitor: Metrics (recorded data points)
CREATE TABLE IF NOT EXISTS monitor_metrics (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    value           DOUBLE PRECISION NOT NULL,
    unit            VARCHAR(32),
    node_id         BIGINT,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitor_metrics_name_time ON monitor_metrics(name, recorded_at DESC);
CREATE INDEX idx_monitor_metrics_node_name ON monitor_metrics(node_id, name, recorded_at DESC);

-- Flow: Flow Logs
CREATE TABLE IF NOT EXISTS flow_logs (
    id              BIGSERIAL PRIMARY KEY,
    src_ip          VARCHAR(45) NOT NULL,
    dst_ip          VARCHAR(45) NOT NULL,
    src_port        INTEGER NOT NULL,
    dst_port        INTEGER NOT NULL,
    protocol        VARCHAR(16) NOT NULL DEFAULT 'TCP',
    bytes_sent      BIGINT NOT NULL DEFAULT 0,
    bytes_received  BIGINT NOT NULL DEFAULT 0,
    packets_sent    BIGINT NOT NULL DEFAULT 0,
    packets_received BIGINT NOT NULL DEFAULT 0,
    duration        REAL,
    application     VARCHAR(128),
    node_id         BIGINT,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_flow_logs_time ON flow_logs(recorded_at DESC);
CREATE INDEX idx_flow_logs_src_ip ON flow_logs(src_ip);
CREATE INDEX idx_flow_logs_dst_ip ON flow_logs(dst_ip);

-- Flow: Saved Queries
CREATE TABLE IF NOT EXISTS flow_saved_queries (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    query           TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Flow: Monitor Records
CREATE TABLE IF NOT EXISTS flow_monitor_records (
    id              BIGSERIAL PRIMARY KEY,
    node_id         BIGINT,
    total_bytes     BIGINT NOT NULL DEFAULT 0,
    total_packets   BIGINT NOT NULL DEFAULT 0,
    conn_count      INTEGER NOT NULL DEFAULT 0,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_flow_monitor_time ON flow_monitor_records(recorded_at DESC);

-- Reports: Custom Reports
CREATE TABLE IF NOT EXISTS report_custom (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    description     VARCHAR(256),
    filters         TEXT,
    schedule        VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reports: Combined Reports
CREATE TABLE IF NOT EXISTS report_combined (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    description     VARCHAR(256),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reports: Timing Reports
CREATE TABLE IF NOT EXISTS report_timing (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    description     VARCHAR(256),
    metric          VARCHAR(128) NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    interval        VARCHAR(16) NOT NULL DEFAULT '5m',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Reports: Manual Reports
CREATE TABLE IF NOT EXISTS report_manual (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    description     VARCHAR(256),
    content         TEXT,
    format          VARCHAR(16) NOT NULL DEFAULT 'markdown',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
