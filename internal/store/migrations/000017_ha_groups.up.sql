-- waf-control · HA 主备组 for 防护实例页 (NW · 06)
--
-- 前端 src/pages/instance/index.tsx 的『HA 主备状态』卡片之前是写死的 4 行表。
-- 这里建 ha_groups 表，主备节点引用 cluster_members.node_id（弱关联，不外键，
-- 因为 agent 自注册时 node_id 才进来；删 cluster 不应级联 ha）。

BEGIN;

CREATE TABLE IF NOT EXISTS ha_groups (
    id           BIGSERIAL PRIMARY KEY,
    name         VARCHAR(32)  NOT NULL UNIQUE,
    primary_node VARCHAR(128) NOT NULL,
    standby_node VARCHAR(128) NOT NULL,
    vip          VARCHAR(64)  NOT NULL,
    state        VARCHAR(16)  NOT NULL DEFAULT 'ok',  -- ok / warn / critical
    last_switch  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ha_groups_state ON ha_groups(state);

-- 种子数据：与 mocks/nebula.ts 之前硬编码的 HA-01..HA-04 对齐，
-- 真实环境会被管理员通过 PUT /api/v1/ha-groups/{id} 覆盖。
INSERT INTO ha_groups (name, primary_node, standby_node, vip, state) VALUES
    ('HA-01', 'waf-01', 'waf-02', '10.0.1.100', 'ok'),
    ('HA-02', 'waf-03', 'waf-04', '10.0.2.100', 'ok'),
    ('HA-03', 'waf-05', 'waf-06', '10.0.3.100', 'warn'),
    ('HA-04', 'waf-07', 'waf-08', '10.0.4.100', 'ok')
ON CONFLICT (name) DO NOTHING;

COMMIT;
