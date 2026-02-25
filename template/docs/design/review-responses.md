# DoorX 架构 Review - 回复与决策记录

> **类型**: 决策记录
> **状态**: 🚧 进行中（5 项待确认）
> **最后更新**: 2026-02-04
> **关联文档**: [architecture.md](./architecture.md)

本文档记录架构评审的回复和最终决策。

### 决策状态汇总

| 状态 | 数量 | 说明 |
|------|------|------|
| ✅ 已决策 | 19 | 可执行 |
| 🔄 待确认 | 5 | 需要讨论 |

**待确认的核心问题**：
1. **2.2** ServiceInstance 多服务模型
2. **3.1** Application/Domain 分层架构
3. **4** 安全模型详细设计
4. **5** 数据同步方案（应用层/DB 层/混合）
5. **9.4** API 版本策略

---

## ✅ 已决策事项

### 1. 技术选型

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **1.1 数据库选型** | ✅ PostgreSQL 统一 | 本地 daemon 服务，用户自行安装（裸机/Docker），保持一致性 |
| **1.2 前端技术栈** | ✅ Zustand + React + shadcn/ui | 确定 |
| **1.3 后端框架** | ✅ 保持 Kratos | 团队已有成熟使用经验 |

### 2. 数据模型

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **2.1 图片存储** | ✅ 文件系统 + DB 存路径 | 单独路径存储 |
| **2.3 软删除** | ✅ 添加 | 所有核心实体添加 `DeletedAt`/`ArchivedAt` |

### 3. 架构设计

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **3.2 多窗口同步** | ✅ WebSocket | 成熟方案 |
| **3.3 Electron+Go 集成** | ✅ 前端独立 + 后台服务独立 | 后台服务开机启动 (daemon) |

### 4. 安全与权限

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **4.2 IPC 安全** | ✅ 互信凭证 | 部署环境视为安全，初始化时生成互信凭证 |
| **4.3 隐私保护** | ✅ 需要安全模型 | 待设计 |

### 5. 同步与一致性

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **5.2 冲突解决** | ✅ 无冲突时自动合并 | 有冲突时提示 |
| **5.3 Router 与 Sync** | ✅ 数据库同步机制 | 属于微服务范畴 |

### 6. Electron 与 Go 集成

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **6.1 进程管理** | ✅ daemon 自动拉起 | OS 级服务管理 |
| **6.2 端口分配** | ✅ 固定端口 | 建议固定端口 |
| **6.3 升级机制** | ✅ v1 兼容 | 自动升级，保持兼容 |

### 7. 执行与路由

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **7.1 Execution Node 输入** | ✅ 通过数据库提取 | 节点从云端 DB 拉取任务 |
| **7.2 节点健康** | ✅ 心跳 + 服务注册 | 标准机制 |
| **7.3 Action 不可用** | ✅ Fallback 提示 | 可自动或用户确认 |

### 8. 插件系统

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **8.1 插件机制** | ✅ gRPC | 采用 gRPC-based plugins |
| **8.2 Tool Registry** | 🔄 待提供方案 | 需要我提供详细设计 |

### 9. MVP 与开发计划

| 问题 | 决策 | 备注 |
|-----|------|-----|
| **9.1 时间线** | ✅ 可调整 | 预估时间可变更 |
| **9.2 Phase 7 风险** | ✅ 实验性质 | 游戏对话总结为实验性功能 |
| **9.4 API 版本** | 🔄 待决策 | alpha/beta vs 直接 v1 |

---

## 🔍 需要深入讨论的问题

### 2.2 多服务模型（替代多租户）

**你的思路**: 不做多租户，只考虑用户本身多服务使用。组织内也可能存在多个服务。

**我的理解**: 你倾向于一种"多实例"模型，而非传统 SaaS 的多租户。

**需要澄清的问题**:

