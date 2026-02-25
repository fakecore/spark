# DoorX Device Mesh — 详细架构设计

> **版本**: v2.2
> **日期**: 2026-02-25
> **状态**: 设计确认，待实施
> **前置文档**: `docs/design/device-mesh-architecture.md`（初始草案）
> **v2.1 变更**: 安全模型升级（HMAC→Ed25519、Hub CA + mTLS、证书生命周期管理）、租约缓存与续期、Relay session 优化、能力指标与 Broker 加权、Action 可靠性提示
> **v2.2 变更**: 新增主控 Agent 架构（Brain Agent + 子 Agent 派发）、agent.exec 能力类型、沙箱与权限边界模型、用户审批流、审计防篡改机制

---

## 1. 设计原则

1. **复用优先** — 在现有 ExecutionNode、Plugin Gateway、Change DAG 基础上扩展，不另起炉灶
2. **gRPC 统一** — 发现走 Hub gRPC，数据面走 P2P gRPC（mTLS），可降级到 Hub Relay
3. **硬件即 Action** — 硬件能力抽象为 Action，Agent/Plugin 通过现有 Invoke 接口调用
4. **全链路审计** — 每个操作产生审计事件，本地持久化 + 异步上报 Hub
5. **UUID 全追踪** — 每个请求携带 request_id + trace_id，贯穿全链路
6. **安全分级** — 默认面向可信局域网（trusted_lan），可升级到公网模式（public_internet），安全基础设施（CA、mTLS、Ed25519）从第一天开始内置
7. **主控大脑** — 一个全知全能的主控 Agent（Brain）与用户对话，可将任务下发到任意 Edge 节点直接执行或唤醒子 Agent 执行，所有行为受沙箱约束、策略管控、审计留痕，不可自行篡改

---

## 2. 整体架构

```
                          ┌─────────────┐
                          │    用户      │
                          │   (Dylan)   │
                          └──────┬──────┘
                            对话 / 授权
                                 │
                          ┌──────▼──────┐
                          │ Brain Agent │  ← 全知全能主控大脑
                          │  全部 mesh  │
                          │  actions    │
                          │  可见+可调  │
                          └──────┬──────┘
                                 │ 直接调用 / 派发子 Agent
                     ┌───────────┴───────────┐
                     ▼                       ▼
              ┌──────────────────────────┐
              │        Cloud Hub          │
              │    (现有 Sync Hub 扩展)    │
              │                           │
              │  MeshCoordinatorService   │  ← 新增 gRPC 服务
              │  ├── CapabilityRegistry   │  ← 能力注册表
              │  ├── CapabilityBroker     │  ← 能力匹配
              │  ├── LeaseAuthority       │  ← 租约签发
              │  ├── DeviceEnrollment     │  ← 设备接入管理
              │  ├── ActionPublisher      │  ← 能力→Action 自动发布
              │  └── AuditCollector       │  ← 审计汇总
              └─────┬───────────┬─────────┘
               gRPC │           │  gRPC
           ┌────────┘           └────────┐
           ▼                             ▼
    ┌──────────────┐   gRPC P2P    ┌──────────────┐
    │   Edge A     │   (mTLS)      │   Edge B     │
    │   (macOS)    │ ◄───────────► │   (iPhone)   │
    │              │               │              │
    │ mesh.Node    │  P2P → Relay  │ mesh.Node    │
    │ ├ CapHost    │  降级         │ ├ CapHost    │
    │ ├ Sandbox    │               │ ├ Sandbox    │
    │ ├ Providers  │               │ ├ Providers  │
    │ ├ AgentExec  │  ← 子Agent   │ ├ AgentExec  │
    │ ├ PolicyEng  │               │ ├ PolicyEng  │
    │ └ AuditLog   │               │ └ AuditLog   │
    │              │               │              │
    │ Caps:        │               │ Caps:        │
    │  llm.gpu     │               │  audio.input │
    │  agent.exec  │               │  agent.exec  │
    │  display     │               │  audio.output│
    └──────────────┘               └──────────────┘
```

---

## 3. 节点与 Workspace 模型

### 3.1 节点统一为 Edge

所有设备都是 Edge 节点。macOS 主力机、iPhone、Homelab GPU 服务器——不在类型上区分，用 capability 声明区分能做什么，用 policy 配置区分愿意做什么。

现有 `ExecutionNode` 保持不变，通过 `node_id` 关联新增的 mesh 表。

### 3.2 设备接入 Workspace

```
设备接入流程:

  1. 用户在主设备（macOS）登录 DoorX
     → 已有 workspace 身份 (user_id + workspace_id)

  2. 用户想把新设备（iPhone）接入同一 workspace
     → 主设备调用 Hub API 生成邀请码（短时效 token，含 workspace_id + user_id）
     → 新设备扫码或输入邀请码
     → Hub 验证邀请码，为新设备创建 enrollment 记录
     → Hub 下发 workspace 凭证（sync token + mesh 签名密钥）
     → 新设备的 node_id 绑定到 {workspace_id, user_id}

  3. 新设备上线后
     → 向 Hub 注册能力（附带 workspace 凭证）
     → Hub 记录能力到 mesh_capabilities 表
     → 同 workspace 的所有用户可发现这些能力
```

### 3.3 Workspace 内能力可见性与共享

```
workspace "team-alpha" 内的能力视图:

  Dylan 的设备:
    macOS   → [llm.gpu, display, audio.output]
    iPhone  → [audio.input, audio.output, video.input]

  Alice 的设备:
    MacBook → [llm.gpu, display]
    Homelab → [llm.gpu(4090), storage]

  可见性: 同 workspace 内所有能力互相可见
  可调用: 取决于 policy
    - same_owner（Dylan 调自己的 iPhone）→ 策略更宽松
    - same_workspace（Dylan 调 Alice 的 Homelab）→ 需要更严格的授权
```

---

## 4. 硬件能力即 Action

### 4.1 核心思路

硬件能力注册到 Hub 后，Hub 自动生成对应的 Action 定义并注册到 Action Catalog。Agent/Plugin 通过现有的 `Plugin Gateway → Invoke` 路径调用，与调用普通插件完全一致。

```
设备注册能力 → Hub 自动发布 Action → Agent 可发现并调用

  iPhone 注册 audio.input
      │
      ▼
  Hub CapabilityRegistry 记录
      │
      ▼
  Hub ActionPublisher 自动生成 Action 定义:
    {
      id: <uuid>,
      name: "mesh.audio.capture",
      type: "mesh_capability",
      description: "从设备采集音频",
      input_schema: {
        format: "opus|pcm_s16le",
        sample_rate: 16000,
        duration_ms: 5000,       // 可选，0=持续
        ...
      },
      output_schema: {
        session_id: "string",    // 长时会话返回 session_id
        data: "bytes",           // 短时调用返回数据
        ...
      },
      provider: "mesh",
      metadata: {
        capability_type: "audio.input",
        // 不绑定具体 node，运行时由 Broker 动态选择最优节点
      },
    }
      │
      ▼
  Action Catalog 注册
      │
      ▼
  Agent tool_use 可调用 "mesh.audio.capture"
```

### 4.2 调用模式

| 模式 | 适用场景 | 行为 |
|------|---------|------|
| **短时调用** | "录 5 秒音频并转写"、"用 GPU 推理一次" | Action 阻塞等待，返回结果 |
| **长时会话** | "持续监听麦克风"、"开始视频流" | Action 返回 session_id，后续通过 Session API 读写流 |

### 4.3 Agent 视角

