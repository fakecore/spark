# DoorX Plugin Platform & Capability Contract Spec (v1)

最后更新：2026-02-19

---

## 1. 文档目的

本文件把“**注册驱动（Registry-driven）**”的插件平台落成可执行规范，回答以下问题：

1. 第三方插件如何接入（安装、注册、升级、卸载）。
2. 平台如何统一暴露能力给插件（包括你自己写的插件）。
3. LLM、Workflow、Sync Infra、节点执行面如何与插件协同。
4. 如何保证安全、审计、可回滚，不因插件或第三方 Provider 数据异常影响主系统。

本规范与以下文档协同：

- `docs/design/architecture.md`
- `docs/design/llm-module.md`
- `docs/design/llm-master-agent-architecture.md`

---

## 2. 设计原则（必须遵守）

1. **注册优先，配置次之**：能力通过注册协议发现，不通过硬编码 if/else 暴露。
2. **能力最小化授权**：插件默认无权限（default deny），只能拿到显式授权能力。
3. **协议稳定，实现可替换**：插件接口/事件协议稳定，运行时（in-process/sidecar/wasm）可替换。
4. **失败不扩散**：插件故障不拖垮核心链路；必须支持超时、隔离和熔断。
5. **全链路可审计**：谁调用了什么能力、在何时、以何身份、结果如何，必须可追踪。

---

## 3. 现状评估（基于当前代码）

当前工程已有清晰骨架：

- 启动组装：`cmd/server/main.go`
- 模块分层：`internal/service/service.go`、`internal/biz/biz.go`、`internal/data/data.go`、`internal/workflow/workflow.go`、`internal/server/server.go`
- 执行面：`internal/biz/execution/usecase.go` + `internal/server/execution_node_endpoints.go`
- 基础设施与热更新：`internal/infrastructure/infrastructure.go`、`pkg/config/watcher/watcher.go`、`pkg/utils/provider/*.go`

但对插件平台仍缺三件核心能力：

1. **统一插件运行时协议**（现在没有标准 Plugin RPC/事件契约）。
2. **统一能力网关**（插件拿能力仍可能绕业务边界）。
3. **统一注册中心**（尚未形成“插件 manifest -> 快照 -> 路由”的主控链路）。

---

## 4. 目标分层（插件平台视角）

```text
Edge Layer
  - HTTP/gRPC/WS API
  - Plugin Admin API (install/enable/disable)

Application Layer
  - PluginManager (lifecycle/orchestration)
  - CapabilityGateway (统一能力出口)
  - AgentRuntime / WorkflowRuntime / LLMRuntime

Domain Layer
  - Plugin manifest model
  - Capability model
  - Policy model (who can do what)
  - Event contract model

Adapter Layer
  - Plugin runtime adapters (inproc/grpc/wasm)
  - Third-party connectors (IM/LLM/Storage/Webhook)
  - Infra adapters (NATS/Postgres/Redis/Oss)

Infra Layer
  - Registry store, snapshot manager, watcher
  - Audit log, metrics, tracing
  - Signature verification / trust anchors
```

边界要求：

- 插件不得直接访问 `gorm.DB` / `nats.Conn` / 系统密钥。
- 插件只能通过 `CapabilityGateway` 调用平台能力。
- `CapabilityGateway` 再调用内部 `biz/service/workflow/llm runtime`。

---

## 5. 统一插件模型（Manifest + Contract）

### 5.1 插件包结构（建议）

```text
plugin-package/
  manifest.json
  schemas/
    config.schema.json
    tools/*.schema.json
  runtime/
    plugin.bin | plugin.wasm
  checksums.txt
  signature.sig
```

### 5.2 Manifest 最小字段

```json
{
  "apiVersion": "projecttemplate.plugin/v1",
  "id": "im.feishu.bot",
  "name": "Feishu Bot Connector",
  "version": "1.2.0",
  "type": "im-connector",
  "runtime": "grpc-sidecar",
  "entrypoint": "127.0.0.1:19001",
  "capabilities": [
    "im.receive",
    "im.send",
    "conversation.read",
    "conversation.write",
    "llm.chat.invoke"
  ],
  "hooks": ["onStart", "onMessage", "onConfigChange", "onStop"],
  "configSchema": "schemas/config.schema.json",
  "toolSchemas": ["schemas/tools/send_message.schema.json"],
  "permissions": {
    "workspaceScope": "bound",
    "networkEgress": "restricted",
    "riskLevel": "medium"
  },
  "signature": {
    "alg": "ed25519",
    "keyId": "projecttemplate-plugin-marketplace",
    "digest": "sha256:..."
  }
}
```

### 5.3 生命周期状态机

`draft -> installed -> verified -> active -> deprecated -> disabled -> removed`

关键规则：

- 只有 `verified` 以上状态允许被加载。
- `disabled` 状态不可接收事件和调用。
- `deprecated` 可运行，但管理面必须告警。

---

