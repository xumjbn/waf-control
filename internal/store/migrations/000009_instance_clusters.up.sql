-- waf-control · clusters for 防护实例页 (NW · 06)
--
-- 前端 mocks/nebula.ts Cluster 形态：id/name/nodes/vip/algo/state/site_count
-- 现 instancemgmt 域是 agent gRPC 只读视图，没有 cluster 概念。
-- 这里新建 clusters 表 + node_clusters 多对多，cluster CRUD 走数据库。
-- node 实例本身仍由 agent 注册维护，新增 node 走配置下发 + 等心跳。

BEGIN;

CREATE TABLE IF NOT EXISTS clusters (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(64) NOT NULL UNIQUE,
    vip         VARCHAR(64) NOT NULL DEFAULT '',
    algo        VARCHAR(32) NOT NULL DEFAULT 'round-robin',
    state       VARCHAR(16) NOT NULL DEFAULT 'ok',
    site_count  INTEGER NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 节点对集群的归属（agent 注册时由管理员人工分配）
CREATE TABLE IF NOT EXISTS cluster_members (
    cluster_id BIGINT NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    node_id    VARCHAR(128) NOT NULL,
    role       VARCHAR(16) NOT NULL DEFAULT 'primary',     -- primary / standby
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (cluster_id, node_id)
);
CREATE INDEX IF NOT EXISTS idx_cluster_members_node ON cluster_members(node_id);

INSERT INTO clusters (name, vip, algo, state, site_count, description) VALUES
    ('CLU-WWW',    '10.0.1.100', 'round-robin', 'ok',   3, '官网主集群'),
    ('CLU-API',    '10.0.2.100', 'least-conn',  'ok',   4, 'API 网关集群'),
    ('CLU-MOBILE', '10.0.3.100', 'ip-hash',     'warn', 2, '移动端集群 · 降级中'),
    ('CLU-INNER',  '10.0.4.100', 'round-robin', 'ok',   3, '内网业务集群')
ON CONFLICT (name) DO NOTHING;

COMMIT;
