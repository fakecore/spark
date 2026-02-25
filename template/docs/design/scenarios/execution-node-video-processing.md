# 场景：独立执行节点完成长视频整理与字幕翻译

## 背景与目标

在现有架构（Go 核心 + Router + Sync + Execution Node）下，支持“指定某台独立服务器”执行长时间视频任务：抽取音频、转写、翻译、整理，并将产物写回指定目录或对象存储，最终同步到云端与客户端。

目标：
- 视频可能位于执行节点本地磁盘
- 执行节点不运行主库，仅运行 runner
- 任务执行稳定、可见进度、可回收结果

---

## 参与角色

- **Client UI**：用户发起 Action，选择执行节点
- **Cloud Control Plane**：路由与编排（Router + Task Queue）
- **Execution Node**：独立服务器 runner
- **Object Storage**：可选，用于大文件中转与产物存储

---

## 交互与执行流程（推荐）

### 1) 用户侧发起

- 用户在 UI 选择 Action：`视频整理 + 字幕翻译`
- 指定执行节点（`cloud@node_id`）
- 选择输入来源：
  - `node://`（节点本地文件）
  - `s3://`（对象存储）

### 2) 云端路由

- Router 验证权限 + 节点能力（ASR / GPU / 视频处理）
- 下发任务到指定节点（含输入引用、输出约定）

### 3) 节点执行

- 解析任务描述
- 执行链：
  1) `ffmpeg` 抽取音频
  2) ASR 转写
  3) 内容整理/摘要
  4) 字幕翻译
  5) 产物落盘

### 4) 结果回写

- 小产物（字幕/摘要/索引）直接回写云端主库
- 大产物上传对象存储，仅回写引用
- 本地通过 Sync 获得更新

---

## 输入/输出约定（Action 级）

建议在 Action 定义中明确：

- `inputRef`: `node://` 或 `s3://`
- `outputDir`: 相对工作目录，防止越权写入
- `artifacts`: 产物清单（SRT/VTT/MD/JSON）

示例任务描述：

```json
{
  "action": "video_subtitle_translate",
  "target": "cloud@node_id",
  "inputRef": "node:///data/videos/foo.mp4",
  "outputDir": "outputs/foo/",
  "artifacts": ["subtitle.srt", "summary.md", "transcript.json"]
}
```

---

## 用户体验（建议）

- 节点选择器显示：地域、能力标签、负载
- 长任务面板：进度、预计耗时、取消/重试
- 任务完成后自动打开产物目录或展示下载链接

---

## 最终效果（端到端）

用户能看到的最终效果应满足：

- 在 UI 里选择“视频整理 + 字幕翻译”，并指定执行节点
- 任务开始后实时看到进度（抽取音频 / 转写 / 翻译 / 产物生成）
- 任务完成后拿到：字幕文件（SRT/VTT）、摘要（MD）、结构化转写（JSON）
- 产物默认进入“与视频同级目录的输出文件夹”，并同步到云端可下载

如果视频本就位于节点磁盘，整个流程无需上传大文件；仅产物回写云端。

---

## 任务状态机（建议）

```
created → queued → running → (succeeded | failed | canceled)
                ↘ retrying ↗
```

关键状态字段：
- `phase`: 当前阶段（extract_audio / asr / summarize / translate / publish）
- `progress`: 0-100
- `eta`: 预计完成时间
- `attempt`: 重试次数

---

## 控制面接口（建议草图）

```
POST /api/v1/tasks                 # 创建任务
GET  /api/v1/tasks/:id             # 查询任务状态
POST /api/v1/tasks/:id/cancel      # 取消任务

POST /api/v1/nodes/register        # 节点注册
POST /api/v1/nodes/heartbeat       # 心跳
POST /api/v1/nodes/lease           # 拉取任务（节点主动拉取）
POST /api/v1/nodes/report          # 上报进度/结果
```

节点建议“主动拉取任务”，避免防火墙/NAT 影响。

---

## 数据流与目录策略（建议）

### 输入策略

1) **节点本地视频（优先）**  
   - `inputRef = node:///data/videos/foo.mp4`  
   - 节点直接读取，无需上传

2) **云端对象存储**  
   - `inputRef = s3://bucket/path/foo.mp4`  
   - 节点下载到本地工作目录

### 输出策略

- `outputDir` 必须是节点工作目录下的相对路径  
- 产物写到 `outputDir`，由节点回写到云端主库  
- 小产物（文本/元数据）直接回写主库  
- 大产物（字幕/大文本）上传对象存储，主库仅保存引用

推荐目录结构：

```
node_workdir/
  jobs/<task_id>/
    input/
    output/
    logs/
```

---

## 执行节点的运行要求（最小可行）

- `ffmpeg`（音频抽取）
- ASR 引擎（本地模型或云端 SDK）
- 翻译模型/LLM 客户端
- 与控制面通信的 runner（可容器化部署）

节点无需数据库，仅需：
- 临时工作目录
- 与云端的通信凭证

---

## 失败处理与幂等（建议）

- 任务采用 `task_id` + `attempt` 唯一标识  
- 输出产物带版本号或覆盖策略  
- 重试从上一个阶段恢复（如音频已生成则跳过抽取）  
- 失败后保留中间产物用于诊断  

---

## 安全与权限边界（建议）

- 节点只允许写入工作目录白名单  
- `outputDir` 必须校验不可越权（禁止 `..` 等）  
- 节点 token 可被吊销  
- 任务描述带签名，防篡改  

---

## 架构适配性与缺口

### 当前架构已满足

- Router 支持 `cloud@node_id`
- Execution Node 作为独立 runner 不需要主库
- Sync 能将结果同步到客户端

### 需要补齐/明确的能力（建议）

1) **任务控制面**
   - 需要 Task Queue / Task 状态机（pending/running/failed/succeeded）
   - 进度上报与取消接口

2) **节点注册与能力发现**
   - 节点能力声明（GPU/ASR/视频处理）
   - 心跳与失效机制

3) **数据就近与存储抽象**
   - 统一输入引用协议（node:// / s3://）
   - 大文件的拉取/上传工具

4) **输出与权限边界**
   - 输出目录必须在白名单内
   - 产物落盘路径安全校验

5) **幂等与重试**
   - 支持任务重试与断点续跑
   - 防重复写入与版本冲突

6) **安全与隔离**
   - 节点 Token 与任务签名
   - 最小权限运行（容器/沙箱）

---

## 结论

该场景在当前架构下**可行**，但需要补齐“任务控制面 + 节点能力发现 + 数据引用协议 + 产物安全边界”等基础设施。补齐后即可稳定支撑独立服务器执行长视频任务。