## 6. 能力规范（Capability Contract）

### 6.1 能力命名约定

`<domain>.<resource>.<verb>`，示例：

- `llm.chat.invoke`
- `llm.model.list`
- `workflow.task.enqueue`
- `conversation.read`
- `conversation.write`
- `event.publish`
- `im.send`
- `im.receive`

### 6.2 能力暴露模型

平台只暴露 Capability API，不暴露底层对象：

- 不允许：插件直接执行 SQL / 直接操作 NATS / 读写任意文件。
- 允许：插件调用 `CapabilityGateway.Invoke(capability, input)`。

### 6.3 能力令牌（Capability Token）

平台按风险策略签发短期令牌（run-scope 或 per-call），绑定：

- `plugin_id`
- `workspace_id`
- `user_or_system_actor`
- `allowed_capabilities[]`
- `exp`（过期时间，统一使用 `iat/exp`，不在 token 内保留独立 `ttl` 字段）
- `request_id`

任一字段不匹配，网关拒绝调用。

说明：

- `ttl` 仅用于策略配置输入（如 60s/10m），签发时必须转换为 `iat/exp`。
- token 生效与失效以 `iat/exp` 为唯一语义来源。

---

## 7. 统一调用协议

### 7.1 Plugin Runtime gRPC（建议）

`PluginService` 最小接口：

- `Handshake(PluginHello) -> PluginReady`
- `Health(HealthRequest) -> HealthReply`
- `Invoke(InvokeRequest) -> InvokeReply`
- `OnEvent(EventEnvelope) -> Ack`
- `Configure(ConfigPatch) -> Ack`
- `Shutdown(ShutdownRequest) -> Ack`

### 7.2 事件协议（建议 CloudEvents 风格）

```json
{
  "id": "evt-...",
  "source": "projecttemplate.workflow",
  "type": "im.message.received",
  "time": "2026-02-19T08:00:00Z",
  "subject": "workspace:ws-1",
  "trace_id": "...",
  "data": {
    "conversation_id": "...",
    "text": "hello"
  }
}
```

要求：

- 所有插件事件必须带 `trace_id` 与 `workspace_id`（在 subject 或 data 中）。
- 事件投递语义按 at-least-once，插件必须实现幂等。

---

## 8. 模块调用细节（你关心的“足够细节”）

### 8.1 LLM -> Plugin Tool 调用链

1. `AgentRuntime` 生成 tool call（内部统一格式）。
2. `PolicyEngine` 校验本次会话允许哪些 capability。
3. `CapabilityGateway` 检查 token + 配额 + 风险级别。
4. `PluginManager` 选择目标插件实例并调用 `Invoke`。
5. 返回结果写入 `ActionRun/Conversation`，并记录审计日志。

### 8.2 Workflow -> Plugin 调用链

1. `workflow` 任务触发（如 `ChangeIngestor`/`OutboxPublisher` 后续步骤）。
2. 生成 `EventEnvelope` 发布到插件订阅通道。
3. 插件消费后返回 Ack；失败进入重试/死信。
4. 超重试阈值后标记 `plugin_event_failed`，不阻塞核心同步链路。

### 8.3 IM Bot 入站消息链

1. 第三方 IM 回调进入 `PluginAdapter`（或 sidecar connector）。
2. 标准化为 `im.message.received` 事件。
3. 路由到 `AgentRuntime`（可触发 LLM）。
4. 结果通过插件 `im.send` 能力发回第三方 IM。

### 8.4 节点间数据流转（与现有执行面对齐）

1. 节点注册/心跳/租约/回报沿用执行控制面：
   - `internal/server/execution_node_endpoints.go`
   - `internal/biz/execution/usecase.go`
2. 插件不直接参与节点租约协议。
3. 插件只在业务层接受“已授权的任务事件”或“已授权的能力调用”。

---

## 9. PluginManager / Registry 设计

### 9.1 注册流程

`discover -> verify -> compile -> activate`

与 LLM registry 一致采用快照模型：

- 发现：本地目录 + 远程仓库。
- 验签：签名、公钥、版本兼容性。
- 编译：manifest + schema -> `PluginRegistrySnapshot`。
- 激活：原子切换 active snapshot。

### 9.2 回滚机制

- 新快照校验失败：保留旧快照。
- 激活后异常率升高：自动回滚到 last-known-good。
- 管理面支持手工指定 snapshot generation 回滚。

---

## 10. 安全规范

### 10.1 必做项

1. 第三方插件必须签名验证。
2. 所有插件调用必须经 `CapabilityGateway`。
3. 高风险能力（`proc.exec`、`fs.write`、`network.egress`）默认禁用。
4. 审计日志必须包含：`plugin_id`、`capability`、`workspace_id`、`actor`、`request_id`、`result`。

### 10.2 运行时隔离策略

- `builtin`：仅官方内置，代码审计后可进程内运行。
- `third-party`：默认 sidecar 或 wasm，进程级隔离。
- 插件崩溃不得影响核心服务进程（隔离 + 超时 + 熔断）。

