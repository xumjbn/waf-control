-- waf-control · alert channel 完整 CRUD + config_json
--
-- 前端 PageAlert 渠道类型：email / wechat / dingtalk / pagerduty / webhook / sms
-- 当前 alert_channels 表只有 name/kind/target/is_enabled，扩展：
--   - config       JSONB    各 kind 的额外配置（webhook URL、密钥、模板）
--   - description  VARCHAR  备注
--   - severity     VARCHAR  最低触发等级（info/warn/critical）
--   - kind 增加 CHECK 约束（软约束：仍接受未来扩展）

BEGIN;

ALTER TABLE alert_channels ADD COLUMN IF NOT EXISTS config      JSONB        NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE alert_channels ADD COLUMN IF NOT EXISTS description VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE alert_channels ADD COLUMN IF NOT EXISTS severity    VARCHAR(16)  NOT NULL DEFAULT 'warn';

CREATE INDEX IF NOT EXISTS idx_alert_channels_kind ON alert_channels(kind);

COMMIT;
