# DoorX Sync Kernel Spec (Object-Scoped)

> **版本**: v1.0
> **日期**: 2026-02-06
> **状态**: 草案

本文件定义 DoorX 的“同步与重放内核”（Sync Kernel）必须满足的硬约束与行为规范。

目标：让任意节点在“相同的 Change 集合”下，最终得到 **完全一致** 的 `object_heads` 与 `object_state`（确定性收敛）；并且在乱序、重复投递、部分失败、NATS 不可用的情况下仍可通过 gRPC 对账补齐恢复一致。

本规范借鉴 any-sync 的 objecttree 核心机制（attached/unAttached/waitList、OrderId、rebuildFromStorage、snapshot counter），但以 DoorX 的 Postgres + ChangeStore 模型落地。

---

## 1. 核心决策

### 1.1 分片单位：每个 object 一棵树

- **对象树（ObjectTree）** 以 `(workspace_id, object_id)` 为作用域。
- 任何与 DAG 构建、heads 维护、重放、快照、GC 相关的计算，都必须在 object-scoped 范围内完成。

理由：锁粒度清晰、吞吐可控、与 `object_heads/object_state` 表结构天然匹配。

### 1.2 NATS JetStream：部署必选，但不作为一致性依赖

- NATS 用于：任务队列 + 变更通知（加速对账）。
- **一致性来源**：以 ChangeStore（PostgreSQL 的 `changes/object_heads/object_state/snapshots`）为权威。
- **NATS Down 必须可降级**：客户端/节点可通过 gRPC 定时对账与拉取补齐最终收敛（更慢但正确）。

---

## 2. 术语与不变量（Invariants）

### 2.1 Change 与 DAG

- Change 是最小同步单元：`id`, `workspace_id`, `object_id`, `object_type`, `parents[]`, `author_node`, `op`, `payload`, `created_at`。
- DAG 的边由 `parents` 定义；同一 object 的 changes 形成一个（可能暂时不连通的）有向无环图。

### 2.2 Heads

- `heads` 表示某 object 当前“未被任何 change 作为 parent 引用”的叶子集合。
- 任意节点在相同 change 集合下计算出来的 heads 必须一致（顺序无关）。

### 2.3 确定性（Determinism）

对同一 object：

- 给定相同的 changes 集合，`object_heads` 与 `object_state` 的最终结果必须完全一致。
- Merge/Apply 禁止使用非确定性来源：`time.Now()`、随机数、外部 I/O 返回值作为合并依据。
- 合并策略必须版本化：同一 `object_type` 的策略变更必须通过 `strategy_version` 显式切换。

### 2.4 幂等（Idempotency）

- Change 的写入必须幂等：重复 `PushChanges` 同一 change 不应产生副作用。
- NATS 投递语义按 at-least-once 设计，消费者必须按 `change_id`/`task_id` 去重。

### 2.5 Change 不可变性与一致性校验（Integrity）

对同一 `workspace_id`：

- **Change ID 不可变**：同一 `change_id` 一旦存在，则除 `order_id`（允许从空->被 Hub 回填）外，其余字段必须完全一致：
  `workspace_id/object_id/object_type/parents/op/payload/author_node/created_at`。
- **不一致必须拒绝**：若收到相同 `change_id` 但内容不一致，Hub 必须拒绝写入并返回 `ErrChangeMismatch`（视为数据损坏或恶意节点）。
- **parents 归一化**：`parents[]` 在语义上视为集合（顺序无关）；入库与对外返回前必须：去重 + 按 UUID 字典序排序。
- **heads 归一化**：`heads[]` 在语义上视为集合（顺序无关）；入库与对外返回前必须：去重 + 按 UUID 字典序排序。
- **object_type 不可漂移**：同一 `(workspace_id, object_id)` 的 `object_type` 必须保持不变；若出现不一致必须拒绝 `ErrObjectTypeMismatch`。

---

## 3. 数据模型（PostgreSQL）

### 3.1 changes 表（追加写）

最小字段（示意）：