```go
// Agent 的 tool 列表——调用方式完全一致，但 metadata 携带可靠性提示
tools := []Tool{
    {Name: "search_web", ...},           // 普通插件
    {Name: "mesh.audio.capture", ...},   // → 远程麦克风
    {Name: "mesh.gpu.inference", ...},   // → 远程 GPU
    {Name: "mesh.video.capture", ...},   // → 远程摄像头
}

// 调用方式完全一致
result := agent.Invoke("mesh.audio.capture", map[string]any{
    "format": "opus",
    "sample_rate": 16000,
    "duration_ms": 5000,
})

// Action metadata 中携带可靠性与调度提示（Agent 框架层消费，Agent 本身无需感知）
// mesh Action 的 metadata 示例:
//   provider: "mesh"
//   reliability: "best_effort"       // 区别于普通插件的 "guaranteed"
//   retry_strategy: "reacquire"      // 失败时需重新获取租约，而非简单重试
//   timeout_hint_ms: 10000           // 建议超时（含网络 + 设备响应）
//   failover: true                   // 支持自动故障转移到同类型其他节点
```

### 4.4 与现有 Plugin Gateway 的集成

扩展 `core.Capability` 和 `gateway.isSupportedCapability`，新增 mesh 相关能力类型：

```go
// 新增 capability 类型
CapabilityMeshInvoke    Capability = "mesh.invoke"       // 调用远程设备能力
CapabilityAudioCapture  Capability = "audio.capture"     // 使用远程麦克风
CapabilityAudioPlayback Capability = "audio.playback"    // 使用远程扬声器
CapabilityGPUCompute    Capability = "gpu.compute"       // 使用远程 GPU
CapabilityVideoCapture  Capability = "video.capture"     // 使用远程摄像头
CapabilityAgentExec     Capability = "agent.exec"        // 在远程节点执行子 Agent
```

mesh.Node 注册为一个 `CapabilityInvoker` 实现，Plugin Gateway 遇到 `mesh.*` 类型时路由给 mesh.Node 处理。

---

## 5. 主控 Agent 架构（Brain Agent）

### 5.1 核心模型

主控 Agent（Brain）是用户的统一交互入口——一个全知全能的大脑，与用户对话，代用户完成所有事情。Brain 可以：

1. **直接调用**远程设备硬件能力（已有 mesh.* actions）
2. **派发子 Agent**到本地或远程 Edge 节点执行复杂任务
3. **编排多步任务**，跨多个节点协调执行

```
用户: "帮我把 iPhone 录的会议音频转写成文字，然后整理成会议纪要存到 NAS"

Brain Agent 规划:
  Step 1: mesh.audio.capture  → iPhone (录音)
  Step 2: mesh.gpu.inference  → Homelab GPU (语音转写)
  Step 3: mesh.agent.exec     → macOS (子 Agent: 整理文本 + 写入 NAS)

Brain 看到的 tool 列表:
  ┌──────────────────────────────────────────────────────────────┐
  │ search_web           │ 普通插件                              │
  │ mesh.audio.capture   │ → 远程麦克风（哪个节点由 Broker 决定）│
  │ mesh.gpu.inference   │ → 远程 GPU                           │
  │ mesh.video.capture   │ → 远程摄像头                          │
  │ mesh.agent.exec      │ → 在指定节点启动子 Agent              │
  │ mesh.fs.read         │ → 远程文件读取                        │
  │ mesh.fs.write        │ → 远程文件写入                        │
  └──────────────────────────────────────────────────────────────┘

所有 tool 调用方式一致，Brain 无需关心底层是硬件还是子 Agent。
```

### 5.2 子 Agent 执行（agent.exec）

子 Agent 是一种特殊的能力类型——"在目标节点上启动一个受约束的 Agent 实例"。

```go
// 子 Agent 调用——Brain 视角
result := agent.Invoke("mesh.agent.exec", map[string]any{
    // 任务描述
    "task":        "整理桌面文件并归档到 NAS",
    "context":     "用户的桌面有大量截图和文档，按日期归档",

    // 沙箱约束（见 5.3）
    "sandbox": map[string]any{
        "type":            "filesystem",
        "allowed_actions": []string{"fs.read", "fs.write:/Users/dylan/Desktop/**", "nas.upload"},
        "denied_actions":  []string{"fs.write:/etc/**", "process.exec"},
        "max_duration_sec": 300,
    },

    // 目标节点（可选，不指定则由 Broker 选择）
    "target_node": "macOS-main",
})

// 子 Agent 返回
// {
//   "session_id": "uuid",         // 长时任务返回 session_id
//   "status": "completed",        // "running"|"completed"|"failed"|"approval_pending"
//   "result": { ... },
//   "audit_trail": ["uuid1", "uuid2", ...],  // 审计事件 ID 列表
// }
```

#### 子 Agent 生命周期

```
Brain 调用 mesh.agent.exec
    │
    ▼
mesh.Node.Invoke() → 获取租约 → transport.Dial → 目标节点
    │
    ▼
目标节点 CapabilityHost
    │
    ├── 1. 验证租约 + mTLS 身份
    ├── 2. 解析 SandboxPolicy（从请求 params 中）
    ├── 3. 创建沙箱环境（文件系统隔离、网络限制等）
    ├── 4. 启动子 Agent 实例
    │      ├── 子 Agent 只能调用 SandboxPolicy.allowed_actions 内的工具
    │      ├── 每个工具调用经过 sandbox.Enforcer 拦截
    │      └── 超出权限 → 触发审批流或直接拒绝
    ├── 5. 子 Agent 执行任务
    │      ├── 每个操作产生审计事件
    │      └── 超时自动终止
    └── 6. 返回结果 + 审计链
```

#### 子 Agent 与 Brain 的关系

```
  Brain（主控）                    子 Agent（被派发）
  ─────────────                   ──────────────────
  全局可见所有能力                  只能看到 sandbox 允许的能力
  可以派发任意子 Agent              不能派发子 Agent（除非显式授权）
  持有用户会话上下文                只持有任务上下文（Brain 传入的 task + context）
  决策权在 Brain                   执行权在子 Agent
  可以主动终止子 Agent              不能主动终止 Brain 或其他子 Agent
  审计记录标记 role=brain          审计记录标记 role=sub_agent, parent=<brain_id>
```

### 5.3 沙箱与权限边界

所有通过 mesh 执行的操作（无论是 Brain 直接调用还是子 Agent 执行）都受沙箱策略约束。

#### 沙箱策略模型

```go
// SandboxPolicy — 定义一次调用/会话的权限边界
type SandboxPolicy struct {
    // 沙箱类型
    // "none"       — 无隔离（仅审计，适用于可信场景）
    // "capability" — 能力级隔离（只限制可调用的 action 列表）
    // "filesystem" — 文件系统隔离（chroot / 路径白名单）
    // "full"       — 完全隔离（独立进程空间、网络隔离、文件隔离）
    Type string

    // 允许的操作（白名单，支持 glob 模式）
    AllowedActions []string   // ["fs.read", "fs.write:/tmp/**", "net.http:*.internal"]

    // 禁止的操作（黑名单，优先于白名单）
    DeniedActions []string    // ["fs.write:/etc/**", "process.exec:rm"]

    // 资源限制
    MaxMemoryMB    int        // 内存上限
    MaxCPUPercent  int        // CPU 占用上限
    MaxDurationSec int        // 最大执行时间

    // 网络限制
    NetworkPolicy  string     // "none"|"local_only"|"allowlist"
    AllowedHosts   []string   // NetworkPolicy=allowlist 时生效

    // 子 Agent 是否可以再派发子 Agent（默认 false）
    AllowSubAgentSpawn bool

    // 审批策略
    ApprovalPolicy ApprovalPolicy
}
```

#### 审批策略

