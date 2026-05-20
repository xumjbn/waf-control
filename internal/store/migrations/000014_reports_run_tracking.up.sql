-- waf-control · 报表执行追踪字段 + 统一列表 metadata
-- 配合 NW · 08 报表中心的"上次/下次执行时间"列与"执行/下载"按钮。

BEGIN;

ALTER TABLE report_custom   ADD COLUMN IF NOT EXISTS last_run_at  TIMESTAMPTZ;
ALTER TABLE report_custom   ADD COLUMN IF NOT EXISTS next_run_at  TIMESTAMPTZ;
ALTER TABLE report_custom   ADD COLUMN IF NOT EXISTS is_enabled   BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE report_combined ADD COLUMN IF NOT EXISTS last_run_at  TIMESTAMPTZ;
ALTER TABLE report_combined ADD COLUMN IF NOT EXISTS is_enabled   BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS cron         VARCHAR(64) NOT NULL DEFAULT '0 0 * * *';
ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS last_run_at  TIMESTAMPTZ;
ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS next_run_at  TIMESTAMPTZ;
ALTER TABLE report_timing   ADD COLUMN IF NOT EXISTS is_enabled   BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE report_manual   ADD COLUMN IF NOT EXISTS file_path    VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE report_manual   ADD COLUMN IF NOT EXISTS file_size    BIGINT       NOT NULL DEFAULT 0;

COMMIT;
