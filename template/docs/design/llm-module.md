# DoorX LLM Module Design (v1)

最后更新：2026-02-07

本文件定义 DoorX 在 Go 核心里的 **LLM 模块**（LLM Engine / Client Layer）V1 设计。目标是为 Action / Chat / Workflow 提供统一、可替换、可审计的 LLM 调用能力，并能与 Change DAG 的 local-first / federation 语义兼容。

---

## 1. 目标与非目标

目标（V1）：

- OpenAI-compatible API 为默认生态接口（可接 OpenAI/兼容网关/自建服务）。
- 统一的请求/响应结构（文本输出、usage、finish_reason、provider request id）。
- 清晰的错误分类与重试策略（rate limit / transient / permanent）。
- 审计与可复现性：记录 *prompt/params/model* 与输出，使执行结果可追踪。
- 与 ActionRun 的落库模型兼容：LLM 输出作为事实写入 Change DAG，不要求确定性重放。

非目标（V1 暂不做）：

- 多模态、embedding、图像生成、语音等全量能力（先聚焦 text/chat completion）。
- 完整的 per-workspace secrets 管理与 UI 配置（先以配置文件/env 为主，接口预留）。
- Router/Execution Node 远程执行协议（先在本地/Hub 同构服务内执行）。

---

## 2. 分层与代码布局（建议）

目标是避免 “biz 直接依赖某个 provider 的 HTTP SDK”：

- `internal/biz/llm`
  - 定义领域接口与核心类型：`Client`, `Request`, `Response`, `Usage`, `Error`
  - 定义策略：重试、超时、日志/审计字段、token budget（可选）
- `internal/infrastructure/llmopenai`（或 `internal/infrastructure/llm/provider/openai`）
  - OpenAI-compatible HTTP 实现（base_url + api_key + model）
  - streaming（SSE）在 V1.1 实现
- `internal/service/*`
  - 业务 API（Chat/Action/Workflow）只依赖 `biz/llm.Client`

依赖方向：

```text
service -> biz -> (interfaces) <- infrastructure
```

---

## 3. 核心接口草案

### 3.1 请求与响应

V1 以 Chat Completions 形态为主（同时也能表达纯 prompt）。

```go
// internal/biz/llm/types.go
package llm

type Message struct {
  Role string // system|user|assistant|tool
  Name string // optional
  Content string
}

type Request struct {
  Provider string // "openai" | ...
  Model    string

  Messages []Message

  Temperature *float32
  MaxTokens   *int32

  // For audit & idempotency across retries.
  RequestID string // caller supplied, stable

  // Caller context (must be carried through logs/traces).
  WorkspaceID string
  ObjectID    string // e.g. action_run id
}

type Usage struct {
  PromptTokens     int32
  CompletionTokens int32
  TotalTokens      int32
}

type Response struct {
  Text         string
  FinishReason string
  Usage        Usage

  ProviderRequestID string
  Model             string
}
```

### 3.2 Client 接口

```go
type Client interface {
  Complete(ctx context.Context, req *Request) (*Response, error)
  // Stream(ctx, req) (Stream, error) // V1.1
}
```

说明：

- `Provider`/`Model` 放在 request 里，便于 Router/策略在上层动态选择。
- `RequestID` 由调用方生成（例如 action_run_id + step_id），用于 provider side 的 idempotency key（若支持）与本地去重。

---

## 4. 错误模型与重试策略

### 4.1 错误分类（建议）

LLM 模块应把 provider 的错误映射为稳定的类别：

- `ErrAuth`：密钥无效/无权限（不重试）
- `ErrInvalidRequest`：参数错误/超限（不重试）
- `ErrRateLimited`：429（可重试，带 backoff）
- `ErrTransient`：超时、5xx、连接错误（可重试）
- `ErrCanceled`：ctx cancel/timeout（不重试，由上层决定）

### 4.2 重试（建议默认）

- 只对 `RateLimited/Transient` 重试
- backoff：指数退避 + 抖动，上限 30s，总时长受 `ctx` 控制
- 每次重试都要写 audit log（不泄漏敏感内容）

---

## 5. 审计与数据落地（local-first + federation）

### 5.1 关键原则

- **LLM 调用结果不要求确定性重放**：同一 prompt 可能产生不同输出。DoorX 的一致性边界是：
  - “执行时产生的输出”作为事实写入 Change DAG
  - 其他节点同步到这个事实后展示相同结果

因此，LLM 调用应把 *输入* 和 *输出* 都写入 ActionRun（或 Chat message）对象状态，避免未来重放时“再次调用 LLM”导致不一致。

### 5.2 建议写入 ActionRun 的结构

V1 先把这些字段嵌在 `action_run.output`（后续可演进为独立对象类型 `llm_call`）：

```json
{
  "text": "final text",
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "request_id": "run-uuid",
    "usage": {"prompt": 12, "completion": 34, "total": 46},
    "finish_reason": "stop",
    "provider_request_id": "xxx",
    "input_hash": "sha256(...)" // 可选：用于去重与审计，不存明文 prompt
  }
}
```

日志（`action_run.logs`）写：

- 重试次数、耗时、provider endpoint（不含 key）、错误分类

敏感信息处理：

- 默认不要把 API Key、完整 prompt 明文写入日志
- 若要存 prompt（用于审计/可复现），建议只存到 DB（object_state/change payload），并支持 workspace 级加密（P2）

---

## 6. 与 Action 系统的集成点

### 6.1 Action Definition（扩展）

在现有 `definition.kind` 基础上新增：

- `kind=llm`：同步调用 LLM，输出写入 `action_run.output`

示例：

```json
{
  "kind": "llm",
  "provider": "openai",
  "model": "gpt-4o-mini",
  "messages": [
    {"role":"system","content":"You are a translator."},
    {"role":"user","content":"Translate to zh: {{.text}}"}
  ],
  "temperature": 0.2
}
```

上层执行流程：

1. `ExecuteAction` 创建 `action_run`（running）
2. executor 解析 `definition.kind`
3. kind=llm 时调用 `llm.Client.Complete`
4. 把输出/usage 写入 `action_run`，状态置为 `completed/failed`

### 6.2 Streaming（V1.1）

- HTTP：SSE（与 `docs/design/architecture.md` 一致）
- 落地语义：
  - 流式 token 只用于 UI
  - 最终仍以 “final output” 写入 ActionRun（append-only）

---

## 7. 配置（V1 方案）

V1 推荐最小配置：

- provider base_url（可选，默认 OpenAI）
- api_key（env 或配置文件，避免提交到仓库）
- 超时与重试参数

后续（P1）：

- per-workspace provider 配置覆盖
- secrets manager（vault/OS keychain）

---

## 8. 验收标准（落地优先级）

V1（进入业务层前建议做到）：

- `llm.Client` 接口 + OpenAI-compatible 实现（非 streaming）
- Action 支持 `kind=llm` 的同步执行，ActionRun 写入 output + usage
- 至少 1 个容器级测试：执行 LLM Action 时可通过 fake provider / httptest server 验证请求与落库结构

V1.1：

- SSE streaming
- rate limit / retry 可观测性指标