```go
// ApprovalPolicy — 当操作超出沙箱边界时的处理方式
type ApprovalPolicy struct {
    // 审批模式
    // "never"          — 越权直接拒绝，不询问用户
    // "on_escalation"  — 仅敏感操作需要用户确认
    // "always"         — 每个操作都需要用户确认（最严格）
    Mode string

    // 敏感操作列表（Mode="on_escalation" 时生效）
    // 匹配到的操作需要用户确认，其他操作按 AllowedActions 执行
    SensitivePatterns []string  // ["fs.write:/etc/**", "process.exec:*", "net.http:*.external"]

    // 审批超时（秒），超时视为拒绝
    TimeoutSec int

    // 审批通知渠道
    NotifyChannels []string    // ["push", "desktop", "email"]
}
```

#### 审批流程

```
子 Agent 请求执行 fs.write:/etc/hosts
    │
    ▼
sandbox.Enforcer 检查
    │
    ├── AllowedActions 匹配? → 否
    ├── DeniedActions 匹配? → 是 (fs.write:/etc/**)
    │
    ▼
ApprovalPolicy.Mode = "on_escalation"
    │
    ├── SensitivePatterns 匹配 fs.write:/etc/** → 需要审批
    │
    ▼
approval.Manager 发起审批
    │ 1. AUDIT: approval.requested {action: "fs.write:/etc/hosts", agent: sub_agent_id}
    │ 2. 推送通知到用户设备（Brain 所在节点）
    │ 3. 用户界面展示:
    │    "子 Agent [整理文件] 请求写入 /etc/hosts，是否允许？"
    │    [允许本次] [允许后续同类] [拒绝]
    │
    ▼
用户响应
    │
    ├── 允许本次 → 执行，AUDIT: approval.granted {scope: "once"}
    ├── 允许后续同类 → 临时将该模式加入 AllowedActions，AUDIT: approval.granted {scope: "pattern"}
    └── 拒绝 → 返回 PermissionDenied，AUDIT: approval.denied {reason: "user_rejected"}

超时（默认 60s）→ 视为拒绝，AUDIT: approval.denied {reason: "timeout"}
```

### 5.4 权限不可篡改原则

Agent（无论 Brain 还是子 Agent）**不能修改自身的安全约束**：

```
不可篡改的边界:

  1. SandboxPolicy
     → 由 Brain 在 mesh.agent.exec 调用时设定
     → 子 Agent 进程内无法修改（Enforcer 在 CapabilityHost 层，非 Agent 进程空间）
     → Brain 的 SandboxPolicy 由系统配置 / 用户 Workspace 设置决定

  2. AuditLog
     → Agent 进程无数据库直接写权限
     → 审计写入通过 CapabilityHost 内的 audit.Logger（独立于 Agent 进程）
     → Agent 只能触发审计记录的产生（通过执行操作），不能修改/删除已有记录

  3. PolicyEngine
     → 策略配置存储在 Hub 或节点本地配置文件
     → Agent 没有修改策略配置的 Action（系统层面不暴露该能力）
     → 策略变更只能通过管理员 API（需独立身份验证，不经过 Agent 通道）

  4. LeaseToken
     → Hub 签发，Ed25519 签名
     → Agent 无法伪造或延长（不持有 Hub 私钥）

  用户显式授权变更:
     → 用户可以通过管理界面调整 Agent 的默认 SandboxPolicy
     → 用户可以在审批流中临时扩展权限
     → 所有权限变更本身也产生审计记录（meta-audit）
```

---

## 6. 通信架构

### 6.1 双链路：P2P 优先，Hub Relay 降级

```
优先级:
  1. Edge A ──gRPC 直连──► Edge B     (最低延迟)
  2. Edge A ──► Hub Relay ──► Edge B   (P2P 失败时自动降级)

降级触发条件:
  - P2P 连接超时（可配置，默认 3s）
  - P2P 连接被 RST/refused
  - 目标节点在 Hub 注册时未提供可达 endpoint

降级对上层透明:
  - transport 层自动切换
  - 审计日志记录实际路径 (transport: "direct" | "relay")
```

### 6.2 Hub Relay 实现

Hub 新增 `RelayStream` RPC：

```protobuf
// Hub 端
service MeshCoordinatorService {
  // ... 其他 RPC ...

  // 双向流中继：两端 Edge 各自与 Hub 建立 stream，Hub 透传
  rpc RelayStream(stream RelayFrame) returns (stream RelayFrame);
}

message RelayFrame {
  string session_token = 1;    // Hub 分配的 relay session token（首帧认证后获得）
  string target_node_id = 2;   // 目标节点（仅首帧需要）
  string lease_token = 3;      // 租约 token（仅首帧携带，用于认证）
  bytes payload = 4;           // 透传数据
  map<string, string> metadata = 5;
}

// Relay 认证流程:
// 1. 首帧: 携带 lease_token + target_node_id，Hub 验证后返回 session_token
// 2. 后续帧: 仅携带 session_token（轻量），Hub 在内存中维护 session → 路由映射
// 3. session_token 与 lease 生命周期绑定，lease 释放时自动失效
```

### 6.3 Edge P2P gRPC 服务

每个 Edge 节点启动一个 Mesh gRPC server（独立端口，或复用现有 gRPC server）：

```protobuf
service MeshPeerService {
  // 能力调用（request-response）
  rpc Invoke(PeerInvokeRequest) returns (PeerInvokeReply);

  // 流式传输（双向流）
  rpc DataStream(stream DataChunk) returns (stream DataChunk);

  // 健康探测
  rpc PeerHealth(PeerHealthRequest) returns (PeerHealthReply);
}

message PeerInvokeRequest {
  RequestMeta meta = 1;
  string lease_token = 2;
  string capability_type = 3;
  string command = 4;           // "create_session" | "control" | "close"
  map<string, string> params = 5;
  bytes payload = 6;
}

message DataChunk {
  string session_id = 1;
  int64 sequence = 2;
  int64 timestamp_ms = 3;
  bytes data = 4;
  map<string, string> metadata = 5;
}
```

---

## 7. 安全模型

### 7.1 信任链

```
                    Hub（信任锚 + 内置 CA）
                   /            \
     签发节点证书 /              \ 签发节点证书
      + 公钥分发 /                \ + 公钥分发
              /                    \
        Edge A                   Edge B
        (调用方)                 (提供方)
              \                  /
               \ mTLS + Lease  /
                \  (Hub签发)  /
                 ──────────►
```

Hub 同时承担两个角色：
- **CA 角色**：为每个 Edge 签发 TLS 客户端/服务端证书，用于 P2P mTLS
- **Lease Authority**：签发 LeaseToken，用 Ed25519 私钥签名，Edge 用 Hub 公钥验证

### 7.2 三层安全

#### 第一层：调用方权限（谁有资格发起调用）

```
Action/Agent 发起调用
    │
    ▼
Plugin Gateway
    │ 1. token 验证（现有 token.Service）
    │ 2. capability 检查（action 是否有 mesh.invoke 权限）
    │ 3. policy 评估（现有 policy.Engine）
    ▼
mesh.Node.Acquire()
    │ 检查本地租约缓存（命中且未过期 → 跳过 Hub 请求）
    │
    ▼ (缓存未命中)
Hub.RequestLease()
    │ 4. workspace 权限验证
    │ 5. 调用方节点可信度检查
    │ 6. Mesh Policy 评估（by_capability / by_relationship）
    ▼
签发 LeaseToken 或拒绝
```

#### 第二层：Hub 签发租约（信任传递）

