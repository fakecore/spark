# DoorX Infra 层项目状态 Checklist

基于 arch/infra 定义范围，交叉验证 architecture.md、sync-kernel-spec.md、implementation-roadmap.md（Phase 1）、infra-todo.md 与核心链路代码。

最后更新：2026-02-12

---

## 一、核心链路完成度验证

| # | 检查项 | 状态 | 备注 |
|---|--------|------|------|
| 1.1 | ChangeStore 核心表（changes/object_heads/object_state/snapshots/outbox/cursors/archive）已创建 | ✅ 已完成 | migration 与 model 层均齐备 |
| 1.2 | Hub 写入路径（HubPushChanges / ApplyRemoteChangesTx）可用 | ✅ 已完成 | change_store.go 实现完整 |
| 1.3 | Sync gRPC 四个 API（GetHeads/GetChangesAfter/GetChanges/PushChanges）可用 | ✅ 已完成 | change_sync.go |
| 1.4 | Workspace 级 token 鉴权（gRPC 拦截器 + edge client 注入） | ✅ 已完成 | sync_auth_grpc.go + hub_client.go |
| 1.5 | Transactional outbox + publisher（DB commit 后发 JetStream, at-least-once） | ✅ 已完成 | outbox_publisher.go, SKIP LOCKED 多进程安全 |
| 1.6 | JetStream fan-in ingestion（edge ChangeTask → hub 入库 + ack outbox）+ DLQ | ✅ 已完成 | change_ingestor.go |
| 1.7 | Edge 侧滑动窗口（max in-flight + durable acks） | ✅ 已完成 | edge_window.go |
| 1.8 | Edge push/pull loop（push via JetStream, pull via gRPC + cursors, reconcile） | ✅ 已完成 | edge_sync.go |
| 1.9 | Hub materializer（订阅变更通知 → 重放 → 更新 object_state；可选 reconcile/snapshot） | ✅ 已完成 | change_materializer.go |
| 1.10 | GC（changes → changes_archive 归档） | ✅ 已完成 | change_gc.go |

---

## 二、Sync Kernel Spec 硬约束落地验证

| # | 检查项 | 状态 | 备注 |
|---|--------|------|------|
| 2.1 | Parents 归一化（去重 + 排序） | ✅ 已落地 | change_store.go:579 normalizeUUIDSet |
| 2.2 | Change immutability 校验（除 order_id 外字段一致） | ✅ 已落地 | change_store.go:764 |
| 2.3 | object_type 稳定性约束 | ✅ 已落地 | change_store.go:821 |
| 2.4 | order_id 由 Hub 分配，19 位递增，object-scoped advisory lock 串行化 | ✅ 已落地 | change_store.go:814, pg_advisory_xact_lock |
| 2.5 | 缺父处理：PushChanges 返回 missing_parent_ids | ✅ 已落地 | change_store.go:664 |
| 2.6 | 缺父处理：重放时 topo 排序检测 ErrMissingParents | ✅ 已落地 | rebuild.go:56 |
| 2.7 | 缺父处理：edge pull 路径自愈拉取缺父 | ✅ 已落地 | edge_sync.go:537 |
| 2.8 | TopologicalOrder 实现（materialize/rebuild 已使用） | ✅ 已落地 | topo.go:24, infra-todo.md 此条已过时 |
| 2.9 | Snapshot 生成并持久化 | ✅ 已落地 | change_materializer.go maybeCreateSnapshot |
| 2.10 | RebuildFromStorageAndUpsertObjectState 实现（含 snapshot baseline） | ✅ 已落地 | rebuild_storage.go:27 |

---

## 三、多节点同步能力验证

| # | 检查项 | 状态 | 备注 |
|---|--------|------|------|
| 3.1 | 多 Edge 共享同一 workspace 数据并同步 | ✅ 支持 | edge push via JetStream → hub → edge pull via gRPC |
| 3.2 | Hub 多实例 HA（同 PG + 同 NATS）水平扩容 | ✅ 基本支持 | advisory lock 防回退, outbox SKIP LOCKED, durable consumer 共享消费 |
| 3.3 | 多 Hub Active-Active / Hub DR / P2P | ❌ 不支持 | 架构文档明确后续演进 (architecture.md:306) |

---

## 四、P0 级问题（影响正确性/与 spec 不一致）

