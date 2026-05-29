-- waf-control · 系统升级任务（执行流水 + 进度 + 日志），支撑 PageUpgrade 实时轮询。
-- 一个升级包可对应多次任务（重试、回滚等），通过 package_id 关联。

CREATE TABLE IF NOT EXISTS upgrade_tasks (
    id           BIGSERIAL PRIMARY KEY,
    package_id   BIGINT      NOT NULL REFERENCES system_upgrades(id) ON DELETE CASCADE,
    status       VARCHAR(16) NOT NULL DEFAULT 'queued',     -- queued / running / done / failed
    progress     INT         NOT NULL DEFAULT 0,            -- 0-100
    log_json     JSONB       NOT NULL DEFAULT '[]'::jsonb,  -- 追加式日志行 [{t, l, k}]
    error_msg    TEXT,
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_pkg    ON upgrade_tasks(package_id);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_status ON upgrade_tasks(status);
CREATE INDEX IF NOT EXISTS idx_upgrade_tasks_pkg_status ON upgrade_tasks(package_id, status);
