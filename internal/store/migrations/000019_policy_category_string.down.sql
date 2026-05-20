BEGIN;
DROP INDEX IF EXISTS idx_policies_category;
ALTER TABLE policies DROP COLUMN IF EXISTS category;
COMMIT;
