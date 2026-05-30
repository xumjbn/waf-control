-- waf-control · 报表产物表。报表生成（手动触发 / cron 调度）把 CSV 内容落库，
-- 下载端点直接服务最近一次产物，避免依赖可写文件系统。

CREATE TABLE IF NOT EXISTS report_outputs (
    id          BIGSERIAL PRIMARY KEY,
    report_kind VARCHAR(16)  NOT NULL,           -- custom / combined / timing / manual
    report_id   BIGINT       NOT NULL,
    filename    VARCHAR(256) NOT NULL,
    content     TEXT         NOT NULL,           -- CSV 文本（含 UTF-8 BOM）
    row_count   INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_report_outputs_lookup
    ON report_outputs(report_kind, report_id, created_at DESC);
