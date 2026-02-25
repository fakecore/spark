# DoorX 简化设计方案

> **日期**: 2026-02-26
> **目标**: 将 DoorX 从复杂的 AI 工具平台简化为标准 Go 后端脚手架
> **方案**: 渐进式精简（方案 A）

---

## 概述

将 DoorX 简化为标准的 Go 后端脚手架，保留 Kratos 框架、PostgreSQL、Redis 和基础系统模块，移除 Change DAG、NATS JetStream、AI 业务逻辑等复杂特性。

**简化后的定位**：
- 标准 RBAC 后端模板
- 适合快速启动新项目
- 代码清晰，易于扩展

---

## 简化后的架构

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (Optional)                  │
│                    HTTP/gRPC Client                     │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Go Service (Kratos v2)                     │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Interface Layer                                 │   │
│  │ - HTTP Handler / gRPC Service                   │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Service Layer                                   │   │
│  │ - User/Role/Menu/Permission (system)            │   │
│  │ - Basic CRUD operations                         │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Biz Layer                                       │   │
│  │ - Use case logic                                │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Data Layer                                      │   │
│  │ - GORM models + Repository                      │   │
│  │ - Redis cache (optional)                        │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Infrastructure                                  │   │
│  │ - PostgreSQL / Logging / Config / Health        │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

**架构变化**：
- ❌ 移除：Sync Service、Change DAG、NATS JetStream、WebSocket
- ✅ 保留：Kratos 框架、标准三层架构、系统模块

---

## 删除清单

### 需要删除的目录/文件

| 目录/文件 | 原因 |
|-----------|------|
| `internal/biz/action/` | AI Action 业务逻辑 |
| `internal/biz/execution/` | 执行节点相关 |
| `internal/biz/changedag/` | Change DAG 核心逻辑 |
| `internal/workflow/` | 工作流、GC、HubLease 等 |
| `internal/plugin/` | 插件系统 |
| `internal/service/action/` | Action 服务 |
| `internal/service/sync/` | 同步服务 |
| `internal/infrastructure/` | NATS、EdgeWindow 等基础设施 |
| `internal/server/ws_endpoints.go` | WebSocket 端点 |
| `internal/server/hub_lease_endpoints.go` | Hub 租约端点 |
| `internal/server/infra_sync_endpoints.go` | 同步基础设施端点 |
| `internal/server/infra_exec_endpoints.go` | 执行节点端点 |
| `internal/testutil/natstest/` | NATS 测试工具 |
| `pkg/utils/provider/nats.go` | NATS provider |
| `api/proto/plugin/` | 插件相关 proto |
| `api/action/v1/` | Action proto |
| `api/sync/v1/` | 同步 proto |

### Proto 生成代码清理

```bash
# 删除后重新生成
rm -f api/action/v1/*.pb.go
rm -f api/sync/v1/*.pb.go
rm -f api/proto/plugin/v1/*.pb.go
```

### 需要保留的核心

| 目录 | 用途 |
|------|------|
| `internal/service/system/` | 系统模块 |
| `internal/biz/biz.go` | Biz 层入口 |
| `internal/data/` | 数据层（需精简） |
| `internal/server/` | HTTP/gRPC 服务端（需精简） |
| `internal/middleware/` | 中间件 |
| `pkg/` | 通用工具包 |
| `api/system/v1/` | 系统 API |
| `api/common/v1/` | 通用 API |

---

## 数据层精简

### `internal/data/` 调整

**删除的 Repository**：
- `internal/data/repository/changedag/` - 整个目录删除
- `internal/data/repository/execution/` - 整个目录删除

**保留的 Repository**：
- `internal/data/repository/system/` - 系统模块（用户、角色、菜单等）
- `internal/data/db_query.go` - GORM 查询入口
- `internal/data/data.go` - 数据层入口（需移除 NATS 相关代码）

### `internal/data/data.go` 修改

```go
// 需要移除的代码
- NATS 连接初始化
- ChangeStore、ObjectState 等相关依赖注入

// 保留
- PostgreSQL (Data)
- Redis (可选)
- Transaction 管理
```

### 数据库迁移精简

