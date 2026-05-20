BEGIN;
DROP INDEX IF EXISTS idx_licenses_is_active;

ALTER TABLE licenses DROP COLUMN IF EXISTS edition;
ALTER TABLE licenses DROP COLUMN IF EXISTS issued_at;
ALTER TABLE licenses DROP COLUMN IF EXISTS grace_until;
ALTER TABLE licenses DROP COLUMN IF EXISTS contact_email;
ALTER TABLE licenses DROP COLUMN IF EXISTS customer;

ALTER TABLE system_settings DROP COLUMN IF EXISTS description;
COMMIT;
