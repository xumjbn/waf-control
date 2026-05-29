-- waf-control · NW · 04 Bot 管理 + API 安全 Tab 后端
--
-- 前端 policy 页面 BotPanel 之前是：Donut 5 行硬编码 + 挑战模式 5 个 Toggle
-- 仅本地 state；ApiSecurityPanel 之前是 4 KPI + 端点表 5 行硬编码。本次设计
-- 两张表 + 一张 KPI 视图：
--   · bot_challenges      ：挑战模式开关（站点级别 1:N）
--   · api_endpoints       ：站点已注册的 API 端点（带 schema 校验状态）
--   · view: api_kpi       ：未授权拦截 / JWT 重放 / 脱敏 count 派生

BEGIN;

CREATE TABLE IF NOT EXISTS bot_challenges (
    site_id     BIGINT      NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    challenge   VARCHAR(32) NOT NULL,  -- js / tls / dev / slider / behave
    enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    config      JSONB       NOT NULL DEFAULT '{}'::jsonb,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (site_id, challenge)
);
CREATE INDEX IF NOT EXISTS idx_bot_challenges_site ON bot_challenges(site_id);

CREATE TABLE IF NOT EXISTS api_endpoints (
    id             BIGSERIAL PRIMARY KEY,
    site_id        BIGINT       NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    method         VARCHAR(10)  NOT NULL,  -- GET / POST / PUT / DELETE / PATCH / OPTIONS
    path           VARCHAR(512) NOT NULL,
    auth_type      VARCHAR(32)  NOT NULL DEFAULT 'JWT', -- None / JWT / JWT+MFA / OAuth / APIKey
    rate_limit     VARCHAR(64)  NOT NULL DEFAULT '',    -- 自由文本，例如 '100/s' 或 '5/min/IP'
    schema_status  VARCHAR(16)  NOT NULL DEFAULT 'pending',  -- pending / imported / failed
    qps            INTEGER      NOT NULL DEFAULT 0,     -- 最近窗口 qps（统计任务回写）
    status         VARCHAR(16)  NOT NULL DEFAULT 'ok',  -- ok / warn / down
    description    TEXT         NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (site_id, method, path)
);
CREATE INDEX IF NOT EXISTS idx_api_endpoints_site ON api_endpoints(site_id);

COMMIT;