---

## 11. 可观测性规范

指标（Prometheus）：

- `plugin_invocation_total{plugin,capability,status}`
- `plugin_invocation_latency_ms{plugin,capability}`
- `plugin_event_consume_lag_ms{plugin,event_type}`
- `plugin_crash_total{plugin}`
- `plugin_registry_generation{generation}`

日志：

- 每次调用写 structured log，字段必须与 trace/span 关联。

追踪：

- 插件调用链必须继承上游 trace id，形成端到端链路。

---

## 12. 与当前代码目录映射（建议演进）

### 12.1 新增目录

```text
internal/plugin/
  core/                # manifest/capability/domain errors
  registry/            # loader/validator/snapshot manager/watcher
  runtime/             # grpc runtime/inproc runtime/wasm runtime
  gateway/             # capability gateway + token check
  policy/              # plugin permission policy
  manager/             # lifecycle orchestrator

api/proto/plugin/v1/   # plugin admin + runtime contract
```

### 12.2 与现有模块关系

- `internal/server`：新增 Plugin Admin API 路由。
- `internal/workflow`：新增事件分发到 plugin bus。
- `internal/biz/llm`：tool invocation 通过 plugin gateway。
- `internal/infrastructure`：接入签名验证器、runtime 进程管理、事件持久化。

---

## 13. 分阶段落地（可执行）

### Phase A（2~4 天）：基础注册与只读能力

- 完成 manifest/schema 校验。
- 完成插件注册快照 + 原子切换 + 回滚。
- 先开放只读能力：`llm.model.list`、`conversation.read`。

验收：

- 插件安装/启用/禁用可用。
- 异常插件不影响主链路。

### Phase B（4~7 天）：能力网关与调用链

- CapabilityGateway + token 实装。
- 插件 `Invoke` 协议落地。
- 接通 LLM tool call -> plugin invoke。

验收：

- 未授权 capability 调用全部拒绝。
- 调用链可追踪到 plugin_id/request_id。

### Phase C（5~10 天）：IM Bot 与事件驱动

- 标准事件协议 + 重试/死信。
- 接入一个 IM bot 示例插件（飞书/企业微信二选一）。
- 接入 workflow 事件订阅。

验收：

- 入站消息 -> Agent -> 回发全链路跑通。
- 失败重试与审计完整。

---

## 14. 非目标（当前阶段不做）

1. 插件市场（Marketplace）完整商业化流程。
2. 跨组织插件分发与计费系统。
3. 所有运行时（先 grpc-sidecar，后续再扩 wasm）。

---

## 15. 评审清单（你可直接用于评估）

1. 插件是否必须注册后才能调用能力（无旁路）？
2. 是否能在不发版的情况下完成插件新增/升级/禁用？
3. 是否支持能力级别授权与审计？
4. 插件崩溃是否会影响核心服务可用性？
5. LLM/Workflow/IM 这三条链路是否走同一能力网关规范？

只要 1~5 都是“是”，这套架构就满足你要的统一规范目标。

---

## 16. 开发者接入规范（SDK + Manifest）

本节回答开发者最关心的三个问题：

1. 怎么接入（开发流程）？
2. 什么时候能用平台能力（LLM/storage/workflow/event）？
3. 能不能“直接操作”底层资源？

### 16.1 开发者接入流程（必须遵循）

1. 使用 DoorX Plugin SDK 初始化插件项目。
2. 编写 `manifest.json`，声明：
   - `capabilities[]`
   - `runtime`
   - `hooks[]`
   - `configSchema`
3. 实现 SDK 约定的 `PluginService`（Handshake/Invoke/OnEvent/Health）。
4. 本地打包并生成签名（第三方插件必签名）。
5. 平台安装插件并执行 `discover -> verify -> compile -> activate`。
6. 管理员在 workspace 绑定插件实例，并授予 capability 策略。
7. 只有进入 `active` 状态后，插件才可接收事件和调用能力。

### 16.2 能力可用时机（Grant Timing）

manifest 声明 != 实际权限。

- **manifest 声明**：插件“申请”要什么。
- **policy 授权**：平台“批准”给什么。
- **runtime token**：本次调用“临时可用”什么。

三者必须同时满足，能力才可用。

### 16.3 关键结论：LLM/storage 是否都能操作？

结论：**不能直接操作底层；可以通过能力网关按授权调用**。

- 不允许：插件直接访问 DB、NATS、Redis、系统密钥、任意文件。
- 允许：插件通过 SDK 调用 `CapabilityGateway` 暴露的 API。
- 可调用范围取决于本次 token 的 `allowed_capabilities[]`。

也就是说：

- 你可以给插件用平台 LLM；
- 你也可以给插件做存储读写；
- 但都必须是“平台包装过的能力接口”，不是裸资源直连。

### 16.4 建议能力分级（便于评审）

`llm.*`

