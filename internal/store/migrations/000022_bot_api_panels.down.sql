BEGIN;
DROP INDEX IF EXISTS idx_api_endpoints_site;
DROP TABLE IF EXISTS api_endpoints;
DROP INDEX IF EXISTS idx_bot_challenges_site;
DROP TABLE IF EXISTS bot_challenges;
COMMIT;