```go
// LeaseToken 结构
type LeaseToken struct {
    LeaseID        string    // UUID
    CapabilityID   string    // 目标能力 ID
    CapabilityType string    // "audio.input" 等
    ProviderNodeID string    // 提供方节点
    LesseeNodeID   string    // 调用方节点
    LesseeUserID   string    // 调用方用户
    WorkspaceID    string    // workspace
    Scopes         []string  // 允许的操作范围
    IssuedAt       int64     // 签发时间
    ExpiresAt      int64     // 过期时间
    Renewable      bool      // 是否允许续期
    MaxRenewals    int       // 最大续期次数
    Signature      []byte    // Ed25519 签名
}

// Hub 签发时:
// 1. 用 Hub Ed25519 私钥对 token 内容签名
// 2. 各 Edge 在注册时获得 Hub 公钥（只能验签，不能伪造）
// 3. Edge 本地验证签名，无需回调 Hub
//
// 租约缓存与续期:
// - 短时调用：单次租约，用完即释放
// - 长时/高频调用：签发可续期租约（Renewable=true）
//   Edge 在 ExpiresAt 前主动续期，Hub 无需重新评估完整策略
//   MaxRenewals 限制续期次数，超出后必须重新申请
// - 同 capability_type + 同 provider 的活跃租约可复用
```

#### 第三层：提供方验证（被调用方安全）

```
Edge B 收到 P2P 请求（已通过 mTLS 建立传输层信任）
    │
    ▼
1. mTLS 身份确认
    ├── 对端证书由 Hub CA 签发
    └── 证书中 node_id 与请求中 lessee_node_id 一致
    │
2. LeaseToken 签名验证（Ed25519 公钥验签）
    │
3. LeaseToken 字段验证
    ├── 过期检查
    ├── capability_type 匹配
    ├── lessee_node_id 匹配 mTLS 证书身份
    └── workspace_id 匹配本地 workspace
    │
4. 本地策略评估
    ├── 能力是否仍然可用（设备上下文检查）
    ├── 并发会话数限制
    └── 资源阈值检查
    │
5. 通过 → 创建 Session + 审计日志
   拒绝 → 返回原因 + RetryAfter + 审计日志
```

### 7.3 Hub 内置 CA 与证书管理

Hub 作为 Mesh 网络的 CA，为每个 Edge 签发 X.509 证书，用于 P2P gRPC mTLS。

```
证书签发流程:

  1. Hub 启动时生成 CA 根证书（自签名，Ed25519）
     - 根证书私钥仅 Hub 持有，安全存储
     - 根证书公钥通过 EnrollDevice 下发给所有 Edge

  2. Edge 注册时申请节点证书
     Edge → Hub: EnrollDevice(invite_code)
     Edge 本地生成密钥对 → 发送 CSR 给 Hub
     Hub 签发节点证书:
       Subject: CN=<node_id>, O=<workspace_id>
       SAN: DNS:<node_id>.mesh.projecttemplate.local
       有效期: 45 天（默认）
       扩展字段:
         x-projecttemplate-node-id: <node_id>
         x-projecttemplate-workspace-id: <workspace_id>
         x-projecttemplate-user-id: <user_id>
     Hub → Edge: {
       node_cert: <签发的节点证书>,
       ca_cert: <Hub CA 根证书>,
       hub_verify_key: <Hub Ed25519 公钥，用于验证 LeaseToken>,
     }

  3. 证书续期（自动）
     - Edge 在证书到期前 7 天通过 CapabilityHeartbeat 自动续期
     - Hub 验证节点仍为活跃状态后签发新证书
     - 新旧证书重叠期 48 小时，平滑过渡

  4. 证书吊销
     - 设备撤销时 Hub 将证书加入 CRL
     - CRL 通过 CapabilityHeartbeat 响应分发给所有 Edge
     - Edge 本地缓存 CRL，P2P 连接时检查对端证书
     - CRL 较小（仅含活跃 workspace 的吊销记录）

安全等级（按部署环境可配置）:

  trusted_lan（默认）:
    - mTLS 启用，但允许 Hub CA 自签名
    - CRL 检查为 soft-fail（获取失败不阻断连接）
    - LeaseToken 验签始终强制

  public_internet:
    - mTLS 强制，Hub CA 可对接外部 CA（如 Let's Encrypt）
    - CRL 检查为 hard-fail
    - LeaseToken 验签始终强制
    - 额外启用 IP allowlist / rate limiting
```

### 7.4 Ed25519 签名密钥管理

```
LeaseToken 验签密钥分发（与 CA 证书独立）:

  Hub 持有:
    - Ed25519 私钥（仅用于签发 LeaseToken）
    - 定期轮换（建议 90 天）

  Edge 持有:
    - Hub Ed25519 公钥（仅能验签，无法伪造 token）
    - 在 EnrollDevice 时首次获得
    - 通过 CapabilityHeartbeat 响应更新

  密钥轮换:
    - Hub 生成新密钥对 (key_version + 1)
    - 通过 CapabilityHeartbeat 响应下发新公钥
    - Edge 同时接受当前和上一版本公钥（平滑过渡）
    - 旧版本公钥在全部 Edge 确认更新后废弃
```

---

## 8. 全链路审计

### 8.1 审计事件模型

```protobuf
message MeshAuditEvent {
  string event_id = 1;           // UUID
  string request_id = 2;         // 关联的请求 ID
  string trace_id = 3;           // 分布式追踪 ID
  int64 timestamp = 4;

  string event_type = 5;         // 事件类型（见下表）
  string source_node_id = 6;     // 发起方节点
  string target_node_id = 7;     // 目标方节点

  string capability_type = 8;
  string capability_id = 9;
  string lease_id = 10;
  string session_id = 11;

  string outcome = 12;           // "granted"|"denied"|"error"|"timeout"
  string reason = 13;
  int32 status_code = 14;
  int64 duration_ms = 15;

  string workspace_id = 16;
  string user_id = 17;
  map<string, string> metadata = 18;
}
```

### 8.2 审计事件类型

| event_type | 触发时机 | 关键信息 |
|---|---|---|
| `device.enrolled` | 新设备接入 workspace | node_id, user_id, enrollment_method |
| `device.revoked` | 设备从 workspace 移除 | node_id, reason |
| `capability.registered` | Edge 向 Hub 注册能力 | node_id, capability_type, spec |
| `capability.unregistered` | 能力注销 | node_id, capability_type, reason |
| `capability.queried` | 查询能力 | 查询条件, 结果数量 |
| `capability.status_changed` | 能力状态变更 | old_status → new_status, reason |
| `lease.requested` | 请求租约 | 调用方, 目标能力 |
| `lease.granted` | 租约签发 | lease_id, 有效期 |
| `lease.denied` | 租约被拒 | 拒绝原因, 命中策略 |
| `lease.released` | 租约释放 | 正常/超时/撤销 |
| `lease.revoked` | 提供方主动撤销 | 撤销原因 |
| `stream.created` | 流建立 | transport(direct/relay), 协商参数 |
| `stream.closed` | 流关闭 | 持续时间, 传输量, 关闭原因 |
| `invoke.called` | 能力被调用 | 调用方, 能力类型, 参数摘要 |
| `invoke.completed` | 调用完成 | 耗时, 结果状态 |
| `policy.evaluated` | 策略评估 | 命中规则, 决策链 |
| `failover.triggered` | 故障转移 | 原节点 → 新节点, 原因 |
| `transport.fallback` | 通信降级 | direct → relay, 原因 |

### 8.3 审计存储

```
Edge 本地:
  → 写入 mesh_audit_events 表（保证不丢）
  → 日志文件（structured JSON，可用于实时监控）

异步上报:
  → 通过 NATS 发送到 Hub（复用现有 outbox 模式）
  → Hub 端写入汇总表
  → 支持聚合查询、告警、合规审计
```

### 8.4 审计防篡改

Agent（无论 Brain 还是子 Agent）不能修改已产生的审计记录。

