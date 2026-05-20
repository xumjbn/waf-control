BEGIN;
DROP INDEX IF EXISTS idx_alert_channels_kind;
ALTER TABLE alert_channels DROP COLUMN IF EXISTS severity;
ALTER TABLE alert_channels DROP COLUMN IF EXISTS description;
ALTER TABLE alert_channels DROP COLUMN IF EXISTS config;
COMMIT;