- `llm.model.list`：读取可用模型目录。
- `llm.chat.invoke`：调用平台统一 LLM Runtime。

`storage.*`

- `storage.object.read`：读对象存储（受 workspace 限制）。
- `storage.object.write`：写对象存储（受路径白名单限制）。
- `storage.kv.read`：读插件命名空间 KV。
- `storage.kv.write`：写插件命名空间 KV。

`workflow.*`

- `workflow.task.enqueue`：投递任务。
- `workflow.event.subscribe`：订阅事件。

`conversation.*`

- `conversation.read`
- `conversation.write`

### 16.5 权限拒绝语义（Denied Behavior）

SDK 调用能力被拒绝时，平台必须返回标准错误：

- `PERMISSION_DENIED`：能力不在授权集合。
- `CAPABILITY_DISABLED`：能力被管理员禁用。
- `TOKEN_EXPIRED`：临时 token 过期。
- `WORKSPACE_SCOPE_MISMATCH`：跨 workspace 调用。
- `RATE_LIMITED`：超过调用配额。

插件必须实现：

- 业务级降级（例如跳过非关键步骤）
- 幂等重试（仅对 `RATE_LIMITED`/`TRANSIENT`）
- 明确日志记录（包含 request_id）

### 16.6 撤权与生效规则（Revocation）

平台撤销授权后：

1. 新签发 token 立即不包含被撤销能力。
2. 已签发 token 到期后自动失效。
3. 高风险撤权可触发“即时失效列表”（token denylist）实现强制中断。

默认建议：

- 普通能力：TTL 5~15 分钟。
- 高风险能力：TTL <= 60 秒，且支持即时吊销。

### 16.7 SDK 最小接口（开发者视角）

```go
type HostCapabilities interface {
    Invoke(ctx context.Context, capability string, input map[string]any) (map[string]any, error)
}

type Plugin interface {
    OnStart(ctx context.Context, host HostCapabilities) error
    OnEvent(ctx context.Context, evt EventEnvelope) error
    Invoke(ctx context.Context, req InvokeRequest) (InvokeReply, error)
    OnStop(ctx context.Context) error
}
```

开发者只需要关注 capability 名称与输入输出 schema，不需要了解内部 DB/NATS/LLM provider 细节。

### 16.8 新闻助手示例（能力申请）

`manifest.json` 示例能力：

- `workflow.event.subscribe`
- `llm.chat.invoke`
- `storage.kv.read`
- `storage.kv.write`
- `conversation.write`
- `im.send`（若需发到 IM）

若管理员只授予了其中 4 项，插件就只能使用这 4 项；其余调用必须被拒绝并返回标准错误。

### 16.9 安装时权限确认（风险分级 + yes/no）

安装/启用插件时，平台必须展示“权限申请清单”，并按风险等级处理：

- `low`：默认自动通过（可在组织策略中改为需确认）。
- `medium`：默认需要一次确认（yes/no）。
- `high`：必须逐项确认（yes/no），且默认拒绝。
- `critical`：必须二次确认（yes/no + 管理员确认），支持审批流。

建议风险映射：

- `low`：`llm.model.list`、`conversation.read`（限定 workspace）。
- `medium`：`llm.chat.invoke`、`workflow.task.enqueue`。
- `high`：`storage.object.write`、`im.send`（外发）。
- `critical`：`proc.exec`、`network.egress` unrestricted、`fs.write` unrestricted。

确认界面建议包含：

1. 权限名称（capability id）
2. 风险等级（颜色 + 文案）
3. 影响范围（workspace/resource/path）
4. 数据方向（只读/写入/外发）
5. 有效期（永久/本次/时间窗）

### 16.10 确认结果的策略落地

yes/no 结果必须落地为可审计策略，不是只存在前端状态：

- `allow`：写入 `PluginGrantPolicy`，用于后续 token 签发。
- `deny`：写入拒绝策略，调用时返回 `PERMISSION_DENIED`。
- `allow-once`：仅本次 run 生效，run 结束即失效。
- `allow-until`：到期自动失效。

审计要求：

- 记录 `who/when/plugin_id/capability/decision/reason/ttl`。
- 高风险能力必须记录“确认人 + 确认方式（交互/审批）”。

### 16.11 运行时交互语义（与你说的流程一致）

当插件首次调用高风险能力且策略未决时：

1. 网关返回 `CAPABILITY_REQUIRES_CONSENT`。
2. 前端弹出 yes/no 确认。
3. 用户确认后，平台写入策略并重试调用。
4. 用户拒绝则保持拒绝策略并返回失败给插件。

这保证了：

- 低风险体验流畅（自动通过）。
- 高风险可控（必须显式确认）。
- 插件行为可追踪、可撤销、可复核。

---

## 17. 架构责任拆分（组件级）

为避免职责漂移，插件平台按以下组件拆分：

