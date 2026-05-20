-- waf-control · NW · 09 系统管理：settings 描述 + license 完整字段

BEGIN;

ALTER TABLE system_settings ADD COLUMN IF NOT EXISTS description VARCHAR(255) NOT NULL DEFAULT '';

ALTER TABLE licenses ADD COLUMN IF NOT EXISTS customer       VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS contact_email  VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS grace_until    TIMESTAMPTZ;
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS issued_at      TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE licenses ADD COLUMN IF NOT EXISTS edition        VARCHAR(32)  NOT NULL DEFAULT 'community';

CREATE INDEX IF NOT EXISTS idx_licenses_is_active ON licenses(is_active);

-- 默认设置项（PageSystem 期望的最小键集）
INSERT INTO system_settings (key, value, category, description) VALUES
  ('platform.name',     'OpenWAF',       'basic',  '平台名称'),
  ('platform.timezone', 'Asia/Shanghai', 'basic',  '默认时区'),
  ('platform.lang',     'zh-CN',         'basic',  '默认语言'),
  ('alert.retention_days',     '90',     'data',   '告警保留天数'),
  ('log.retention_days',       '30',     'data',   '日志保留天数'),
  ('security.session_timeout', '3600',   'security','会话超时秒')
ON CONFLICT (key) DO NOTHING;

COMMIT;
