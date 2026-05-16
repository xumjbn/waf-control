CREATE TABLE IF NOT EXISTS tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_type  VARCHAR(16) NOT NULL DEFAULT 'refresh',
    token_hash  VARCHAR(256) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tokens_user_id ON tokens(user_id);
CREATE INDEX idx_tokens_expires_at ON tokens(expires_at) WHERE revoked = FALSE;

CREATE TABLE IF NOT EXISTS operation_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL,
    method      VARCHAR(10) NOT NULL,
    path        VARCHAR(512) NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    client_ip   VARCHAR(45),
    request_body  TEXT,
    response_body TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oplog_user_id ON operation_logs(user_id);
CREATE INDEX idx_oplog_created_at ON operation_logs(created_at DESC);
