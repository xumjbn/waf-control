# WAF Backend 迁移设计文档

## 1. 项目背景

将原有 WAF 管理系统从多语言、多组件架构迁移为统一的 Go 后端：

| 原系统组件 | 语言 | 功能 |
|---|---|---|
| system-service/managerd | Go (gorilla/mux) | 管理平面 API，25 个业务模块 |
| asg-service | Python 2.7 | 节点级 agent（心跳、监控、bypass、网络配置） |

## 2. 原系统架构分析

### 2.1 managerd 架构

- **路由**：gorilla/mux，集中注册在 `api/api.go`
- **模式**：Backend Interface 模式 — 每个模块定义 `Backend` 接口，handler 通过 `GetBackend()` 调用
- **中间件**：auth wrapper + oplog wrapper 包裹每个 handler
- **存储**：三重存储
  - MySQL：业务数据（站点、策略、负载均衡等）
  - etcd：配置 + 认证 + 节点信息
  - Elasticsearch：日志查询（攻击日志、流量日志）

### 2.2 asg-service 架构

- **通信**：RabbitMQ RPC（管理端调用 agent 执行命令）
- **心跳**：etcd lease 续期，lease 到期 = 节点离线
- **功能**：系统监控、硬件 bypass 控制、网络接口配置、固件升级、防篡改、日志采集、磁盘监控

### 2.3 25 个 API 模块清单

identity, asgs, protect_sites, policies, lb, node, network, acl, ha, security, license, system, upgrade, autoscaling, asgpool, failover, reports, logs/attack, logs/operate, logs/flow, logs/antivirus, logs/antitamper, monitor/attack, monitor/system, appident

## 3. 存储层变更分析

### 3.1 去除 etcd 的影响与替代方案

原系统使用 etcd 的场景及替代：

| 原 etcd 用途 | 为何能替代 | PostgreSQL 替代方案 |
|---|---|---|
| Token 存储（利用 TTL 自动过期） | JWT 自带过期语义，无需外部 TTL | `tokens` 表存刷新 token + 黑名单 |
| 节点心跳（lease 续期） | 单管理节点场景不需要分布式一致性 | `heartbeats` 表 + `updated_at` 字段定时扫描 |
| ACL 规则存储 | 无需 watch 推送，API 拉取即可 | `acl_policies` 表 |
| 系统配置 KV | 配置变更低频，不需要 watch | `system_settings` 表（key-value） |
| 节点/接口信息 | 普通 CRUD | `nodes`, `node_interfaces` 表 |
| 分布式锁（HA leader election） | 单节点时不需要；HA 时用 pg advisory lock | `pg_advisory_lock` 或 `SELECT FOR UPDATE` |

**去除 etcd 的收益：**
- 减少一个运维组件，降低部署复杂度
- 减少一类故障面（etcd 集群脑裂、磁盘满等）
- 开发调试只需 PostgreSQL 一个数据库

**风险与应对：**
- 如果未来需要多管理节点 HA：PostgreSQL 本身支持流复制 + `pg_advisory_lock` 实现 leader election
- 如果需要配置实时推送：可加 PostgreSQL LISTEN/NOTIFY 或引入轻量消息队列

### 3.2 去除 MySQL 的影响

原 MySQL 存储的业务数据直接迁移到 PostgreSQL 对应表，无功能损失。PostgreSQL 在 JSONB、全文索引、分区表方面均优于 MySQL。

### 3.3 去除 Elasticsearch 的影响

| 原 ES 用途 | PostgreSQL 替代方案 | 性能考量 |
|---|---|---|
| 攻击日志全文搜索 | GIN 索引 + JSONB + `to_tsvector` | 百万级数据量足够，亿级需考虑 TimescaleDB |
| 流量日志按时间查询 | 分区表（按天/月） | 配合 BRIN 索引，查询性能优秀 |
| 日志聚合统计 | PostgreSQL 窗口函数 + 物化视图 | 满足报表需求 |

