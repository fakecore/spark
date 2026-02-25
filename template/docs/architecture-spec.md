# DoorX Architecture Specification

> 本文档定义了 DoorX 项目各层的职责、依赖方向和禁止事项。
> 所有代码贡献必须遵守本规范，Code Review 时以此为依据。

## 1. 分层架构总览

```
┌───────────────────────────────────────────────────────────┐
│  API Layer (proto定义)                                     │
│  api/**/*.proto                                           │
│  职责: 定义 HTTP/gRPC 接口契约、请求响应结构、校验规则         │
├───────────────────────────────────────────────────────────┤
│  Service Layer (薄适配层)                                   │
│  internal/service/                                        │
│  职责: DTO 转换、请求参数校验、编排调用 usecase               │
├───────────────────────────────────────────────────────────┤
│  Business Layer (业务核心)                                  │
│  internal/biz/                                            │
│  职责: 业务规则、领域逻辑、验证、状态转换、接口定义            │
├───────────────────────────────────────────────────────────┤
│  Data Layer (数据持久化)                                    │
│  internal/data/                                           │
│  职责: 实现 biz 层定义的 Repo 接口，执行 CRUD               │
├───────────────────────────────────────────────────────────┤
│  Infrastructure Layer (基础设施)                            │
│  internal/infrastructure/                                 │
│  职责: 数据库连接、缓存、消息队列、Casbin 初始化              │
└───────────────────────────────────────────────────────────┘
```

## 2. 依赖方向规则

```
唯一合法方向（自上而下 + 依赖倒置）:

  service  →  biz  ←(interface)  data  →  infrastructure
     ↓          ↓                  ↓           ↓
   api/v1    pkg/*              dal/model    gorm/redis/nats
```

### 允许的 import 关系

