-- waf-control · NW · 11 系统升级：补齐 type / notes / is_current /
-- is_latest / changes_summary / checksum / channel 字段

BEGIN;

ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS type             VARCHAR(16)  NOT NULL DEFAULT 'patch'; -- patch / minor / major / security
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS channel          VARCHAR(16)  NOT NULL DEFAULT 'stable'; -- stable / beta / dev
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS notes            TEXT         NOT NULL DEFAULT '';
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS changes_summary  TEXT         NOT NULL DEFAULT '';
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS checksum         VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS download_url     VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS is_current       BOOLEAN      NOT NULL DEFAULT FALSE;
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS is_latest        BOOLEAN      NOT NULL DEFAULT FALSE;
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS released_at      TIMESTAMPTZ;
ALTER TABLE system_upgrades ADD COLUMN IF NOT EXISTS applied_at       TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_system_upgrades_is_current ON system_upgrades(is_current);
CREATE INDEX IF NOT EXISTS idx_system_upgrades_is_latest  ON system_upgrades(is_latest);

COMMIT;