```
场景 A: 个人用户
┌─────────────────────────────────────┐
│  User A                             │
│  ├─ Device 1 (Desktop + Daemon)     │
│  ├─ Device 2 (Laptop + Daemon)      │
│  └─ Cloud Service (可选)            │
│                                     │
│  数据同步: 云端为主，多设备同步      │
└─────────────────────────────────────┘

场景 B: 团队/组织
┌─────────────────────────────────────┐
│  Organization X                     │
│  ├─ Member A (Device + Daemon)      │
│  ├─ Member B (Device + Daemon)      │
│  ├─ Shared Cloud Service            │
│  └─ Execution Nodes (GPU 服务器)    │
│                                     │
│  问题:                              │
│  1. 成员间如何共享 Action/Prompt?   │
│  2. 执行节点选择范围?               │
│  3. 数据隔离级别?                   │
└─────────────────────────────────────┘
```

**建议方案**:

采用"**服务实例**"（Service Instance）作为核心隔离单元：

```go
// 用户账户
type User struct {
    ID       string
    Email    string
    Name     string
}

// 服务实例 - 数据隔离的基本单元
type ServiceInstance struct {
    ID          string
    Name        string
    Type        InstanceType // personal | team
    OwnerID     string       // 创建者
    
    // 成员（如果是团队类型）
    Members     []InstanceMember
    
    // 关联的资源
    Resources   InstanceResources
}

type InstanceType string
const (
    InstanceTypePersonal InstanceType = "personal"
    InstanceTypeTeam     InstanceType = "team"
)

type InstanceMember struct {
    UserID string
    Role   MemberRole // owner | admin | member
}

type InstanceResources struct {
    // 该实例关联的所有服务节点
    Daemons     []DaemonInfo     // 本地 daemon 列表
    CloudNodes  []CloudNodeInfo  // 云端节点
    ExecNodes   []ExecNodeInfo   // 执行节点
}
```

**关键设计**:
- 一个用户可以创建多个 ServiceInstance（个人 + 多个团队）
- 登录时选择进入哪个 Instance
- 所有数据（Conversation、Action、Prompt）归属到 Instance
- 同步在 Instance 内部进行

**待确认**:
1. 是否采用 ServiceInstance 模型？
2. 团队成员权限粒度？（只读？可执行？可修改？）
3. 个人 Instance 和 Team Instance 的功能差异？

---

### 3.1 Application 与 Domain 层边界

**你的回复**: 我也没想明白，可以深入聊一下

**问题本质**: 在分层架构中，业务逻辑应该放在哪里？

**当前分层**:
```
Interface Layer (API Handler)
    ↓
Application Layer (Services: Chat/Action/Prompt/Tool/Window)
    ↓
Domain Layer (Core: Action Engine/LLM Engine/Tool Registry)
    ↓
Infrastructure Layer (DB/Event Log)
```

**我的建议**:

采用**领域驱动设计 (DDD) Lite** 方案：

```go
// ============================================
// Domain Layer - 核心业务规则（无外部依赖）
// ============================================

// 只包含业务逻辑，不涉及存储、网络
type ActionExecutor struct {
    // 纯内存的执行逻辑
}

func (e *ActionExecutor) Execute(ctx context.Context, action Action, input map[string]any) (Result, error) {
    // 1. 验证输入（业务规则）
    if err := e.validateInput(action, input); err != nil {
        return Result{}, err
    }
    
    // 2. 执行（可能是纯计算，或调用外部服务）
    // 但 Domain 层不直接调用 HTTP/DB，通过接口抽象
    return e.doExecute(ctx, action, input)
}

// Domain 只定义接口，不实现
type ToolInvoker interface {
    Invoke(ctx context.Context, toolName string, args map[string]any) (any, error)
}

type LLMClient interface {
    Chat(ctx context.Context, messages []Message) (Response, error)
}

// ============================================
// Application Layer - 业务流程编排
// ============================================

type ActionService struct {
    // 依赖注入
    actionRepo    ActionRepository  // 基础设施接口
    executor      *ActionExecutor   // Domain 层核心
    toolInvoker   ToolInvoker       // 基础设施实现
    llmClient     LLMClient         // 基础设施实现
    eventBus      EventBus          // 基础设施
}

func (s *ActionService) ExecuteAction(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
    // Application 层负责：
    
    // 1. 获取数据（协调基础设施）
    action, err := s.actionRepo.Get(ctx, req.ActionID)
    if err != nil {
        return nil, err
    }
    
    // 2. 权限检查（调用 Auth 服务）
    if err := s.checkPermission(ctx, action); err != nil {
        return nil, err
    }
    
    // 3. 调用 Domain 层执行业务逻辑
    result, err := s.executor.Execute(ctx, action, req.Input)
    if err != nil {
        return nil, err
    }
    
    // 4. 保存结果（协调基础设施）
    if err := s.saveExecutionLog(ctx, action, result); err != nil {
        // 日志但不阻断
    }
    
    // 5. 发送事件（协调基础设施）
    s.eventBus.Publish(ctx, ActionExecutedEvent{...})
    
    return result, nil
}

// ============================================
// 关键区别
// ============================================

// Domain: "是什么" - 业务规则、计算逻辑
// - Action 如何执行
// - Prompt 如何渲染变量
// - Chain 如何串联步骤

// Application: "怎么做" - 流程编排、协调
// - 先查权限，再执行，再存日志
// - 先截图，再调用 AI，再显示结果
// - 失败时如何重试/降级
```

