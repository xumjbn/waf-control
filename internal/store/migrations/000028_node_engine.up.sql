-- waf-control · 节点后端引擎类型（nginx / openresty / safeline）。
-- agent 注册时通过 RegisterRequest.labels["engine"] 上报，control 存这里，
-- 实例列表 UI 显示节点用的是哪种引擎。现有行默认 nginx。

ALTER TABLE nodes
  ADD COLUMN IF NOT EXISTS engine VARCHAR(16) NOT NULL DEFAULT 'nginx';
