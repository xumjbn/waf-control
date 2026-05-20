-- waf-control · site runtime/binding fields for NW · 03 UI
--
-- 前端 mocks/nebula.ts Site 形态需要 rps / blocked_rate / instance(label)。
-- 这些是运行时指标，但 UI 列表里要每行展示，因此 sites 表上再加 cached
-- 字段；由监控管道定时回写，UI 直接读取免 N+1。

BEGIN;

ALTER TABLE sites ADD COLUMN IF NOT EXISTS rps           DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS blocked_rate  DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS instance_label VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS metrics_updated_at TIMESTAMPTZ;

COMMIT;