**数据量评估：** WAF 设备日志量通常在每天 10 万 ~ 100 万条，PostgreSQL 完全胜任。如果单台设备日志超过亿级，后续可引入 TimescaleDB 扩展或独立的 ClickHouse。

### 3.4 去除 RabbitMQ 的影响

| 原 RabbitMQ 用途 | 替代方案 | 理由 |
|---|---|---|
| 管理端 → agent RPC 调用 | gRPC 双向流 | 直连更简单，延迟更低，类型安全 |
| 异步任务队列 | gRPC + goroutine | 任务量不大，无需独立队列 |
| 事件广播 | gRPC server-side streaming | 管理端主动推送配置变更给 agent |

## 4. 技术选型决策

| 层 | 选择 | 对比考虑 | 决策理由 |
|---|---|---|---|
| Web 框架 | chi | gin, echo, fiber | 标准 net/http 兼容，零魔法，中间件生态丰富 |
| 数据库驱动 | pgx | database/sql, gorm | 原生 PostgreSQL 协议，连接池内置，性能最优 |
| 配置 | viper | envconfig, koanf | 支持 TOML/YAML/ENV 多源，热重载 |
| 日志 | slog | zap, zerolog | Go 1.21+ 标准库，零外部依赖 |
| 认证 | JWT | session, OAuth | 无状态，适合 API 服务 |
| 迁移 | golang-migrate | goose, atlas | 纯 SQL 迁移文件，工具链成熟 |
| 节点通信 | gRPC | REST, NATS | 强类型、双向流、代码生成 |
| 定时任务 | gocron | cron | Go 原生，支持分布式锁 |

## 5. 分阶段实施计划

### Phase 1：项目骨架 ✅ 已完成
- Go module 初始化
- chi router + 中间件（recovery、requestID、logger、CORS）
- PostgreSQL 连接池
- 配置加载（viper + TOML）
- 数据库迁移框架 + 初始 users/roles schema
- Makefile、.gitignore、git init

### Phase 2：身份认证模块
- 用户 CRUD、角色 CRUD
- JWT 签发/验证/刷新
- RBAC 中间件
- 操作日志中间件
- 密码加密（bcrypt）

### Phase 3：核心业务 API
- ASG 设备管理
- 防护站点
- WAF 策略（规则、分类、变更历史）
- 节点管理
- 网络配置（接口、网桥、bond、路由）

### Phase 4：辅助业务 API
- 负载均衡（VIP、Pool、Health Monitor）
- ACL 策略
- HA 集群
- 系统设置、升级、License
- 报表、自动伸缩、应用识别
- 日志查询（攻击/操作/流量/防病毒/防篡改）

### Phase 5：节点 Agent
- gRPC proto 定义（server ↔ agent 通信协议）
- 心跳上报
- 系统资源监控
- 网络配置执行
- 硬件 bypass 控制
- 固件升级执行
- 防篡改、日志采集、磁盘监控

### Phase 6：联调与测试
- 集成测试
- 与 waf-admin 前端联调
- 性能测试与优化

## 6. 数据库 Schema 规划

### 替代 etcd 的表
```sql
users, roles, user_roles        -- 身份认证
tokens                          -- JWT 黑名单/刷新 token
nodes, node_interfaces          -- 节点信息
acl_policies                    -- ACL 规则
system_settings                 -- KV 系统配置
heartbeats                      -- 心跳记录
```

### 原 MySQL 业务表（直接迁移）
```sql
asgs, sites, protect_assoc
policies, policy_rules, policy_categories, policy_change_history
lb_vips, lb_pools, lb_members, lb_health_monitors
asg_pools, asg_autoscaling
operation_logs
```

### 替代 Elasticsearch 的表
```sql
attack_logs       -- GIN 索引 + JSONB，支持全文搜索
flow_logs         -- 分区表按天，BRIN 索引
antivirus_logs
antitamper_logs
```