```sql
CREATE TABLE changes (
  id UUID PRIMARY KEY,
  workspace_id VARCHAR(64) NOT NULL,
  object_id UUID NOT NULL,
  object_type VARCHAR(64) NOT NULL,
  parents UUID[] NOT NULL,

  -- OrderId：用于增量遍历/重放，不用于业务胜负
  -- 说明：
  -- - V1: 由 Hub 分配 canonical order_id，并在同步时回填到各副本（跨副本一致）。
  -- - 非 Hub 节点离线新增 Change 时允许 order_id 为空；正确性不依赖 order_id。
  order_id VARCHAR(64),

  author_node VARCHAR(64) NOT NULL,
  op VARCHAR(16) NOT NULL,
  payload JSONB NOT NULL,
  created_at BIGINT NOT NULL
);

-- order_id 非空时在 object 内唯一，既用于增量扫描也用于约束正确性
CREATE UNIQUE INDEX uq_changes_object_order ON changes (workspace_id, object_id, order_id)
  WHERE order_id IS NOT NULL;
CREATE INDEX idx_changes_ws_id ON changes (workspace_id, id);
```

关于分区：建议 V1 先按时间（月）RANGE 分区，以便 GC 时可以 DROP PARTITION。workspace 级 LIST 分区在 workspace 数量增长后维护成本较高，建议后续用 HASH(workspace_id) 替代。

### 3.2 object_heads 表（派生数据）

```sql
CREATE TABLE object_heads (
  workspace_id VARCHAR(64) NOT NULL,
  object_id UUID NOT NULL,
  heads UUID[] NOT NULL,
  updated_at BIGINT NOT NULL,
  PRIMARY KEY (workspace_id, object_id)
);
```

### 3.3 object_state 表（物化查询态）

```sql
CREATE TABLE object_state (
  workspace_id VARCHAR(64) NOT NULL,
  object_id UUID NOT NULL,
  object_type VARCHAR(64) NOT NULL,
  state JSONB NOT NULL,
  updated_at BIGINT NOT NULL,
  PRIMARY KEY (workspace_id, object_id)
);
```

### 3.4 snapshots 表（可选但推荐）

快照必须携带可重放基线信息：

- `base_change_id`：快照覆盖的基线 change
- `base_order_id`：快照对应的 order 游标（用于增量加载）