**另一个视角 - 按复杂度分层**:

如果团队对 DDD 不熟悉，也可以采用更简单的分层：

```
Handler Layer (HTTP Handler)
    ↓ 调用
Service Layer (业务逻辑)
    ↓ 调用
Store Layer (数据访问)
```

**Kratos 项目结构参考**:

```
internal/
├── biz/          # Domain Layer (Kratos 叫 biz)
│   ├── action.go
│   └── action_executor.go
├── service/      # Application Layer
│   ├── action_service.go
│   └── action_grpc.go
├── data/         # Infrastructure Layer
│   ├── action_repo.go
│   └── tool_invoker.go
└── server/       # Interface Layer
    ├── http.go
    └── grpc.go
```

**待决策**:
1. 采用 DDD Lite 分层，还是简单三层架构？
2. 是否遵循 Kratos 默认的 biz/service/data 分层？

---

### 4. 安全模型设计

**你的回复**: 定义一套安全模型，敏感操作权限有哪些，action声明，系统级操作需要组件自行组织，沙箱执行

**我的方案建议**:

```go
// ============================================
// 1. 权限定义
// ============================================

type Permission string
const (
    // 系统级权限
    PermClipboardRead     Permission = "system:clipboard:read"
    PermClipboardWrite    Permission = "system:clipboard:write"
    PermScreenshot        Permission = "system:screenshot"
    PermGlobalHotkey      Permission = "system:hotkey"
    PermFileRead          Permission = "system:file:read"
    PermFileWrite         Permission = "system:file:write"
    PermProcessExec       Permission = "system:process:exec"
    PermNetwork           Permission = "system:network"
    PermAudioCapture      Permission = "system:audio:capture"
    PermScreenCapture     Permission = "system:screen:capture"
    
    // 数据级权限
    PermActionCreate      Permission = "data:action:create"
    PermActionDelete      Permission = "data:action:delete"
    PermActionShare       Permission = "data:action:share"
    PermConvDelete        Permission = "data:conversation:delete"
)

// 风险等级
type RiskLevel string
const (
    RiskLow      RiskLevel = "low"      // 纯计算，无外部影响
    RiskMedium   RiskLevel = "medium"   // 读取系统信息
    RiskHigh     RiskLevel = "high"     // 修改系统状态（剪贴板、文件）
    RiskCritical RiskLevel = "critical" // 执行代码、网络访问
)

// ============================================
// 2. Action/Tool 权限声明
// ============================================

type Action struct {
    ID          string
    Name        string
    
    // 权限声明
    RequiredPermissions []Permission
    RiskLevel           RiskLevel
    
    // 执行约束
    Constraints *ExecutionConstraints
}

type ExecutionConstraints struct {
    MaxExecutionTime time.Duration
    MaxMemoryMB      int64
    AllowedPaths     []string  // 文件访问白名单
    BlockedPaths     []string  // 文件访问黑名单
    AllowNetwork     bool
    AllowedHosts     []string  // 网络白名单
}

// ============================================
// 3. 用户授权存储
// ============================================

type UserPermissionGrant struct {
    UserID      string
    Permission  Permission
    GrantedAt   int64
    GrantedBy   string  // user | system
    ExpiresAt   *int64  // 可选过期
    Scope       GrantScope // global | instance | action
}

type GrantScope struct {
    Type   string // global | instance | action
    ID     string // instance_id or action_id
}

// ============================================
// 4. 运行时权限检查
// ============================================

type PermissionChecker struct {
    grantRepo GrantRepository
}

func (c *PermissionChecker) Check(ctx context.Context, userID string, perm Permission, scope GrantScope) error {
    // 1. 检查是否已授权
    grant, err := c.grantRepo.Get(ctx, userID, perm, scope)
    if err != nil {
        return ErrPermissionDenied
    }
    
    // 2. 检查是否过期
    if grant.ExpiresAt != nil && *grant.ExpiresAt < time.Now().Unix() {
        return ErrPermissionExpired
    }
    
    return nil
}

// ============================================
// 5. 首次使用授权流程
// ============================================

func (s *ActionService) ExecuteAction(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
    action := s.getAction(req.ActionID)
    
    // 检查所有必需权限
    for _, perm := range action.RequiredPermissions {
        if err := s.permChecker.Check(ctx, req.UserID, perm, GrantScope{Type: "action", ID: action.ID}); err != nil {
            if err == ErrPermissionDenied {
                // 返回需要授权的错误，前端展示授权对话框
                return nil, &PermissionRequiredError{
                    Permission: perm,
                    ActionName: action.Name,
                    RiskLevel:  action.RiskLevel,
                }
            }
            return nil, err
        }
    }
    
    // 执行...
}
```

