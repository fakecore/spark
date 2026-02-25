# DoorX LLM Master Agent 架构设计（文件流热刷新）

最后更新：2026-02-19

---

## 1. 背景与目标

你提出的核心问题是正确的：

- **模型/Provider 不能写死在代码里**，否则每次模型上下线都要发版。
- LLM 层需要支持 **本地模型 + 云端 API**，并为未来 **voice/vision** 预留扩展位。
- 主 Agent 需要成为可控的“执行中枢”：可解释项目、可调度其他 Agent、可调用工具能力，但必须有清晰安全边界。

本设计目标：

1. 用“**文件化 Registry + 热刷新**”替代硬编码配置。
2. 在现有 DoorX 分层（service -> biz -> infrastructure）中落地 `llm-provider` 与 `master agent`。
3. 保持与 `docs/design/llm-module.md` 的 V1 思路兼容，并平滑扩展到多模态与多 Agent。

---

## 2. 关键决策（ADR）

### ADR-1: Provider/Model 全部走文件流

- Provider、Model、能力、价格、生命周期、路由标签，统一来自文件（JSON manifest + model/provider files）。
- 运行时只读取内存快照，不在请求路径做磁盘 IO。
- 文件更新后由 watcher 触发 reload，生成新快照并原子替换。

### ADR-2: 运行时“读写分离”

- **读路径无锁/低锁**：请求只读原子指针当前快照。
- **写路径串行**：reload 用互斥锁串行执行，避免并发覆盖。
- 模式参考现有 `pkg/utils/provider/*.go` 的 hot reload 管理器（atomic + mutex + 失败不切换）。

### ADR-3: 失败即回滚（保持旧快照）

- 新文件解析、校验、能力索引构建任一步失败，则保持旧快照继续服务。
- 对外返回“上一次成功版本 + 当前失败状态”，不让请求面感知不一致状态。

### ADR-4: 能力驱动路由，不靠 if-else 模型名分支

- 路由依据 capability（chat/tool_use/vision/audio/...）与 policy（成本、延迟、workspace 约束）。
- 模型名只作为候选集元素，不参与架构硬编码。

---

## 3. 建议目录结构（代码层）

```text
internal/
  biz/
    llm/
      types.go                # Unified request/response/message schema
      capabilities.go         # Capability flags and negotiation primitives
      provider.go             # Provider-agnostic adapter interface
      router.go               # Model/provider selection policy interface
      errors.go               # Stable error taxonomy
      registry.go             # In-memory snapshot interfaces
      agent_orchestrator.go   # Master agent orchestration interface

  infrastructure/
    llmregistry/
      loader.go               # Load files (providers/models/manifests)
      validator.go            # Schema + referential checks
      builder.go              # Build indexes (by capability/modality/tier)
      manager.go              # Atomic snapshot swap + versioning
      watcher.go              # fsnotify + debounce + reload trigger

    llmprovider/
      openai/
        client.go             # Provider adapter impl
        mapper.go             # request/response normalization
      anthropic/
      ollama/                 # local model provider example

    agentruntime/
      toolgate.go             # tool capability gate / allowlist check
      sandbox_policy.go       # process/file/network policy decisions
      audit_sink.go           # structured audit events

  service/
    llm_service.go            # Chat/Complete/Stream API entry
    agent_service.go          # Master agent control APIs
```

说明：

- `biz/llm` 只定义抽象，不依赖任何具体 HTTP SDK。
- `infrastructure/llmregistry` 专注“文件 -> 可用快照”。
- `infrastructure/llmprovider/*` 专注协议适配和字段归一化。
- `agentruntime` 专注“能做什么、谁可以做、怎么审计”。

---

## 4. Registry 数据模型（文件侧）

可复用你给的 `llm-providers` 思路：

- `providers/{id}.json`: base_url、auth_type、streaming、json_mode、function_calling 等。
- `models/{provider}/{model}.json`: modalities、capabilities、specs、pricing、lifecycle。
- `manifests/latest.json`: 版本、summary、索引、聚合视图。

建议新增（DoorX 侧私有字段，可放 `_projecttemplate`）：

- `_projecttemplate.routing_tier`: `budget|standard|premium|offline_first`
- `_projecttemplate.workspace_allow`: `[*]` 或 workspace id 列表
- `_projecttemplate.safety_profile`: `strict|standard|permissive`
- `_projecttemplate.tool_profile`: `none|basic|full`

---

## 5. 运行时核心对象

```go
type RegistrySnapshot struct {
    Version       string
    GeneratedAt   time.Time
    Providers     map[string]ProviderSpec
    Models        map[string]ModelSpec           // key: provider/model
    IndexByCap    map[string][]ModelRef
    IndexByMod    map[string][]ModelRef
    IndexByTier   map[string][]ModelRef
    ValidationLog []string
}
```

```go
type RegistryManager interface {
    Current() *RegistrySnapshot
    Reload(ctx context.Context) error
    LastSuccessVersion() string
    LastReloadError() error
}
```