```sql
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

## 4. OrderId（增量遍历键）

### 4.1 语义

- `order_id` 用于：
  - 增量同步：从上次 `order_id` 继续拉取
  - 构建/重建：按 order 流式迭代 changes
  - 快照基线：以 `base_order_id` 作为快照后的起点

### 4.2 生成原则

- 同一 `(workspace_id, object_id)` 下，**已分配的** `order_id` 必须单调递增。
- 并发写入会发生：因此必须定义一个可重复的“序号分配”方法。

建议两种实现路径（择一落地）：

1) **Hub 分配**：Hub 接收 `PushChanges` 时，为每个 change 分配 `order_id`（推荐，简单可靠）。
2) **客户端分配**：客户端生成 order_id，但必须保证与 Hub 的收敛规则一致，复杂度更高。

注意：`order_id` 不作为一致性/正确性的来源；业务冲突由 merge 策略解决。
（merge 策略允许使用 **Hub 分配的 canonical `order_id`** 作为确定性的 tie-break，但不得要求离线时必须具备 `order_id` 才能写入。）

V1 决策（落地约束）：

- **采用 Hub 分配**：Hub 为同一 `(workspace_id, object_id)` 维护单调递增序号，并在写入 `changes` 时分配 canonical `order_id`。
- **跨副本一致**：同一个 `change_id` 在所有副本上最终应拥有相同的 `order_id`（以 Hub 为准）。
- **离线可写**：非 Hub 节点离线新增 change 时允许 `order_id` 为空；在 `PushChanges` 被 Hub 接收后由 Hub 回填（同步返回或后续对账补齐）。
- **单节点模式**：本地 `Go Service + Postgres` 即该 workspace 的 Hub 角色，因此本地写入即可分配 `order_id`，无需额外部署独立 Hub 服务。

### 4.3 格式与比较（必须）

- `order_id` 作为“可比较的游标”，必须满足：**字符串比较顺序 == 逻辑顺序**。
- V1 规范格式：`order_id = %019d`（19 位十进制、左侧补 0，范围为 `1..9_223_372_036_854_775_807`）。
- `order_id` 在非 Hub 节点可为空（`NULL` 或空字符串；推荐落库为 `NULL`）。

### 4.4 Hub 分配算法（必须满足的可见性不变量）

为了让 `order_id` 可作为增量拉取游标，Hub 必须保证：对同一 `(workspace_id, object_id)`，**不会出现“提交晚的 change 拿到更小/相等的 order_id”**，否则客户端会因为游标推进而永久漏同步。

硬约束：

- `order_id` 分配必须与写入 `changes` 处于**同一个 DB 事务**内。
- 对同一 `(workspace_id, object_id)`，分配 `order_id` 的写入必须**串行化**（二选一实现即可）：
  1) `pg_advisory_xact_lock(hash(workspace_id, object_id))`（推荐，避免额外表）；或
  2) 锁住某个 object 级行（例如 `object_heads` 或单独的 cursor 表）使用 `SELECT ... FOR UPDATE`。

建议实现（示意）：

1) 在事务内获取 object 级锁
2) 读取该 object 当前最大 `order_id`（或 cursor）
3) 为本批次待写入 changes 依次分配 `NextOrderId(max)`
4) 批量 upsert changes（幂等）并返回 `change_id -> order_id` 映射
5) 提交事务

说明：`order_id` 不要求连续，但必须单调递增且对外可作为游标安全推进。

### 4.5 回填（Backfill）与不可变规则（必须）

- Hub 接收 `PushChanges` 时，若 change 的 `order_id` 为空：
  - Hub 必须分配 canonical `order_id` 并持久化；
  - Hub 必须在响应中返回 `change_id -> order_id` 映射；
  - 客户端/节点收到后必须更新本地 ChangeStore（把 `order_id` 从空回填为 canonical 值）。
- `order_id` 只允许发生一次变更：`NULL/"" -> canonical`；一旦有 canonical 值，后续不得改变。

### 4.6 增量游标（Cursor）推进规则（必须）

- 当节点使用 `order_id` 做增量拉取时，其本地游标只能推进到“本次成功持久化并进入 ObjectTree 的最大 order_id”。
- Hub 必须保证：对同一 object，未来不会再出现 `order_id <= 已下发游标` 的新 change。

---

## 5. ObjectTree：乱序容忍与批量 flush

### 5.1 内存结构（借鉴 any-sync）

每个 object 维护一棵内存树：

```go
type ObjectTree struct {
  workspaceID string
  objectID    uuid.UUID
  objectType  string

  attached   map[uuid.UUID]*Change
  unattached map[uuid.UUID]*Change
  // parent -> children waiting for this parent
  waitList   map[uuid.UUID][]uuid.UUID

  heads map[uuid.UUID]struct{}

  // last applied order cursor (optional, for incremental rebuild)
  lastOrderID string
}
```

### 5.2 AddChanges 行为

输入：一批 changes（来自 gRPC GetChanges 或本地新增）。

要求：

- 去重：已存在的 change_id 必须跳过。
- 乱序：若 parents 未全部到达，则进入 `unattached` 并登记到 `waitList[parent]`。
- 解锁：当某个 parent 变为 attached 时，应立即尝试 attach waitList 中等待它的 children。

输出：

- 可 attach 的 change 必须进入 attached。
- heads 必须更新（增删叶子）。

### 5.3 Flush 行为（单事务）

当收到一批 changes 后，不应逐条写 DB + 逐条 Apply：

- 先 `AddChanges` 在内存里解决乱序/缺父
- 再在一个事务里：
  1) 批量插入 changes（幂等 upsert）
  2) 更新 object_heads
  3) 应用/合并后写 object_state（见第 6 节）
  4) 必要时写 snapshots

事务失败则全部回滚。

---

## 6. Apply / Merge / Registry

### 6.1 MergeRegistry（按 object_type）

- 每个 `object_type` 必须注册一个 merge 策略（或声明该类型暂不物化）。
- 策略必须包含 `strategy_version`，用于重放兼容。

未知类型建议行为：

- 允许 changes 入库与同步
- 不更新 object_state（或写入 opaque state），直到策略上线后可通过 rebuild 回放生成

### 6.2 Merge Change 的确定性

当出现多 heads：

- 使用策略对并发分支合并得到新状态
- 生成一个 Merge Change：其 `parents` 指向所有 heads

硬约束：

- **Merge Change ID 建议确定性生成**（避免多个节点生成不同 merge 导致再次分叉）：
  `merge_id = hash(sorted(parents) + object_type + strategy_version + merge_payload_canonical)`

### 6.3 冲突、决议变更（Resolution Change）与 Revert/Undo

当同一 object 出现 **多 heads** 时，有两类处理路径：

1) **自动合并（推荐优先）**：若该 `object_type` 的策略能确定性地合并并发分支，则生成一个 **Merge Change**（见 6.2），其 `parents` 指向当时的所有 heads，使 DAG 收敛回单 head。
2) **人工决议（必要时）**：若策略无法自动合并（例如同一字段出现业务上不可自动决胜的并发写），则进入冲突态（可选记录到 `conflict_logs`），等待用户/业务策略生成一个“决议变更”收敛。

**决议变更（Resolution Change）定义**：

- 它本质上仍是一个普通 Change（`op` 可为 `set`/`patch`/`delete` 等）。
- 关键约束：其 `parents` **必须同时指向当时的所有 heads**，以保证一次决议能把 DAG 从多 head 收敛回单 head。
- payload 必须显式给出最终结果（例如 patch 出最终 title），而不是只写“revert 到某个 change_id”并依赖运行时推导（避免不确定性）。

提交预条件（防止竞态导致“无意分叉”）：

- 对于 **多父变更**（通常是 merge/resolution change，`len(parents) > 1`），Hub 在写入前应校验：`parents(set) == current_heads(set)`。
- 若不满足，则返回冲突错误（例如 `ErrHeadsChanged`）并携带最新 `heads`，客户端应刷新并提示用户基于最新 heads 重新决议后再提交。

**Revert**：是一种常见的决议变更，语义是“把对象状态改回到某个历史版本/某个值”，但仍遵循 append-only（不删除历史）。

**Undo Revert（撤销 Revert）**：再新增一个新的决议变更（append-only），其 `parents` 指向当前 heads（通常只有刚才的 revert head），把状态改回 revert 之前的值或其他期望值。历史中会同时存在“revert”与“undo revert”，物化态以最新 heads（或最新决议）为准。

示例（同一对象 `conv_01` 的 title 并发更新）：

```text
base: c0(title="init")