**UI 授权流程**:

```
┌─────────────────────────────────────────┐
│  "截图分析" Action 需要以下权限:          │
│                                         │
│  ⚠️ 风险等级: 中等                       │
│                                         │
│  [✓] system:screenshot                  │
│      └─ 捕获屏幕内容                     │
│                                         │
│  [✓] system:network                     │
│      └─ 上传图片到 AI 服务               │
│                                         │
│  [  ] system:clipboard:read             │
│      └─ 读取剪贴板内容（可选）           │
│                                         │
│  □ 记住我的选择（不再询问）              │
│                                         │
│  [  允许  ]  [  拒绝  ]                 │
└─────────────────────────────────────────┘
```

**审计日志**:

```go
type AuditLog struct {
    ID          string
    Timestamp   int64
    
    // 谁
    UserID      string
    InstanceID  string
    DeviceID    string
    
    // 做了什么
    Action      string      // action:execute | permission:grant
    Resource    ResourceRef // action | conversation | tool
    
    // 执行详情
    Input       any         // 脱敏后
    Output      any         // 脱敏后
    Error       string
    
    // 权限相关
    PermissionsUsed []Permission
    RiskLevel       RiskLevel
    
    // 位置信息
    Target      string      // local | cloud | cloud@node_id
    IP          string
}
```

**待确认**:
1. 权限粒度是否合适？
2. 授权是否区分"全局授权"和"单次授权"？
3. 审计日志保留策略？

---

### 5. 数据库同步方案

**你的回复**: 想借助 db 的同步协议，属于微服务范畴

**分析**:

数据库层同步（如 PostgreSQL Logical Replication）确实可以简化应用层设计，但需要考虑：

```
方案对比:

┌─────────────────────────────────────────────────────────────────┐
│ 方案 A: 应用层同步（原设计）                                     │
│ ├─ 优点: 灵活，可以处理业务逻辑冲突                              │
│ ├─ 缺点: 复杂，需要实现版本号、事件队列                           │
│ └─ 适用: 复杂冲突处理、部分字段同步                               │
├─────────────────────────────────────────────────────────────────┤
│ 方案 B: 数据库同步 (Logical Replication)                         │
│ ├─ 优点: 简单，DB 自动处理                                       │
│ ├─ 缺点: 全表同步，无法处理业务冲突                               │
│ └─ 适用: 数据一致性优先、简单场景                                 │
├─────────────────────────────────────────────────────────────────┤
│ 方案 C: 混合方案（推荐）                                         │
│ ├─ 核心数据: 应用层同步（Conversation、Action 定义）              │
│ ├─ 大文件: 对象存储同步                                          │
│ └─ 配置/日志: 数据库同步                                          │
└─────────────────────────────────────────────────────────────────┘
```

