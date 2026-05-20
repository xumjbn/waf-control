BEGIN;
DROP INDEX IF EXISTS idx_system_upgrades_is_latest;
DROP INDEX IF EXISTS idx_system_upgrades_is_current;

ALTER TABLE system_upgrades DROP COLUMN IF EXISTS applied_at;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS released_at;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS is_latest;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS is_current;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS download_url;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS checksum;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS changes_summary;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS notes;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS channel;
ALTER TABLE system_upgrades DROP COLUMN IF EXISTS type;
COMMIT;