A: a1(title="1", parents=[c0]) -> a2(title="2", parents=[a1])
B: b1(title="45", parents=[c0]) -> b2(title="222", parents=[b1])

heads = {a2, b2}  // 冲突
```

用户选择“回到 2”（revert/resolve）：

```json
{
  "id": "r1",
  "object_id": "conv_01",
  "parents": ["a2", "b2"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/title","value":"2"}
  ]
}
```

此时：

```text
heads = {r1}
```

之后用户又想撤销该 revert（undo revert），只需再追加一条 change：

```json
{
  "id": "u1",
  "object_id": "conv_01",
  "parents": ["r1"],
  "op": "patch",
  "payload": [
    {"op":"replace","path":"/title","value":"222"}
  ]
}
```

最终：

```text
heads = {u1}
```

若 undo revert 期间同时发生了其他并发写入，`parents` 需要覆盖“当时的所有 heads”，以避免再次分叉。

---

## 7. Rebuild from Storage（容错基座）

触发场景：

- AddChanges 检测到结构不一致（例如形成环、缺根且无法通过补齐解决）
- ApplyChange/ValidateChange 失败
- Flush 成功后内存态与 DB 不一致

行为：

1) 读取该 object 的最新 snapshot（若存在）作为基线 `state` 与 `base_order_id`
2) 重放 canonical changes：
   - `order_id IS NOT NULL AND order_id > base_order_id` 按 `order_id` 升序读取
3) 重放 local-only changes（离线未回填 order_id 的变更）：
   - `order_id IS NULL` 按 `(created_at, id)` 确定性排序读取
4) 将以上 changes 统一交给 ObjectTree 的 `AddChanges`，重建 attached/unattached/heads/state
5) 写回 `object_heads/object_state`（可选）

快照约束（必须）：

- snapshots 仅作为优化，不参与同步正确性。
- 创建 snapshot 时，该 object 的所有已知 changes 必须都具备 canonical `order_id`（否则 snapshot 无法用 `base_order_id` 做稳定增量回放）。

要求：rebuild 必须是确定性的，且可重复执行。

---

## 8. 同步协议与 NATS 降级

### 8.1 gRPC 对账（权威路径）

#### 8.1.1 RPC 列表（V1）

- `GetHeads(workspace_id, object_ids[]) -> { entries[] }`
- `GetChanges(workspace_id, change_ids[]) -> { changes[], missing_ids[] }`
- `GetChangesAfter(workspace_id, object_id, after_order_id, limit) -> { changes[], next_after_order_id, has_more }`
- `PushChanges(workspace_id, changes[]) -> { accepted, assigned_order_ids[], missing_parent_ids[], rejected[], ignored_change_ids[] }`
- `GetIgnoreList(workspace_id, object_id) -> { change_ids[] }`
- `RegisterNode(workspace_id, node_id, role, ...) -> {}`
- `ReportMissingParents(workspace_id, node_id, object_id, missing_parent_ids[]) -> {}`
- `ResolveMissingParent(workspace_id, object_id, missing_parent_id, strategy=PRUNE, ...) -> { ignored_change_ids[], ... }`

> 说明：V1 仍保留 `GetChanges(workspace_id, change_ids[])` 作为修复路径（缺父补齐/自愈），并新增 `GetChangesAfter` 作为高效增量拉取路径。

#### 8.1.2 GetHeads 语义（必须）

- 返回的 `heads[]` 必须去重 + 排序（见 2.5）。
- 推荐额外返回：
  - `last_order_id`（该 object 当前最大 canonical `order_id`），用于客户端判断是否追平。
  - `heads_updated_at`（Hub `object_heads.updated_at`），用于在“heads 变化但 order_id 未前进”的情况下仍能对账收敛（见 8.1.7）。

#### 8.1.3 GetChangesAfter 语义（必须）

- 仅返回 Hub 已分配 canonical `order_id` 的 changes（在 Hub 上应等价于“该 object 的全量 changes”）。
- 返回的 changes 必须按 `order_id` 升序排列。
- `after_order_id` 为空表示从头开始；否则返回 `order_id > after_order_id` 的 changes。
- `next_after_order_id` 等于本次返回 changes 中最后一个 change 的 `order_id`（若本次为空则保持不变）。

#### 8.1.4 GetChanges（按 ID 修复）语义（必须）

- 用途：当节点在 Apply/AddChanges 过程中发现缺父（unattached）时，用父 change_id 精确拉取补齐。
- Hub 响应应区分：已找到的 `changes[]` 与未找到的 `missing_ids[]`。

#### 8.1.5 PushChanges 语义（必须）

Hub 在处理 `PushChanges` 时：

- 必须执行 2.5 的一致性校验（`ErrChangeMismatch` / `ErrObjectTypeMismatch`）。
- 必须为所有接受的 changes 分配 canonical `order_id`（若请求中为空则回填；若请求中已有值也以 Hub 为准）。
- 允许乱序与缺父：parents 未到达时可先入库为 unattached（ObjectTree 机制），并在响应中返回 `missing_parent_ids[]` 提示对端补齐。
- **注意**：由于允许缺父写入，canonical `order_id` **不等价于拓扑序**；正确的重放顺序必须以 parents DAG 的 **拓扑排序** 为准（order_id 仅作为增量游标/性能优化）。
- 若 change 被 Hub 判定为“已被忽略/剪枝”（见 8.1.7），Hub 仍可接受该 change（保留审计），但必须在响应中返回 `ignored_change_ids[]`，并保证该 change 不进入 canonical DAG（不影响 heads/state，且不会再通过 GetChangesAfter/GetChanges 对外提供）。

多父变更（merge/resolution，`len(parents) > 1`）提交预条件（继承 6.3，必须）：

- Hub 写入前必须校验：`parents(set) == current_heads(set)`。
- 若不满足，必须拒绝并返回 `ErrHeadsChanged`，并携带最新 heads（用于前端刷新与重试）。

回填（必须）：

- `assigned_order_ids[]` 必须覆盖本次 accepted 的所有 changes（至少包含“从空回填”的条目）。
- 对同一 `change_id` 的重复 Push，Hub 必须返回相同的 `order_id`（幂等）。

#### 8.1.6 推荐对账算法（可实现为后台循环）

1) 拉取 heads：`GetHeads`
2) 增量拉取：对每个 object 使用 `GetChangesAfter(after_order_id)` 直到追平
3) Apply：收到 changes 后执行 ObjectTree `AddChanges` + `Flush`
4) 自愈补齐：若仍存在 unattached（缺父），收集缺失 parent ids，调用 `GetChanges` 补齐并重复 Apply
5) 推送本地新增：`PushChanges`；若返回 `missing_parent_ids[]`，则补齐并重试

#### 8.1.7 缺父永久丢失（Declared Lost）与确定性剪枝（必须）

背景：系统允许缺父写入以提升可用性与离线写入能力，但在极端情况下，某个 parent change 可能在所有节点都不存在（真正丢失），导致 materialize/rebuild 永远返回 `ErrMissingParents`，对象查询态无法前进。

V1 策略：把复杂度留在 Hub，并提供一条**确定性**的“剪枝/隔离（ignore）”路径，使所有节点最终收敛到同一结果。

核心机制：

- Hub 维护 `sync_ignored_changes`（ignore list）：被 ignore 的 change **不属于 canonical DAG**，必须从：
  - `GetChangesAfter` / `GetChanges` 输出
  - `GetHeads`/`object_heads` 计算
  - materialize/rebuild 的输入集合
  中排除（确定性收敛）。
- Hub 维护 workspace node registry（`RegisterNode`）与 missing-parent 观测上报（`ReportMissingParents`）作为“宣告丢失”的证据来源。

宣告丢失（Declared Lost）的推荐前置条件（建议）：

1) **节点集合闭合**：workspace 的“预期节点集合”可枚举（node registry），并明确哪些节点已退役。
2) **通信确认**：预期节点均在 staleness 窗口内有心跳（last_seen_at 足够新），且都上报“缺父”（或达到操作员设定的最小确认数）。
3) **等待窗口**：missing parent 已持续超过一个合理时间窗（覆盖正常延迟/重试/备份恢复窗口）。

一旦满足条件，操作员/Hub 执行：

- `ResolveMissingParent(strategy=PRUNE)`：
  - 将 `missing_parent_id` 本身写入 ignore list（即使该 change 当前并不存在），用于处理“晚到父节点”。
  - 计算并写入 ignore list：所有依赖该 parent 的 descendants（传递闭包）。
  - 重新计算并更新 `object_heads`，并触发/执行 materialize 使 `object_state` 恢复可用。

晚到父节点（关键语义）：

- 若某 parent 已被 Declared Lost，之后该 parent change 又“回来了”（晚到/延迟恢复/补发），**默认必须继续忽略**（进入 ignore list，不自动复活 canonical 历史），以避免各节点在不同时间点“回滚/翻转”导致非确定性。
- 若确实要复活，需要显式的人工流程（撤销 ignore + 重建 + 校验），不作为 V1 默认行为。

举例（PRUNE）：

- 当前 object 的 DAG 中存在：
  - `P`（parent，缺失，所有节点均找不到）
  - `C1` parents=`[P]`
  - `C2` parents=`[C1]`
  - `D`（另一条分支/不依赖 P）
- 在未处理前，materialize 会因 `C1` 缺父 `P` 而长期 `ErrMissingParents`。
- 当满足“节点集合闭合 + 通信确认 + 等待窗口”后执行 `ResolveMissingParent(PRUNE)`：
  - ignore: `{P, C1, C2}`
  - canonical 重放集合变为 `{D,...}`，materialize 恢复可用；`object_heads` 会从 `{C2, D}` 收敛到 `{D}`（示意）。
  - 若之后 `P` 晚到：仍被 ignore，不自动改变 canonical 历史；需要人工流程才允许“复活”。

### 8.2 NATS 通知（加速路径，不影响正确性）

- Change 写入成功后发布通知到 `projecttemplate.changes.*`，触发订阅方更快发起对账。
- 消息建议携带：`workspace_id`, `object_id`, `new_head_ids[]` 或 `new_change_ids[]`。
- JetStream 采用 at-least-once：
  - 发布侧建议用 `Nats-Msg-Id = change_id` 以在 DuplicateWindow 内去重。
  - 消费侧必须按 `change_id` 幂等去重。

### 8.3 NATS Down 降级

当 NATS 不可用：

- 不依赖通知；由客户端/节点定时触发 gRPC heads 对账
- 最终一致性仍需保证

### 8.4 DB 与 NATS 的原子性（Transactional Outbox，Hub 必须）

目标：避免“DB 已提交，但消息未发布”的空洞。

硬约束：

- Hub 在写入 changes 的同一事务内，必须把要发布的事件写入 outbox 表；发布 NATS 必须由 outbox 异步投递器完成。
- outbox 投递必须按 at-least-once；允许重复投递，但消费者必须按 `change_id`/`task_id` 幂等去重（见 2.4）。

最小 outbox 表（示意）：

```sql
CREATE TABLE outbox_events (
  id UUID PRIMARY KEY,
  workspace_id VARCHAR(64) NOT NULL,
  topic VARCHAR(128) NOT NULL,
  dedup_key VARCHAR(128) NOT NULL,
  payload JSONB NOT NULL,
  status VARCHAR(16) NOT NULL, -- pending|published|failed
  attempts INT NOT NULL DEFAULT 0,
  next_attempt_at BIGINT NOT NULL,
  last_error TEXT,
  created_at BIGINT NOT NULL,
  published_at BIGINT
);