**我的建议 - 混合方案**:

```go
// 1. 核心业务数据 - 应用层同步（需要冲突处理）
// Conversation, Action, Prompt 定义
// 使用 SyncEvent + 版本号

type SyncEvent struct {
    ID         string
    EntityType string      // conversation | action | prompt
    EntityID   string
    Operation  string      // create | update | delete
    Version    int64       // 单调递增
    Payload    []byte
    Timestamp  int64
}

// 2. 大文件/产物 - 对象存储同步
// 图片、视频、音频产物
// 使用 S3-compatible API

// 3. 配置/审计日志 - 数据库同步或无需同步
// AuditLog - 只写不同步（云端聚合）
// UserSettings - 应用层同步或数据库同步
```

**如果坚持数据库同步**:

需要明确：
1. 使用 PostgreSQL Logical Replication 还是第三方工具？
2. 冲突时以谁为准？（云端主还是最后写入者？）
3. 离线时的处理方式？（本地写，上线后同步？）
4. 同步延迟的容忍度？

**待确认**:
1. 采用哪种同步方案？
2. 如果使用 DB 同步，具体技术选型？

---

### 8.2 Tool Registry 方案

**我的建议**:

```go
// ============================================
// 1. Tool 定义
// ============================================

type Tool struct {
    // 基础信息
    Metadata ToolMetadata
    
    // 接口契约
    Interface ToolInterface
    
    // 实现信息
    Implementation ToolImplementation
    
    // 权限与安全
    Security ToolSecurity
    
    // 生命周期
    Lifecycle ToolLifecycle
}

type ToolMetadata struct {
    ID          string
    Name        string
    Description string
    Category    string      // "builtin" | "plugin" | "community"
    Author      string
    Version     semver.Version
    Tags        []string
    Icon        string
    
    // 归属
    InstanceID  string      // 归属的 ServiceInstance
    Visibility  Visibility  // private | instance | public
}

type ToolInterface struct {
    // 输入 Schema (JSON Schema)
    InputSchema map[string]any
    
    // 输出 Schema (JSON Schema)
    OutputSchema map[string]any
    
    // 示例
    Examples []ToolExample
}

type ToolExample struct {
    Name        string
    Description string
    Input       map[string]any
    Output      any
}

type ToolImplementation struct {
    Type ImplementationType // "builtin" | "grpc" | "wasm" | "http"
    
    // Type=builtin: 内置代码
    // Type=grpc: gRPC 服务地址
    // Type=wasm: WASM 模块字节码
    // Type=http: HTTP 端点
    Config map[string]any
}

type ToolSecurity struct {
    RequiredPermissions []Permission
    RiskLevel           RiskLevel
    Constraints         ExecutionConstraints
    SandboxConfig       *SandboxConfig
}

type ToolLifecycle struct {
    State       ToolState   // draft | active | deprecated | disabled
    CreatedAt   int64
    UpdatedAt   int64
    PublishedAt *int64
    
    // 版本管理
    Replaces    []string    // 替换的旧版本 Tool ID
    ReplacedBy  *string     // 被哪个新版本替换
}

type ToolState string
const (
    ToolStateDraft      ToolState = "draft"
    ToolStateActive     ToolState = "active"
    ToolStateDeprecated ToolState = "deprecated"
    ToolStateDisabled   ToolState = "disabled"
)

// ============================================
// 2. Tool Registry 接口
// ============================================

type ToolRegistry interface {
    // CRUD
    Register(ctx context.Context, tool *Tool) error
    Get(ctx context.Context, id string) (*Tool, error)
    Update(ctx context.Context, tool *Tool) error
    Delete(ctx context.Context, id string) error
    
    // 查询
    List(ctx context.Context, filter ToolFilter) ([]*Tool, error)
    Search(ctx context.Context, query string) ([]*Tool, error)
    
    // 发现
    Discover(ctx context.Context, capability string) ([]*Tool, error)
    
    // 执行
    Invoke(ctx context.Context, toolID string, input map[string]any) (any, error)
}

// ============================================
// 3. 版本管理
// ============================================

// 语义化版本控制
type VersionPolicy struct {
    Current     semver.Version
    MinCompatible semver.Version  // 最小兼容版本
    
    // 兼容性声明
    BreakingChanges []string       // 破坏性变更说明
    DeprecationNotes []string      // 废弃功能说明
}

// 版本解析策略
func ResolveToolVersion(registry ToolRegistry, name string, constraint string) (*Tool, error) {
    // 解析约束: 
    // "latest" -> 最新稳定版
    // ">=1.0.0 <2.0.0" -> 兼容 1.x
    // "^1.2.0" -> 兼容 1.2.0 及以上，但 <2.0.0
    // "~1.2.0" -> 兼容 1.2.x
}

// ============================================
// 4. 插件发现机制
// ============================================

// 本地扫描
type LocalScanner struct {
    PluginDirs []string
}

func (s *LocalScanner) Scan(ctx context.Context) ([]*Tool, error) {
    // 扫描 ~/.projecttemplate/plugins/ 目录
    // 读取 plugin.json 或 projecttemplate-plugin.yaml
    // 验证并注册
}

// 远程市场
type MarketplaceClient struct {
    Endpoint string
}

func (c *MarketplaceClient) Search(ctx context.Context, query string) ([]*MarketplaceItem, error) {
    // 搜索远程市场
    // 返回可安装的插件列表
}

func (c *MarketplaceClient) Install(ctx context.Context, toolID string) error {
    // 下载插件
    // 验证签名
    // 安装到本地
}

// ============================================
// 5. 插件生命周期
// ============================================

```
状态流转:

   ┌─────────┐    开发完成    ┌─────────┐
   │  Draft  │ ────────────▶ │ Active  │
   └─────────┘               └────┬────┘
                                  │
                    新版本发布     │ 发现问题
                    (Breaking)     │
                         │         │
                         ▼         ▼
                   ┌─────────┐  ┌─────────┐
                   │Deprecated│  │Disabled │
                   └────┬────┘  └─────────┘
                        │
                        │ 一段时间后
                        ▼
                   ┌─────────┐
                   │ Removed │
                   └─────────┘