```
防篡改策略:

  1. 进程隔离
     → Agent 进程无数据库直接写权限
     → 审计写入通过 CapabilityHost 内的 audit.Logger（独立 goroutine / 独立进程）
     → Agent 只能通过执行操作间接触发审计记录，不能修改/删除

  2. 存储层强制
     → 数据库: mesh_audit_events 表仅授予 INSERT 权限（无 UPDATE/DELETE）
     → 本地文件: append-only 模式写入 + 文件权限 0644（Agent 进程不可写）

  3. 双写交叉验证
     → 每条审计事件同时写入:
        a) 本地 mesh_audit_events 表
        b) 本地 append-only 日志文件
        c) 异步上报 Hub（NATS outbox）
     → Hub 端可交叉验证: Edge 上报记录 vs Hub 自身记录（如 lease 签发记录）
     → 不一致时触发告警: audit.integrity_violation

  4. 审计链完整性（public_internet 模式启用）
     → 每条审计事件包含 prev_hash 字段（前一条事件的 SHA-256 哈希）
     → 形成 hash chain，篡改中间任何一条会导致链断裂
     → Hub 定期校验各 Edge 上报的 hash chain 连续性
     → trusted_lan 模式下为可选（减少性能开销）

  5. 元审计（Meta-audit）
     → 权限变更（SandboxPolicy 调整、审批授权等）本身产生审计记录
     → 审计系统自身的配置变更也产生审计记录
     → 形成完整的"谁在什么时候改了什么权限"记录链
```

### 8.5 新增审计事件类型（Agent 相关）

| event_type | 触发时机 | 关键信息 |
|---|---|---|
| `agent.spawned` | Brain 派发子 Agent | parent_agent_id, sandbox_policy, target_node |
| `agent.completed` | 子 Agent 执行完成 | duration, result_status, actions_executed |
| `agent.terminated` | 子 Agent 被终止 | reason (timeout/user/brain/error) |
| `sandbox.violation` | 操作超出沙箱边界 | attempted_action, policy_rule, outcome |
| `approval.requested` | 越权操作请求用户审批 | action, agent_id, sandbox_context |
| `approval.granted` | 用户批准越权操作 | scope (once/pattern), user_id |
| `approval.denied` | 用户/超时拒绝越权操作 | reason (user_rejected/timeout) |
| `approval.escalated` | 权限升级生效 | old_scope → new_scope, ttl |
| `policy.modified` | 管理员修改策略配置 | old_policy → new_policy, admin_user_id |
| `audit.integrity_violation` | 审计链完整性校验失败 | node_id, expected_hash, actual_hash |

---

## 9. 策略引擎

### 9.1 策略评估优先级

当多层策略冲突时，按以下优先级（高到低）：

```
1. by_context（场景策略） — 如 gaming_mode deny 覆盖一切
2. by_capability + by_relationship 交集 — 更具体的规则优先
3. by_capability（能力类型策略）
4. by_relationship（设备关系策略）
5. default（全局默认）
```

### 9.2 Provider 端本地策略

Provider 节点可以独立于 Hub 做本地策略评估：

```go
type LocalPolicyEngine struct {
    // 资源条件检查器
    conditions []AvailabilityCondition

    // 并发限制
    maxSessions map[string]int

    // 用户上下文
    contextProvider UserContextProvider
}

// AvailabilityCondition — 可插拔的可用性检查
type AvailabilityCondition interface {
    Name() string
    Check(ctx context.Context) (available bool, reason string)
}

// 内置条件：GPU 资源检查、进程检测、电量检查等
// 基于资源指标（GPU 利用率、显存）而非进程名黑名单
```

---

## 10. 数据模型

### 10.1 新增表