CREATE UNIQUE INDEX uq_outbox_dedup ON outbox_events (workspace_id, topic, dedup_key);
CREATE INDEX idx_outbox_pending ON outbox_events (status, next_attempt_at);
```

投递要求：

- 发布变更通知时，发布侧建议设置 `Nats-Msg-Id = dedup_key`（通常为 `change_id`），并仍以消费侧去重为最终保证。

### 8.5 限流与分页（必须）

- `GetChangesAfter` 必须提供 `limit`，并在响应里给出 `has_more`/`next_after_order_id`，用于分页与 backpressure。
- `GetChanges(change_ids[])` 必须限制每次请求的 ids 数量（例如 1k），超限返回 `ErrTooManyIds`。

---

## 9. Hub 故障切换与灾难恢复（V1：人工仲裁）

本节定义 Hub 作为“中央协调点”时的故障语义与恢复流程。V1 采用人工方式选择新 Hub，以避免自动选举引入 split-brain 风险。

注意：Hub 的 **HA（多实例/集群）** 目标是提高可用性与容量，但推荐形态仍应保持 **单个逻辑可写入口**。
HA 不等于“多 Hub Active-Active 同时可写”；后者会引入 split-brain，需要额外仲裁与协议约束。

### 9.1 目标与非目标

目标：

- Hub 临时不可用时，终端仍可本地读写与产生日志（local-first）。
- Hub 数据丢失时，可由任意终端或新服务器重建 Hub，并通过各终端的 ChangeStore 回灌恢复到最终一致。

非目标（V1 不做）：

- 自动选主（quorum/raft）
- 无中心 P2P 发现与直连同步

### 9.2 故障分类

1) **Hub 服务不可用，但数据仍在**：例如进程挂掉/网络隔离。
   - 影响：跨设备同步暂停；本地仍可写。
   - 恢复：Hub 恢复后，通过常规 gRPC 对账补齐收敛。

2) **Hub 数据丢失（灾难）**：例如数据库损坏且无备份。
   - 影响：跨设备同步暂停；需要重建 Hub。

### 9.3 人工切换流程（选定新 Hub）

V1 要求：在任何时刻，一个 workspace 只允许存在一个“被承认的 Hub”。

建议操作流程：

1) 人工指定新的 Hub 入口（例如新的 server 地址 + 新的 workspace hub_id）。
2) 通过配置分发/二维码/邀请链接让所有终端更新其 Hub 地址。
3) 旧 Hub 若仍存活，必须被显式下线或隔离，避免双 Hub 并存。

### 9.4 从终端回灌重建 Hub（Reseed）

假设新 Hub 的 ChangeStore 为空或不完整。

对每个终端节点：

1) 连接新 Hub，进行 workspace 级握手。
2) `GetHeads` 对账：终端拿到 Hub 当前 heads（可能为空）。
3) 终端计算“Hub 缺失的 change_id 集合”，并批量 `PushChanges` 回灌。
4) Hub 幂等写入 changes，并重放更新 `object_heads/object_state`。
5) 重复步骤直到对账无差异。

要求：

- PushChanges 必须幂等（重复回灌不会产生副作用）。
- Hub 必须能处理乱序与缺父（ObjectTree 机制 + rebuild）。
- 若终端之间存在分歧（各自离线写入），Hub 必须通过 Merge 收敛为确定性结果（由策略版本化保证）。

### 9.5 Split-brain 防护（V1 最小约束）

V1 不做自动仲裁时，最小要求是“组织流程约束”：

- **单 Hub 约束**：人工操作必须保证同一 workspace 只有一个可写 Hub。
- **写入端约束**：终端只允许向其当前配置的 Hub PushChanges。

未来（V2+）可引入：

- workspace 级 lease（租约）/epoch（任期）机制
- 多节点仲裁（quorum）或基于云服务的轻量仲裁

### 9.6 V1 实现语义（2026-02）

为降低 P2 收尾复杂度，V1 已落地如下约束：

1) **Leader 只做 PG Lease（shared PG 内部协调）**
- 数据模型：`hub_leases(workspace_id, holder_id, epoch, acquired_at, renewed_at, expires_at)`
- 作用范围：Hub 后台协调与 admin 动作（如定时任务、状态暴露），不是跨库自动仲裁协议
- 非目标：不引入 NATS KV lease，不实现跨 PG Active-Active 自动选主

2) **PushChanges 不是 leader-only**
- 在 shared PG 架构下，写入一致性仍以数据库事务与约束为真相源
- lease 用于“避免多实例重复协调”，而不是限制所有写流量只能进 leader

3) **Reseed 仍是人工仲裁流程**
- 当新 Hub 空库或缺数据时，边缘节点可按 canonical `order_id` 回灌
- 该流程是 V1 DR 操作手册的一部分，不代表跨库自动故障切换

4) **WebSocket 事件仅 best-effort**
- subject 规范：`projecttemplate.events.<workspace_id>.<event_type>.<object_id>`
- 连接鉴权：建连校验 JWT；长连接周期校验，过期主动 close，客户端重连
- 一致性来源：gRPC/HTTP 对账（`GetHeads`/`GetChangesAfter`），WS 不承担一致性语义

---

## 10. 最小测试矩阵（建议）

必须通过的用例（每条都应可自动化）：

- 乱序到达：child 先于 parent 到达，最终应 attach 成功并 heads 正确
- 重复投递：重复 Push 同一 change，不应改变 heads/state
- change mismatch：重复 Push 同一 change_id 但内容不同，必须拒绝 `ErrChangeMismatch`
- 并发分叉：两端并发产生不同 head，最终通过 merge 收敛到同一 head
- merge change 重复生成：不同节点同时生成 merge，不应导致永久分叉（依赖确定性 merge_id 或仲裁规则）
- snapshot + GC：GC 后仍能从 snapshot 基线重放得到一致 state
- order_id backfill：客户端离线 change(order_id=NULL) Push 到 Hub 后，Hub 回填 order_id；客户端更新后可用 cursor 增量拉取且不漏变更
- cursor 安全性：并发写入同一 object 时，`GetChangesAfter` 不得出现“先看到更大 order_id，后出现更小/相等 order_id”的情况
- outbox：模拟 DB commit 后进程崩溃，恢复后 outbox 仍能补发 NATS 通知；且重复通知不影响正确性
- NATS down：无通知情况下靠轮询对账最终收敛