```

**待确认**:
1. 是否接受此方案？
2. 插件市场的优先级？（V1 还是后续版本）

---

### 9.4 API 版本策略

**选项对比**:

| 方案 | 优点 | 缺点 |
|-----|------|-----|
| **Alpha/Beta** | 明确标注不稳定，允许快速迭代 | 用户感知为"未完成" |
| **直接 v1** | 简洁，用户感知稳定 | 一旦发布难以修改，前期容错空间小 |
| **日期版本** | 清晰表达发布时间 | 不表达兼容性 |

**我的建议**:

采用 **"稳定版本 + 预览版本"** 策略：

```
API 端点设计:

/v1/...              # 稳定版本，向后兼容保证
/v1beta/...          # 预览版本，可能变更
/v1alpha/...         # 实验版本，随时可能删除

版本升级策略:
- v1 发布后，保持 6-12 个月向后兼容
- 新功能先在 beta 测试，稳定后进入 v1
- 破坏性变更仅在 major 版本升级时进行
```

**MVP 建议**:
- 对外使用 `/api/v1`（简洁，表达信心）
- 内部保留扩展空间（响应中包含 `api_version: "1.0"`）
- 文档中注明 "v1 API 在正式版发布前可能调整"

**待确认**: 采用哪种方案？

---

## 📋 待办清单

### 需要我提供的文档

- [ ] Tool Registry 详细设计文档
- [ ] 安全模型详细设计文档
- [ ] ServiceInstance 模型详细设计（确认 2.2 方案后）

### 需要你确认的问题

- [ ] **2.2** 是否采用 ServiceInstance 多服务模型？
- [ ] **3.1** Application/Domain 分层采用 DDD Lite 还是简单三层？
- [ ] **4** 安全模型方案是否接受？
- [ ] **5** 数据库同步采用应用层、DB 层还是混合方案？
- [ ] **9.4** API 版本采用 Alpha/Beta 还是直接 v1？

---

*文档版本: 2026-02-04*