1. `PluginRegistry`
   - 负责插件包发现、验签、schema 校验、快照编译、原子切换。
   - 只负责“插件是否可用”，不负责授权决策。

2. `PluginManager`
   - 负责实例生命周期：install/enable/disable/restart/health。
   - 负责把事件与调用路由到正确插件实例。

3. `PolicyEngine`
   - 负责能力授权判定（声明、组织策略、用户确认、风险等级、资源范围、TTL）。
   - 输出标准决策：`ALLOW | DENY | REQUIRE_CONSENT`。

4. `CapabilityGateway`
   - 插件对平台能力的唯一入口。
   - 负责 token 校验、配额、审计、熔断与错误语义统一。

5. `RuntimeAdapter`
   - 负责插件运行形态（builtin/grpc-sidecar/wasm）和隔离边界。
   - 不参与授权逻辑，只执行调用转发。

6. `Audit & Telemetry`
   - 负责调用日志、策略变更日志、指标、trace 关联。

关键约束：

- `PluginManager` 不能绕过 `CapabilityGateway` 直接给插件能力。
- `PolicyEngine` 必须纯决策，不做 RPC/IO；避免与执行耦合。
- `RuntimeAdapter` 不应持有长期密钥。

---

## 18. 策略判定引擎（PolicyEngine）

### 18.1 判定输入

- `plugin_id`
- `workspace_id`
- `actor`（user/system/scheduler）
- `capability`
- `resource_scope`（路径/对象/会话/频道）
- `risk_level`
- `requested_ttl`

### 18.2 优先级（从高到低）

1. **系统硬禁令**（如 critical 能力全局禁用）
2. **组织安全策略**（org policy）
3. **workspace 策略**
4. **插件实例策略**
5. **用户交互确认策略**（yes/no）
6. **默认策略**（risk-based default）

同一 capability 多条规则冲突时：

- `deny` 优先于 `allow`
- 精确资源范围优先于通配范围
- 短 TTL 优先于长 TTL

### 18.3 标准决策算法

```text
if hard_deny -> DENY
else if explicit_deny -> DENY
else if consent_required && not_confirmed -> REQUIRE_CONSENT
else if explicit_allow -> ALLOW(with scope + ttl)
else if default_allow_by_risk -> ALLOW(with constrained scope + ttl)
else -> DENY
```

---

## 19. 权限状态机（Capability Grant Lifecycle）

### 19.0 CAP 是否需要 Lifecycle（结论）

需要，且分两层管理：

1. **Capability Catalog Lifecycle（平台能力定义）**
   - 状态：`active -> deprecated -> disabled -> removed`
   - 作用：控制能力版本演进、灰度下线、兼容窗口。

2. **Capability Grant Lifecycle（插件实例授权）**
   - 状态：`undeclared -> declared -> pending_consent -> granted -> active -> revoked -> expired`
   - 作用：控制某个插件实例在某个 workspace 的实际可用权限。

治理约束：

- Catalog 层 `disabled/removed` 时，实例层一律不可 `active`。
- Catalog `deprecated` 时，允许调用但必须告警并给出迁移提示。
- 禁止“仅看 manifest 声明即放权”，必须经过 Grant 生命周期。

每个 `plugin_id + workspace_id + capability` 的授权状态为：

`undeclared -> declared -> pending_consent -> granted -> active -> revoked -> expired`

状态含义：

- `declared`：manifest 声明了能力，仅表示“申请”。
- `pending_consent`：策略要求用户/管理员确认。
- `granted`：授权成立，但未进入运行上下文。
- `active`：当前 run/token 正在生效。
- `revoked`：被撤销，后续调用拒绝。
- `expired`：授权时间窗结束。

触发事件：

- install/enable -> `declared`
- consent yes -> `granted`
- run start + token mint -> `active`
- admin revoke -> `revoked`
- ttl timeout -> `expired`

### 19.1 consent 与 grant 转换规则

- `pending_consent` + user `yes` -> `granted`
- `pending_consent` + user `no` -> `declared`（并写入 deny policy）
- `granted` + run start -> `active`
- `active` + run end -> `granted`（授权仍在，但 run token 失效）
- `granted` + `allow_until` 到期 -> `expired`
- 任意状态 + admin revoke -> `revoked`

`allow_once` 特例：首次成功调用后立即 `expired`（或回落 `declared`，由策略实现决定），不得跨 run 复用。

---

## 20. Runtime Token 合约

token 由平台签发，插件不可自行伪造或扩权。

### 20.1 必含 claims

- `iss`：projecttemplate-gateway
- `aud`：projecttemplate-capability-gateway
- `plugin_id`
- `workspace_id`
- `run_id`
- `allowed_capabilities[]`
- `resource_constraints`
- `iat` / `exp`
- `jti`（便于吊销）

约束：

- `exp` 必须 `<= run_end_time + skew`（建议 `skew <= 120s`）。
- 未绑定 `run_id` 的插件调用 token 视为无效。

### 20.2 TTL 建议

