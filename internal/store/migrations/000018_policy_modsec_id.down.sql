BEGIN;
DROP INDEX IF EXISTS uq_policies_modsec_id;
ALTER TABLE policies DROP COLUMN IF EXISTS modsec_id;
COMMIT;
