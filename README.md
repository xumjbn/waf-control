# waf-control

OpenWAF / NebulaWAF 控制面 —— Go 1.22 + chi v5 + pgx v5 + PostgreSQL 16 + JWT。

负责：

- 给 **waf-admin** 提供 REST API（站点 / 策略 / 日志 / 告警 / 报表 / 系统 / 升级 / 监控 / 防护实例 / 身份）
- 给 **waf-agent** 提供 gRPC 流（register / heartbeat / config push / deploy result）+ REST 上报入口（攻击日志 / 命中计数 / 节点指标）

```
                 REST          REST/gRPC
   waf-admin ───────▶ waf-control ◀────── waf-agent (× N)
                          │
                          ▼
                      PostgreSQL
```

## 技术栈

- Go 1.22（vendored 依赖）
- chi/v5 路由 + 自写 middleware（auth / logging / recovery）
- pgx/v5 + 内置 migrations（`internal/store/migrations/`，up/down 配对）
- golang-jwt/v5（JWT，HMAC）
- swaggo（OpenAPI 文档生成）
- grpc-go（与 waf-agent 通信，proto 在 `proto/agent/agent.proto`）

## 目录结构

```
cmd/
├── waf-control/     # 主入口：HTTP + gRPC server
└── seedpw/          # 离线工具：生成 bcrypt 密码 UPDATE 语句

internal/
├── config/          # 环境变量装载
├── server/          # chi 路由组装 + middleware 链
├── middleware/      # auth / logging / recovery
├── identity/        # 用户/角色/项目 + JWT 签发（Postgres + InMemory 双 store）
├── domain/          # 按业务域切分
│   ├── identity/    # 用户登录/查询自身/列表
│   ├── site/        # 站点 CRUD + metrics 上报入口
│   ├── instancemgmt/# 防护实例集群（NW · 06）
│   ├── policy/      # 策略规则 + 命中计数（NW · 04）
│   ├── logs/        # 攻击日志 + 封禁IP / 加白 / 关联事件（NW · 05）
│   ├── alert/       # 告警渠道 6 类（邮件/微信/钉钉/PagerDuty/Webhook/SMS）
│   ├── reports/     # 报表统一列表 + 执行追踪
│   ├── system/      # 系统设置 + license + 升级
│   ├── monitor/     # KPI 看板聚合（attack_logs + alert_events + metrics）
│   └── aggregation/ # 跨域聚合占位
├── deploy/          # 配置下发逻辑
├── store/migrations/# 数据库迁移（000001 ~ 000016）
└── httputil/        # 通用 JSON 响应

proto/agent/         # gRPC 协议
```

## 快速开始

```bash
# 1. 起 Postgres（任意方式）
docker run -d --name pg -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16

# 2. 跑迁移
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/wafcontrol?sslmode=disable"
make migrate-up

# 3. 启动 server
make run-server
# 或 go run ./cmd/waf-control

# 默认监听：HTTP 9200 / gRPC 50051
```

环境变量在 `internal/config/config.go`，主要：

| 变量 | 含义 | 默认 |
| --- | --- | --- |
| `HTTP_ADDR` | REST 监听地址 | `:9200` |
| `GRPC_ADDR` | gRPC 监听地址 | `:50051` |
| `DATABASE_URL` | Postgres DSN | — |
| `JWT_SECRET` | HMAC 密钥 | — |

## 数据库迁移

- 迁移文件：`internal/store/migrations/`，命名 `NNNNNN_<desc>.{up,down}.sql`，必须成对。
- **每个 domain 还有自己的 `EnsureSchema`**（启动时跑 `ALTER TABLE … ADD COLUMN IF NOT EXISTS`），作为迁移落地不及时的兜底；正规生产环境仍以 migration 为准。
- 加字段流程：写 migration → 跑 `make migrate-up` 验证 → 同步 EnsureSchema → 改 repo/handler。

## 常用命令

```bash
make build              # 编译到 ./bin/
make test               # 跑单测
make test-integration   # tests/ 下的集成测试
make lint               # go vet
make generate           # 重新生成 proto 代码
make migrate-up         # 应用迁移
make migrate-down       # 回滚一步
```

## 接口约定

- 所有业务接口前缀 `/api/v1/...`
- 鉴权：`Authorization: Bearer <JWT>`（登录端点 `POST /api/v1/auth/login` 除外）
- 错误格式：`{ "code": "...", "message": "..." }`，由 `internal/httputil` 统一封装
- 时间字段一律 RFC3339

## 仓库归属

`waf-control` 是 OpenWAF monorepo（[xumjbn/OpenWAF](https://github.com/xumjbn/OpenWAF)）下的 git 子模块。
与 `waf-admin`（前端）、`waf-agent`（节点 sidecar）三件套配套部署。