| # | 检查项 | 严重性 | 现状 | 建议 |
|---|--------|--------|------|------|
| 4.1 | **computeObjectHeads 未 UNION changes_archive** | **BUG** | ✅ 已修复：computeObjectHeads 现在 UNION `changes_archive`；补充断 DAG + archive heads 测试 | - |
| 4.2 | **Edge push 无 gRPC PushChanges fallback** | **可用性缺陷** | ✅ 已修复：JetStream 不可用/Publish 失败时，edge 自动降级为 gRPC `PushChanges`；补充降级测试 | - |
| 4.3 | **NATS Down 时 edge 新对象发现不足** | **数据缺口** | ✅ 已修复：reconcile 增加 best-effort full scan（`GetHeads(object_ids=空)`）用于新对象发现；补充测试 | - |

---

## 五、P1 级问题（可用但离完整形态有差距）

| # | 检查项 | 现状 | 建议 |
|---|--------|------|------|
| 5.1 | **Snapshot 未接入 materializer 主流程** | ✅ 已完成：materializer 主流程使用 snapshot baseline 加速（topo replay + snapshot baseline） | - |
| 5.2 | **JetStream stream 配置过于最小化** | ✅ 已完成：`projecttemplate_changes/projecttemplate_acks/projecttemplate_tasks/projecttemplate_dlq` streams 增加 `MaxAge/MaxBytes/Replicas/Duplicates` | - |
| 5.3 | **多实例启动 stream/consumer 创建竞争** | ✅ 已完成：stream/consumer 采用 Update→Add 幂等收敛 + 竞争容错 | - |
| 5.4 | **安全边界仅 workspace token，无细粒度 capability** | ✅ 已完成：增加 read/write token capability（兼容 legacy `workspace_tokens/default_token`） | - |
| 5.5 | **Edge gRPC 连接无 TLS** | ✅ 已完成：edge client 支持可选 TLS（CA file / server name / skip verify） | - |
| 5.6 | **MySQL 支持名义存在但实际不兼容** | ✅ 已完成：sync kernel 启用/配置时强制要求 Postgres driver（避免误导） | - |
| 5.7 | **DSN 明文日志输出** | ✅ 已完成：DB DSN 日志脱敏（隐藏密码） | - |
| 5.8 | **Snapshot 无删除/保留策略** | ✅ 已完成：snapshot keep-last retention（可配置，默认保留最近 10 个/对象） | - |
| 5.9 | **GC 多实例无分布式锁** | ✅ 已完成：GC transaction 使用 `pg_try_advisory_xact_lock` 互斥 | - |
| 5.10 | **Outbox 无"卡住事件"检测** | ✅ 已完成：outbox stuck pending 检测（日志告警） | - |
| 5.11 | **缺父永久丢失无确定性处理策略** | ✅ 已完成：引入 ignore list（`sync_ignored_changes`）+ node registry/report + `ResolveMissingParent(PRUNE)`，并在 spec 明确“Declared Lost/晚到父节点继续忽略”语义 | - |

---

## 六、P2 级问题（文档明确后续演进）

| # | 检查项 | 现状 |
|---|--------|------|
| 6.1 | 多 Hub / Hub DR / P2P | ✅ 已完成 V1 范围：PG Lease（shared PG 协调）+ Edge reseed；❌ Active-Active/P2P 仍为后续立项 |
| 6.2 | WebSocket / UI 实时通知 | ✅ 已完成：`/api/v1/ws` best-effort 推送 + `/infraSync` WS 刷新（轮询兜底） |
| 6.3 | Router / Execution Node 基础设施 | ✅ 已完成：`execution_nodes/execution_tasks` + `/api/v1/nodes/*` + SKIP LOCKED claim + 回写 `action_run` |
| 6.4 | E2EE / 密钥轮换 | ✅ 已完成 V1 范围：workspace secrets SSE + key rotation/reencrypt；❌ Change DAG 全量 E2EE 非本期目标 |
| 6.5 | 系统级 Metrics (Prometheus) / Tracing (OpenTelemetry) | ✅ 已完成：`/metrics`、HTTP/gRPC/job/lease/ws/exec/secrets 指标、OTel 配置化 |
| 6.6 | Secrets/Vault 统一管理 | ✅ 已完成：env/file/vault provider 抽象 + keyring + node token 管理 |
| 6.7 | Sync 状态 Dashboard（只读 HTTP + Web 页面 /infraSync，轮询刷新） | ✅ 已完成 |

---

## 七、文档与代码一致性验证