```go
type ProviderAdapter interface {
    Complete(ctx context.Context, req *llm.Request) (*llm.Response, error)
    Stream(ctx context.Context, req *llm.Request) (llm.Stream, error)
    Supports(capability string) bool
}
```

---

## 6. 热刷新流程（文件流）

1. `watcher` 监听 registry 目录（含 debounce）。
2. 触发 `loader` 读取 provider/model/manifest 文件。
3. `validator` 执行校验：
   - JSON 格式校验
   - provider_id/model 引用一致性
   - capability/modality 字段合法性
4. `builder` 构建查询索引。
5. `manager` 原子替换快照指针。
6. 记录 reload 审计事件（成功/失败、版本、耗时、错误摘要）。

失败策略：

- 任一步失败 -> **不替换当前快照**；
- 对外继续使用旧版本，内部暴露告警与失败原因。

---

## 7. Master Agent 设计

### 7.1 职责边界

Master Agent 负责：

- 任务理解与计划（Plan）。
- 模型/Provider 选择（Route）。
- 子 Agent 调度（Dispatch）。
- 工具调用编排（Tool orchestration）。
- 输出汇总与解释（Synthesize）。

Master Agent 不直接负责：

- 具体 provider 协议细节（交给 adapter）。
- 低层权限判定实现（交给 ToolGate/PolicyEngine）。

### 7.2 子 Agent 生命周期

1. Master 生成子任务（含能力需求与资源上限）。
2. PolicyEngine 评估是否允许创建该子 Agent。
3. Orchestrator 选择模型并启动执行。
4. ToolGate 在每次工具调用前做 capability check。
5. AuditSink 全量记录关键动作（谁发起、调用什么、结果如何）。

---

## 8. 多模态扩展（voice/vision）

统一消息结构建议升级为“分片内容数组”：

```go
type ContentPart struct {
    Type string // text|image|audio|tool_result|...
    Text string
    URI  string
    Mime string
}

type Message struct {
    Role    string
    Name    string
    Content []ContentPart
}
```

这样未来支持 voice/vision 不需要重写 `Request` 主结构，只增 part 类型与 provider mapper。

---

## 9. 安全与治理边界

### 9.1 三层保护

1. **Registry 层**: 模型能力声明与 workspace 可用性约束。
2. **Policy 层**: 运行时授权（用户/角色/workspace/场景）。
3. **Execution 层**: 工具调用 gate（文件、进程、网络、代码执行）。

### 9.2 必须有的硬限制

- 工具能力默认拒绝（default deny）。
- 子 Agent 默认无“创建新 Agent”权限，需显式授予。
- 进程读取/控制能力独立 capability，不与普通文件读写捆绑。
- 审计日志必须包含：workspace、actor、agent_id、tool、target、result、request_id。

---

## 10. 与现有 DoorX 代码风格对齐点

可直接复用的工程模式：

- `pkg/config/watcher/watcher.go`: fsnotify + debounce + reload。
- `pkg/config/watcher/watcher.go`: provider 更新失败时保持旧配置。
- `pkg/utils/provider/database.go`: 读路径原子指针，写路径加锁，失败不切换。
- `pkg/utils/provider/hotreload_concurrency_test.go`: 验证读路径不会被更新锁阻塞。

LLM Registry/Provider 可以采用同样的并发与回滚语义，降低引入风险。

与现有执行编排面结合建议：

- `internal/biz/execution/usecase.go` 已有节点注册、心跳、任务租约、回报链路，可作为“子 Agent 执行面”的基础。
- `internal/server/execution_node_endpoints.go` 的 register/heartbeat/lease/report 端点可承载后续 agent worker 协议。
- `internal/workflow/hublease/hub_lease.go` 已有 workspace 级领导者语义，可用于主 Agent 的协调者选主，避免多实例重复调度。

---

## 11. 分阶段落地（建议）

### Phase A（先把架子立住）

- 实现 `llmregistry`：加载 + 校验 + 原子快照 + 热刷新。
- 实现一个 provider adapter（OpenAI-compatible）。
- `llm.Client` 改为从 registry 挑模型，不再依赖硬编码列表。

### Phase B（主 Agent 可控执行）

- 增加 `agent_orchestrator` + `toolgate` + 审计链路。
- 支持子 Agent 调度，但默认禁用“自我复制/创建 agent”。

### Phase C（多模态）

- Message part schema 升级。
- 增加 voice/vision adapter mapper。
- 引入按 modality/capability 的动态路由策略。

---

## 12. 验收标准

1. 更新 registry 文件后无需发版，系统可在运行中识别新模型。
2. 非法配置不会污染在线快照（旧版本继续可用）。
3. 同一请求路径在 reload 期间无明显阻塞。
4. 主 Agent 所有 tool 调用可审计、可回溯。
5. 新增 voice/vision 能力不需要修改核心 orchestrator 主流程。

---

## 13. 非目标（当前阶段不做）

- 立刻实现完整多 provider 全功能（先做主干抽象 + 1~2 个 adapter）。
- 在 V1 引入复杂 RL 路由策略（先规则路由 + capability gating）。
- 开放无限制的 agent 自创建能力（先最小可控集）。