- `low/medium`：5~15 分钟
- `high`：<= 60 秒
- `critical`：单次请求 token（one-shot）

### 20.3 续签规则

- 插件本身不能续签。
- 只能由 host runtime 在策略仍满足时续签。
- 撤权后不再续签。

### 20.4 签发/刷新策略（是否每次调用都刷新）

不采用“一刀切每次调用都刷新”，采用风险分级策略：

1. `low/medium`
   - run/session 级短 token（5~15 分钟）。
   - 到期或接近到期时由 host 续签。

2. `high`
   - 超短 token（<= 60 秒）。
   - 建议高频刷新（按批次调用或小时间窗）。

3. `critical`
   - one-shot token（单次请求一签一用）。
   - 调用完成即失效，不可复用。

统一要求：

- 无论 token 是否过期，每次调用都必须经过 Gateway 实时判定（含撤权检查）。
- 插件不得持久化长期 token，不得自行续签。
- 撤权后新调用立即拒绝；存量 token 依赖短 TTL + denylist 快速失效。

### 20.5 为什么默认 run 结束即失效（核心思考点）

默认按 run 收敛权限，不是为了“增加复杂度”，而是为了建立稳定安全边界：

1. **边界清晰**：run 是最自然的执行上下文边界。run 结束后应当“零残留权限”。
2. **爆炸半径可控**：即使 token 泄漏，也只影响短时间和单个 run。
3. **撤权延迟可控**：新 run 不再签发，存量 token 快速过期，撤权能快速生效。
4. **避免跨上下文复用**：上一个 run 的 token 不应跨用户/workspace/任务语义继续可用。
5. **审计可归因**：`run_id` 绑定后，风险事件可直接回溯到具体执行链路。

### 20.6 何时不必“每次调用都重签”

`run-scope` 与 `per-call` 不是同义词：

- `run-scope`：权限绑定一个 run 生命周期。
- `per-call`：每次 API 调用都新签 token。

默认建议：

- `low/medium` 用 run-scope 短 token，不做每次调用重签。
- `high` 采用超短 token，可按小时间窗批量复用。
- `critical` 才采用 per-call one-shot。

性能与安全平衡原则：

- **用户会话可长、插件权限要短**：长会话保留在 host；插件侧始终拿短 token。
- **减少重签成本**：host 端做静默续签，不要求用户重复确认。
- **拒绝长期插件凭证**：插件不持有 refresh token、不持久化 access token。

### 20.7 run 结束失效规则（强制）

run 生命周期结束（completed/failed/canceled/timeout）后：

1. 该 run 下全部 token 立即标记不可续签。
2. Gateway 对该 `run_id` 的后续调用直接拒绝（即使 `exp` 未到）。
3. in-flight 请求默认允许当前请求完成，但后续请求一律拒绝。

本规则优先级高于普通 `exp` 检查。

---

## 21. 端到端链路规范（时序）

### 21.1 安装与授权链路

```text
User/Admin -> PluginAdminAPI: Install(package)
PluginAdminAPI -> PluginRegistry: discover/verify/compile/activate
PluginRegistry -> PolicyEngine: evaluate declared capabilities
PolicyEngine -> UI: required consents (by risk)
UI -> PolicyEngine: yes/no decisions
PolicyEngine -> GrantStore: persist grants/denies
PluginManager: enable instance if minimal grants satisfied
```

### 21.2 定时任务（每天 8 点）链路

```text
Scheduler(08:00) -> PluginManager: start run(plugin_instance)
PluginManager -> PolicyEngine: build effective capability set
PolicyEngine -> Gateway: mint runtime token
Plugin -> CapabilityGateway: invoke(llm.chat.invoke / storage.kv.read ...)
Gateway -> Platform services: execute with scope checks
Gateway -> Audit: write invocation records
Plugin -> conversation/im: publish output
```

### 21.3 高风险首次调用链路（yes/no）

```text
Plugin -> Gateway: invoke(high-risk capability)
Gateway -> PolicyEngine: decision?
PolicyEngine => REQUIRE_CONSENT
Gateway -> Plugin: CAPABILITY_REQUIRES_CONSENT
UI -> User: yes/no prompt
User -> PolicyEngine: confirm or deny
Plugin retries -> Gateway -> ALLOW or DENY
```

### 21.4 撤权链路

```text
Admin -> PolicyEngine: revoke(capability)
PolicyEngine -> GrantStore: mark revoked
PolicyEngine -> TokenService: denylist(jti)/stop renewal
Gateway: reject subsequent calls immediately
```

撤权传播语义：

- 对同一 `plugin_id + workspace_id + capability` 的所有活跃 token（含派生 token）统一生效。
- 跨插件委托场景下，若 caller 能力被撤销，callee 的派生 token 同步失效。

### 21.5 插件间调用链路（Plugin-to-Plugin）

