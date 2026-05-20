-- waf-control · site_modules 站点级别防护模块开关 + 等级
--
-- 模型：site N:M module，每条 binding 有 enabled + level。
-- 同 module（sqli/xss/rce/lfi-rfi/bot/rate-limit/ip-reputation/virtual-patches）
-- 通过 policies.category 与具体 ModSec 规则关联：
--   site_modules(site_id, module='sqli', level='medium')
--     ↘ 命中 → 由 agent 加载 policies WHERE category='sqli' AND severity≥medium
--
-- level 语义：
--   high   → 启用 critical + high + medium + low 全套（最严格，可能误报）
--   medium → 启用 critical + high + medium       （推荐）
--   low    → 仅启用 critical                     （宽松）

BEGIN;

CREATE TABLE IF NOT EXISTS site_modules (
    site_id    BIGINT      NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    module     VARCHAR(32) NOT NULL,  -- sqli / xss / rce / lfi-rfi / bot / rate-limit / ip-reputation / virtual-patches
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    level      VARCHAR(8)  NOT NULL DEFAULT 'medium', -- low / medium / high
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (site_id, module)
);

CREATE INDEX IF NOT EXISTS idx_site_modules_site ON site_modules(site_id);

COMMIT;