**保留的表**（系统相关）：
- `sys_user` - 用户
- `sys_role` - 角色
- `sys_menu` - 菜单
- `sys_role_dept` - 角色部门关联
- `sys_user_post` - 用户岗位关联
- `sys_user_online` - 在线用户
- `sys_user_o_auth` - OAuth 用户
- `sys_config` - 系统配置
- `sys_dict_type` / `sys_dict_value` - 字典
- `sys_post` - 岗位
- `sys_dept` - 部门
- `sys_file` - 文件
- `sys_message_*` - 消息相关
- `sys_login_log` / `sys_oper_log` - 日志
- `casbin_rule` - 权限规则

**删除的表**：
- `changes` - 变更 DAG
- `object_heads` - 对象头
- `object_state` - 对象状态
- `snapshots` - 快照
- `outbox_events` - 事件箱

---

## 配置精简

### `internal/conf/conf.proto` 修改

**需要移除的字段**：

```protobuf
// 移除 NATS 相关
message NATS {
  string url = 1;
  bool jetstream_enabled = 2;
  string jetstream_domain = 3;
}

// 移除同步相关
message Sync {
  bool enabled = 1;
  string hub_url = 2;
}

// 从 Data 中移除
message Data {
  Database database = 1;
  Redis redis = 2;
  NATS nats = 3;        // 删除
  Sync sync = 4;         // 删除
}
```

**精简后的配置结构**：

```protobuf
message Data {
  Database database = 1;
  Redis redis = 2;
}

message Database {
  string driver = 1;
  string source = 2;
}

message Redis {
  Network network = 1;
  string addr = 2;
  string password = 3;
  int32 db = 4;
}
```

### `docker/backend/config/config.yaml` 精简

```yaml
server:
  http:
    addr: 0.0.0.0:9988
    timeout: 30s
  grpc:
    addr: 0.0.0.0:9989
    timeout: 30s

data:
  database:
    driver: postgres
    source: host=127.0.0.1 user=projecttemplate password=projecttemplate dbname=projecttemplate port=5432 sslmode=disable TimeZone=Asia/Shanghai
  redis:
    addr: 127.0.0.1:6379
    db: 0
```

---

## 服务端入口精简

### `internal/server/` 端点注册

**需要移除的端点注册**：
- `wsServer` - WebSocket 服务
- `hubLeaseServer` - Hub 租约服务
- `infraSyncServer` - 同步基础设施服务
- `infraExecServer` - 执行节点服务

**保留的端点**：
- `systemHTTPServer` - 系统模块 HTTP
- `systemGRPCServer` - 系统模块 gRPC
- `metricsServer` - 指标服务（可选）

### `cmd/server/main.go` 精简

```go
// 需要移除的初始化
- NATS JetStream 初始化
- ChangeStore 初始化
- Sync Service 初始化
- WebSocket 初始化
- Plugin System 初始化

// 保留
- HTTP Server
- gRPC Server
- PostgreSQL
- Redis
- Logger
- Config
```

---

## Makefile 和依赖清理

### `Makefile` 精简

**需要移除的 target**：
- `nats-setup` - NATS 设置
- `nats-streams-create` - NATS Stream 创建
- `test-sync` - 同步测试
- `test-nats` - NATS 测试

**保留的核心 target**：
- `make build` - 构建
- `make run` - 运行
- `make migrate-up/down` - 数据库迁移
- `make test` - 测试
- `make proto` - Proto 生成
- `make dev-up/down` - 开发环境

### `docker/docker-compose.yml` 精简

```yaml
services:
  postgres:
    image: postgres:16-alpine

  redis:
    image: redis:7-alpine

  # 删除 nats 服务
```

### `go.mod` 依赖清理

```bash
go mod tidy  # 自动清理未使用的依赖
```

手动检查以下依赖是否仍被使用：
- `github.com/nats-io/nats.go`
- `github.com/nats-io/nkeys`

---

## 编译修复步骤

按以下顺序执行，确保每步编译通过：

1. **删除业务代码目录**
2. **删除 proto 定义并重新生成** (`make proto`)
3. **修改 data.go** 移除 NATS/ChangeStore
4. **修改 conf.proto** 并重新生成
5. **修改 server/http.go** 和 **grpc.go**
6. **修改 cmd/server/main.go**
7. **go mod tidy** 清理依赖
8. **make build** 验证编译

---

## 验收标准

- [ ] `make dev-up` 成功启动 Postgres + Redis
- [ ] `go run cmd/server/main.go` 启动不报错
- [ ] `make build` 编译成功
- [ ] `/healthz` 返回 200
- [ ] 系统 API（用户、角色、菜单）可正常调用
- [ ] 单元测试通过