```text
Plugin A -> CapabilityGateway: invoke(plugin.invoke, target=Plugin B, action=X)
Gateway -> PolicyEngine: check caller grant + target export + scope
PolicyEngine -> TokenService: mint derived token (A->B, least privilege)
Gateway -> PluginManager: route to Plugin B runtime
Plugin B -> Gateway: optional host capability calls (re-checked)
Gateway -> Plugin A: return result + audit ids
```

关键语义：

- 插件间调用只能走 Gateway，禁止直连。
- `Plugin A` 的权限不会自动传递给 `Plugin B`（禁止权限穿透）。
- 派生 token 必须绑定 `caller_plugin_id`、`callee_plugin_id`、`run_id`、`trace_id`。

---

## 22. 数据模型（治理层）

以下为设计层建议结构（用于评审，不要求当前实现）：

### 22.1 `plugin_registry_snapshots`

- `generation` (pk)
- `created_at`
- `checksum`
- `status` (`active|failed|rolled_back`)
- `manifest_index_json`

### 22.2 `plugin_instances`

- `id` (pk)
- `plugin_id`
- `workspace_id`
- `state` (`installed|enabled|disabled|error`)
- `runtime_type`
- `config_json`
- `created_at` / `updated_at`

### 22.3 `plugin_grant_policies`

- `id` (pk)
- `plugin_id`
- `workspace_id`
- `capability`
- `decision` (`allow|deny|allow_once|allow_until`)
- `resource_scope_json`
- `risk_level`
- `granted_by`
- `reason`
- `expires_at` (nullable)
- `created_at`

### 22.4 `plugin_invocation_audit`

- `id` (pk)
- `request_id`
- `run_id`
- `plugin_id`
- `workspace_id`
- `capability`
- `decision`
- `result_status`
- `latency_ms`
- `error_code`
- `created_at`

---

## 23. API 合同（设计层）

### 23.1 管理面 API

- `POST /api/v1/plugins/install`
- `POST /api/v1/plugins/{instance_id}/enable`
- `POST /api/v1/plugins/{instance_id}/disable`
- `GET /api/v1/plugins/{instance_id}/permissions`
- `POST /api/v1/plugins/{instance_id}/consent`
- `POST /api/v1/plugins/{instance_id}/revoke`

### 23.2 运行面 API（Gateway）

- `POST /api/v1/plugin-gateway/invoke`
  - 入参：`capability`, `input`, `token`
  - 出参：`output` 或标准错误码
- `POST /api/v1/plugin-gateway/invoke-plugin`
  - 入参：`target_plugin_instance_id`, `action`, `input`, `token`
  - 出参：`output` 或标准错误码

### 23.3 错误码（最小集合）

- `PERMISSION_DENIED`
- `CAPABILITY_DISABLED`
- `CAPABILITY_REQUIRES_CONSENT`
- `WORKSPACE_SCOPE_MISMATCH`
- `TOKEN_EXPIRED`
- `TOKEN_INVALID`
- `RATE_LIMITED`
- `UPSTREAM_UNAVAILABLE`

---

## 24. 失败与降级规则

1. 插件进程异常：标记实例 `error`，不影响主服务；根据策略自动重启。
2. Gateway 异常：插件调用失败，返回 `UPSTREAM_UNAVAILABLE`，核心链路继续。
3. PolicyEngine 异常：按 fail-closed 处理（拒绝高风险调用）。
4. 审计写入失败：调用可成功，但必须打告警并异步补写。
5. 定时任务重入：同一 `plugin_instance + schedule_slot` 幂等锁，防重复执行。
6. 插件间调用失败：仅影响当前调用链，不级联阻断其他插件实例。

---

## 25. 插件间调用规范（是否允许）

结论：**允许，但默认受限且必须可审计。**

### 25.0 交互范式（主张）

V1 采用 **Action-first** 模型：

- 插件对外能力优先包装为 `exported actions`（同步 RPC 语义）。
- 跨插件异步协作走事件流（event bus），不直接替代 action 调用。

原因：

1. Action 有清晰 `input/output schema`，更适合做网关校验和权限治理。
2. Action 天然适配你现有 `ExecuteAction` 主链路，便于统一审计与追踪。
3. 事件流更适合广播/异步任务，不适合所有“请求-响应”场景。

### 25.1 默认策略

- V1 默认关闭“任意插件互调”。
- 只有显式授予 `plugin.invoke` 且目标插件显式 `export` 的 action 才可调用。

### 25.2 Manifest 约束（建议）

调用方（caller）声明：

- `requires: ["plugin.invoke:plugin-b"]`

被调用方（callee）声明：

- `exports.actions[]`（可被外部调用的 action 列表）
- 每个 action 的 `input_schema` / `output_schema` / `risk_level`

Action-first 示例（callee manifest 段）：

```json
{
  "exports": {
    "actions": [
      {
        "name": "news.fetch_digest",
        "input_schema": {"type": "object", "properties": {"topic": {"type": "string"}}},
        "output_schema": {"type": "object", "properties": {"items": {"type": "array"}}},
        "risk_level": "medium"
      }
    ]
  }
}
```