| # | 检查项 | 状态 | 备注 |
|---|--------|------|------|
| 7.1 | infra-todo.md "parents topo 排序兜底" 已实现 | ✅ 已更新 | infra-todo.md 已同步为“已补齐” |
| 7.2 | infra-todo.md "object_state 物化策略" 已实现 | ✅ 已更新 | infra-todo.md 已同步为“已补齐” |
| 7.3 | infra-todo.md "Sync 安全边界" 部分实现 | ✅ 已更新 | read/write capability token 已落地（兼容 legacy token） |
| 7.4 | infra-todo.md "Snapshot/GC/Rebuild" 已实现 | ✅ 已更新 | snapshot baseline/retention + GC lock 已落地 |
| 7.5 | architecture.md "NATS Down 降级到 gRPC" 未完整落地 | ✅ 已落地 | edge push gRPC fallback + full reconcile 已补齐 |

---

## 八、测试覆盖验证

| # | 测试包 | 状态 | 覆盖范围 |
|---|--------|------|----------|
| 8.1 | `./internal/data/repository/changedag` | ✅ PASS | ChangeStore 核心读写、archive 查询 |
| 8.2 | `./internal/service/sync` | ✅ PASS | gRPC 同步 API、两节点测试 |
| 8.3 | `./internal/workflow/syncjob` | ✅ PASS | Ingestor、Outbox publisher |
| 8.4 | `./internal/workflow/edgesync` | ✅ PASS | Edge push/pull/reconcile |
| 8.5 | `./internal/workflow/materializejob` | ✅ PASS | Materializer + snapshot |
| 8.6 | `./internal/workflow/gcjob` | ✅ PASS | GC 归档 |
| 8.7 | `./internal/infrastructure` | ✅ PASS | 基础设施初始化 |
| 8.8 | GC 后 computeObjectHeads 正确性测试 | ✅ PASS | 覆盖 archive heads + disconnected DAG 场景 |
| 8.9 | JetStream 不可用时 edge push 降级测试 | ✅ PASS | 覆盖 PublishTask 失败→gRPC PushChanges backfill |
| 8.10 | NATS Down 时全量 reconcile 新对象发现测试 | ✅ PASS | 覆盖 full scan 发现新对象并 pull/materialize |

---

## 九、总体评估

| 维度 | 评估 |
|------|------|
| **最小闭环（V1 Hub 模式）** | 基本跑通，代码层面齐全 |
| **与 Spec 一致性** | 硬约束已落地；P0 偏差已修复 |
| **多 Edge 数据同步** | 正常路径支持；NATS Down 通过 gRPC 对账（full reconcile）与 push fallback 可最终收敛（更慢） |
| **Hub HA 水平扩容** | 基本支持（锁 + 幂等 + durable consumer + GC lock + stream/consumer 幂等创建） |
| **生产就绪度** | 中等（已补 TLS 可选 / stream 保留策略 / DSN 脱敏；仍缺 metrics/tracing/告警与 secrets 统一管理） |
| **文档时效性** | infra-status-checklist.md / infra-todo.md 已同步；后续随架构演进持续更新 |

---

## 十、下一步建议（P2/运维）

1. 增加告警阈值与看板：outbox/sync/materialize/GC/lease/ws/exec/secrets 的积压、延迟、错误率
2. 将 execution node 运行时（runner）接入生产并压测（长任务、重试、断点恢复）
3. 多 Hub Active-Active / P2P（独立立项，含 split-brain 仲裁与跨库数据路径）
4. 是否将 business change DAG 扩展到 E2EE（独立立项，不与本期 SSE 混用）

---

## 十一、演示步骤（离线系统接入 Hub 并同步）

1. 打开 `/infraSync`，选择目标 `workspace_id`。
2. 观察 `sync_nodes`：离线节点上线后 `last_seen_at` 连续刷新，状态从 stale 变为 online。
3. 在离线节点产生变更后恢复网络，观察：
   - `changes` 出现新 `author_node` 与递增 `created_at`
   - Hub lease 概览中的 holder/epoch 可见
4. 若是新 Hub 回灌场景，开启 edge reseed 后观察 `recent changes` 快速增长并最终稳定。
5. 若启用 execution node，观察 `execution_nodes` 与 `execution_tasks`：
   - task `queued -> running -> completed/failed`
   - `claimed_by` 与节点心跳一致
6. 若启用 secrets keyring，观察概览中的 active key 及 key 数量；执行 rotate + reencrypt 后可看到 key_id 切换。
