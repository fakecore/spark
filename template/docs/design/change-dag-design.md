# DoorX Change DAG：模型、业务数据结构与多节点合并

> **版本**: v2.0
> **日期**: 2026-02-05
> **状态**: 草案
> **作者**: Claude

---

## 1. 为什么是 Change DAG

Change DAG 的核心是把“对象状态”拆成“变更序列”，用父子依赖描述因果关系。它不依赖全局时间戳，也不依赖单点仲裁。多节点离线写入后，只要补齐变更，就能收敛到一致状态。

这带来三个直接收益：

- **离线优先**：本地先写，回连后补齐变更。
- **可并发**：多个节点同时改同一对象，不会静默覆盖。
- **可合并**：冲突通过合并策略与“合并变更”来收敛。

---

## 2. 核心概念

- **Change**：最小同步单元。
- **Parents**：该变更依赖的父变更，决定因果顺序。
- **DAG**：变更之间的有向无环图。
- **Heads**：当前没有被其他变更引用的末端变更集合。
- **Materialized State**：物化状态，业务查询使用。
- **Snapshot**：状态快照，用于加速回放与 GC。

---

## 3. Change 数据结构

```go
// internal/pkg/changedag/change.go
package changedag

type Change struct {
    ID          uuid.UUID   `json:"id"`
    WorkspaceID string      `json:"ws_id"`
    ObjectID    uuid.UUID   `json:"obj_id"`
    ObjectType  string      `json:"obj_type"`
    Parents     []uuid.UUID `json:"parents"`

    // OrderId：用于增量遍历/重放游标，不用于业务胜负。
    // V1 建议：由 Hub 分配 canonical order_id，并在同步时回填到各副本（跨副本一致）。
    // 非 Hub 节点离线新增 Change 时允许 order_id 为空。
    OrderID    string `json:"order_id,omitempty"`
    AuthorNode string `json:"author"`

    Op      string          `json:"op"`      // set, patch, add, remove, delete, merge
    Payload json.RawMessage `json:"payload"`

    CreatedAt int64 `json:"created_at"`
}
```

### 变更载荷建议

- 标量字段：`{"path":"/name","value":"xxx"}`
- JSON Patch：`[{"op":"replace","path":"/name","value":"xxx"}]`
- OR-Set：`{"add":[{"v":"tag","id":"uuid"}],"remove":["uuid"]}`
- Delete：`{"reason":"user"}`
- Merge：`{"heads":["c1","c2"],"strategy":"action_v1"}`

---

## 4. 存储结构

```sql
CREATE TABLE changes (
    id UUID PRIMARY KEY,
    workspace_id VARCHAR(64) NOT NULL,
    object_id UUID NOT NULL,
    object_type VARCHAR(64) NOT NULL,
    parents UUID[] NOT NULL,
    order_id VARCHAR(64),
    author_node VARCHAR(64) NOT NULL,
    op VARCHAR(16) NOT NULL,
    payload JSONB NOT NULL,
    created_at BIGINT NOT NULL
);

-- order_id 非空时在 object 内唯一（同时用于增量扫描）
CREATE UNIQUE INDEX uq_changes_object_order ON changes (workspace_id, object_id, order_id)
  WHERE order_id IS NOT NULL;
CREATE INDEX idx_changes_ws_id ON changes (workspace_id, id);

CREATE TABLE object_heads (
    workspace_id VARCHAR(64) NOT NULL,
    object_id UUID NOT NULL,
    heads UUID[] NOT NULL,
    updated_at BIGINT NOT NULL,
    PRIMARY KEY (workspace_id, object_id)
);

CREATE TABLE object_state (
    workspace_id VARCHAR(64) NOT NULL,
    object_id UUID NOT NULL,
    object_type VARCHAR(64) NOT NULL,
    state JSONB NOT NULL,
    updated_at BIGINT NOT NULL,
    PRIMARY KEY (workspace_id, object_id)
);

CREATE TABLE snapshots (
    id UUID PRIMARY KEY,
    workspace_id VARCHAR(64) NOT NULL,
    object_id UUID NOT NULL,
    base_change_id UUID NOT NULL,
    base_order_id VARCHAR(64) NOT NULL,
    state JSONB NOT NULL,
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_snapshots_object_created ON snapshots (workspace_id, object_id, created_at);
```

