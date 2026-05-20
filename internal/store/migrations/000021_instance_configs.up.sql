-- waf-control · instance_configs 节点管理员可编辑配置
--
-- 设计稿 waf-admin/src/pages/instance/InstanceDetail.tsx 配置 Tab 字段：
--   · role               数据面 / 控制面 / 边缘
--   · gateway            默认网关
--   · dns                DNS 服务器（逗号分隔）
--   · tags               标签（逗号分隔）
--   · enabled            实例启用开关（关闭=停接流量，保留配置）
--   · max_connections    资源限制 - 最大连接数
--   · max_qps            资源限制 - QPS 上限
--   · cpu_soft_limit     资源限制 - CPU 软限制 %
--   · maintenance_window 维护窗口 - 每周维护时段
--
-- 这些字段不在 agent 自报心跳里（心跳是运行时观测值），需要管理员通过
-- UI 写入并由 control 下发给 agent。

BEGIN;

CREATE TABLE IF NOT EXISTS instance_configs (
    node_id            VARCHAR(128) PRIMARY KEY,
    role               VARCHAR(16)  NOT NULL DEFAULT 'data',          -- data / control / edge
    gateway            VARCHAR(64)  NOT NULL DEFAULT '',
    dns                VARCHAR(256) NOT NULL DEFAULT '',              -- 逗号分隔
    tags               VARCHAR(256) NOT NULL DEFAULT '',              -- 逗号分隔
    enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
    max_connections    INTEGER      NOT NULL DEFAULT 50000,
    max_qps            INTEGER      NOT NULL DEFAULT 20000,
    cpu_soft_limit     INTEGER      NOT NULL DEFAULT 80,              -- 百分比
    maintenance_window VARCHAR(64)  NOT NULL DEFAULT '周日 02:00 - 04:00',
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMIT;