```sql
-- 设备接入记录
CREATE TABLE mesh_device_enrollments (
    id UUID PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    device_name VARCHAR(128),
    enrollment_method VARCHAR(32) NOT NULL,  -- "invite_code"|"manual"|"auto"
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- "active"|"revoked"
    enrolled_at BIGINT NOT NULL,
    revoked_at BIGINT,
    metadata JSONB,

    UNIQUE(node_id, workspace_id),
    INDEX idx_mesh_enrollments_ws (workspace_id, status),
    INDEX idx_mesh_enrollments_user (user_id)
);

-- 能力注册表（独立于 ExecutionNode）
CREATE TABLE mesh_capabilities (
    id UUID PRIMARY KEY,
    node_id VARCHAR(64) NOT NULL,           -- → execution_nodes.node_id
    workspace_id VARCHAR(64) NOT NULL,
    owner_user_id VARCHAR(64) NOT NULL,     -- 设备所有者

    type VARCHAR(64) NOT NULL,              -- "audio.input"|"gpu.compute"|...
    spec JSONB NOT NULL,                    -- 能力规格
    status VARCHAR(32) NOT NULL DEFAULT 'available', -- "available"|"busy"|"degraded"|"offline"
    status_reason TEXT,

    -- 实时指标（Broker 加权选择用）
    metrics JSONB,                          -- {"utilization": 0.3, "latency_ms": 5, "battery_pct": 85}

    -- 访问端点（P2P 直连地址）
    endpoint JSONB,                         -- {"addr": "192.168.1.100:19400", "tls": true}

    -- 策略配置
    policy JSONB,

    -- 自动发布的 Action ID
    published_action_id UUID,

    -- 版本号（乐观锁）
    version INT NOT NULL DEFAULT 1,

    updated_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,

    INDEX idx_mesh_caps_node (node_id),
    INDEX idx_mesh_caps_ws_type (workspace_id, type, status),
    INDEX idx_mesh_caps_owner (owner_user_id)
);

-- 能力租约
CREATE TABLE mesh_leases (
    id UUID PRIMARY KEY,
    capability_id UUID NOT NULL,            -- → mesh_capabilities.id
    capability_type VARCHAR(64) NOT NULL,

    -- 提供方
    provider_node_id VARCHAR(64) NOT NULL,

    -- 调用方
    lessee_node_id VARCHAR(64) NOT NULL,
    lessee_user_id VARCHAR(64),

    -- 授权
    workspace_id VARCHAR(64) NOT NULL,
    token_hash VARCHAR(64),                 -- LeaseToken 哈希（用于撤销查询）
    scopes JSONB,

    -- 时间
    issued_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL,
    released_at BIGINT,
    last_used_at BIGINT,

    -- 状态
    status VARCHAR(32) NOT NULL,            -- "active"|"expired"|"released"|"revoked"

    -- 版本号
    version INT NOT NULL DEFAULT 1,

    INDEX idx_mesh_leases_cap (capability_id, status),
    INDEX idx_mesh_leases_lessee (lessee_node_id),
    INDEX idx_mesh_leases_expires (expires_at),
    INDEX idx_mesh_leases_ws (workspace_id, status)
);

-- 流会话
CREATE TABLE mesh_streams (
    id UUID PRIMARY KEY,
    lease_id UUID NOT NULL,                 -- → mesh_leases.id

    transport_type VARCHAR(32) NOT NULL,    -- "direct"|"relay"
    direction VARCHAR(16) NOT NULL,         -- "in"|"out"|"bidir"

    -- 连接信息
    local_addr VARCHAR(256),
    remote_addr VARCHAR(256),

    -- 统计
    bytes_sent BIGINT DEFAULT 0,
    bytes_recv BIGINT DEFAULT 0,

    -- 时间
    established_at BIGINT NOT NULL,
    closed_at BIGINT,
    close_reason VARCHAR(256),

    INDEX idx_mesh_streams_lease (lease_id),
    INDEX idx_mesh_streams_closed (closed_at)
);

-- 子 Agent 执行会话
CREATE TABLE mesh_agent_sessions (
    id UUID PRIMARY KEY,
    lease_id UUID NOT NULL,                 -- → mesh_leases.id

    -- Agent 身份
    parent_agent_id VARCHAR(64),            -- Brain 或上级 Agent 的 ID
    agent_role VARCHAR(32) NOT NULL,        -- "brain"|"sub_agent"

    -- 任务
    task_summary TEXT,                      -- 任务描述摘要
    target_node_id VARCHAR(64) NOT NULL,

    -- 沙箱配置
    sandbox_policy JSONB NOT NULL,          -- SandboxPolicy 完整快照

    -- 执行统计
    actions_attempted INT DEFAULT 0,        -- 尝试执行的操作数
    actions_completed INT DEFAULT 0,        -- 成功完成的操作数
    actions_denied INT DEFAULT 0,           -- 被沙箱/策略拒绝的操作数
    approvals_requested INT DEFAULT 0,      -- 触发用户审批的次数
    approvals_granted INT DEFAULT 0,        -- 用户批准的次数

    -- 状态
    status VARCHAR(32) NOT NULL,            -- "running"|"completed"|"failed"|"terminated"|"approval_pending"
    status_reason TEXT,

    -- 时间
    started_at BIGINT NOT NULL,
    completed_at BIGINT,
    max_duration_sec INT,

    -- 版本号
    version INT NOT NULL DEFAULT 1,

    INDEX idx_mesh_agent_lease (lease_id),
    INDEX idx_mesh_agent_parent (parent_agent_id),
    INDEX idx_mesh_agent_node (target_node_id, status),
    INDEX idx_mesh_agent_ws (status, started_at)
);

-- 用户审批记录
CREATE TABLE mesh_approval_records (
    id UUID PRIMARY KEY,
    agent_session_id UUID NOT NULL,         -- → mesh_agent_sessions.id

    -- 请求内容
    requested_action VARCHAR(256) NOT NULL, -- 请求执行的操作
    sandbox_rule_hit VARCHAR(256),          -- 命中的沙箱规则
    request_context JSONB,                  -- 请求上下文（供用户判断）

    -- 审批结果
    outcome VARCHAR(32) NOT NULL,           -- "pending"|"granted"|"denied"|"timeout"
    granted_scope VARCHAR(32),              -- "once"|"pattern"|"session"（仅 granted 时有值）
    granted_pattern VARCHAR(256),           -- 授权的 pattern（scope=pattern 时）
    responded_by VARCHAR(64),               -- 响应的用户 ID
    response_channel VARCHAR(32),           -- "push"|"desktop"|"email"

    -- 时间
    requested_at BIGINT NOT NULL,
    responded_at BIGINT,
    timeout_sec INT NOT NULL,

    INDEX idx_mesh_approval_session (agent_session_id),
    INDEX idx_mesh_approval_outcome (outcome, requested_at)
);

-- 审计事件
CREATE TABLE mesh_audit_events (
    id UUID PRIMARY KEY,
    request_id VARCHAR(64),
    trace_id VARCHAR(64),
    timestamp BIGINT NOT NULL,

    event_type VARCHAR(64) NOT NULL,
    source_node_id VARCHAR(64),
    target_node_id VARCHAR(64),

    capability_type VARCHAR(64),
    capability_id UUID,
    lease_id UUID,
    session_id UUID,
    agent_session_id UUID,                  -- → mesh_agent_sessions.id（Agent 相关事件）

    outcome VARCHAR(32),
    reason TEXT,
    status_code INT,
    duration_ms BIGINT,

    workspace_id VARCHAR(64),
    user_id VARCHAR(64),
    metadata JSONB,

    -- 审计链完整性（public_internet 模式启用）
    prev_event_id UUID,                     -- 前一条审计事件 ID
    prev_hash VARCHAR(64),                  -- 前一条审计事件的 SHA-256 哈希

    INDEX idx_mesh_audit_ts (workspace_id, timestamp),
    INDEX idx_mesh_audit_type (event_type, timestamp),
    INDEX idx_mesh_audit_node (source_node_id, timestamp),
    INDEX idx_mesh_audit_request (request_id),
    INDEX idx_mesh_audit_agent (agent_session_id, timestamp)
);
```

### 10.2 与现有模型的关系

```
execution_nodes (不变)
    │
    │ node_id 关联
    ▼
mesh_device_enrollments — 哪些 node 属于哪个 workspace/user
mesh_capabilities       — node 提供的能力（含 agent.exec）
    │
    │ capability_id 关联
    ▼
mesh_leases             — 能力的租约
    │
    │ lease_id 关联
    ├───────────────────┐
    ▼                   ▼
mesh_streams          mesh_agent_sessions — 子 Agent 执行会话
  数据流会话              │
                         │ agent_session_id 关联
                         ▼
                      mesh_approval_records — 用户审批记录

mesh_audit_events       — 全操作审计（关联 lease_id + agent_session_id）
```

---

## 11. gRPC 服务定义

### 11.1 Hub 端：MeshCoordinatorService

```protobuf
// api/proto/mesh/v1/mesh_coordinator.proto

service MeshCoordinatorService {
  // ===== 设备接入 =====
  // 生成邀请码
  rpc CreateInviteCode(CreateInviteCodeRequest) returns (CreateInviteCodeReply);
  // 使用邀请码接入
  rpc EnrollDevice(EnrollDeviceRequest) returns (EnrollDeviceReply);
  // 撤销设备
  rpc RevokeDevice(RevokeDeviceRequest) returns (RevokeDeviceReply);

  // ===== 能力注册 =====
  rpc RegisterCapabilities(RegisterCapabilitiesRequest) returns (RegisterCapabilitiesReply);
  rpc UnregisterCapabilities(UnregisterCapabilitiesRequest) returns (UnregisterCapabilitiesReply);
  rpc CapabilityHeartbeat(CapabilityHeartbeatRequest) returns (CapabilityHeartbeatReply);

  // ===== 能力查询 =====
  rpc QueryCapabilities(QueryCapabilitiesRequest) returns (QueryCapabilitiesReply);

  // ===== 租约管理 =====
  rpc RequestLease(RequestLeaseRequest) returns (RequestLeaseReply);
  rpc ReleaseLease(ReleaseLeaseRequest) returns (ReleaseLeaseReply);
  rpc RevokeLease(RevokeLeaseRequest) returns (RevokeLeaseReply);

  // ===== Hub Relay =====
  rpc RelayStream(stream RelayFrame) returns (stream RelayFrame);
}

// 所有请求都包含追踪元数据
message RequestMeta {
  string request_id = 1;      // UUID，调用方生成
  string trace_id = 2;        // 分布式追踪 ID
  string source_node_id = 3;
  int64 timestamp = 4;
}
```

### 11.2 Edge 端：MeshPeerService

```protobuf
// api/proto/mesh/v1/mesh_peer.proto

service MeshPeerService {
  // 能力调用（request-response）
  rpc Invoke(PeerInvokeRequest) returns (PeerInvokeReply);

  // 流式传输（双向流）
  rpc DataStream(stream DataChunk) returns (stream DataChunk);

  // 健康探测
  rpc PeerHealth(PeerHealthRequest) returns (PeerHealthReply);
}

message PeerInvokeRequest {
  RequestMeta meta = 1;
  string lease_token = 2;      // Hub 签发的租约 token
  string capability_type = 3;
  string command = 4;           // "create_session"|"control"|"close_session"
  map<string, string> params = 5;
  bytes payload = 6;
}

message PeerInvokeReply {
  string request_id = 1;
  bool success = 2;
  string session_id = 3;       // 会话 ID（create_session 时返回）
  map<string, string> result = 4;
  bytes payload = 5;
  string error = 6;
}

message DataChunk {
  string session_id = 1;
  int64 sequence = 2;          // 单调递增序列号
  int64 timestamp_ms = 3;
  bytes data = 4;
  map<string, string> metadata = 5;
}
```