---

## 5. 业务数据结构建议

### 5.1 Action

```go
type Action struct {
    ID          uuid.UUID `json:"id"`
    WorkspaceID string    `json:"workspace_id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Definition  JSONB     `json:"definition"`
    Behavior    JSONB     `json:"behavior"`
    Tags        []string  `json:"tags"`
}
```

**Action 的典型变更**

```json
{
  "id": "c1",
  "obj_id": "a1",
  "parents": ["c0"],
  "op": "patch",
  "payload": [{"op":"replace","path":"/name","value":"英文翻译"}]
}
```

### 5.2 Conversation / Message

建议 Message 独立成对象，Conversation 维护消息顺序。

```go
type Conversation struct {
    ID          uuid.UUID `json:"id"`
    WorkspaceID string    `json:"workspace_id"`
    Title       string    `json:"title"`
    MessageIDs  []string  `json:"message_ids"` // 可用 RGA/LSEQ
}

type Message struct {
    ID             uuid.UUID `json:"id"`
    WorkspaceID    string    `json:"workspace_id"`
    ConversationID uuid.UUID `json:"conversation_id"`
    Role           string    `json:"role"`
    Content        string    `json:"content"`
}
```

**Message 新增变更**

```json
{
  "id": "m2",
  "obj_id": "conv1",
  "parents": ["m1"],
  "op": "add",
  "payload": {"message_id":"msg_01","position":"after:msg_00"}
}
```

### 5.3 Prompt

```go
type Prompt struct {
    ID          uuid.UUID `json:"id"`
    WorkspaceID string    `json:"workspace_id"`
    Name        string    `json:"name"`
    Content     string    `json:"content"`
    Tags        []string  `json:"tags"`
}
```

### 5.4 结合 DoorX 业务功能的 Change 示例

| 功能 | 对象类型 | 典型变更 | 合并策略 |
|------|---------|---------|---------|
| Action 编辑 | `action` | `patch` /name /definition | 标量 LWW + JSON 字段合并 |
| Action 执行 | `action_run` | `set`/`patch` 状态与输出 | LWW（状态机）+ 追加日志 |
| Workflow 编排 | `workflow` | `add` 节点/边 | OR-Set（节点/边） |
| Conversation | `conversation`/`message` | `add` 消息、`patch` 内容 | RGA/LSEQ（顺序）+ LWW |
| Prompt | `prompt` | `patch` 模板内容 | LWW |
| Tool 注册 | `tool_def` | `set` schema/version | LWW |
| Skill 调用 | `skill_run` | `set`/`patch` 结果与日志 | LWW + 追加日志 |
| Router 执行路由 | `route_decision` | `set` target/理由 | LWW |
| 远程 Tool 执行 | `tool_run` | `set`/`patch` 状态与输出 | LWW（状态机） |
| Workspace 成员 | `workspace_member` | `add/remove` 成员 | OR-Set（成员） |
| 多窗口同步 | `window_state` | `patch` 位置/尺寸 | LWW |
| 租约锁 | `lease` | `set` owner/expires | LWW + TTL 校验 |

#### Action 执行（ActionRun）

```json
{
  "id": "run_01_c2",
  "obj_id": "run_01",
  "obj_type": "action_run",
  "parents": ["run_01_c1"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/status","value":"completed"},
    {"op":"replace","path":"/output_ref","value":"blob://result/123"},
    {"op":"replace","path":"/ended_at","value":1738732800123}
  ]
}
```

#### Workflow 编排（新增节点与边）

```json
{
  "id": "wf_c10",
  "obj_id": "wf_01",
  "obj_type": "workflow",
  "parents": ["wf_c09"],
  "op": "add",
  "payload": {"node_id":"n3","node_type":"action","config":{"action_id":"a1"}}
}
```

```json
{
  "id": "wf_c11",
  "obj_id": "wf_01",
  "obj_type": "workflow",
  "parents": ["wf_c10"],
  "op": "add",
  "payload": {"edge_id":"e2","from":"n2","to":"n3"}
}
```

#### Conversation 消息追加（Message 独立对象 + 顺序引用）

```json
{
  "id": "msg_c1",
  "obj_id": "msg_01",
  "obj_type": "message",
  "parents": [],
  "op": "set",
  "payload": {"role":"user","content":"你好"}
}
```

```json
{
  "id": "conv_c8",
  "obj_id": "conv_01",
  "obj_type": "conversation",
  "parents": ["conv_c7"],
  "op": "add",
  "payload": {"message_id":"msg_01","position":"tail"}
}
```

#### 多窗口同步（WindowState）

```json
{
  "id": "win_c3",
  "obj_id": "win_01",
  "obj_type": "window_state",
  "parents": ["win_c2"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/x","value":120},
    {"op":"replace","path":"/y","value":80},
    {"op":"replace","path":"/width","value":1100},
    {"op":"replace","path":"/height","value":720}
  ]
}
```

#### 租约锁（Lease）

```json
{
  "id": "lease_c5",
  "obj_id": "lease_action_a1",
  "obj_type": "lease",
  "parents": ["lease_c4"],
  "op": "set",
  "payload": {"owner":"node_a","expires_at":1738732805000}
}
```

#### Router 执行路由（RouteDecision）

```json
{
  "id": "route_c2",
  "obj_id": "route_req_01",
  "obj_type": "route_decision",
  "parents": ["route_c1"],
  "op": "set",
  "payload": {"target":"cloud@node_gpu_1","reason":"large_ctx_remote","fallback":"local"}
}
```

#### Skill 调用（SkillRun）

```json
{
  "id": "skill_c3",
  "obj_id": "skill_run_01",
  "obj_type": "skill_run",
  "parents": ["skill_c2"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/status","value":"completed"},
    {"op":"add","path":"/logs/-","value":"claude code --skill translate ok"},
    {"op":"replace","path":"/output_ref","value":"blob://skill/out/88"}
  ]
}
```

#### 远程 Tool 执行（ToolRun）

```json
{
  "id": "tool_c4",
  "obj_id": "tool_run_02",
  "obj_type": "tool_run",
  "parents": ["tool_c3"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/status","value":"running"},
    {"op":"replace","path":"/node_id","value":"node_cloud_2"}
  ]
}
```

```json
{
  "id": "tool_c5",
  "obj_id": "tool_run_02",
  "obj_type": "tool_run",
  "parents": ["tool_c4"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/status","value":"completed"},
    {"op":"replace","path":"/output_ref","value":"blob://tool/out/991"}
  ]
}
```

#### Workspace 成员变更（WorkspaceMember）

```json
{
  "id": "ws_c9",
  "obj_id": "ws_team_a",
  "obj_type": "workspace_member",
  "parents": ["ws_c8"],
  "op": "add",
  "payload": {"member_id":"user_123","role":"editor"}
}
```

```json
{
  "id": "ws_c10",
  "obj_id": "ws_team_a",
  "obj_type": "workspace_member",
  "parents": ["ws_c9"],
  "op": "remove",
  "payload": {"member_id":"user_456"}
}
```

### 5.5 端到端场景回放（基于现有业务功能）

下面给出完整的“变更序列 → 合并 → 最终状态”预览，使用当前 DoorX 规划中的核心功能场景。

#### 5.5.1 Action 并发编辑（Name 与 Definition）

初始物化状态：

```json
{
  "id": "a1",
  "name": "翻译",
  "definition": {"model":"gpt-4o","lang":"en"},
  "behavior": {"stream": true}
}
```

节点 A 与节点 B 并发变更：

```json
{
  "id": "c1",
  "obj_id": "a1",
  "obj_type": "action",
  "parents": ["c0"],
  "op": "patch",
  "payload": [{"op":"replace","path":"/name","value":"英文翻译"}]
}
```

```json
{
  "id": "c2",
  "obj_id": "a1",
  "obj_type": "action",
  "parents": ["c0"],
  "op": "patch",
  "payload": [{"op":"replace","path":"/definition/model","value":"gpt-4.1"}]
}
```

合并变更（Merge Change）：

```json
{
  "id": "cm1",
  "obj_id": "a1",
  "obj_type": "action",
  "parents": ["c1","c2"],
  "op": "merge",
  "payload": {"strategy":"action_v1"}
}
```

最终物化状态：

```json
{
  "id": "a1",
  "name": "英文翻译",
  "definition": {"model":"gpt-4.1","lang":"en"},
  "behavior": {"stream": true}
}
```

#### 5.5.2 Conversation 并发追加消息

初始物化状态：

```json
{
  "id": "conv_01",
  "title": "周会",
  "message_ids": ["m1"]
}
```

节点 A 与节点 B 离线追加消息：

```json
{
  "id": "m2_c1",
  "obj_id": "m2",
  "obj_type": "message",
  "parents": [],
  "op": "set",
  "payload": {"role":"user","content":"我来汇报进展"}
}
```

```json
{
  "id": "conv_c9",
  "obj_id": "conv_01",
  "obj_type": "conversation",
  "parents": ["conv_c8"],
  "op": "add",
  "payload": {"message_id":"m2","position":"tail"}
}
```

```json
{
  "id": "m3_c1",
  "obj_id": "m3",
  "obj_type": "message",
  "parents": [],
  "op": "set",
  "payload": {"role":"assistant","content":"收到，继续"}
}
```

```json
{
  "id": "conv_c10",
  "obj_id": "conv_01",
  "obj_type": "conversation",
  "parents": ["conv_c8"],
  "op": "add",
  "payload": {"message_id":"m3","position":"tail"}
}
```

合并与顺序确定性（RGA/LSEQ + NodeID 决胜）后最终物化状态：

```json
{
  "id": "conv_01",
  "title": "周会",
  "message_ids": ["m1","m2","m3"]
}
```

#### 5.5.3 Workflow 并发编排（节点与边）

初始物化状态：

```json
{
  "id": "wf_01",
  "nodes": ["n1"],
  "edges": []
}
```

节点 A 与节点 B 并发新增节点：

```json
{
  "id": "wf_c20",
  "obj_id": "wf_01",
  "obj_type": "workflow",
  "parents": ["wf_c19"],
  "op": "add",
  "payload": {"node_id":"n2","node_type":"action","config":{"action_id":"a1"}}
}
```

```json
{
  "id": "wf_c21",
  "obj_id": "wf_01",
  "obj_type": "workflow",
  "parents": ["wf_c19"],
  "op": "add",
  "payload": {"node_id":"n3","node_type":"tool","config":{"tool_id":"t1"}}
}
```

随后各自添加边：

```json
{
  "id": "wf_c22",
  "obj_id": "wf_01",
  "obj_type": "workflow",
  "parents": ["wf_c20"],
  "op": "add",
  "payload": {"edge_id":"e1","from":"n1","to":"n2"}
}
```

```json
{
  "id": "wf_c23",
  "obj_id": "wf_01",
  "obj_type": "workflow",
  "parents": ["wf_c21"],
  "op": "add",
  "payload": {"edge_id":"e2","from":"n1","to":"n3"}
}
```

最终物化状态（OR-Set 合并）：

```json
{
  "id": "wf_01",
  "nodes": ["n1","n2","n3"],
  "edges": ["e1","e2"]
}
```

#### 5.5.4 Router 决策 + 远程 Tool 执行

路由决策（选择远程节点执行）：

```json
{
  "id": "route_c2",
  "obj_id": "route_req_01",
  "obj_type": "route_decision",
  "parents": ["route_c1"],
  "op": "set",
  "payload": {"target":"cloud@node_gpu_1","reason":"large_ctx_remote","fallback":"local"}
}
```

工具执行状态流转：

```json
{
  "id": "tool_c4",
  "obj_id": "tool_run_02",
  "obj_type": "tool_run",
  "parents": ["tool_c3"],
  "op": "patch",
  "payload": [{"op":"replace","path":"/status","value":"running"}]
}
```

```json
{
  "id": "tool_c5",
  "obj_id": "tool_run_02",
  "obj_type": "tool_run",
  "parents": ["tool_c4"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/status","value":"completed"},
    {"op":"replace","path":"/output_ref","value":"blob://tool/out/991"}
  ]
}
```

最终物化状态：

```json
{
  "id": "tool_run_02",
  "status": "completed",
  "output_ref": "blob://tool/out/991",
  "node_id": "node_cloud_2"
}
```

#### 5.5.5 租约锁并发抢占（Lease）

节点 A 与节点 B 并发抢占同一资源时，建议采用“单点仲裁 + 确定性决胜”：

- V1 推荐：由 Hub 写入并分配 `order_id`，以 `order_id` 作为顺序键（必要时再用 `change_id`/`author_node` 打平）。
- 落败方的 Change 仍会进入变更历史，但业务层可将其标记为抢占失败（UI 提示）。

胜出后的物化状态示例：

```json
{
  "id": "lease_action_a1",
  "owner": "node_a",
  "expires_at": 1738732805000
}
```

---

## 6. 多节点合并的基本逻辑

多节点合并的关键在于：

- DAG 的 **parents** 定义因果顺序。
- 并发修改会形成 **多 heads**。
- 收敛通过创建 **Merge Change** 完成。

### 6.1 同步流程（Heads 对账）

1. 节点 A 调用 Hub `GetHeads`。
2. 通过 `GetChangesAfter` 按 `order_id` 增量拉取变更（cursor-based）。
3. 若 Apply 过程中发现缺父（unattached），用 `GetChanges(change_ids)` 精确补齐缺失父变更。
4. `PushChanges` 推送本地新增变更（Hub 回填 canonical `order_id` 并返回映射）。
5. 本地 `ApplyChange` 更新物化状态与 heads。

### 6.2 Merge Change 的产生

当一个对象出现多个 heads：

- 选择合并策略，生成合并后的状态。
- 创建一个 **Merge Change**，`parents` 指向所有 heads。
- `Merge Change` 写入后，heads 收敛为该变更。

这样所有节点最终都会收敛到同一个 head。

---

## 7. 合并策略（按对象类型）

| 数据类型 | 推荐策略 |
|---------|---------|
| 标量字段 | LWW（Hub `order_id` + tie-break）或冲突提示 + 决议变更 |
| Map/Object | 递归按字段合并 |
| Set/Tags | OR-Set |
| List/顺序结构 | RGA/LSEQ |
| Delete | DeleteChange 覆盖 |

**说明**：同一对象类型必须使用确定性策略，保证所有节点在相同变更集下得到相同结果。

---

## 8. ApplyChange 伪代码

```go
func ApplyChange(state ObjectState, change Change) ObjectState {
    switch change.Op {
    case "set":
        return applySet(state, change.Payload)
    case "patch":
        return applyJSONPatch(state, change.Payload)
    case "add":
        return applyAdd(state, change.Payload)
    case "remove":
        return applyRemove(state, change.Payload)
    case "delete":
        return markDeleted(state)
    case "merge":
        return applyMerge(state, change.Payload)
    default:
        return state
    }
}
```

---

## 9. Action 配置与执行的规则（必须）

以下 4 类规则是 DoorX 在 Change DAG 之上的最小“硬规则”，避免执行态混乱与合并不确定性。

### 9.1 状态机规则（ActionRun）

```text
pending -> running -> completed/failed/canceled
failed -> running (retry)
```

约束：
- 终态（completed/failed/canceled）不可回退为非终态
- 非法状态跳转直接拒绝 ApplyChange

### 9.2 字段合并策略（Action）

- `name/description`：LWW
- `definition`：JSON Patch 按字段合并
- `behavior`：JSON Patch 按字段合并
- `tags`：OR-Set

> 规则必须确定性，保证同一变更集下所有节点合并结果一致。

### 9.3 幂等与终态保护（ActionRun）

- 相同 `ActionRun` 的终态只允许写入一次
- 重复执行消息必须可被去重（idempotent）
- 日志采用 append-only

### 9.4 Schema 校验（Action 定义）

- `definition`/`behavior` 必须通过 JSON Schema 校验
- 校验失败的变更禁止入库（ChangeStore 拒绝）
- 校验规则版本化（如 `schema_version` 字段）

---

## 10. Action 底层实现参考（代码骨架）

```go
// internal/pkg/changedag/merge.go
func MergeAction(local, remote Action) Action {
    // LWW: name/description
    // JSON patch: definition/behavior
    // OR-Set: tags
    return merged
}

