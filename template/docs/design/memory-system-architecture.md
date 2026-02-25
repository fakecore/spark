# DoorX Memory System 架构设计

> **类型**: 核心设计
> **状态**: 🚧 草案
> **最后更新**: 2026-02-25
> **前置依赖**: `llm-master-agent-architecture.md`, `architecture.md`, `plugin-platform-capability-spec.md`

---

## 目录

- [1. 背景与目标](#1-背景与目标)
- [2. 关键决策 (ADR)](#2-关键决策-adr)
- [3. 接口契约 (Interface Contract)](#3-接口契约-interface-contract)
- [4. 记忆类型体系](#4-记忆类型体系)
- [5. 数据模型](#5-数据模型)
- [6. 场景配置 (Scene Profile)](#6-场景配置-scene-profile)
- [7. 核心流程](#7-核心流程)
- [8. 检索策略](#8-检索策略)
- [9. 生命周期管理](#9-生命周期管理)
- [10. 代码布局](#10-代码布局)
- [11. 集成点](#11-集成点)
- [12. 分阶段实施](#12-分阶段实施)
- [13. 验收标准](#13-验收标准)
- [14. 非目标](#14-非目标)
- [更新日志](#更新日志)

---

## 1. 背景与目标

### 1.1 问题

DoorX 定位为**全能型个人 Agent**——不仅能做项目开发，也能陪伴聊天、辅助学习、管理任务。当前 LLM 层（`internal/biz/llm`）是**无状态**的：每次 `Chat(ctx, req)` 调用都从零开始，Agent 不记得昨天的对话，不知道用户的偏好，不了解项目的架构规则。

现有的短期方案（如 Claude Code 的 MEMORY.md）存在硬伤：
- 200 行硬截断，不可扩展
- 无相关性过滤，每次全量注入
- 无结构化查询能力
- 单机文件，无多设备同步

### 1.2 目标

设计一套**通用记忆系统**，满足以下要求：

1. **全场景覆盖**：陪伴/聊天、学习、项目开发、任务管理等场景，通过 Scene Profile 切换记忆策略。
2. **Change DAG 统一存储**：所有记忆作为 Change DAG 的 object_type 写入，天然获得多设备同步能力。
3. **结构化优先**：V1 使用标签/类型/路径等结构化检索，接口预留 embedding 语义检索位。
4. **自动化生命周期**：记忆有衰减、强化、整合机制，不需要用户手动管理。
5. **与 Master Agent 深度集成**：记忆提取和注入是 Agent 执行循环的一部分，不是独立旁路。

### 1.3 参考系统

- **Claude Code**: MEMORY.md 方案——简单但不可扩展，作为反面参考。
- **OpenMemory (CaviraOSS)**: Hierarchical Sectored Graph 架构——选择性借鉴其 temporal facts、三级衰减、SimHash 去重、Reflection 整合、复合评分机制。不采用其 5-sector 分类、regex 分类器、multi-sector embedding、waypoint graph。

---

## 2. 关键决策 (ADR)

### ADR-1: Change DAG 统一存储

所有记忆写入现有 Change DAG（`object_type = "memory"` / `"landmark"`），查询走 `object_state` 物化视图。

**理由**:
- 多设备同步零成本（复用 `GetHeads` / `PushChanges` / `GetChangesAfter`）。
- 冲突解决复用 DAG 合并语义。
- 不引入新的存储基础设施。

**代价**:
- 复杂查询（如按 weight 排序、tags 过滤）需要在 `object_state.state` JSONB 上建索引。
- V1 可接受：记忆量级在万条以内，JSONB 查询性能足够。

### ADR-2: Scene Profile 驱动，不硬编码场景

不同场景（聊天/开发/学习）的记忆策略差异通过 **Scene Profile 配置** 实现，不在核心流程中 if-else 分支。

**理由**:
- 场景可扩展——新增场景只需定义 Profile，不改核心代码。
- 用户可自定义场景。
- 与 LLM Master Agent 的 Router 设计理念一致。

### ADR-3: 提取用 LLM，分类不用 regex

记忆提取（从对话中抽取 facts）使用 LLM 调用，不使用 OpenMemory 式的正则匹配。

**理由**:
- DoorX 是中英双语项目，regex 不可靠。
- LLM 提取质量远高于 pattern matching。
- 提取是异步后台任务，延迟可接受。
- 用便宜模型（如 haiku 级别）控制成本。

### ADR-4: 结构化检索优先，Embedding 预留

V1 完全使用结构化查询（FactKind 过滤 + tags 匹配 + path 关联 + weight×recency 排序）。Embedding 接口在类型系统中预留，Phase B 引入 pgvector。

**理由**:
- 开发场景的查询多数是精确的（"这个文件上次改出过什么问题？"、"项目有什么架构约束？"），结构化查询更直接、更可靠、更便宜。
- 避免 V1 引入 embedding provider 依赖。
- Embedding 对模糊语义检索（"类似的讨论"、"相关话题"）有价值，但不是 V1 刚需。

### ADR-5: 借鉴 OpenMemory 的 Temporal Facts

项目事实会随时间变化（"DoorX 用 NATS JetStream" → 可能将来换方案）。借鉴 OpenMemory 的 SPO 三元组 + `valid_from/valid_to` 时间窗口。

**理由**:
- 新事实写入时自动关闭旧事实，不删除历史。
- Agent 可回答"之前用的什么？"这类时间相关问题。
- 学习场景也需要：知识点可能被更正。

### ADR-6: SimHash 去重 + LLM 合并

两级去重策略：
1. **SimHash**（O(1)）：内容哈希的 hamming distance ≤ 3 视为疑似重复，直接强化已有记忆。
2. **LLM 判定**（可选）：Phase B 引入，对 SimHash 无法判定的近似内容做语义合并。

**理由**:
- SimHash 成本几乎为零，足以过滤明显重复。
- 避免每次写入都调用 LLM 判重。

### ADR-7: 全接口契约，实现可拔插

记忆模块的每个环节（存储、提取、去重、检索、注入、衰减、整合）都定义为 `biz/memory/` 下的**纯接口**。当前的 Change DAG 存储、LLM 提取、SimHash 去重等只是 V1 默认实现。

**理由**:
- 记忆系统是 AI 领域最活跃的方向之一，新方案（如 MemGPT、GraphRAG、long-term memory as a service）快速演进。
- 接口稳定、实现可换。未来换成第三方记忆服务（OpenMemory、Mem0）、或更先进的检索算法，只需提供新实现 + fx.Provide 替换注入，不改 biz 层和 service 层。
- 与 `llm.Provider`（biz 定义接口，infrastructure 提供 OpenAI/Ollama 实现）和 `changedag.ChangeStoreRepo`（biz 定义接口，data 提供 GORM 实现）的设计一致。

**原则**:
- 接口定义在 `internal/biz/memory/`，只依赖本包类型和标准库。
- 接口方法签名不暴露任何实现细节（不出现 gorm、pgvector、NATS 等）。
- 通过 `fx.Provide` + `fx.As(new(Interface))` 注入，切换实现只改 wire 不改调用方。

---

## 3. 接口契约 (Interface Contract)

记忆模块的可拔插性通过六个核心接口实现。每个接口独立可替换。

### 3.1 接口总览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       biz/memory/ (接口层 — 稳定契约)                        │
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ MemoryStore │  │  Extractor  │  │  Retriever   │  │   Maintainer     │  │
│  │  写入/读取   │  │  对话→facts │  │  召回+评分    │  │ 衰减/整合/清理   │  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬───────┘  └───────┬──────────┘  │
│         │                │                │                   │             │
│  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴───────┐  ┌───────┴──────────┐  │
│  │ Deduplicator│  │  Assembler  │  │ SceneDetector│  │ EmbeddingProvider│  │
│  │  去重判定    │  │ facts→prompt│  │  场景识别     │  │  向量化 (预留)   │  │
│  └─────────────┘  └─────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
                              │ fx.Provide + fx.As
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                   V1 默认实现（可整体或逐个替换）                              │
│                                                                             │
│  ChangeDAGMemoryStore    LLMExtractor        StructuralRetriever            │
│  SimHashDeduplicator     PromptAssembler     KeywordSceneDetector           │
│  TimerMaintainer         (EmbeddingProvider → Phase B)                      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 MemoryStore — 记忆持久化

MemoryStore 是唯一的记忆读写入口。V1 实现基于 Change DAG + object_state JSONB 查询；未来可换成专用 DB、第三方 API、甚至内存缓存。

```go
// internal/biz/memory/store.go

// MemoryStore 管理记忆的持久化与查询。
// 实现应保证：
// 1. 写入幂等（同 ID 重复写入不报错）。
// 2. 查询结果不含 soft-deleted 记忆。
// 3. 线程安全。
type MemoryStore interface {
    // ── 写入 ──

    // Put 创建或整体覆盖一条记忆。
    Put(ctx context.Context, fact *MemoryFact) error

    // Patch 部分更新（weight、last_used、consolidated 等字段级更新）。
    Patch(ctx context.Context, workspaceID string, factID uuid.UUID, patch FactPatch) error

    // SoftDelete 软删除。
    SoftDelete(ctx context.Context, workspaceID string, factID uuid.UUID) error

    // ── 单条读取 ──

    Get(ctx context.Context, workspaceID string, factID uuid.UUID) (*MemoryFact, error)

    // ── 批量查询 ──

    // Query 是通用查询入口。所有过滤、排序、分页通过 QueryOpts 表达。
    // 实现方负责将 QueryOpts 翻译为底层查询（JSONB、SQL、向量检索等）。
    Query(ctx context.Context, opts QueryOpts) (*QueryResult, error)

    // ── Landmark（同接口，不同 object_type）──

    PutLandmark(ctx context.Context, lm *Landmark) error
    PatchLandmark(ctx context.Context, workspaceID string, lmID uuid.UUID, patch LandmarkPatch) error
    SoftDeleteLandmark(ctx context.Context, workspaceID string, lmID uuid.UUID) error
    QueryLandmarks(ctx context.Context, opts LandmarkQueryOpts) ([]*Landmark, error)
}

// QueryOpts 描述一次记忆查询的所有条件。
// 字段为零值表示"不过滤该维度"。
type QueryOpts struct {
    WorkspaceID string
    UserID      string

    // 类型过滤
    Kinds []FactKind

    // 标签过滤（OR 语义：命中任一 tag 即匹配）
    Tags []string

    // 代码路径前缀匹配（Refs 中任一 path 匹配即命中）
    PathPrefix string

    // Triple 精确匹配
    TripleSubject   string
    TriplePredicate string

    // 时间有效性：只返回 "当前有效" 的 temporal facts
    ValidAt *int64

    // 排除已整合的记忆
    ExcludeConsolidated bool

    // 分页与排序
    OrderBy string // "weight_recency" | "created_at" | "access_n"
    Limit   int
    Offset  int
}

type QueryResult struct {
    Facts []*MemoryFact
    Total int
}

// FactPatch 表示对 MemoryFact 的部分更新。
// nil 字段不更新。
type FactPatch struct {
    Weight       *float32
    AccessN      *int
    LastUsed     *int64
    Consolidated *bool
    ValidTo      *int64
    Tags         *[]string // 整体替换
}

type LandmarkPatch struct {
    Stale     *bool
    StaleNote *string
    Weight    *float32
}

type LandmarkQueryOpts struct {
    WorkspaceID string
    Kinds       []LandmarkKind
    PathPrefix  string // Refs 中任一 path 匹配
    IncludeStale bool  // 默认不含 stale landmarks
    Limit       int
}
```

**替换示例**：
- V1: `ChangeDAGMemoryStore` — 写 Change DAG，查 object_state JSONB。
- 替换方案 A: `PostgresMemoryStore` — 独立 memories 表，支持 GIN 索引。
- 替换方案 B: `OpenMemoryStore` — 代理到 OpenMemory HTTP API。
- 替换方案 C: `InMemoryStore` — 纯内存，用于测试。

### 3.3 Extractor — 记忆提取

从对话历史中提取值得记忆的 facts。V1 用 LLM；未来可以用微调模型、规则引擎、或外部服务。

```go
// internal/biz/memory/extractor.go

// Extractor 从对话中提取结构化记忆。
// 实现可以是：LLM 调用、规则引擎、微调分类器、外部 API。
type Extractor interface {
    // Extract 接受对话消息和场景上下文，返回提取到的 facts。
    // 返回空切片表示"无值得记忆的内容"，不视为错误。
    Extract(ctx context.Context, opts ExtractOpts) ([]MemoryFact, error)
}

type ExtractOpts struct {
    WorkspaceID string
    UserID      string
    Scene       string        // 当前场景标识
    Messages    []llm.Message // 完整对话历史
}
```

**替换示例**：
- V1: `LLMExtractor` — 调用 `llm.Provider.Chat()` + 结构化输出 prompt。
- 替换方案 A: `RuleExtractor` — 正则/关键词规则（轻量但精度低）。
- 替换方案 B: `HybridExtractor` — 规则预筛 + LLM 精提。
- 替换方案 C: `RemoteExtractor` — 调用外部记忆服务 API。

### 3.4 Deduplicator — 去重

判定新 fact 是否与已有记忆重复，如果重复则返回应该强化的已有记忆 ID。

```go
// internal/biz/memory/dedup.go

// Deduplicator 判定一条新记忆是否与已有记忆重复。
type Deduplicator interface {
    // IsDuplicate 检查 candidate 是否与已有记忆重复。
    // 返回值：
    //   - existing != nil: 重复，应该强化 existing 而不是写入新记忆。
    //   - existing == nil, err == nil: 非重复，可以写入。
    IsDuplicate(ctx context.Context, store MemoryStore, candidate *MemoryFact) (existing *MemoryFact, err error)
}
```

**替换示例**：
- V1: `SimHashDeduplicator` — 计算 simhash，hamming distance ≤ 3 视为重复。
- 替换方案 A: `EmbeddingDeduplicator` — 向量 cosine similarity > 0.95 视为重复。
- 替换方案 B: `LLMDeduplicator` — 调用 LLM 判定语义等价。
- 替换方案 C: `CompositeDeduplicator` — SimHash 快筛 + Embedding 精判。

### 3.5 Retriever — 记忆召回

根据当前上下文，从存储中召回最相关的记忆。这是记忆系统最可能被替换的组件。

```go
// internal/biz/memory/retriever.go

// Retriever 根据上下文和场景配置召回相关记忆。
// 实现负责评分、排序和 top-K 截取。
type Retriever interface {
    // Recall 返回与当前上下文最相关的记忆集合。
    Recall(ctx context.Context, opts RecallOpts) (*RecallResult, error)
}

type RecallOpts struct {
    WorkspaceID string
    UserID      string
    Profile     SceneProfile  // 场景配置（决定加载哪些 kind、top-K 等）

    // 上下文信号（Retriever 实现自行决定如何利用）
    Query         string   // 用户最新消息或对话摘要
    ContextTags   []string // 从对话中提取的标签
    AffectedPaths []string // 涉及的代码文件路径（dev 场景）
}

type RecallResult struct {
    // 按类型分组的记忆，方便 Assembler 分区注入
    Groups []RecallGroup
}

type RecallGroup struct {
    Label string       // 显示标签："Rules", "Architecture", "Context" 等
    Facts []ScoredFact // 已排序
}

type ScoredFact struct {
    Fact  MemoryFact
    Score float64 // 0~1，实现自行定义评分含义
}
```

**替换示例**：
- V1: `StructuralRetriever` — JSONB 结构化查询 + weight×recency×tag 复合评分。
- 替换方案 A: `EmbeddingRetriever` — 结构化预筛 + pgvector cosine top-K。
- 替换方案 B: `GraphRetriever` — 基于记忆关联图谱的图遍历召回。
- 替换方案 C: `HybridRetriever` — 结构化 + 向量 + graph 的加权融合。
- 替换方案 D: `RemoteRetriever` — 代理到外部记忆服务（OpenMemory / Mem0 API）。

### 3.6 Assembler — Prompt 注入

将召回结果格式化并注入到 system prompt 中。看似简单但值得抽象——不同的注入策略（纯文本 vs XML tags vs structured JSON）对 LLM 效果差异大。

```go
// internal/biz/memory/assembler.go

// Assembler 将召回的记忆格式化并注入到 system prompt。
type Assembler interface {
    // Inject 接受基础 system prompt 和召回结果，返回增强后的 system prompt。
    Inject(base string, recall *RecallResult, profile SceneProfile) string
}
```

**替换示例**：
- V1: `MarkdownAssembler` — 按分组输出 markdown 列表。
- 替换方案 A: `XMLAssembler` — 用 `<memory>` XML 标签包裹（某些模型对 XML 理解更好）。
- 替换方案 B: `CompactAssembler` — 极简关键词输出（token 预算紧张时）。

### 3.7 Maintainer — 生命周期管理

衰减、整合、清理等后台维护操作。

```go
// internal/biz/memory/maintainer.go

// Maintainer 执行记忆的后台维护任务。
// 每个方法可独立调度（不同频率）。
type Maintainer interface {
    // Decay 对记忆执行权重衰减。
    Decay(ctx context.Context, workspaceID string) error

    // Reflect 聚类碎片记忆，生成整合摘要。
    Reflect(ctx context.Context, workspaceID string, userID string) error

    // Prune 清理过期的软删除记忆和极低权重记忆。
    Prune(ctx context.Context, workspaceID string) error

    // CheckLandmarkStaleness 检查 landmark 引用是否仍然有效。
    CheckLandmarkStaleness(ctx context.Context, workspaceID string) error
}
```

**替换示例**：
- V1: `DefaultMaintainer` — 三级衰减 + 基于 subject/tags 聚类整合。
- 替换方案 A: `LLMMaintainer` — 用 LLM 做语义聚类和摘要生成。
- 替换方案 B: `NoOpMaintainer` — 不维护（测试用或外部服务已处理）。

### 3.8 SceneDetector — 场景识别

```go
// internal/biz/memory/scene.go

// SceneDetector 根据对话上下文判断当前场景。
type SceneDetector interface {
    Detect(ctx context.Context, messages []llm.Message) string
}
```

### 3.9 EmbeddingProvider — 向量化（Phase B 预留）

```go
// internal/biz/memory/embedding.go

// EmbeddingProvider 生成文本的向量表示。
// Phase B 启用。接口现在定义，V1 不要求实现。
type EmbeddingProvider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
    Model() string
}

// VectorStore 持久化和检索向量。
// Phase B 启用。
type VectorStore interface {
    Store(ctx context.Context, factID uuid.UUID, vector []float32) error
    Search(ctx context.Context, query []float32, opts VectorSearchOpts) ([]VectorMatch, error)
    Delete(ctx context.Context, factID uuid.UUID) error
}

type VectorSearchOpts struct {
    WorkspaceID string
    UserID      string
    Kinds       []FactKind
    TopK        int
    MinScore    float32
}

type VectorMatch struct {
    FactID uuid.UUID
    Score  float32
}
```

### 3.10 MemoryService — 面向 Agent 的统一入口

MemoryService 是编排层，组合上述接口完成端到端流程。它不是可替换接口，而是消费所有接口的 **biz usecase**。

```go
// internal/biz/memory/service.go

// MemoryService 编排记忆系统的端到端流程。
// 它消费上述所有接口，对外提供 Agent 需要的高层操作。
// 这不是一个可替换接口——它是粘合层。
type MemoryService struct {
    store    MemoryStore
    extract  Extractor
    dedup    Deduplicator
    retrieve Retriever
    assemble Assembler
    maintain Maintainer
    detect   SceneDetector
}

// NewMemoryService 所有依赖通过 fx 注入。
// 切换任意子组件只需替换对应的 fx.Provide。
func NewMemoryService(
    store    MemoryStore,
    extract  Extractor,
    dedup    Deduplicator,
    retrieve Retriever,
    assemble Assembler,
    maintain Maintainer,
    detect   SceneDetector,
) *MemoryService {
    return &MemoryService{
        store: store, extract: extract, dedup: dedup,
        retrieve: retrieve, assemble: assemble,
        maintain: maintain, detect: detect,
    }
}

// PrepareContext 是对话开始时的入口：检测场景 → 召回记忆 → 注入 prompt。
func (s *MemoryService) PrepareContext(ctx context.Context, basePrompt string, opts PrepareOpts) (string, error)

// AfterConversation 是对话结束后的入口：提取 → 去重 → 存储。
func (s *MemoryService) AfterConversation(ctx context.Context, opts AfterConversationOpts) error

// Remember 是 Agent 主动记忆的入口（tool_use 调用）。
func (s *MemoryService) Remember(ctx context.Context, fact *MemoryFact) error

// Recall 是 Agent 主动回忆的入口（tool_use 调用）。
func (s *MemoryService) Recall(ctx context.Context, opts RecallOpts) (*RecallResult, error)
```

### 3.11 fx 注入示意

```go
// internal/biz/biz.go — V1 默认 wiring

// V1 默认实现
fx.Provide(
    dagstore.NewChangeDAGMemoryStore,         // MemoryStore 实现
    llmextract.NewLLMExtractor,               // Extractor 实现
    simhash.NewSimHashDeduplicator,           // Deduplicator 实现
    structural.NewStructuralRetriever,        // Retriever 实现
    markdown.NewMarkdownAssembler,            // Assembler 实现
    defaultmaint.NewDefaultMaintainer,        // Maintainer 实现
    keyword.NewKeywordSceneDetector,          // SceneDetector 实现
)

// 接口绑定
fx.Provide(
    fx.Annotate(dagstore.NewChangeDAGMemoryStore,     fx.As(new(memory.MemoryStore))),
    fx.Annotate(llmextract.NewLLMExtractor,           fx.As(new(memory.Extractor))),
    fx.Annotate(simhash.NewSimHashDeduplicator,       fx.As(new(memory.Deduplicator))),
    fx.Annotate(structural.NewStructuralRetriever,    fx.As(new(memory.Retriever))),
    fx.Annotate(markdown.NewMarkdownAssembler,        fx.As(new(memory.Assembler))),
    fx.Annotate(defaultmaint.NewDefaultMaintainer,    fx.As(new(memory.Maintainer))),
    fx.Annotate(keyword.NewKeywordSceneDetector,      fx.As(new(memory.SceneDetector))),
)

// 编排层
fx.Provide(memory.NewMemoryService)

// ── 未来替换示例 ──
// 要换成 OpenMemory 后端？只需：
//   fx.Annotate(openmemory.NewOpenMemoryStore,       fx.As(new(memory.MemoryStore))),
//   fx.Annotate(openmemory.NewOpenMemoryRetriever,   fx.As(new(memory.Retriever))),
// 其余组件不变。
```

### 3.12 接口替换矩阵

| 接口 | V1 默认实现 | 未来替换选项 | 替换动机 |
|------|-----------|------------|---------|
| **MemoryStore** | ChangeDAGMemoryStore | PostgresStore, OpenMemoryStore, Mem0Store | 需要更复杂索引或使用第三方服务 |
| **Extractor** | LLMExtractor | RuleExtractor, HybridExtractor, FineTunedExtractor | 降成本、提速、提精度 |
| **Deduplicator** | SimHashDedup | EmbeddingDedup, LLMDedup, CompositeDedup | 更精确的语义去重 |
| **Retriever** | StructuralRetriever | EmbeddingRetriever, GraphRetriever, HybridRetriever | 语义检索、关联推理 |
| **Assembler** | MarkdownAssembler | XMLAssembler, CompactAssembler, JSONAssembler | 不同模型适配 |
| **Maintainer** | DefaultMaintainer | LLMMaintainer, NoOpMaintainer | 更智能整合 |
| **SceneDetector** | KeywordDetector | LLMDetector, MLClassifier | 更准确的场景识别 |
| **EmbeddingProvider** | (Phase B) | OpenAIEmbedding, OllamaEmbedding, LocalEmbedding | 语义检索能力 |
| **VectorStore** | (Phase B) | PgVectorStore, QdrantStore, InMemoryVectorStore | 向量持久化 |

---

## 4. 记忆类型体系

### 4.1 设计原则

不采用 OpenMemory 的 5 认知扇区（episodic/semantic/procedural/emotional/reflective），而是基于 **实际用途** 定义 FactKind。原因：
- DoorX 是个人 agent，不是认知科学模拟器。
- "emotional" 对开发无意义，"procedural" 与 "knowledge" 区分模糊。
- 基于用途的分类更利于检索策略：不同 kind 有不同的衰减率和检索优先级。

### 4.2 FactKind 分类

```
FactKind
├── Personal（跨场景通用）
│   ├── preference    用户偏好         "喜欢暗色主题"、"翻译时保留术语原文"
│   ├── correction    用户纠正         "不要用 gorm 的 AutoMigrate"
│   └── persona       用户画像         "是后端工程师"、"在杭州"
│
├── Knowledge（知识性记忆）
│   ├── knowledge     通用知识         "Go 1.22 引入了 range over func"
│   ├── convention    项目规约         "biz 层不能 import infrastructure"
│   └── decision      决策记录         "选用 pgvector 而非 Milvus"
│
├── Episodic（事件性记忆）
│   ├── incident      问题事件         "改 sync 逻辑时引入了死锁"
│   ├── resolution    解决方案         "通过 context timeout 解决了死锁"
│   ├── review        评审反馈         "PR #42 review 指出缺少错误处理"
│   └── milestone     里程碑           "v1.0 发布"、"第三章学完"
│
└── Meta（元记忆）
    └── reflection    自动整合         由 Reflect 流程生成，聚合碎片记忆
```

### 4.3 衰减策略映射

| FactKind | 衰减策略 | 说明 |
|----------|---------|------|
| preference, persona | **不衰减** | 长期稳定的用户特征 |
| convention | **不衰减** | 项目规则，除非被用户主动修改 |
| decision | **极慢衰减** (λ=0.002) | 决策通常长期有效 |
| knowledge | **慢衰减** (λ=0.005) | 知识可能过时但衰减缓慢 |
| correction | **慢衰减** (λ=0.005) | 纠正信息长期有价值 |
| incident, resolution | **正常衰减** (λ=0.015) | 事件记忆随时间淡化 |
| review | **正常衰减** (λ=0.015) | 评审反馈时效性较强 |
| milestone | **慢衰减** (λ=0.005) | 里程碑值得长期记忆 |
| reflection | **极慢衰减** (λ=0.001) | 整合后的高阶洞察最有价值 |

### 4.4 Landmark（独立类型）

Landmark 不是 FactKind 的一种，而是独立的 object_type。原因：它不是一句话的 fact，而是**带有代码引用的结构化架构知识**。

```
LandmarkKind
├── pattern     架构模式     "所有 usecase 通过 fx.Provide 注入"
├── module      子系统       "sync 子系统: workflow/syncjob/"
├── boundary    边界约束     "service→biz→data 依赖方向"
└── hotspot     热点文件     "router.go 经常改，改动要小心"
```

Landmark 不按时间衰减，而是**按代码变更失效**——引用的文件被大幅修改或删除时标记为 stale。

---

## 5. 数据模型

### 5.1 MemoryFact（Change DAG object_type = "memory"）

```go
// internal/biz/memory/types.go

type MemoryFact struct {
    ID          uuid.UUID `json:"id"`
    WorkspaceID string    `json:"workspace_id"`
    UserID      string    `json:"user_id"`

    // ── 内容 ──
    Kind    FactKind `json:"kind"`
    Subject string   `json:"subject"`  // 简短主题，用于去重和显示
    Content string   `json:"content"`  // 具体内容（1~3 句话）

    // ── 结构化三元组（可选，借鉴 OpenMemory Temporal KG）──
    Triple *Triple `json:"triple,omitempty"`

    // ── 时间有效性（借鉴 OpenMemory）──
    ValidFrom *int64 `json:"valid_from,omitempty"` // 事实开始生效时间
    ValidTo   *int64 `json:"valid_to,omitempty"`   // 事实失效时间（被新事实取代）

    // ── 检索维度 ──
    Tags  []string  `json:"tags"`            // 业务标签
    Refs  []CodeRef `json:"refs,omitempty"`  // 关联代码位置（开发场景）
    Scope string    `json:"scope"`           // global | workspace

    // ── 去重 ──
    SimHash uint64 `json:"simhash"` // 内容指纹，hamming distance 去重

    // ── 生命周期 ──
    Source       string  `json:"source"`        // 来源（conversation_id / action_run_id）
    Weight       float32 `json:"weight"`        // 重要性 0~1
    AccessN      int     `json:"access_n"`      // 被召回次数（用于强化）
    LastUsed     int64   `json:"last_used"`     // 最后一次被召回的时间
    Consolidated bool    `json:"consolidated"`  // 是否已被 Reflect 整合

    // ── Embedding 预留（V1 不填充，Phase B 启用）──
    EmbeddingModel string    `json:"embedding_model,omitempty"` // 生成 embedding 的模型
    EmbeddingDim   int       `json:"embedding_dim,omitempty"`   // 向量维度
    Embedding      []float32 `json:"-"`                         // 不序列化到 Change payload

    // ── 元数据 ──
    CreatedAt int64 `json:"created_at"`
    UpdatedAt int64 `json:"updated_at"`
    Deleted   bool  `json:"deleted"`
}

type FactKind string

const (
    // Personal
    FactPreference FactKind = "preference"
    FactCorrection FactKind = "correction"
    FactPersona    FactKind = "persona"

    // Knowledge
    FactKnowledge  FactKind = "knowledge"
    FactConvention FactKind = "convention"
    FactDecision   FactKind = "decision"

    // Episodic
    FactIncident   FactKind = "incident"
    FactResolution FactKind = "resolution"
    FactReview     FactKind = "review"
    FactMilestone  FactKind = "milestone"

    // Meta
    FactReflection FactKind = "reflection"
)

type Triple struct {
    Subject   string `json:"subject"`   // "projecttemplate", "sync-module", "biz-layer"
    Predicate string `json:"predicate"` // "uses", "depends_on", "forbids", "prefers"
    Object    string `json:"object"`    // "NATS JetStream", "infrastructure import"
}

type CodeRef struct {
    Path   string `json:"path"`             // "internal/biz/changedag/usecase.go"
    Symbol string `json:"symbol,omitempty"` // "ChangeStoreRepo.InsertChange"
    Line   int    `json:"line,omitempty"`
    Note   string `json:"note,omitempty"`   // "这里定义了核心同步接口"
}
```

### 5.2 Landmark（Change DAG object_type = "landmark"）

```go
type Landmark struct {
    ID          uuid.UUID    `json:"id"`
    WorkspaceID string       `json:"workspace_id"`

    Kind    LandmarkKind `json:"kind"`
    Name    string       `json:"name"`    // "fx dependency injection"
    Summary string       `json:"summary"` // 一两句话描述

    Refs []CodeRef `json:"refs"` // 关联的文件/符号

    // 生命周期
    Weight    float32 `json:"weight"`
    Stale     bool    `json:"stale"`      // 引用的代码已变更
    StaleNote string  `json:"stale_note"` // 为何标记为 stale

    CreatedAt int64 `json:"created_at"`
    UpdatedAt int64 `json:"updated_at"`
    Deleted   bool  `json:"deleted"`
}

type LandmarkKind string

const (
    LandmarkPattern  LandmarkKind = "pattern"
    LandmarkModule   LandmarkKind = "module"
    LandmarkBoundary LandmarkKind = "boundary"
    LandmarkHotspot  LandmarkKind = "hotspot"
)
```

### 5.3 Materializer

```go
// internal/biz/memory/materialize.go

const (
    ObjectTypeMemory   = "memory"
    ObjectTypeLandmark = "landmark"
)

type MemoryMaterializer struct{}

func (m *MemoryMaterializer) ObjectType() string { return ObjectTypeMemory }

func (m *MemoryMaterializer) Apply(cur []byte, op string, payload []byte) ([]byte, error) {
    switch op {
    case "set":
        return payload, nil
    case "patch":
        return jsonpatch.Apply(cur, payload)
    case "delete":
        return softDelete(cur)
    case "merge":
        return jsonMerge(cur, payload)
    default:
        return nil, fmt.Errorf("unknown op %q for %s", op, ObjectTypeMemory)
    }
}

type LandmarkMaterializer struct{}

func (m *LandmarkMaterializer) ObjectType() string { return ObjectTypeLandmark }

func (m *LandmarkMaterializer) Apply(cur []byte, op string, payload []byte) ([]byte, error) {
    // 同 MemoryMaterializer
}
```

### 5.4 Change DAG 写入示例

```go
func (uc *MemoryUsecase) createFact(ctx context.Context, ws, userID string, fact *MemoryFact) error {
    fact.ID = uuid.New()
    fact.CreatedAt = time.Now().UnixMilli()
    fact.UpdatedAt = fact.CreatedAt
    fact.SimHash = computeSimHash(fact.Content)

    if fact.Weight == 0 {
        fact.Weight = initialWeight(fact.Kind)
    }

    payload, _ := json.Marshal(fact)

    ch := &model.Change{
        ID:          uuid.New(),
        WorkspaceID: ws,
        ObjectID:    fact.ID,
        ObjectType:  ObjectTypeMemory,
        Parents:     pq.StringArray{},
        AuthorNode:  "local",
        Op:          "set",
        Payload:     datatypes.JSON(payload),
        CreatedAt:   fact.CreatedAt,
    }

    if _, _, _, _, err := uc.changeRepo.HubPushChanges(ctx, ws, []*model.Change{ch}); err != nil {
        return fmt.Errorf("push memory change: %w", err)
    }

    return uc.refreshObjectState(ctx, ws, fact.ID, ObjectTypeMemory)
}
```

### 5.5 object_state 查询辅助

V1 利用 PostgreSQL JSONB 操作符实现结构化查询：

```sql
-- 按 FactKind 查询（convention 全量加载）
SELECT * FROM object_state
WHERE workspace_id = $1
  AND object_type = 'memory'
  AND COALESCE((state->>'deleted')::bool, false) = false
  AND state->>'kind' = 'convention';

-- 按 tags 模糊匹配
SELECT * FROM object_state
WHERE workspace_id = $1
  AND object_type = 'memory'
  AND state->'tags' ?| ARRAY['sync', 'nats'];

-- 按 CodeRef 路径匹配
SELECT * FROM object_state
WHERE workspace_id = $1
  AND object_type = 'memory'
  AND EXISTS (
    SELECT 1 FROM jsonb_array_elements(state->'refs') AS r
    WHERE r->>'path' LIKE 'internal/biz/changedag/%'
  );

-- 按 weight × recency 排序
SELECT *, (
    (state->>'weight')::float *
    EXP(-0.01 * (EXTRACT(EPOCH FROM NOW()) * 1000 - (state->>'last_used')::bigint) / 86400000)
) AS score
FROM object_state
WHERE workspace_id = $1
  AND object_type = 'memory'
  AND COALESCE((state->>'deleted')::bool, false) = false
ORDER BY score DESC
LIMIT $2;
```

---

## 6. 场景配置 (Scene Profile)

### 6.1 设计理念

Agent 在不同时刻做不同的事（聊天 vs 编程 vs 学习），需要的记忆策略不同。Scene Profile 定义了：

- 激活哪些 FactKind
- 检索优先级和 top-K 配额
- 是否加载 Landmark
- 注入 prompt 的格式和位置

```go
// internal/biz/memory/scene.go

type SceneProfile struct {
    Name string `json:"name"` // "companion", "dev", "learning", "task"

    // 哪些 kind 始终加载（全量注入 system prompt）
    AlwaysLoad []FactKind `json:"always_load"`

    // 哪些 kind 按相关性检索（top-K）
    Retrieve []RetrieveRule `json:"retrieve"`

    // 是否加载 Landmark
    LoadLandmarks bool `json:"load_landmarks"`

    // Prompt 注入配置
    Injection InjectionConfig `json:"injection"`
}

type RetrieveRule struct {
    Kinds []FactKind `json:"kinds"`
    TopK  int        `json:"top_k"`
    // 未来扩展：MinWeight, MaxAge, RequireTags 等
}

type InjectionConfig struct {
    // system prompt 中记忆区块的标题
    SectionTitle string `json:"section_title"`
    // 最大 token 预算（记忆区块占用的上限）
    MaxTokens int `json:"max_tokens"`
}
```

### 6.2 内置 Scene Profile

#### Companion（陪伴/聊天）

```go
var SceneCompanion = SceneProfile{
    Name:          "companion",
    AlwaysLoad:    []FactKind{FactPreference, FactPersona},
    Retrieve: []RetrieveRule{
        {Kinds: []FactKind{FactKnowledge, FactMilestone}, TopK: 5},
        {Kinds: []FactKind{FactCorrection}, TopK: 3},
    },
    LoadLandmarks: false,
    Injection: InjectionConfig{
        SectionTitle: "About this user",
        MaxTokens:    800,
    },
}
```

注入效果：
```
## About this user
- [persona] 后端工程师，在杭州
- [preference] 喜欢简洁直接的回复风格
- [preference] 翻译时保留技术术语原文
- [knowledge] 最近在学 Rust
- [milestone] 上周完成了 DoorX v1.0 发布
```

#### Dev（项目开发）

```go
var SceneDev = SceneProfile{
    Name:          "dev",
    AlwaysLoad:    []FactKind{FactConvention, FactPreference, FactCorrection},
    Retrieve: []RetrieveRule{
        {Kinds: []FactKind{FactDecision, FactKnowledge}, TopK: 5},
        {Kinds: []FactKind{FactIncident, FactResolution}, TopK: 5},
        {Kinds: []FactKind{FactReview}, TopK: 3},
    },
    LoadLandmarks: true,
    Injection: InjectionConfig{
        SectionTitle: "Project memory",
        MaxTokens:    1500,
    },
}
```

注入效果：
```
## Project memory

### Rules
- [convention] biz 层不能 import infrastructure
- [convention] 所有 repo 必须有 var _ Interface = (*impl)(nil) 断言
- [convention] commit 用 conventional commits 格式
- [correction] 不要用 gorm 的 AutoMigrate

### Architecture
- [landmark:pattern] fx 依赖注入: internal/biz/biz.go, internal/data/repository/repo.go
- [landmark:module] Change DAG 同步: internal/biz/changedag/, internal/workflow/syncjob/
- [landmark:boundary] service→biz→data 依赖方向

### Relevant context
- [decision] 选用 NATS JetStream 而非 Kafka（2026-02-05）
- [incident] 改 syncjob 时引入死锁 → [resolution] 通过 context timeout 解决

### Preferences
- [preference] 用中文写注释和文档
```

#### Learning（学习）

```go
var SceneLearning = SceneProfile{
    Name:          "learning",
    AlwaysLoad:    []FactKind{FactPreference, FactPersona},
    Retrieve: []RetrieveRule{
        {Kinds: []FactKind{FactKnowledge}, TopK: 10},
        {Kinds: []FactKind{FactCorrection}, TopK: 3},
        {Kinds: []FactKind{FactMilestone}, TopK: 3},
        {Kinds: []FactKind{FactReflection}, TopK: 3},
    },
    LoadLandmarks: false,
    Injection: InjectionConfig{
        SectionTitle: "Learning context",
        MaxTokens:    1200,
    },
}
```

#### Task（任务管理）

```go
var SceneTask = SceneProfile{
    Name:          "task",
    AlwaysLoad:    []FactKind{FactPreference},
    Retrieve: []RetrieveRule{
        {Kinds: []FactKind{FactMilestone, FactDecision}, TopK: 5},
        {Kinds: []FactKind{FactKnowledge}, TopK: 3},
    },
    LoadLandmarks: false,
    Injection: InjectionConfig{
        SectionTitle: "Context",
        MaxTokens:    600,
    },
}
```

### 6.3 场景检测

Scene 可以由以下方式确定（优先级从高到低）：

1. **用户显式指定**：API 参数或 UI 切换。
2. **Master Agent 推断**：Agent 根据对话内容判断（如出现代码、文件路径 → dev；出现"学习"、"教我" → learning）。
3. **默认 Companion**：无法判断时使用 companion 场景。

```go
type SceneDetector interface {
    Detect(ctx context.Context, messages []llm.Message) string // "companion"|"dev"|"learning"|"task"
}
```

V1 实现可以是简单的关键词匹配 + 用户上下文；后续可升级为 LLM 分类。

---

## 7. 核心流程

### 7.1 总览

所有流程通过**接口调用**，MemoryService 只做编排，不包含具体实现逻辑。

```
                    ┌──────────────────────────────────────────────┐
                    │     MemoryService（编排层，消费所有接口）        │
                    └──────┬──────────────────┬───────────────┬────┘
                           │                  │               │
                    PrepareContext       AfterConversation   (定时)
                           │                  │               │
                ┌──────────┴───┐       ┌──────┴──────┐   ┌───┴──────────┐
                │              │       │             │   │              │
                ▼              ▼       ▼             ▼   ▼              ▼
         SceneDetector    Retriever  Extractor   Dedup  Maintainer  Maintainer
          .Detect()       .Recall()  .Extract() .IsDup  .Decay()   .Reflect()
                │              │       │             │
                │              ▼       │             ▼
                │          Assembler   │        MemoryStore
                │          .Inject()   │         .Put()
                │              │       │
                ▼              ▼       ▼
            scene_id      enhanced   stored
                          prompt     facts
```

### 7.2 PrepareContext（对话开始 — 召回与注入）

MemoryService.PrepareContext 的编排逻辑（伪代码）：

```go
func (s *MemoryService) PrepareContext(ctx context.Context, basePrompt string, opts PrepareOpts) (string, error) {
    // 1. 场景检测 → SceneDetector 接口
    scene := s.detect.Detect(ctx, opts.Messages)
    profile := GetProfile(scene)

    // 2. 记忆召回 → Retriever 接口
    recall, err := s.retrieve.Recall(ctx, RecallOpts{
        WorkspaceID:   opts.WorkspaceID,
        UserID:        opts.UserID,
        Profile:       profile,
        Query:         opts.Query,
        ContextTags:   opts.ContextTags,
        AffectedPaths: opts.AffectedPaths,
    })
    if err != nil {
        return basePrompt, nil // 降级：无记忆也能工作
    }

    // 3. Prompt 注入 → Assembler 接口
    return s.assemble.Inject(basePrompt, recall, profile), nil
}
```

### 7.3 AfterConversation（对话结束 — 提取与存储）

```go
func (s *MemoryService) AfterConversation(ctx context.Context, opts AfterConversationOpts) error {
    // 1. 提取 → Extractor 接口
    facts, err := s.extract.Extract(ctx, ExtractOpts{
        WorkspaceID: opts.WorkspaceID,
        UserID:      opts.UserID,
        Scene:       opts.Scene,
        Messages:    opts.Messages,
    })
    if err != nil || len(facts) == 0 {
        return err
    }

    // 2. 逐条去重 + 存储
    for _, fact := range facts {
        // 2a. 去重 → Deduplicator 接口
        if existing, _ := s.dedup.IsDuplicate(ctx, s.store, &fact); existing != nil {
            // 重复 → 强化已有记忆
            s.store.Patch(ctx, existing.WorkspaceID, existing.ID, FactPatch{
                AccessN:  ptr(existing.AccessN + 1),
                LastUsed: ptr(time.Now().UnixMilli()),
            })
            continue
        }

        // 2b. Temporal 更新检查
        if fact.Triple != nil {
            s.closeSupersededFacts(ctx, &fact) // 通过 MemoryStore.Query + .Patch
        }

        // 2c. 写入 → MemoryStore 接口
        if err := s.store.Put(ctx, &fact); err != nil {
            return err
        }
    }
    return nil
}
```

### 7.4 Extraction Prompt（V1 LLMExtractor 参考）

V1 的 `LLMExtractor` 实现使用以下 prompt 结构。**这不是接口契约的一部分**——其他 Extractor 实现可以用完全不同的方式：

```
从以下对话中提取值得长期记住的信息。输出 JSON 数组。

每条记忆必须包含：
- kind: preference|correction|persona|knowledge|convention|decision|incident|resolution|review|milestone
- subject: 简短主题（用于去重匹配，<10字）
- content: 具体内容（1~3句话）
- tags: 相关标签数组
- triple: （可选）{subject, predicate, object} 结构化三元组
- refs: （可选）[{path, symbol, note}] 关联的代码位置

规则：
1. 只提取有长期价值的信息，忽略临时性对话。
2. preference/correction 类型要准确反映用户的明确表态。
3. convention 仅限用户或项目文档明确定义的规则。
4. incident 必须记录出了什么问题，resolution 必须记录如何解决。
5. 如果没有值得记忆的内容，返回空数组 []。
```

### 7.5 V1 StructuralRetriever 参考评分公式

V1 `StructuralRetriever` 的复合评分（借鉴 OpenMemory，简化版）。**其他 Retriever 实现可以用完全不同的评分策略**：

```
score = 0.30 × weight        // 记忆自身权重
      + 0.25 × recency       // 时效分：exp(-0.05 × days_since_used)
      + 0.25 × tag_overlap   // 标签匹配度
      + 0.20 × path_overlap  // 代码路径匹配度（dev 场景）
```

### 7.6 Decay（衰减）

定时后台任务，借鉴 OpenMemory 三级衰减。

```go
// internal/biz/memory/decay.go

type DecayTier string
const (
    TierHot  DecayTier = "hot"  // 近期高频访问 → λ 最小
    TierWarm DecayTier = "warm" // 中等活跃
    TierCold DecayTier = "cold" // 久未访问 → λ 最大
)

func pickTier(fact MemoryFact, now int64) DecayTier {
    daysSinceUsed := float64(now-fact.LastUsed) / 86400000
    if daysSinceUsed < 7 && (fact.AccessN > 3 || fact.Weight > 0.7) {
        return TierHot
    }
    if daysSinceUsed < 30 || fact.Weight > 0.4 {
        return TierWarm
    }
    return TierCold
}

var tierMultiplier = map[DecayTier]float64{
    TierHot:  0.3,  // hot 记忆衰减速度 = base λ × 0.3
    TierWarm: 1.0,  // warm 记忆衰减速度 = base λ × 1.0
    TierCold: 2.0,  // cold 记忆衰减速度 = base λ × 2.0
}

func (uc *MemoryUsecase) RunDecay(ctx context.Context, ws string) error {
    facts, _, _ := uc.stateRepo.ListObjectStatesByType(ctx, ws, ObjectTypeMemory, 1000, 0)
    now := time.Now().UnixMilli()

    for _, state := range facts {
        var fact MemoryFact
        json.Unmarshal(state.State, &fact)

        // 不衰减的类型跳过
        baseLambda := decayLambda(fact.Kind)
        if baseLambda == 0 {
            continue
        }

        tier := pickTier(fact, now)
        lambda := baseLambda * tierMultiplier[tier]
        daysSince := float64(now-fact.LastUsed) / 86400000

        newWeight := fact.Weight * float32(math.Exp(-lambda*daysSince))
        if math.Abs(float64(newWeight-fact.Weight)) < 0.001 {
            continue // 变化太小，跳过写入
        }

        // 淘汰：weight 过低且足够老
        if newWeight < 0.05 && daysSince > 90 {
            uc.softDelete(ctx, ws, fact.ID)
            continue
        }

        uc.updateWeight(ctx, ws, fact.ID, newWeight)
    }
    return nil
}
```

### 7.7 Reflect（自动整合）

定时后台任务。聚类相似碎片记忆，生成高阶 reflection。

```go
// internal/biz/memory/reflect.go

func (uc *MemoryUsecase) RunReflect(ctx context.Context, ws, userID string) error {
    facts, _ := uc.listUnconsolidated(ctx, ws, userID)
    if len(facts) < 20 {
        return nil // 记忆太少，不整合
    }

    // 按 subject + tags 聚类
    clusters := clusterBySubjectAndTags(facts)

    for _, cluster := range clusters {
        if len(cluster) < 3 {
            continue // 不够密集
        }

        // 生成整合摘要
        summary := synthesizeCluster(cluster) // 简单拼接，或 LLM 摘要

        reflection := MemoryFact{
            Kind:    FactReflection,
            Subject: cluster[0].Subject,
            Content: summary,
            Tags:    mergeTags(cluster),
            Weight:  avgWeight(cluster) * 1.2, // 升权
            Source:  "reflect",
        }

        uc.createFact(ctx, ws, userID, &reflection)

        // 标记源记忆为已整合（加速衰减）
        for _, f := range cluster {
            uc.markConsolidated(ctx, ws, f.ID)
        }
    }
    return nil
}
```

---

## 8. 检索策略

### 8.1 V1: 结构化检索

```
查询请求
    │
    ├─── Kind 过滤 ──────────── "我要 convention 类型"
    │
    ├─── Tags 匹配 ──────────── "标签包含 sync 或 nats"
    │
    ├─── CodeRef.Path 匹配 ──── "涉及 internal/biz/changedag/ 的记忆"
    │
    ├─── Triple.Subject 匹配 ── "关于 projecttemplate 的事实"
    │
    ├─── ValidTo IS NULL ────── "只看当前有效的事实"
    │
    └─── 复合评分排序 ─────────── weight × recency × tag_match × path_match
```

优点：精确、快速、无外部依赖。
局限：无法处理语义模糊查询（"类似的讨论"）。

### 8.2 Phase B: + Embedding 语义检索

```
查询请求
    │
    ├─── 结构化预筛 ──── 按 kind / tags / scope 缩小候选集
    │
    ├─── Embedding ────── 对 query 生成 embedding
    │
    ├─── pgvector ─────── 在候选集中做 cosine similarity top-K
    │
    └─── 混合评分 ─────── α×semantic_sim + β×structural_score
```

预留接口：

```go
// internal/biz/memory/embedding.go

type EmbeddingProvider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
}

type VectorStore interface {
    Store(ctx context.Context, factID uuid.UUID, vector []float32) error
    Search(ctx context.Context, query []float32, opts VectorSearchOpts) ([]VectorMatch, error)
}

type VectorSearchOpts struct {
    WorkspaceID string
    UserID      string
    Kinds       []FactKind
    TopK        int
    MinScore    float32
}

type VectorMatch struct {
    FactID uuid.UUID
    Score  float32
}
```

Phase B 实现方案：
- `EmbeddingProvider` → 复用 LLM Registry 的 embedding 模型（OpenAI `text-embedding-3-small` 或 Ollama 本地模型）。
- `VectorStore` → PostgreSQL + pgvector 扩展，不引入额外数据库。向量存为独立列或独立表，不放 Change payload（太大）。

---

## 9. 生命周期管理

### 9.1 状态转换

```
                  ┌─── 被召回 ──▶ 强化 (weight↑, access_n++, last_used=now)
                  │
新记忆 ──▶ 活跃 ──┤
                  │
                  └─── 未被召回 ──▶ 衰减 (weight↓)
                                      │
                              ┌───────┴────────┐
                              ▼                ▼
                         weight > 0.05    weight ≤ 0.05
                              │           且 age > 90d
                              │                │
                              ▼                ▼
                          继续衰减          软删除
```

### 9.2 Temporal Validity（时间有效性）

```
事实 A: projecttemplate uses NATS    valid_from: 2026-02-05  valid_to: NULL (当前有效)

写入新事实 B: projecttemplate uses Kafka   valid_from: 2026-06-01

→ 自动关闭 A: valid_to = 2026-05-31

查询 "projecttemplate uses what?"
  → at 2026-03-01 → "NATS"
  → at 2026-07-01 → "Kafka"
  → timeline      → ["NATS (2026-02~2026-05)", "Kafka (2026-06~)"]
```

### 9.3 Consolidation（整合）

```
碎片记忆:
  [incident] sync 改动导致数据丢失 (2月)
  [resolution] 加了 WAL 防护 (2月)
  [incident] sync 并发问题 (3月)
  [resolution] 引入 mutex (3月)
  [review] PR 评审指出 sync 缺少超时 (3月)

  ↓ Reflect 整合

  [reflection] sync 子系统是高风险模块，历史上多次出现并发和数据丢失问题。
               改动时需要：1) WAL 防护 2) mutex 3) 超时控制。
               相关文件: internal/workflow/syncjob/
```

### 9.4 Landmark Staleness（地标失效）

```go
// 后台任务：检查 landmark 引用的文件是否仍然存在/未大幅修改
func (uc *MemoryUsecase) CheckLandmarkStaleness(ctx context.Context, ws string) error {
    landmarks, _ := uc.listLandmarks(ctx, ws)
    for _, lm := range landmarks {
        for _, ref := range lm.Refs {
            // 检查文件是否存在、最后修改时间等
            // 如果文件不存在或大幅变更 → 标记 stale
            if isStale(ref) {
                uc.markLandmarkStale(ctx, ws, lm.ID, fmt.Sprintf("file %s changed significantly", ref.Path))
            }
        }
    }
    return nil
}
```

---

## 10. 代码布局

```
internal/
  biz/
    memory/
      # ── 接口契约（稳定层，不依赖任何实现）──
      types.go              # MemoryFact, Landmark, FactKind, LandmarkKind, Triple, CodeRef
      store.go              # MemoryStore interface + QueryOpts/FactPatch
      extractor.go          # Extractor interface + ExtractOpts
      dedup.go              # Deduplicator interface
      retriever.go          # Retriever interface + RecallOpts/RecallResult/ScoredFact
      assembler.go          # Assembler interface
      maintainer.go         # Maintainer interface
      scene.go              # SceneProfile, SceneDetector interface, 内置 profiles
      embedding.go          # EmbeddingProvider, VectorStore interfaces (Phase B)
      errors.go             # ErrMemoryNotFound, ErrDuplicate, etc.

      # ── 编排层（消费接口，不依赖实现）──
      service.go            # MemoryService: PrepareContext, AfterConversation, Remember, Recall

      # ── Materializer（Change DAG 集成，跟随存储实现）──
      materialize.go        # MemoryMaterializer, LandmarkMaterializer

  infrastructure/
    memory/
      # ── V1 默认实现（每个文件实现一个接口）──
      changedag_store.go    # ChangeDAGMemoryStore → implements MemoryStore
      llm_extractor.go      # LLMExtractor → implements Extractor
      simhash_dedup.go      # SimHashDeduplicator → implements Deduplicator
      structural_retriever.go  # StructuralRetriever → implements Retriever
      markdown_assembler.go # MarkdownAssembler → implements Assembler
      default_maintainer.go # DefaultMaintainer → implements Maintainer
      keyword_detector.go   # KeywordSceneDetector → implements SceneDetector

      # ── Phase B 实现 ──
      # embedding_openai.go   # OpenAIEmbeddingProvider → implements EmbeddingProvider
      # pgvector_store.go     # PgVectorStore → implements VectorStore
      # hybrid_retriever.go   # HybridRetriever → implements Retriever

  service/
    memory/
      memory_service.go     # gRPC/HTTP 服务（list, get, delete, edit memories）

  workflow/
    memoryjob/
      extract_job.go        # 对话结束 → 异步提取 (NATS consumer)
      decay_job.go          # 定时衰减 → Maintainer.Decay()
      reflect_job.go        # 定时整合 → Maintainer.Reflect()
      landmark_check_job.go # Landmark 失效检查 → Maintainer.CheckLandmarkStaleness()

api/
  memory/
    v1/
      memory.proto          # MemoryService proto 定义
```

### 10.1 fx Module 注册

```go
// internal/biz/biz.go — 编排层
fx.Provide(
    memory.NewMemoryService, // 消费所有接口，不依赖实现
    memory.NewMemoryMaterializer,
    memory.NewLandmarkMaterializer,
)

// internal/infrastructure/memory/module.go — V1 默认实现
var Module = fx.Options(
    fx.Provide(
        fx.Annotate(NewChangeDAGMemoryStore,     fx.As(new(memory.MemoryStore))),
        fx.Annotate(NewLLMExtractor,             fx.As(new(memory.Extractor))),
        fx.Annotate(NewSimHashDeduplicator,      fx.As(new(memory.Deduplicator))),
        fx.Annotate(NewStructuralRetriever,      fx.As(new(memory.Retriever))),
        fx.Annotate(NewMarkdownAssembler,        fx.As(new(memory.Assembler))),
        fx.Annotate(NewDefaultMaintainer,        fx.As(new(memory.Maintainer))),
        fx.Annotate(NewKeywordSceneDetector,     fx.As(new(memory.SceneDetector))),
    ),
)

// cmd/server/main.go — 引入
infrastructure.Module,   // 已有
memoryprovider.Module,   // 新增：记忆系统实现层

// ── 替换示例 ──
// 要换 Retriever？只改一行：
//   fx.Annotate(NewHybridRetriever,  fx.As(new(memory.Retriever))),
// 要接入 OpenMemory？替换 Store + Retriever：
//   fx.Annotate(openmemory.NewStore, fx.As(new(memory.MemoryStore))),
//   fx.Annotate(openmemory.NewRetriever, fx.As(new(memory.Retriever))),

// 更新 Materializer 集合
func(
    am *action.ActionMaterializer,
    arm *action.ActionRunMaterializer,
    mm *memory.MemoryMaterializer,      // 新增
    lm *memory.LandmarkMaterializer,    // 新增
) []changedag.Materializer {
    return []changedag.Materializer{am, arm, mm, lm}
}

// internal/service/service.go — 新增
fx.Provide(
    memorysvc.NewMemoryService,
)

// internal/server/grpc.go — 注册
memoryv1.RegisterMemoryServiceServer(s, memorySvc)
```

---

## 11. 集成点

### 11.1 与 LLM Master Agent

Master Agent 的执行循环中嵌入记忆系统：

```
用户消息到达
    │
    ▼
Scene Detector → 确定场景 (companion/dev/learning/task)
    │
    ▼
Retriever.Recall(scene_profile, context)
    │
    ▼
Assembler.Inject(base_system_prompt, recall_result)
    │
    ▼
Master Agent 处理请求（使用注入了记忆的 system prompt）
    │
    ▼
对话结束
    │
    ▼
Extractor.Extract(conversation) → MemoryUsecase.Ingest(facts)  [异步]
```

### 11.2 与 LLM Provider

- **Extract** 调用 `llm.Provider.Chat()` 提取 facts（便宜模型）。
- **Reflect** 可选调用 `llm.Provider.Chat()` 生成整合摘要。
- **Phase B** 的 EmbeddingProvider 通过 LLM Registry 获取 embedding 模型。

### 11.3 与 Plugin 系统

新增 Memory Capability：

```go
CapabilityMemoryRead  Capability = "memory.read"   // 读取记忆
CapabilityMemoryWrite Capability = "memory.write"   // 写入记忆
```

Plugin 可以：
- 读取记忆为自己的上下文（如 code review 插件读取 convention）。
- 写入记忆（如 GitHub 集成插件将 PR review 写为 FactReview）。

### 11.4 与 Device Mesh

Memory 通过 Change DAG 存储 → Change DAG 通过 gRPC Sync 在设备间同步 → Memory 天然跨设备可用。

特殊场景：
- **Edge 设备上的 Agent** 可以直接从本地 object_state 读取记忆，不需要在线。
- **Brain Agent** 在 Hub 上拥有所有设备推送的完整记忆。
- 设备 A 上提取的记忆，通过 Change DAG 同步到设备 B，设备 B 的 Agent 下次对话时自动使用。

### 11.5 与 Action 系统

记忆操作可以暴露为 Action（供 Master Agent tool_use 调用）：

```json
{
    "name": "memory.recall",
    "description": "从记忆系统中检索相关信息",
    "input_schema": {
        "query": "string",
        "kinds": ["string"],
        "top_k": "number"
    }
}
```

```json
{
    "name": "memory.remember",
    "description": "将信息存入记忆系统",
    "input_schema": {
        "kind": "string",
        "subject": "string",
        "content": "string",
        "tags": ["string"]
    }
}
```

这样 Master Agent 可以在对话中主动决定"记住"或"回忆"某些信息。

---

## 12. 分阶段实施

### Phase A: 核心骨架（先跑起来）

| 项目 | 内容 |
|------|------|
| 类型系统 | MemoryFact, Landmark, FactKind, LandmarkKind, Triple, CodeRef |
| Materializer | MemoryMaterializer, LandmarkMaterializer → 注册到 StaticMaterializerRegistry |
| CRUD | MemoryUsecase: createFact, getFact, listFacts, deleteFact → Change DAG |
| 查询 | MemoryQueryRepo: 按 kind/tags/scope 查 object_state JSONB |
| 提取 | Extractor: LLM 从对话提取 facts |
| 去重 | SimHash 快速去重 |
| 注入 | Assembler: 召回结果 → system prompt 片段 |
| API | Proto + Service: list/get/delete memories |
| 场景 | 2 个内置 SceneProfile: companion, dev |

**不做**: embedding, 复杂评分, decay job, reflect job, landmark staleness check, 场景自动检测

### Phase B: 生命周期 + 智能检索

| 项目 | 内容 |
|------|------|
| 衰减 | Decay job: 三级衰减 + 自动淘汰 |
| 强化 | Retriever 召回时自动 touch（access_n++, weight↑） |
| 整合 | Reflect job: 聚类 + 摘要 → FactReflection |
| 时间 | Temporal validity: valid_from/valid_to + 自动关闭 |
| 评分 | 复合评分: weight × recency × tag_match × path_match |
| 场景 | 全部 4 个 SceneProfile + SceneDetector（关键词匹配） |
| Landmark | Landmark CRUD + staleness check |
| Action | memory.recall / memory.remember 暴露为 tool |

**不做**: embedding, pgvector, LLM rerank

### Phase C: 语义检索 + 高级特性

| 项目 | 内容 |
|------|------|
| Embedding | EmbeddingProvider 实现 (OpenAI / Ollama) |
| 向量存储 | pgvector 扩展 + VectorStore 实现 |
| 混合检索 | 结构化预筛 + embedding top-K + 混合评分 |
| LLM Rerank | 对候选集做 LLM 精排 |
| SceneDetector | 升级为 LLM 分类 |
| Plugin | memory.read / memory.write capability 开放给插件 |
| 用户 UI | 记忆管理界面（查看、编辑、删除、搜索） |

---

## 13. 验收标准

### Phase A

1. 对话结束后可异步提取 MemoryFact 并写入 Change DAG。
2. 下次对话开始时，相关记忆自动注入 system prompt。
3. 同一内容不会重复写入（SimHash 去重有效）。
4. 通过 API 可列出、查看、删除记忆。
5. companion 和 dev 两个 SceneProfile 注入不同格式和内容。
6. 记忆通过 Change DAG 在多设备间同步。

### Phase B

1. 长期未使用的记忆 weight 自动下降。
2. 频繁被召回的记忆 weight 自动上升。
3. 3 条以上同主题碎片记忆可被自动整合为 1 条 reflection。
4. Temporal fact 更新时旧值自动关闭，可查历史。
5. 复合评分排序的召回结果比纯 weight 排序更相关。
6. Landmark 引用的文件被删除/大改后标记为 stale。

### Phase C

1. 语义模糊查询（"上次讨论的类似问题"）可返回相关记忆。
2. 混合检索的召回质量优于纯结构化检索。
3. Plugin 可通过 capability 读写记忆。

---

## 14. 非目标

当前阶段明确不做的事：

- **独立向量数据库**（如 Qdrant / Milvus）——pgvector 够用。
- **多用户共享记忆**——V1 记忆绑定到 workspace + user，不跨用户共享。
- **记忆导入/迁移**——不从其他系统（Mem0/Zep）导入。
- **记忆可视化图谱**——不做 waypoint graph 可视化。
- **RL 自适应检索策略**——先用固定权重公式。
- **端到端加密记忆**——V1 明文存储，复用 workspace 级别的加密策略（如有）。
- **记忆容量限制/计费**——V1 不限量。

---

## 更新日志

- **2026-02-25**: v1.1 新增「接口契约」章节（ADR-7），全接口可拔插设计。MemoryStore/Extractor/Deduplicator/Retriever/Assembler/Maintainer/SceneDetector 七个核心接口。实现层从 biz 移至 infrastructure。
- **2026-02-25**: v1.0 初始设计。参考 OpenMemory (CaviraOSS) 的 temporal facts、三级衰减、SimHash 去重、Reflection 整合、复合评分机制。
