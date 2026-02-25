# DoorX Design Index

本目录的目标是提供一小组“可执行、权威”的设计文档入口。

**最后更新**: 2026-02-06

---

## Canonical Docs (Start Here)

- `docs/design/architecture.md`：项目定位、核心组件、数据模型、部署形态（含 Hub HA 语义）
- `docs/design/sync-kernel-spec.md`：同步/重放内核硬约束（object-scoped、OrderId、Rebuild、Snapshot/GC、NATS 降级、Hub DR）
- `docs/design/change-dag-design.md`：Change DAG 详细语义 + 端到端示例（偏算法/模型）
- `docs/design/implementation-roadmap.md`：分阶段实施路线与验收点

## Context & Validation

- `docs/design/scene-analysis.md`：用 5 个场景验证架构
- `docs/design/review-responses.md`：评审结论与待决策项（建议开发前过一遍）

## Supporting Docs (Use As Needed)
- `docs/design/workflow-schema.md`：Workflow YAML Schema
- `docs/design/llm-module.md`：LLM 模块设计（Provider/Streaming/Tool Calling/与 Action 集成）
- `docs/design/llm-master-agent-architecture.md`：主 Agent + 文件流 LLM Provider 热刷新架构
- `docs/design/plugin-platform-capability-spec.md`：插件平台与能力契约规范（注册、鉴权、生命周期、调用链）
- `docs/design/plugin-implementation-checklist.md`：插件平台开发执行清单（跨 session 持久化）
- `docs/design/plugin-implementation-checklist.yaml`：插件平台开发执行清单（machine-readable）
- `docs/design/plugin-rule-freeze.md`：插件规则冻结矩阵（优先级、token、consent、delegation）
- `docs/design/plugin-error-catalog.md`：插件错误码目录（重试性与 UI 行为）
- `docs/design/plugin-db-schema-migration-plan.md`：插件数据模型与迁移/回滚计划
- `docs/design/plugin-threat-model-review.md`：插件平台安全威胁模型评审（blockers + mitigations）
- `docs/design/plugin-release-rollout-plan.md`：插件平台发布分阶段计划（alpha/beta/GA，kill switch，回滚准则）
- `docs/design/error-handling.md`：重试/Fallback 机制
- `docs/engineering/ssh-config.md`：SSH/远程执行配置（工程实践）
- `docs/engineering/database-schema-management.md`：数据库 schema/迁移/生成流程（工程实践）
- `docs/design/architecture-extras.md`：附录材料

## Quick Reading Order

1) `docs/design/architecture.md`
2) `docs/design/sync-kernel-spec.md`
3) `docs/design/scene-analysis.md`
4) `docs/design/implementation-roadmap.md`

---

## 概念速查

### 执行类型

| 类型 | 触发 | 终点 | 状态 | 典型场景 |
|------|------|------|------|---------|
| **Action (Simple)** | 用户/事件 | 明确 | 无 | 翻译、查询 |
| **Action (Chain)** | 用户/事件 | 明确 | 无 | 多步骤工具调用 |
| **Workflow (DAG)** | 用户 | 明确 | 检查点 | NAS 视频处理 |
| **RecurringTask** | 定时/Webhook | 无（持续） | Cursor + Memory | 项目追踪 |

### 核心组件

```
┌─────────────────────────────────────────────────────────┐
│  Frontend (Electron/Web)                                │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│  Go Service (本地/云端同构)                              │
│  ├─ Router          — 执行路由                          │
│  ├─ ActionExecutor  — Action 执行                       │
│  ├─ DAGRunner       — Workflow 编排                     │
│  ├─ Scheduler       — 定时任务调度                      │
│  ├─ Sync            — 数据同步（云端主/本地从）          │
│  └─ EventBus        — 事件驱动                          │
└─────────────────────────────────────────────────────────┘
```

### 验证场景

| 场景 | 描述 | 验证点 |
|------|------|--------|
| 1. 本地复制翻译 | 剪贴板触发翻译 | Router、LocalNode、EventBus |
| 2. 本地语音监听 | 实时转写会议 | AudioPipeline、云端增强 |
| 3. NAS 视频整理 | 长时间工作流 | WorkflowEngine、SSH、CredentialStore |
| 4. 本地对话查询 | 多源数据路由 | IntentRecognizer、LANNode |
| 5. 定期项目追踪 | AI 定时唤醒 | Scheduler、RecurringTaskRunner、StateStore |

---

## 文档维护

- 任何“会影响实现”的结论，必须落入 `docs/design/architecture.md` 或 `docs/design/sync-kernel-spec.md`。
- 讨论过程与历史版本不再保存在仓库内，避免干扰当前实现判断。