// internal/biz/action/run_fsm.go
func AllowTransition(from, to string) bool {
    switch from {
    case "pending":
        return to == "running" || to == "canceled"
    case "running":
        return to == "completed" || to == "failed" || to == "canceled"
    case "failed":
        return to == "running" // retry
    default:
        return false
    }
}

// internal/biz/action/validator.go
func ValidateActionDefinition(def JSONB) error {
    // JSON Schema validate
    return nil
}
```

---

## 11. 冲突与确定性

- 并发变更并不是错误，而是 **多 heads**。
- 只要 merge 策略确定，所有节点都会收敛到相同结果。
- 当业务需要人工决策时，可记录冲突并生成一个“决议变更（Resolution Change）”来收敛。

### 11.1 决议变更（Resolution Change）

决议变更的关键约束是：`parents` 必须覆盖**当时的所有 heads**，这样才能把 DAG 从多 head 收敛回单 head。

- 自动合并：引擎生成 `op=merge` 的 Merge Change（见 6.2）。
- 人工决议：用户选择最终结果后，生成一个普通 Change（例如 `op=patch`），把目标字段改成“最终值”。

提交预条件（推荐）：

- 对于多父变更（`len(parents) > 1`，通常为 merge/resolution），Hub 写入前校验 `parents(set) == current_heads(set)`。
- 若 heads 已变化则拒绝并返回最新 heads，让用户基于最新状态重新决议，避免决议落在过期分支上造成额外分叉。

### 11.2 Revert 与 Undo Revert

**Revert**：决议变更的一种常见形式，把对象状态“改回到某个历史值”。它是 append-only，不删除历史。

**Undo Revert**：撤销 revert 时，再追加一条新的 Change（同样 append-only），其 `parents` 指向当前 heads（通常只有刚才的 revert head），把状态改回 revert 之前的值或其他期望值。

示例（title 并发，先 revert 再撤销）：

```text
heads = {a2(title=2), b2(title=222)}
```

Revert/Resolve：

```json
{ "id":"r1", "parents":["a2","b2"], "op":"patch", "payload":[{"op":"replace","path":"/title","value":"2"}] }
```

Undo Revert：

```json
{ "id":"u1", "parents":["r1"], "op":"patch", "payload":[{"op":"replace","path":"/title","value":"222"}] }
```

---

## 12. Snapshot 与 GC

- Snapshot 解决回放性能问题。
- 达到阈值后生成快照，保留基线 `base_change_id` 与 `base_order_id`。
- GC 时可删除早期 change，但必须保留快照基线后可重放的变更。

---

## 13. 最小落地路径

1. ChangeStore + Heads 表
2. gRPC Sync API（GetHeads/GetChangesAfter/GetChanges/PushChanges）
3. ApplyChange + Merge 策略
4. Snapshot 与 GC
5. NATS JetStream（部署必选，但不作为一致性依赖）

   - 变更通知与任务分发用 NATS JetStream 加速。
   - NATS 不可用时必须可降级：通过 gRPC 对账 + 拉取补齐完成最终一致（只是更慢）。

---

## 14. 结论

Change DAG 是 DoorX 长期可扩展同步的核心。它去掉了对 HLC/NATS 的强依赖，保证离线写入与多节点收敛。合并的关键不是“谁写的晚”，而是“把并发变更显式建模，并用确定性策略收敛”。
