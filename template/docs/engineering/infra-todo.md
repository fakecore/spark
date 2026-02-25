# Infra TODO (DoorX)

目标：记录 Infra 层“仍未完成或需持续推进”的事项，作为后续迭代唯一清单。

范围（Infra）：存储/同步内核、事件投递、执行基础设施、密钥管理、可观测性与运维门禁。不包含业务领域规则与业务 UI。

最后更新：2026-02-12

---

## 当前结论（V1）

1. P0/P1/P2（6.1-6.6）在 V1 范围内已落地，可支撑单 Hub（shared PG）+ 多 Edge 的同步与对账。
2. 可通过 `/infraSync` 观察节点、recent changes、lease、execution、secrets 的运行状态。
3. 一致性语义仍以 gRPC 对账为准，WS 仅 best-effort 通知。

---

## 尚未完成（后续独立立项）

1. Multi-Hub Active-Active（跨 PG）自动仲裁与一致性协议。
2. P2P peer 模式（无 Hub，多 writer 冲突仲裁与合并协议）。
3. Change DAG 全量 E2EE（Hub 不可见明文，含密钥分发/撤权/历史重加密体系）。

---

## 工程化补强（建议近期完成）

1. ✅ 提交门禁统一化已落地：
   - `make check-commit`
   - `.githooks/pre-commit`（`make install-githooks` 安装）
   - CI test job 对齐使用同一套 gate
2. 监控告警固化：补齐 dashboard + alert rule（outbox/ingestor/materializer/lease/ws/exec/secrets）。
3. 运行手册：补齐 Hub 异常、NATS 异常、PG 异常、Declared Lost 人工处理的 SOP。

---

## 对账同步验收入口（可立即执行）

1. 自动化测试：
   - `make test-sync-kernel`
   - `make test-infra-outbox`
   - `go test -v -count=1 ./internal/workflow/edgesync -run TestEdgeSync_NoNATS_DegradesToGRPC_PushAndDiscover`
2. 界面验收：
   - 打开 `/infraSync`，观察节点上线与 `last_seen_at` 刷新。
   - 制造离线写入再恢复网络，观察 recent changes 增长并最终稳定。
   - NATS 不可用时观察仍可通过 gRPC 对账收敛（速度下降但最终一致）。
