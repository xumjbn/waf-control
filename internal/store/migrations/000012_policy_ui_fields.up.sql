-- waf-control · policy UI fields for NW · 04
--
-- 前端 mocks/nebula.ts Rule 列：id, name, scope, field, match, action,
-- priority, enabled, builtin, hits。已有 policies 表只有 name/severity/
-- action/is_enabled/description，扩展剩余字段。

BEGIN;

ALTER TABLE policies ADD COLUMN IF NOT EXISTS scope        VARCHAR(64) NOT NULL DEFAULT '全部站点';
ALTER TABLE policies ADD COLUMN IF NOT EXISTS field        VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE policies ADD COLUMN IF NOT EXISTS match_value  TEXT        NOT NULL DEFAULT '';
ALTER TABLE policies ADD COLUMN IF NOT EXISTS priority     INTEGER     NOT NULL DEFAULT 100;
ALTER TABLE policies ADD COLUMN IF NOT EXISTS builtin      BOOLEAN     NOT NULL DEFAULT FALSE;
ALTER TABLE policies ADD COLUMN IF NOT EXISTS hits         BIGINT      NOT NULL DEFAULT 0;
ALTER TABLE policies ADD COLUMN IF NOT EXISTS last_hit_at  TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_policies_scope    ON policies(scope);
CREATE INDEX IF NOT EXISTS idx_policies_priority ON policies(priority);
CREATE INDEX IF NOT EXISTS idx_policies_builtin  ON policies(builtin);

COMMIT;