---

## 12. 模块设计

### 12.1 目录结构

```
internal/mesh/
├── node.go                         # mesh.Node 生命周期管理
├── module.go                       # fx.Module 注册
│
├── capability/
│   ├── provider.go                 # Provider 接口定义
│   ├── host.go                     # CapabilityHost（本地能力托管）
│   ├── registry.go                 # Hub 端：能力注册表
│   ├── broker.go                   # Hub 端：能力匹配
│   ├── availability.go             # 动态可用性检测
│   ├── failover.go                 # 故障转移管理
│   └── remote_proxy.go             # 远程能力透明代理
│
├── enrollment/
│   ├── service.go                  # 设备接入服务
│   └── invite.go                   # 邀请码管理
│
├── lease/
│   ├── authority.go                # Hub 端：租约签发
│   ├── manager.go                  # Edge 端：租约管理
│   └── token.go                    # LeaseToken 签发/验证
│
├── stream/
│   ├── manager.go                  # 流管理器
│   └── stats.go                    # 流统计收集
│
├── policy/
│   ├── engine.go                   # 策略评估引擎
│   ├── config.go                   # 策略配置加载
│   └── conditions.go               # 可用性条件检查器
│
├── audit/
│   ├── logger.go                   # 审计日志记录器
│   ├── reporter.go                 # 异步上报 Hub
│   └── types.go                    # 审计事件类型定义
│
├── cert/
│   ├── ca.go                       # Hub CA：根证书生成、节点证书签发
│   ├── manager.go                  # Edge 端：证书存储、自动续期
│   └── crl.go                      # CRL 管理与分发
│
├── transport/
│   ├── dialer.go                   # 统一连接器（mTLS + P2P → Relay 降级）
│   ├── peer_server.go              # Edge P2P gRPC server（mTLS）
│   └── relay_client.go             # Hub Relay 客户端
│
├── sandbox/
│   ├── policy.go                   # SandboxPolicy 定义与解析
│   ├── enforcer.go                 # 运行时权限拦截（每次操作前检查）
│   └── approval.go                 # 用户审批流（通知推送、等待确认、超时处理）
│
├── action/
│   └── publisher.go                # Hub 端：能力→Action 自动发布
│
└── providers/                      # 具体能力实现
    ├── audio/
    │   ├── microphone.go
    │   └── speaker.go
    ├── gpu/
    │   └── inference.go
    ├── video/
    │   └── camera.go
    └── agent/
        ├── executor.go             # 子 Agent 执行器（沙箱内启动 Agent 实例）
        └── lifecycle.go            # 子 Agent 生命周期（启动、监控、终止、超时）
```

### 12.2 fx 注入

```go
// internal/mesh/module.go

var Module = fx.Module("mesh",
    // 核心
    fx.Provide(mesh.NewNode),

    // 能力
    fx.Provide(capability.NewHost),
    fx.Provide(capability.NewRegistry),     // Hub only
    fx.Provide(capability.NewBroker),       // Hub only

    // 租约
    fx.Provide(lease.NewAuthority),          // Hub only
    fx.Provide(lease.NewManager),

    // 证书
    fx.Provide(cert.NewCA),                 // Hub only
    fx.Provide(cert.NewManager),

    // 传输
    fx.Provide(transport.NewDialer),
    fx.Provide(transport.NewPeerServer),

    // 策略
    fx.Provide(policy.NewEngine),

    // 审计
    fx.Provide(audit.NewLogger),
    fx.Provide(audit.NewReporter),

    // 设备接入
    fx.Provide(enrollment.NewService),

    // 沙箱
    fx.Provide(sandbox.NewEnforcer),
    fx.Provide(sandbox.NewApprovalManager),

    // 子 Agent 执行
    fx.Provide(agent.NewExecutor),

    // Action 发布
    fx.Provide(action.NewPublisher),         // Hub only
)
```

### 12.3 配置扩展

```protobuf
// 在 conf.proto 的 Bootstrap 中新增
message Mesh {
  bool enabled = 1;

  // P2P gRPC 监听地址
  string peer_addr = 2;                    // 默认 ":19400"

  // 能力声明（静态配置）
  repeated MeshCapabilityDecl capabilities = 3;

  // 策略配置文件路径
  string policy_config_path = 4;

  // 审计配置
  MeshAudit audit = 5;

  // 安全配置
  MeshSecurity security = 6;

  // Brain Agent 策略
  MeshBrainPolicy brain_policy = 7;
}

message MeshSecurity {
  // 安全等级: "trusted_lan"(默认) | "public_internet"
  string security_level = 1;

  // 节点证书有效期（天），默认 45
  int32 cert_ttl_days = 2;

  // 证书自动续期提前天数，默认 7
  int32 cert_renew_before_days = 3;

  // CRL 检查模式: "soft_fail"(默认) | "hard_fail"
  string crl_check_mode = 4;

  // Hub CA 证书路径（public_internet 模式可对接外部 CA）
  string external_ca_cert_path = 5;
  string external_ca_key_path = 6;
}

message MeshCapabilityDecl {
  string type = 1;              // "audio.input"|"gpu.compute"|...
  string provider = 2;          // "builtin.microphone"|"builtin.vllm_proxy"|...
  map<string, string> config = 3;
  MeshCapabilityPolicy policy = 4;
}

message MeshCapabilityPolicy {
  bool require_auth = 1;
  int32 max_sessions = 2;
  bool exclusive = 3;
}

// Brain Agent 默认沙箱配置
message MeshBrainPolicy {
  // Brain Agent 自身的沙箱类型（通常为 "capability" 或 "none"）
  string sandbox_type = 1;

  // Brain 派发子 Agent 时的默认沙箱策略
  MeshSandboxDefaults sub_agent_defaults = 2;

  // 审批策略
  string approval_mode = 3;                // "never"|"on_escalation"|"always"
  int32 approval_timeout_sec = 4;          // 默认 60
  repeated string notify_channels = 5;     // ["push", "desktop"]
}

message MeshSandboxDefaults {
  string type = 1;                         // 默认沙箱类型: "capability"
  repeated string denied_actions = 2;      // 全局黑名单
  int32 max_duration_sec = 3;              // 默认最大执行时间
  int32 max_memory_mb = 4;
  bool allow_sub_agent_spawn = 5;          // 默认 false
}

message MeshAudit {
  bool enabled = 1;
  bool report_to_hub = 2;      // 异步上报 Hub
  int32 retention_days = 3;    // 本地保留天数
}
```

---

## 13. 调用全链路