| 源 (from)              | 目标 (to)                           | 是否允许 |
|------------------------|-------------------------------------|---------|
| service                | biz (usecase 结构体)                 | YES     |
| service                | api/v1 (proto 生成的类型)             | YES     |
| service                | data/do (DTO 对象)                   | YES     |
| service                | data/dal/model (数据模型)             | YES     |
| service                | pkg/* (工具函数)                      | YES     |
| biz                    | data/dal/model (数据模型)             | YES     |
| biz                    | data/do (DTO 对象)                   | YES     |
| biz                    | pkg/* (工具函数)                      | YES     |
| data/repository        | biz (Repo interface)                 | YES     |
| data/repository        | data/dal/model, data/dal/query       | YES     |
| data/repository        | infrastructure                       | YES     |
| data/repository        | pkg/* (工具函数)                      | YES     |

### 禁止的 import 关系

| 源 (from)              | 目标 (to)                           | 原因                      |
|------------------------|-------------------------------------|--------------------------|
| service                | service (另一个 service)             | Service 之间不得互相调用    |
| service                | infrastructure                       | 跨层访问，应通过 FX 注入接口 |
| biz                    | data/repository (具体实现)            | 违反依赖倒置               |
| biz                    | service                              | 反向依赖                  |
| biz                    | infrastructure                       | 跨层访问                  |
| data/repository        | service                              | 反向依赖                  |
| data/repository        | 另一个 data/repository               | Repo 之间不得互相调用       |
| infrastructure         | biz / service / data                 | 反向依赖                  |

## 3. 公共包规范 (`pkg/`)

`pkg/` 存放可跨层复用的通用工具，不包含业务逻辑。所有层均可 import `pkg/*`。

| 子包 | 职责 | 示例 |
|------|------|------|
| `pkg/clog` | 日志适配 | 上下文日志、zap wrapper |
| `pkg/kvstore` | KV 存储抽象 | Redis + 内存 fallback |
| `pkg/utils` | 纯工具函数 | 密码哈希、分页、DTO 转换 |
| `pkg/middleware` | HTTP 中间件 | Auth、Tracing、Logging |
| `pkg/errors` | 错误类型 | 通用错误定义 |
| `pkg/config` | 配置管理 | 环境变量、配置加载 |
| `pkg/trace` | 分布式追踪 | OpenTelemetry |

**准入规则**: 放入 `pkg/` 的代码必须满足：
- 不依赖 `internal/` 下的任何包
- 不包含业务逻辑（纯技术关注点）
- 可被多个模块复用

## 4. 类型策略 (`model` vs `do`)

项目使用两种数据类型贯穿各层：

| 类型 | 包路径 | 用途 | 特点 |
|------|--------|------|------|
| `model` | `internal/data/dal/model` | ORM 映射，数据库表结构 | 由 GORM Gen 生成，字段与数据库列 1:1 对应 |
| `do` | `internal/data/do` | Data Object / DTO | 手动定义，字段可选（指针类型），用于部分更新和聚合查询结果 |

**各层使用规则**:
- **biz 层**: 定义 Repo interface 时，入参/返回值使用 `model` 或 `do`（当前共享类型策略）
- **service 层**: 负责 proto ↔ model/do 的转换
- **data 层**: 直接操作 `model`，返回 `model` 或 `do`

> 注：当前 biz 和 data 共享 `model`/`do` 类型，这是务实的选择。如果未来领域模型和数据库结构差异增大，可考虑在 biz 层定义独立的 domain 类型。

## 5. 错误处理规范

| 层 | 错误类型 | 说明 |
|----|---------|------|
| data/repository | `fmt.Errorf("failed to xxx: %w", err)` | 包装底层错误，提供上下文 |
| biz | `commonv1.ErrorXxx()` / `v1.ErrorXxx()` | 创建领域错误，面向调用方 |
| service | 透传 biz 层错误或做最终格式化 | 不创建新的领域错误 |

**规则**:
- Data 层只做错误包装（`%w`），不创建领域错误
- 业务含义的错误（"用户名已存在"、"权限不足"）只在 biz 层创建
- Service 层一般透传，仅在需要统一格式时做转换

## 6. 各层职责详细规定

### 6.1 Service Layer (`internal/service/`)

**职责**:
- 接收 proto request，转换为 biz 层需要的入参
- 调用一个或多个 usecase 方法
- 将 biz 层返回值转换为 proto response
- 请求级别的参数校验（格式、必填）

**禁止事项**:
- 不得包含业务逻辑（if/else 业务判断）
- 不得直接操作数据库或缓存
- 不得直接 import `infrastructure` 包
- 不得调用其他 service

**关于多 usecase 依赖**:
- Service 可以依赖多个 usecase（例如编排场景）
- 但如果编排逻辑本身是业务规则，应该下沉到一个专门的 usecase

### 6.2 Business Layer (`internal/biz/`)

**职责**:
- 定义 Repo interface（依赖倒置的关键）
- 实现业务规则、业务验证
- 状态转换和领域逻辑
- 跨 Repo 的数据聚合和编排
- 事务边界的控制（通过 context 传递 tx）

**禁止事项**:
- 不得 import data 层的具体实现
- 不得 import infrastructure 包
- 不得直接操作 *gorm.DB 或 *redis.Client

**Usecase 设计原则**:
- 每个 usecase 可以依赖多个 Repo interface
- Usecase 的写操作方法不能只是简单的 `return uc.repo.Xxx()`（透传），必须有业务价值
- 如果写操作 usecase 方法只是透传，说明业务逻辑可能下沉到了 repo 层
- 纯读取的简单查询方法（如 `GetByID`、`List`）允许透传，这是合理的

### 6.3 Data Layer (`internal/data/repository/`)

**职责**:
- 实现 biz 层定义的 Repo interface
- 执行数据库 CRUD 操作
- SQL 查询构建（使用 GORM Gen）
- 数据格式转换（model ↔ do）

**禁止事项**:
- 不得包含业务逻辑（验证、状态判断、权限检查）
- 不得调用其他 Repository
- 不得直接操作 Casbin Enforcer（鉴权是业务逻辑）
- 不得管理事务（事务由 biz 层通过 context 控制）
- 方法命名不应暗示业务语义（用 `Create` 而非 `Register`）

**Repository 方法设计原则**:
- 每个方法只操作自己对应的主表（及其直接关联表的简单 JOIN）
- 方法应该是原子性的单表操作
- 组合操作（如 "创建用户 + 分配角色"）应由 biz 层编排

### 6.4 Infrastructure Layer (`internal/infrastructure/`)

**职责**:
- 数据库连接管理
- 缓存连接管理
- 消息队列连接管理
- Casbin Enforcer 初始化
- fx.Lifecycle 管理

**禁止事项**:
- 不得 import biz / service / data 包
- 不得包含业务逻辑

## 7. 接口定义规范

### Repo Interface 定义位置

Repo interface 必须定义在 `internal/biz/` 层对应的文件中：

```go
// internal/biz/system/user.go

type SystemUserRepo interface {
    // 简单 CRUD - 每个方法对应一个原子操作
    FindByID(ctx context.Context, id int64) (*model.SysUser, error)
    Create(ctx context.Context, user *model.SysUser) (*model.SysUser, error)
    Update(ctx context.Context, user *do.SysUser) (*do.SysUser, error)
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context, params ListUserParams) ([]*model.SysUser, int32, error)
}
```

### Interface 编译期检查

每个 repository 实现文件必须包含接口一致性断言：

```go
var _ system.SystemUserRepo = (*systemUserRepo)(nil)
```

## 8. Casbin / 权限操作规范

### 权限操作的正确位置

| 操作类型            | 正确位置                    | 错误位置               |
|--------------------|-----------------------------|----------------------|
| Casbin Enforce     | biz 层 (PermissionUsecase)   | data/repository      |
| Casbin LoadPolicy  | biz 层 (事务提交后)          | data/repository      |
| CasbinRule CRUD    | data 层 (CasbinRuleRepo)    | 散落在各个 repo 中     |
| 角色权限判断         | biz 层 (PermissionUsecase)   | data/repository      |

### 推荐方案

抽取独立的 `CasbinRuleRepo`，将所有 casbin_rule 表操作集中管理：

```go
// internal/biz/system/casbin.go

type CasbinRuleRepo interface {
    // 用户-角色关系 (g 规则)
    AssignUserRole(ctx context.Context, userID, roleID int64) error
    RevokeUserRole(ctx context.Context, userID int64) error
    ListUserRoleIDs(ctx context.Context, userID int64) ([]int64, error)

    // 角色-菜单权限 (p 规则)
    SetRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error
    ListRoleMenuIDs(ctx context.Context, roleID int64) ([]int64, error)
}
```

## 9. 事务管理规范

### 事务的控制方应在 biz 层

```go
// 正确：biz 层控制事务
func (uc *UserUsecase) CreateWithRoles(ctx context.Context, ...) error {
    return uc.txManager.Transaction(ctx, func(txCtx context.Context) error {
        user, err := uc.userRepo.Create(txCtx, user)
        if err != nil { return err }

        return uc.casbinRepo.AssignUserRoles(txCtx, user.ID, roleIDs)
    })
}

// 错误：data 层自己开事务做复合操作
func (r *userRepo) CreateWithRoles(ctx context.Context, ...) error {
    tx := r.db.Begin()  // 不应该在这里
    ...
}
```

## 10. FX 模块注册规范

每个层的 Module 只注册自己层的依赖：

```go
// internal/data/repository/repo.go
var Module = fx.Options(
    fx.Provide(
        system.NewSystemUserRepo,
        system.NewSystemRoleRepo,
        // ...
    ),
)

// internal/biz/biz.go
var Module = fx.Options(
    fx.Provide(
        system.NewSystemUserUsecase,
        system.NewSystemRoleUsecase,
        // ...
    ),
)
```

## 11. 代码审查检查清单

每次 PR 必须检查以下项目：

- [ ] import 方向是否符合分层规则
- [ ] Repository 方法是否为原子操作（不跨表、不跨 repo）
- [ ] 业务逻辑（验证、权限、状态转换）是否在 biz 层
- [ ] Service 层是否足够薄（只做 DTO 转换和 usecase 调用）
- [ ] Casbin 操作是否集中管理
- [ ] 事务是否由 biz 层控制
- [ ] 新 Provider 是否注册到对应的 fx.Module
- [ ] Usecase 方法是否有实际业务价值（非纯透传）
- [ ] 接口一致性断言 `var _ Interface = (*impl)(nil)` 是否存在