### 25.2.1 典型场景与推荐交互方式

1. **新闻抓取 -> 摘要 -> 分发**
   - 调用链：`news-plugin action` -> `llm-summary-plugin action` -> `im-sender-plugin action`
   - 推荐：Action 串联（需要每步可观测与失败定位）。

2. **价格波动告警广播**
   - 调用链：`market-plugin` 产生事件 -> 多订阅插件处理
   - 推荐：事件流（广播扇出，异步处理更合适）。

3. **用户问答时实时工具联动**
   - 调用链：`chat-plugin` 在单次请求内调用 `doc-search-plugin action`
   - 推荐：Action（低延迟、请求内完成）。

4. **每日8点批处理任务**
   - 调用链：Scheduler 触发 run -> Action DAG 执行
   - 推荐：调度触发 + Action 编排；必要时配合事件做状态通知。

结论：

- 需要“请求-响应 + 强校验 + 强审计”的，优先 Action。
- 需要“异步广播 + 多消费者”的，使用事件。

### 25.3 安全边界

1. 禁止跨 workspace 调用，除非明确跨域策略放行。
2. 禁止循环调用超过阈值（建议最大深度 3）。
3. 禁止把 caller 的高权限能力透传给 callee。
4. 高频互调需要独立配额，避免形成调用风暴。
5. 派生 token capability 必须是 caller 已授权能力与 callee `exports.actions[]` 的交集。

### 25.4 审计与追踪

每次互调记录：

- `caller_plugin_id`
- `callee_plugin_id`
- `action`
- `request_id` / `trace_id`
- `decision` / `result` / `latency`

必须支持从 caller 一键追溯到 callee 执行日志。

### 25.5 插件是否“暴露 Action”与“调用系统 Action”（双向模型）

结论：**是双向模型**。

1. **插件作为 Action Provider（向外提供能力）**
   - 插件在 manifest 里声明 `exports.actions[]`。
   - 平台把这些 action 编入 Action Catalog，供 Agent/Workflow/其他插件调用。

2. **插件作为 Action Consumer（调用平台能力）**
   - 插件通过 SDK 调用 `CapabilityGateway.Invoke("action.invoke", ...)`。
   - 只能调用授权白名单内的系统 action（例如 `system.search`, `system.notify`）。

约束：

- 插件不能直接调用内部 service/biz 对象；必须经网关。
- `exports.actions[]` 是“被调用面”，`requires[]` 是“调用面”，两者独立授权。

### 25.6 Plugin 内置 Sub-Agent Master 模式（你提到的场景）

允许插件内置一个“子 Agent 主控”，但其权限受插件上下文约束，不可越权。

#### 25.6.1 运行方式

1. 平台启动 plugin run，注入 `PluginContext`（含 capability token、预算、trace）。
2. 插件内 sub-agent 做计划（plan/reflect/tool-call）。
3. 每次工具或 action 调用都走 SDK -> Gateway。
4. Gateway 按当前 token + policy 决策允许/拒绝。
5. 结果回到 sub-agent，继续下一步，直到 run 完成。

#### 25.6.2 必须限制

- **无权限升级**：sub-agent 只能使用 plugin 当前 run 的能力子集。
- **无隐式持久权限**：run 结束后 token 失效；下次 run 重新签发。
- **预算控制**：必须设置 `max_steps`、`max_duration_ms`、`max_cost`。
- **循环防护**：限制 `plugin -> action -> plugin` 调用深度，建议 `<=3`。
- **子 Agent 创建能力隔离**：除非显式授予 `agent.spawn`，否则禁止继续创建 agent。

#### 25.6.3 推荐 capability

- `action.invoke`：调用系统 action 的统一入口。
- `llm.chat.invoke`：调用平台 LLM。
- `workflow.task.enqueue`：异步任务委托。
- `conversation.write` / `im.send`：输出结果。

#### 25.6.4 示例链路（新闻插件 + sub-agent）

```text
Scheduler(08:00) -> start plugin run(news)
news-sub-agent -> action.invoke(system.news.fetch)
news-sub-agent -> llm.chat.invoke(summary)
news-sub-agent -> action.invoke(system.dedup.check)
news-sub-agent -> conversation.write / im.send
run end -> token expired
```

这保证了插件可以有较强自治能力，同时仍在平台治理边界内运行。

---

## 26. 设计冻结标准（进入实现前）

满足以下条件才进入开发：

1. 能力命名空间与风险分级表冻结。
2. PolicyEngine 优先级与冲突规则冻结。
3. token claims、TTL、撤权语义冻结。
4. 管理面与运行面 API 错误码冻结。
5. 至少 2 条关键链路（安装授权、定时执行）通过架构评审。
6. 插件间调用策略（是否默认开启、深度限制、透传限制）冻结。

在未冻结前，禁止并行大规模实现，避免返工。