```
┌──────────────────────────────────────────────────────────────────┐
│                     完整调用链路 & 审计点                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Agent tool_use("mesh.audio.capture", {...})                     │
│    │  AUDIT: invoke.called                                       │
│    ▼                                                              │
│  Plugin Gateway                                                  │
│    │  1. Token 验证 (token.Service)                              │
│    │  2. Capability 检查 (audio.capture)                         │
│    │  3. Policy 评估 (policy.Engine)                             │
│    │  AUDIT: policy.evaluated                                    │
│    ▼                                                              │
│  mesh.Node.Invoke()                                              │
│    │  检查本地租约缓存                                             │
│    │  命中且未过期 → 直接复用，跳到 transport.Dial               │
│    │                                                              │
│    ▼ (缓存未命中)                                                 │
│  Hub.RequestLease()  [via 已有 Edge→Hub gRPC 链路]               │
│    │  Hub 端:                                                     │
│    │  4. Workspace 权限验证                                       │
│    │  5. Mesh Policy 评估 (by_capability + by_relationship)      │
│    │  6. CapabilityBroker 选择最优 Provider 节点                  │
│    │     （基于 metrics: 利用率、延迟、电量加权）                  │
│    │  7. 签发 LeaseToken (Ed25519 签名)                          │
│    │     可续期租约: Renewable=true, MaxRenewals=N               │
│    │  AUDIT: lease.requested + lease.granted/denied              │
│    ▼                                                              │
│  transport.Dial(provider_node)                                   │
│    │  建立 mTLS 连接（双向证书验证，Hub CA 签发）                 │
│    │  尝试 P2P 直连                                               │
│    │  失败 → Hub Relay 降级（首帧认证，后续帧轻量 session_token） │
│    │  AUDIT: stream.created (transport: direct/relay)            │
│    │  AUDIT: transport.fallback (如果降级)                        │
│    ▼                                                              │
│  Edge B: MeshPeerService.Invoke()                                │
│    │  8. mTLS 身份确认 (证书 node_id 匹配)                      │
│    │  9. LeaseToken 签名验证 (Ed25519 公钥验签)                  │
│    │ 10. Token 字段校验 (过期、capability、来源与证书一致)        │
│    │ 11. 本地策略评估 (资源可用性、并发限制)                       │
│    │  AUDIT: policy.evaluated (provider-side)                    │
│    ▼                                                              │
│  CapabilityHost.CreateSession()                                  │
│    │ 12. Provider 创建会话                                        │
│    │ 13. 执行操作（采集音频 / GPU 推理 / ...）                    │
│    │  AUDIT: invoke.completed                                    │
│    ▼                                                              │
│  返回结果 → mesh.Node → Plugin Gateway → Agent                  │
│    │  AUDIT: stream.closed                                       │
│    │  可续期租约: 缓存复用，不立即释放                            │
│    │  单次租约: lease.released                                    │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

---

## 14. 待详细设计的内容

以下主题已确定方向但需要进一步细化，标记为后续讨论点：

### 14.1 Provider 接口与硬件适配层 [需细化]

**现状**: 已定义 `Provider` / `Session` 接口框架。

**待定**:
- 各平台（macOS/iOS/Linux）的硬件访问方式差异大，Provider 实现是 Go native 还是通过 FFI/CGo 调用系统 API？
- 音频编解码选型：纯 Go（如 `hraban/opus`）还是 CGo 绑定？
- 视频帧的表示格式和传输编码（H.264/VP8 的 Go 编解码器成熟度）
- Provider 的热插拔：USB 麦克风拔出后如何检测和响应

### 14.2 Action 自动发布的精确语义 [需细化]

**现状**: Hub 端 `ActionPublisher` 将 capability → Action。

**待定**:
- 一个 capability type 发布一个 Action（如所有 `audio.input` 设备共享一个 `mesh.audio.capture` Action），还是每个设备的每个 capability 实例各自发布？推荐前者，运行时动态选节点
- Action 的生命周期——设备下线后 Action 是否标记为 unavailable？还是保留但调用时返回错误？
- Action input/output schema 的版本管理——能力规格升级时 Action schema 如何演化

### 14.3 长时会话的管理 [需细化]

**现状**: 短时调用走 request-response，长时会话返回 session_id。

**待定**:
- Session 的最大生存时间和自动续期机制
- 客户端（Agent）如何消费长时会话的流数据——轮询 Session API 还是 WebSocket 推送
- 会话中断后的恢复策略：从断点续传还是重新开始
- 多个 Agent 能否共享同一个 Session（如多人同时听同一个麦克风）

### 14.4 Hub Relay 的性能与限流 [需细化]

**现状**: Hub 做 bidirectional stream 透传。

**待定**:
- Hub 端的连接数限制和带宽限流策略
- Relay 流量是否计费/配额
- 大流量场景（视频流 relay）下 Hub 的资源消耗评估
- 是否需要 Hub 集群部署来分担 relay 流量

### 14.5 跨 Workspace 能力共享 [需细化]

**现状**: 能力在 workspace 内可见。

**待定**:
- 是否支持跨 workspace 共享（如公共 GPU 集群）
- 如果支持，信任模型如何扩展
- 计费/配额模型

### 14.6 离线与弱网场景 [需细化]

**现状**: 假设节点在线且网络可达。

**待定**:
- 设备频繁上下线（如 iPhone 锁屏/切后台）时的优雅处理
- 能力状态的最终一致性保证——Hub 注册表与实际设备状态的同步延迟
- 心跳间隔和超时阈值的合理配置

### 14.7 数据库选型统一 [需细化]

**现状**: 项目同时支持 PostgreSQL 和 MySQL（GORM 驱动切换）。

**待定**:
- `mesh_audit_events` 表会快速增长，是否需要分区策略或时序数据库
- 审计数据的归档和清理策略（类比现有 Change DAG 的 GC 机制）

### 14.8 可观测性集成 [需细化]

**现状**: 项目已有 OpenTelemetry + Prometheus。

**待定**:
- Mesh 相关的关键指标定义（能力利用率、租约成功率、P2P/Relay 比例、流延迟）
- 分布式追踪如何跨 P2P gRPC 传播 trace context
- 告警规则（能力长时间不可用、租约失败率飙升等）

### 14.9 Migration 与灰度 [需细化]

**现状**: 新增模块，不修改现有表。

**待定**:
- Mesh 功能的 feature flag 控制
- 新旧版本 Edge 节点共存时的兼容性
- 数据库 migration 策略（新增表，无破坏性变更）

---

## 15. 实施路径建议

基于全能力框架的目标，建议分层实施：

| 阶段 | 内容 | 依赖 |
|------|------|------|
| **L1: 骨架** | proto 定义、fx module、配置扩展、数据库 migration | 无 |
| **L2: 证书与安全基础** | Hub CA、节点证书签发/续期/吊销、Ed25519 密钥管理、安全等级配置 | L1 |
| **L3: 注册与发现** | 设备接入、能力注册/查询、Hub CapabilityRegistry、能力指标上报 | L1 + L2 |
| **L4: 租约与策略** | LeaseToken 签发/验证(Ed25519)、策略引擎、租约缓存与续期 | L3 |
| **L5: P2P 通信** | MeshPeerService(mTLS)、transport.Dialer、Hub Relay(session-based) | L2 + L4 |
| **L6: 能力框架** | Provider/Session 接口、CapabilityHost、RemoteProxy、可靠性提示 | L5 |
| **L7: 沙箱与审批** | SandboxPolicy、Enforcer、ApprovalManager、用户通知推送 | L6 |
| **L8: 子 Agent 执行** | agent.exec 能力、AgentExecutor、子 Agent 生命周期管理 | L6 + L7 |
| **L9: Action 集成** | ActionPublisher、Plugin Gateway 扩展、mesh metadata（含 agent.exec Action） | L3 + L8 |
| **L10: 审计** | 审计日志、NATS 上报、审计表、防篡改（hash chain）、元审计 | L1 |
| **L11: 故障转移** | FailoverManager、可用性检测、通信降级 | L6 |

L2（证书与安全基础）提前到注册发现之前，因为 mTLS 和签名是后续所有通信的前提。
L7-L8（沙箱 + 子 Agent）是 Brain Agent 架构的核心，在能力框架之后、Action 集成之前。
L10（审计）可与其他层并行，从 L1 开始就埋入审计点。

---

*本文档基于 `docs/design/device-mesh-architecture.md` 的初始草案修订，反映了架构评审后的设计决策。v2.1 修订了安全模型、租约机制和通信架构。v2.2 新增主控 Agent 架构、沙箱权限模型和审计防篡改机制。*
