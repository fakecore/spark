# DoorX 场景分析

> **类型**: 场景验证
> **状态**: ✅ 完成
> **最后更新**: 2026-02-04
> **关联文档**: [architecture.md](./architecture.md) | [architecture-extras.md](./architecture-extras.md)

本文档通过 5 个具体场景分析 DoorX 系统的行为设计，验证架构的合理性和发现潜在问题。

| 场景 | 描述 | 核心验证点 |
|------|------|-----------|
| 1. 本地复制翻译 | 剪贴板触发即时翻译 | Router、本地优先策略 |
| 2. 本地语音监听 | 实时会议转写 | 音频管道、云端增强 |
| 3. NAS 视频整理 | 长时间工作流 | WorkflowEngine、SSH |
| 4. 本地对话查询 | 多源数据路由 | 意图识别、LAN 节点 |
| 5. 定期项目追踪 | AI 定时唤醒 | Scheduler、状态持久化 |

---

## 场景 1: 本地复制翻译

**用户描述**:
> 本地复制了一段话，进行翻译。是不是负载均衡到多个节点，但是固定一个节点

### 场景分析

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 用户操作流程                                                             │
├─────────────────────────────────────────────────────────────────────────┤
│ 1. 用户在任意应用中复制文本（剪贴板监听触发）                            │
│ 2. DoorX 检测到复制事件                                                 │
│ 3. 识别用户意图：需要翻译                                                │
│ 4. 执行翻译 Action                                                      │
│ 5. 结果通过通知/浮窗展示给用户                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 服务行为设计

#### 1.1 触发机制

```go
// 剪贴板监听服务（本地 Daemon）
type ClipboardMonitor struct {
    eventBus   EventBus
    intentRepo IntentRepository
}

func (m *ClipboardMonitor) OnClipboardChange(ctx context.Context, text string) {
    // 1. 判断是否需要处理（长度、语言等）
    if m.shouldIgnore(text) {
        return
    }

    // 2. 意图识别
    intent := m.intentRepo.Recognize(ctx, text)
    // intent 可能是: translate, summarize, search, none

    if intent.Action == "translate" {
        // 3. 触发 Action
        m.eventBus.Publish(ctx, ActionTriggerEvent{
            ActionName: "translate",
            Input: map[string]any{
                "text": text,
                "from": intent.DetectedLanguage,
                "to":   m.getUserPreferredLanguage(),
            },
        })
    }
}
```

#### 1.2 执行节点选择

**核心问题**: "是否负载均衡到多个节点，但固定一个节点？"

**分析**:

| 方案 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| **固定本地节点** | 始终在用户本地执行 | 延迟最低，无网络依赖 | 本地资源不足时无法处理 |
| **云端负载均衡** | 分发到任意可用云端节点 | 资源无限，可扩展 | 延迟高，依赖网络 |
| **智能路由** | 根据任务复杂度动态选择 | 最优资源利用 | 实现复杂 |

**推荐方案**: **本地优先 + 云端溢出**

```go
type Router struct {
    localNode  *LocalNode
    cloudNodes []*CloudNode
}

func (r *Router) RouteTranslate(ctx context.Context, req TranslateRequest) (*Node, error) {
    // 1. 评估任务复杂度
    complexity := r.estimateComplexity(req)

    // 2. 简单任务（短文本）→ 本地
    if complexity == Low {
        if r.localNode.IsAvailable() {
            return r.localNode, nil
        }
    }

    // 3. 复杂任务（长文本/专业翻译）→ 云端
    //   选择节点时，考虑：
    //   - 同一会话的翻译请求发送到同一节点（保持术语一致性）
    //   - 节点负载
    //   - 地理位置延迟

    sessionID := req.GetSessionID()
    if node := r.sessionCache.Get(sessionID); node != nil {
        if node.IsAvailable() {
            return node, nil  // 固定节点，保持一致性
        }
    }

    // 选择新节点并缓存
    node := r.cloudNodes.SelectLeastLoaded()
    r.sessionCache.Set(sessionID, node, 10*time.Minute)
    return node, nil
}
```

**关于"固定节点"的理解**:

- **会话内固定**: 同一次对话/任务的多次请求发送到同一节点
- **原因**: 翻译术语一致性、上下文记忆、状态保持
- **实现**: 使用 `sessionID` 作为 sticky key

#### 1.3 数据流

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│   Electron   │──────│  Daemon      │──────│  LLM/API     │
│   (Frontend) │      │  (Local)     │      │  (Local/Cloud)│
└──────────────┘      └──────────────┘      └──────────────┘
      ↑                      │
      │                      │
      └────── 结果展示 ───────┘
```

#### 1.4 权限需求

| 权限 | 用途 | 风险等级 |
|------|------|----------|
| `system:clipboard:read` | 读取剪贴板内容 | Low |
| `system:network` | 调用云端翻译 API | Medium |
| `system:notification:show` | 展示翻译结果 | Low |

#### 1.5 潜在问题

1. **剪贴板隐私**: 用户复制敏感内容时是否需要征得同意？
   - 建议: 添加敏感内容检测（密码、Token），跳过处理

2. **翻译质量**: 如何处理专业术语？
   - 建议: 支持用户自定义术语表，作为 Action 输入

3. **离线场景**: 网络断开时如何处理？
   - 建议: 本地缓存轻量级翻译模型，作为降级方案

---

## 场景 2: 本地语音流监听

**用户描述**:
> 本地监听语音流，识别多人声纹，转写文本，可能还会借助线上的能力，比如音频转写，文本转写，对话识别

### 场景分析

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 用户操作流程                                                             │
├─────────────────────────────────────────────────────────────────────────┤
│ 1. 用户启动"会议记录"功能                                               │
│ 2. DoorX 开始监听系统音频流                                             │
│ 3. 实时识别说话人（声纹识别）                                            │
│ 4. 转写音频为文本                                                       │
│ 5. 生成带说话人标记的会议记录                                           │
│ 6. 可选：借助云端进行更精确的转写和分析                                  │
└─────────────────────────────────────────────────────────────────────────┘
```

### 服务行为设计

#### 2.1 组件架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Audio Pipeline                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐  │
│  │ Audio      │───▶│ VAD        │───▶│ Speaker    │───▶│ ASR        │  │
│  │ Capture    │    │ (Voice     │    │ Diarization│    │ (Speech    │  │
│  │            │    │  Activity  │    │ (声纹识别)  │    │  Recognition│  │
│  └────────────┘    │  Detection)│    └────────────┘    │  转写)     │  │
│                    └────────────┘                      └────────────┘  │
│                           │                                   │        │
│                           ▼                                   ▼        │
│                    ┌────────────┐                    ┌────────────┐    │
│                    │ Local      │                    │ Cloud      │    │
│                    │ Processing │                    │ Enhanced   │    │
│                    │ (轻量模型)  │                    │ (高精度)   │    │
│                    └────────────┘                    └────────────┘    │
│                           │                                   │        │
│                           └─────────────┬─────────────────────┘        │
│                                         ▼                              │
│                                  ┌────────────┐                       │
│                                  │ Result     │                       │
│                                  │ Merge      │                       │
│                                  └────────────┘                       │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 2.2 本地/云端协作

```go
type AudioPipeline struct {
    localProcessor  *LocalAudioProcessor
    cloudEnhancer   *CloudAudioEnhancer
    speakerProfiles *SpeakerProfileStore
}

func (p *AudioPipeline) ProcessAudioChunk(ctx context.Context, chunk AudioChunk) error {
    // 1. 本地轻量处理（实时）
    localResult := p.localProcessor.Process(chunk)

    // 2. 判断是否需要云端增强
    needsCloud := p.shouldUseCloud(ctx, localResult)
    // 判断依据：
    // - 说话人数量 > 2（本地声纹识别能力有限）
    // - 转写置信度 < 阈值
    // - 用户请求"高精度模式"

    if needsCloud {
        // 3. 发送音频片段到云端
        cloudResult, err := p.cloudEnhancer.Enhance(ctx, chunk)
        if err == nil {
            // 4. 合并结果（云端结果优先）
            return p.mergeResults(localResult, cloudResult)
        }
        // 降级：使用本地结果
    }

    return localResult
}

func (p *AudioPipeline) shouldUseCloud(ctx context.Context, result LocalResult) bool {
    // 网络可用
    if !p.cloudEnhancer.IsAvailable() {
        return false
    }

    // 说话人数量超过本地能力
    if len(result.DetectedSpeakers) > 2 {
        return true
    }

    // 转写置信度低
    if result.Confidence < 0.8 {
        return true
    }

    // 用户设置
    if p.getUserPreference(ctx, "transcription.quality") == "high" {
        return true
    }

    return false
}
```

#### 2.3 声纹识别设计

```go
type SpeakerProfile struct {
    ID          string
    Name        string  // 用户可编辑的名字，如"张三"
    Voiceprint  []byte  // 声纹特征向量
    Samples     int     // 样本数量
    CreatedAt   int64
    UpdatedAt   int64
}

type SpeakerDiarization struct {
    profileStore *SpeakerProfileStore
    matcher      VoiceprintMatcher
}

func (s *SpeakerDiarization) IdentifySpeaker(ctx context.Context, audio AudioChunk) (*SpeakerProfile, float64, error) {
    // 1. 提取声纹特征
    voiceprint := s.extractVoiceprint(audio)

    // 2. 匹配已知说话人
    profiles := s.profileStore.ListAll()
    bestMatch, score := s.matcher.Match(voiceprint, profiles)

    if score > 0.85 {
        // 高置信度：返回已知说话人
        return bestMatch, score, nil
    }

    if score > 0.6 {
        // 中置信度：可能是新说话人或已知说话人
        // 发送到云端进一步确认
        cloudResult, err := s.cloudIdentify(ctx, voiceprint)
        if err == nil && cloudResult.Confidence > 0.9 {
            return cloudResult.Profile, cloudResult.Confidence, nil
        }
    }

    // 低置信度：新说话人
    return s.profileStore.CreateNew(ctx, voiceprint)
}
```

#### 2.4 数据流

```
实时流式处理:
┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐
│ Audio   │─▶│ VAD     │─▶│ Speaker │─▶│ ASR     │─▶│ Output  │
│ Chunk 1 │  │ Detect  │  │ Identify│  │ Text    │  │ Stream  │
└─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘
     │                                                            │
     │    延迟: < 100ms                                           │
     └────────────────────────────────────────────────────────────┘
                                                                   │
                                                                   ▼
                                                          ┌─────────────┐
                                                          │ Conversation│
                                                          │ Message     │
                                                          │ (实时追加)  │
                                                          └─────────────┘
```

#### 2.5 权限需求

| 权限 | 用途 | 风险等级 |
|------|------|----------|
| `system:audio:capture` | 捕获系统音频 | High |
| `system:screen:record` | （可选）录屏时捕获音频 | High |
| `system:network` | 上传音频到云端 | Medium |

#### 2.6 隐私保护

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 隐私保护策略                                                             │
├─────────────────────────────────────────────────────────────────────────┤
│ 1. 明确授权: 每次录音前需要用户确认（除非设置为"信任应用"）            │
│ 2. 视觉提示: 录音时顶部栏显示红点/麦克风图标                            │
│ 3. 本地优先: 默认使用本地模型，不向外发送                               │
│ 4. 敏感词过滤: 检测到密码、证件号时自动暂停                             │
│ 5. 数据加密: 上传到云端的音频加密传输                                   │
│ 6. 自动删除: 处理后的本地音频自动删除（可配置保留）                     │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 2.7 潜在问题

1. **性能问题**: 实时音频处理对 CPU 要求高
   - 方案: 使用 GPU 加速（如果可用），或降级采样率

2. **多说话人混淆**: 声纹相似的人容易混淆
   - 方案: 支持用户手动标注/修正，模型学习

3. **离线场景**: 云端增强不可用时的体验
   - 方案: 明确告知用户使用"离线模式"，精度可能降低

---

## 场景 3: NAS 视频整理翻译

**用户描述**:
> 指定一个单独nas服务器，对这个服务器的内容，比如视频进行整理，翻译字幕。因为这是个长服务，用云服务器来处理。云上登陆ssh，进行音频提取，专业。提前设计好工作流

### 场景分析

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 用户操作流程                                                             │
├─────────────────────────────────────────────────────────────────────────┤
│ 1. 用户指定 NAS 上的视频文件/目录                                        │
│ 2. 创建"视频处理"工作流:                                                 │
│    - SSH 登录 NAS                                                       │
│    - 提取音频（ffmpeg）                                                 │
│    - 上传音频到云端                                                     │
│    - 语音转写（ASR）                                                    │
│    - 翻译字幕                                                           │
│    - 生成字幕文件（SRT/VTT）                                            │
│    - 上传字幕回 NAS                                                     │
│ 3. 长时间运行，可能需要数小时                                           │
│ 4. 进度通知用户                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 服务行为设计

#### 3.1 工作流定义

```yaml
# workflow: video-subtitle-generation
name: "视频字幕生成"
description: "从视频中提取音频，生成多语言字幕"

variables:
  nas_host: "{user_input}"      # NAS 地址
  nas_video_path: "{user_input}" # 视频路径
  target_languages: ["en", "zh"]

steps:
  # 步骤 1: 连接 NAS
  - id: connect_nas
    type: ssh_connect
    config:
      host: "{{ nas_host }}"
      username: "{{ nas_username }}"
      auth_type: "key"  # 或 "password"
      private_key: "{{ stored_credential }}"

  # 步骤 2: 提取音频
  - id: extract_audio
    type: ssh_command
    depends_on: [connect_nas]
    config:
      command: |
        ffmpeg -i "{{ nas_video_path }}" \
               -vn -acodec pcm_s16le -ar 16000 -ac 1 \
               -y /tmp/extracted_audio.wav
      timeout: 1800  # 30 分钟

  # 步骤 3: 下载音频到云端
  - id: download_audio
    type: scp_download
    depends_on: [extract_audio]
    config:
      source: "/tmp/extracted_audio.wav"
      destination: "{{ cloud_temp_dir }}/audio.wav"

  # 步骤 4: 语音转写
  - id: transcribe
    type: asr_service
    depends_on: [download_audio]
    config:
      audio_path: "{{ cloud_temp_dir }}/audio.wav"
      language: "auto"
      speaker_diarization: true
      output_format: "json"

  # 步骤 5: 翻译字幕
  - id: translate
    type: translate_service
    depends_on: [transcribe]
    config:
      input: "{{ steps.transcribe.output }}"
      target_languages: "{{ target_languages }}"
      preserve_format: true

  # 步骤 6: 生成字幕文件
  - id: generate_srt
    type: generate_subtitle
    depends_on: [translate]
    config:
      transcript: "{{ steps.transcribe.output }}"
      translations: "{{ steps.translate.output }}"
      output_formats: ["srt", "vtt"]

  # 步骤 7: 上传字幕回 NAS
  - id: upload_subtitle
    type: scp_upload
    depends_on: [generate_srt]
    config:
      source: "{{ cloud_temp_dir}}/subtitles/*"
      destination: "{{ nas_video_path | dirname }}/subtitles/"

  # 步骤 8: 清理临时文件
  - id: cleanup
    type: parallel
    depends_on: [upload_subtitle]
    steps:
      - type: ssh_command
        config:
          command: "rm -f /tmp/extracted_audio.wav"
      - type: delete_files
        config:
          path: "{{ cloud_temp_dir }}/audio.wav"
```

#### 3.2 执行节点选择

**关键决策**: 为什么选择云端而非本地执行？

| 因素 | 本地执行 | 云端执行 |
|------|----------|----------|
| **计算资源** | 有限，影响用户使用 | 无限，可扩展 |
| **长时间运行** | 用户关机后中断 | 持续运行 |
| **网络带宽** | 上传音频可能慢 | 云端之间高速 |
| **NAS 访问** | 同网段快 | 跨网段需要 VPN |

**推荐方案**: **云端执行 + 代理访问 NAS**

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          执行架构                                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌─────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────┐  │
│   │  User   │────▶│  Frontend  │────▶│  Router    │────▶│  Cloud  │  │
│   │ Device │     │  (Electron)│     │  (云端)     │     │  Node   │  │
│   └─────────┘     └─────────────┘     └─────────────┘     └────┬────┘  │
│                                                             │         │
│                                                             │         │
│                                    ┌────────────────────────┘         │
│                                    │                                  │
│                                    ▼                                  │
│                          ┌─────────────┐                            │
│                          │  Workflow   │                            │
│                          │  Engine     │                            │
│                          └─────────────┘                            │
│                                    │                                  │
│                    ┌───────────────┼───────────────┐                  │
│                    ▼               ▼               ▼                  │
│            ┌───────────┐    ┌───────────┐    ┌───────────┐            │
│            │ SSH/SCP   │    │ ASR       │    │ Translate │            │
│            │ to NAS    │    │ Service   │    │ Service   │            │
│            └───────────┘    └───────────┘    └───────────┘            │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

```go
type Router struct {
    cloudNodes []*CloudNode
}

func (r *Router) RouteWorkflow(ctx context.Context, wf Workflow) (*Node, error) {
    // 长时间运行的工作流 → 云端
    if wf.EstimatedDuration() > 5*time.Minute {
        return r.cloudNodes.SelectForWorkflow(wf), nil
    }

    // 需要 NAS 访问 → 云端（VPN 配置方便）
    if wf.RequiresNASAccess() {
        return r.cloudNodes.SelectWithVPN(), nil
    }

    // 需要 GPU → 云端
    if wf.RequiresGPU() {
        return r.cloudNodes.SelectWithGPU(), nil
    }

    return r.defaultCloudNode, nil
}
```

#### 3.3 凭证管理

```go
// NAS 访问凭证（敏感，需要安全存储）
type NASCredential struct {
    ID       string
    Name     string  // "我的 NAS"
    Host     string
    Port     int
    Username string
    AuthType AuthType // password | key

    // 加密存储
    Password     *EncryptedField
    PrivateKey   *EncryptedField

    CreatedAt   int64
    LastUsedAt  int64
}

type CredentialStore interface {
    // 存储凭证（加密）
    Store(ctx context.Context, cred *NASCredential) error

    // 获取凭证（需要用户授权）
    Get(ctx context.Context, id string, userConsent bool) (*NASCredential, error)

    // 列出可用的凭证（不包含密码）
    List(ctx context.Context) []*NASCredential
}

// 使用时的授权流程
func (w *WorkflowEngine) ExecuteStep(ctx context.Context, step SSHCommandStep) error {
    // 1. 检查是否有缓存的授权
    if !w.credentialStore.IsAuthorized(step.CredentialID) {
        // 2. 请求用户授权
        consent := w.requestUserConsent(ctx, &ConsentRequest{
            Action:   "ssh.connect",
            Target:   step.CredentialID,
            Duration: 1 * time.Hour,
        })

        if !consent.Approved {
            return fmt.Errorf("user denied credential access")
        }
    }

    // 3. 获取凭证
    cred, err := w.credentialStore.Get(ctx, step.CredentialID, true)
    if err != nil {
        return err
    }

    // 4. 执行 SSH 命令
    return w.executeSSH(ctx, cred, step.Command)
}
```

#### 3.4 进度通知

```go
type WorkflowExecution struct {
    ID          string
    WorkflowID  string
    Status      ExecutionStatus // running | paused | completed | failed
    CurrentStep string
    Progress    float64         // 0.0 - 1.0
    StartedAt   int64
    EstimatedEndAt *int64
    Output      []StepOutput
}

type ProgressNotifier interface {
    // 通知方式
    NotifyProgress(ctx context.Context, exec *WorkflowExecution) error
    NotifyComplete(ctx context.Context, exec *WorkflowExecution) error
    NotifyError(ctx context.Context, exec *WorkflowExecution, err error) error
}

// WebSocket 实时推送
func (n *WebSocketNotifier) NotifyProgress(ctx context.Context, exec *WorkflowExecution) error {
    return n.wsConn.WriteJSON(map[string]any{
        "type":     "workflow.progress",
        "id":       exec.ID,
        "status":   exec.Status,
        "step":     exec.CurrentStep,
        "progress": exec.Progress,
        "eta":      exec.EstimatedEndAt,
    })
}

// 系统通知（完成时）
func (n *SystemNotifier) NotifyComplete(ctx context.Context, exec *WorkflowExecution) error {
    return n.notifier.Show(
        "工作流完成",
        fmt.Sprintf("%s 已完成，共处理 %d 个文件", exec.WorkflowID, len(exec.Output)),
    )
}
```

#### 3.5 权限需求

| 权限 | 用途 | 风险等级 |
|------|------|----------|
| `system:network` | SSH/SCP 连接 NAS | Medium |
| `system:process:exec` | 云端执行 ffmpeg | High |
| `system:file:read` | 读取字幕文件 | Low |
| `system:file:write` | 写入字幕到 NAS | Medium |

#### 3.6 潜在问题

1. **NAS 网络可达性**: 云端如何访问用户内网的 NAS？
   - 方案 A: 用户配置端口转发
   - 方案 B: 使用 Tailscale/ZeroTier 等 VPN
   - 方案 C: NAS 主动连接云端（反向代理）

2. **长时间运行的稳定性**: 工作流中断如何恢复？
   - 方案: 支持检查点（checkpoint），从失败步骤恢复

3. **成本问题**: 云端长时间运行的成本
   - 方案: 显示预估成本，用户确认后再执行

---

## 场景 4: 本地对话查询

**用户描述**:
> 本地发送对话，识别云上，可能是工作节点，也可能是其他的在同个网段的，进行数据查询。

### 场景分析

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 用户操作流程                                                             │
├─────────────────────────────────────────────────────────────────────────┤
│ 1. 用户在 Electron 前端发起对话                                         │
│ 2. LLM 分析用户意图，识别为"数据查询"                                    │
│ 3. Router 决定执行节点：                                                │
│    - 云端工作节点（如查询在线服务）                                      │
│    - 同网段节点（如查询其他设备上的数据）                                │
│ 4. 执行查询，返回结果                                                   │
│ 5. 结果经过 LLM 整理后展示给用户                                        │
└─────────────────────────────────────────────────────────────────────────┘
```

### 服务行为设计

#### 4.1 意图识别与节点选择

```go
type Intent struct {
    Type       IntentType  // query | execute | search | chat
    DataSource DataSource  // cloud | local | lan_device | web
    Query      string
    Parameters map[string]any
}

type DataSource string
const (
    DataSourceCloud     DataSource = "cloud"     // 云端数据库/API
    DataSourceLocal     DataSource = "local"     // 本地文件
    DataSourceLAN       DataSource = "lan"       // 同网段其他设备
    DataSourceWeb       DataSource = "web"       // 互联网搜索
)

type IntentRecognizer struct {
    llmClient LLMClient
}

func (r *IntentRecognizer) Recognize(ctx context.Context, userMessage string) (*Intent, error) {
    // 使用 LLM 识别意图
    prompt := fmt.Sprintf(`
用户消息: %s

请分析用户意图，返回 JSON:
{
  "type": "query",
  "data_source": "cloud|local|lan|web",
  "query": "实际查询内容",
  "parameters": {}
}

判断依据:
- "查询我的文件" → local
- "搜索公司文档" → cloud (工作节点)
- "查一下客厅电脑的温度" → lan (同网段设备)
- "今天天气怎么样" → web
`, userMessage)

    response := r.llmClient.Complete(ctx, prompt)
    return parseIntent(response)
}
```

#### 4.2 节点发现（LAN）

```go
// 局域网节点发现
type LANNodeDiscovery struct {
    registry *NodeRegistry
    mdns     *mDNSResponder
}

func (d *LANNodeDiscovery) Discover(ctx context.Context) []*LANNode {
    // 1. mDNS/Bonjour 发现
    mdnsResults := d.mdns.Browse("_projecttemplate._tcp")

    // 2. 扫描常见端口（如果 mDNS 不可用）
    scanResults := d.scanLocalNetwork()

    // 3. 合并结果
    nodes := append(mdnsResults, scanResults...)

    // 4. 注册到路由表
    for _, node := range nodes {
        d.registry.Register(ctx, node)
    }

    return nodes
}

type LANNode struct {
    ID          string
    Name        string  // "客厅 MacBook"
    IP          string
    Port        int
    Capabilities []string // ["files", "camera", "sensors"]
    AuthToken   string   // 互信凭证
}

// 同网段节点能力
type NodeCapability string
const (
    CapabilityFileQuery   NodeCapability = "files"      // 文件查询
    CapabilityCamera      NodeCapability = "camera"     // 摄像头
    CapabilitySensor      NodeCapability = "sensors"    // 传感器
   CapabilityExecute      NodeCapability = "execute"    // 执行命令
)
```

#### 4.3 路由决策

```go
type Router struct {
    localNode    *LocalNode
    cloudNodes   []*CloudNode
    lanNodes     []*LANNode
    intentRec    *IntentRecognizer
}

func (r *Router) RouteUserQuery(ctx context.Context, userMessage string) (*RouteDecision, error) {
    // 1. 意图识别
    intent, err := r.intentRec.Recognize(ctx, userMessage)
    if err != nil {
        return nil, err
    }

    // 2. 根据数据源选择节点
    switch intent.DataSource {
    case DataSourceLocal:
        return &RouteDecision{
            Node:   r.localNode,
            Reason: "查询本地数据",
        }, nil

    case DataSourceCloud:
        // 选择云端工作节点
        node := r.selectBestCloudNode(ctx, intent)
        return &RouteDecision{
            Node:   node,
            Reason: "查询云端工作数据",
        }, nil

    case DataSourceLAN:
        // 选择同网段节点
        node, err := r.selectLANNode(ctx, intent)
        if err != nil {
            return nil, fmt.Errorf("no available LAN node: %w", err)
        }
        return &RouteDecision{
            Node:   node,
            Reason: "查询同网段设备",
        }, nil

    case DataSourceWeb:
        // 云端节点（有公网访问能力）
        return &RouteDecision{
            Node:   r.cloudNodes.SelectWithInternetAccess(),
            Reason: "互联网搜索",
        }, nil

    default:
        return nil, fmt.Errorf("unknown data source: %s", intent.DataSource)
    }
}

func (r *Router) selectLANNode(ctx context.Context, intent *Intent) (*LANNode, error) {
    // 根据意图筛选有对应能力的节点
    var candidates []*LANNode
    for _, node := range r.lanNodes {
        if node.HasCapability(intent.RequiredCapability) {
            candidates = append(candidates, node)
        }
    }

    if len(candidates) == 0 {
        return nil, fmt.Errorf("no node with capability: %s", intent.RequiredCapability)
    }

    // 选择最佳节点（优先本地网络，延迟低）
    return r.selectLowestLatency(ctx, candidates), nil
}
```

#### 4.4 查询执行

```go
// 统一的查询接口
type QueryExecutor interface {
    Execute(ctx context.Context, query Query) (*QueryResult, error)
}

// 本地文件查询
type LocalFileQuery struct {
    fileIndex *FileIndex
}

func (q *LocalFileQuery) Execute(ctx context.Context, query Query) (*QueryResult, error) {
    // 1. 全文搜索
    files := q.fileIndex.Search(ctx, query.QueryString)

    // 2. 返回结果
    return &QueryResult{
        Source:    "local",
        Files:     files,
        Count:     len(files),
    }, nil
}

// 同网段节点查询（通过 HTTP API）
type LANNodeQuery struct {
    httpClient *http.Client
    node       *LANNode
}

func (q *LANNodeQuery) Execute(ctx context.Context, query Query) (*QueryResult, error) {
    // 1. 构造请求
    req := &NodeQueryRequest{
        Query: query.QueryString,
        Filter: query.Filter,
    }

    // 2. 发送到目标节点
    resp, err := q.httpClient.PostContext(ctx,
        q.node.QueryEndpoint(),
        "application/json",
        toJSON(req),
    )
    if err != nil {
        return nil, fmt.Errorf("query LAN node failed: %w", err)
    }
    defer resp.Body.Close()

    // 3. 解析结果
    var result NodeQueryResponse
    json.NewDecoder(resp.Body).Decode(&result)

    return &QueryResult{
        Source: "lan",
        NodeID: q.node.ID,
        Data:   result.Data,
    }, nil
}

// 云端工作节点查询
type CloudWorkQuery struct {
    db     *sql.DB
    client *APIClient
}

func (q *CloudWorkQuery) Execute(ctx context.Context, query Query) (*QueryResult, error) {
    // 1. 查询云端数据库
    rows, err := q.db.QueryContext(ctx, query.SQL)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // 2. 或者调用云端 API
    resp, err := q.client.Query(ctx, query.QueryString)
    if err != nil {
        return nil, err
    }

    return &QueryResult{
        Source: "cloud_work",
        Data:   resp.Data,
    }, nil
}
```

#### 4.5 结果整合

```go
// 多源查询结果整合
type ResultAggregator struct {
    llmClient LLMClient
}

func (a *ResultAggregator) Aggregate(ctx context.Context, results []*QueryResult, userQuery string) (*AggregatedResult, error) {
    // 1. 收集所有结果
    var allResults []string
    for _, r := range results {
        allResults = append(allResults, r.Format())
    }

    // 2. 使用 LLM 整理结果
    prompt := fmt.Sprintf(`
用户问题: %s

查询结果:
%s

请将以上结果整理成简洁、友好的回答，只保留相关信息。
`, userQuery, strings.Join(allResults, "\n\n"))

    answer := a.llmClient.Complete(ctx, prompt)

    return &AggregatedResult{
        Answer:      answer,
        Sources:     extractSources(results),
        RawResults:  results,
    }, nil
}
```

#### 4.6 数据流

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          查询路由流程                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────┐                                                            │
│  │  User   │                                                            │
│  │"查一下  │                                                            │
│  │ 客厅电脑 │                                                            │
│  │ 的温度" │                                                            │
│  └────┬────┘                                                            │
│       │                                                                  │
│       ▼                                                                  │
│  ┌─────────────┐       ┌─────────────┐                                  │
│  │   Intent    │──────▶│ "查询传感器" │                                  │
│  │ Recognizer  │       │ "lan_device" │                                  │
│  └─────────────┘       └──────┬──────┘                                  │
│                                │                                         │
│                                ▼                                         │
│                    ┌───────────────────────┐                             │
│                    │  Router               │                             │
│                    │  - 发现 LAN 节点       │                             │
│                    │  - 匹配能力           │                             │
│                    │  - 选择最佳节点       │                             │
│                    └───────┬───────────────┘                             │
│                            │                                             │
│            ┌───────────────┼───────────────┐                             │
│            ▼               ▼               ▼                             │
│     ┌──────────┐    ┌──────────┐    ┌──────────┐                       │
│     │  Local   │    │   Cloud  │    │    LAN   │                       │
│     │  Files   │    │   Work   │    │  Node    │ ◀── 选中               │
│     └──────────┘    └──────────┘    └─────┬────┘                       │
│                                        │                               │
│                                        ▼                               │
│                               ┌──────────────┐                          │
│                               │  LAN Node    │                          │
│                               │  Query API   │                          │
│                               │  /sensors    │                          │
│                               └──────┬───────┘                          │
│                                      │                                  │
│                                      ▼                                  │
│                               ┌──────────────┐                          │
│                               │  Result      │                          │
│                               │  Aggregator  │                          │
│                               └──────┬───────┘                          │
│                                      │                                  │
│                                      ▼                                  │
│                               ┌──────────────┐                          │
│                               │  客厅电脑温度  │                          │
│                               │  目前是 24°C  │                          │
│                               └──────────────┘                          │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 4.7 权限需求

| 权限 | 用途 | 风险等级 |
|------|------|----------|
| `system:network` | LAN 节点通信 | Medium |
| `system:file:read` | 读取本地文件 | Medium |
| `system:clipboard:read` | （可选）读取剪贴板作为查询内容 | Low |

#### 4.8 互信凭证（LAN 节点）

```go
// 同网段节点互信
type NodeAuth struct {
    LocalNodeID  string
    TrustedNodes []string
    SharedSecret []byte  // 预共享密钥
}

func (n *LANNode) Authenticate(req *NodeQueryRequest) bool {
    // 验证请求签名
    expectedMAC := hmac.New(sha256.New, n.SharedSecret)
    expectedMAC.Write([]byte(req.Query))
    expectedMAC.Write([]byte(req.Timestamp))

    return hmac.Equal(req.Signature, expectedMAC.Sum(nil))
}
```

#### 4.9 潜在问题

1. **LAN 节点发现不稳定**: mDNS 可能被防火墙阻止
   - 方案: 支持手动添加节点 IP

2. **跨网段访问**: 多个子网时无法发现
   - 方案: 支持配置网关/中继节点

3. **查询结果质量**: 多源结果可能冲突
   - 方案: LLM 去重、置信度评分

---

## 场景 5: 定期项目追踪

**用户描述**:
> 关注 langchain 项目的 issue 和 changelog，定期整理报告给我。追踪 React 19 的 RFC 进展，有重要更新通知我。

### 场景分析

```
┌─────────────────────────────────────────────────────────────────────────┐
│ 用户操作流程                                                             │
├─────────────────────────────────────────────────────────────────────────┤
│ 1. 用户描述追踪需求（项目、关注点、频率、通知方式）                       │
│ 2. DoorX 创建 RecurringTask（定期任务）                                 │
│ 3. 定时器按 schedule 唤醒 AI                                            │
│ 4. AI 恢复上下文，执行追踪任务：                                         │
│    - 获取上次状态（cursor）                                             │
│    - 拉取新数据（issues, releases, RSS...）                             │
│    - 分析、整理、生成报告                                               │
│    - 判断是否需要即时通知                                               │
│    - 更新状态                                                           │
│ 5. 输出报告/通知给用户                                                  │
│ 6. 等待下次唤醒                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 核心概念：RecurringTask（定期唤醒 AI）

与 Action/Workflow 的区别：

| 执行类型 | 触发方式 | 终点 | 状态 | 典型场景 |
|---------|---------|------|------|---------|
| **Action** | 用户/事件 | 明确 | 无 | 翻译、查询 |
| **Workflow** | 用户 | 明确 | 检查点 | 视频处理 |
| **RecurringTask** | 定时/Webhook | 无 | Cursor + Memory | 项目追踪 |

**本质**：`RecurringTask = 定时器 + 上下文恢复 + AI 执行 + 状态保存`

### 服务行为设计

#### 5.1 数据模型

```go
type RecurringTask struct {
    ID          string
    Name        string           // "LangChain 项目追踪"

    // 任务定义（AI 的"记忆"）
    SystemPrompt string          // AI 的角色和目标
    UserPrompt   string          // 用户原始请求

    // 调度配置
    Schedule    ScheduleConfig   // cron / interval / webhook

    // 状态（AI 的"笔记"）
    State       TaskState

    // 输出配置
    Outputs     []OutputConfig   // 通知、文件、邮件...

    // 生命周期
    Status      TaskStatus       // active | paused | archived
    CreatedAt   time.Time
    LastRunAt   *time.Time
    NextRunAt   *time.Time
}

type ScheduleConfig struct {
    Type     ScheduleType  // cron | interval | webhook
    Cron     string        // "0 9 * * 1" (每周一早上9点)
    Interval time.Duration // 6h
    Webhook  string        // 接收外部推送的 URL
}

type TaskState struct {
    LastRun  time.Time
    Cursor   map[string]any  // 上次处理到哪（AI 可读写）
    Memory   string          // AI 的自由笔记（可选）
    History  []RunSummary    // 历史运行摘要
}

type RunSummary struct {
    RunAt      time.Time
    ItemsFound int
    Summary    string  // AI 生成的摘要
    Notified   bool    // 是否发送了通知
}
```

#### 5.2 AI 可用的 Tool

```go
// 状态管理 Tool（AI 自己管理状态）
type StateTool struct{}

func (t *StateTool) GetCursor(key string) (any, error)
func (t *StateTool) SetCursor(key string, value any) error
func (t *StateTool) GetMemory() (string, error)
func (t *StateTool) SetMemory(content string) error
func (t *StateTool) AppendHistory(summary string) error

// 数据源 Tool
type GitHubTool struct{}

func (t *GitHubTool) ListIssues(repo string, since time.Time, labels []string) ([]Issue, error)
func (t *GitHubTool) ListReleases(repo string, since string) ([]Release, error)
func (t *GitHubTool) GetIssue(repo string, number int) (*Issue, error)

type RSSFetchTool struct{}

func (t *RSSFetchTool) Fetch(url string, since time.Time) ([]FeedItem, error)

type WebScrapeTool struct{}

func (t *WebScrapeTool) FetchPage(url string, selector string) (string, error)
func (t *WebScrapeTool) CompareWithLast(url string) (*PageDiff, error)

// 输出 Tool
type OutputTool struct{}

func (t *OutputTool) Notify(title, body string, priority string) error
func (t *OutputTool) SaveReport(path, content string) error
func (t *OutputTool) SendEmail(to, subject, body string) error
```

#### 5.3 执行流程

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        定期唤醒 AI 执行流程                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐                                                        │
│  │  Scheduler  │  定时器触发 / Webhook 触发                              │
│  │  (cron)     │                                                        │
│  └──────┬──────┘                                                        │
│         │                                                                │
│         ▼                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  构造 AI Prompt                                                  │    │
│  │                                                                  │    │
│  │  System: {{ task.SystemPrompt }}                                │    │
│  │                                                                  │    │
│  │  Context:                                                        │    │
│  │  - 用户请求: {{ task.UserPrompt }}                              │    │
│  │  - 上次运行: {{ state.LastRun }}                                │    │
│  │  - 当前状态: {{ state.Cursor | toJSON }}                        │    │
│  │  - 历史摘要: {{ state.History | last 3 }}                       │    │
│  │                                                                  │    │
│  │  Task: 现在请执行追踪任务，使用提供的 Tool 完成工作。           │    │
│  │                                                                  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│         │                                                                │
│         ▼                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  AI 执行（ReAct 循环 + Tool 调用）                               │    │
│  │                                                                  │    │
│  │  1. state.get_cursor("last_issue_time")                         │    │
│  │  2. github.list_issues(repo, since=last_time)                   │    │
│  │  3. github.list_releases(repo, since=last_tag)                  │    │
│  │  4. [分析数据，判断重要性]                                       │    │
│  │  5. output.notify(...) // 如果有紧急更新                        │    │
│  │  6. output.save_report(...)                                     │    │
│  │  7. state.set_cursor("last_issue_time", new_time)               │    │
│  │  8. state.append_history("发现 N 个更新...")                    │    │
│  │                                                                  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│         │                                                                │
│         ▼                                                                │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  保存状态 & 等待下次触发                                         │    │
│  │                                                                  │    │
│  │  - 更新 task.State                                               │    │
│  │  - 更新 task.LastRunAt                                           │    │
│  │  - 计算 task.NextRunAt                                           │    │
│  │                                                                  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 5.4 用户交互流程

```
用户: "关注 langchain 项目的 issue 和 changelog，定期整理报告给我"

DoorX (意图识别 → 创建 RecurringTask):

DoorX: "我来帮你创建一个追踪任务：

  📊 追踪目标: langchain-ai/langchain
  📋 追踪内容: Issues + Releases
  ⏰ 检查频率: 每天早上 9:00
  📬 通知方式: 有重要更新时推送，每周生成汇总报告

  确认创建吗？或者你想调整什么？"

用户: "每周一汇总就行，重要 bug 马上通知我"

DoorX: "好的，已调整：
  - 每周一 9:00 生成周报
  - 标记为 bug 且 priority:high 的 issue 立即通知

  ✅ 追踪任务已创建 (ID: task-abc123)

  你可以随时说「暂停 langchain 追踪」或「查看追踪历史」"
```

#### 5.5 YAML 定义格式

```yaml
# recurring_task: langchain-tracker

name: "LangChain 项目追踪"
description: "追踪 LangChain 的 issues 和 releases，定期生成报告"

system_prompt: |
  你是一个项目追踪助手。你的任务是定期追踪指定项目的更新，并生成报告。

  你可以使用以下工具：
  - github.list_issues: 获取 issue 列表
  - github.list_releases: 获取 release 列表
  - state.get_cursor/set_cursor: 读写你的状态（记住上次处理到哪）
  - state.append_history: 记录本次运行摘要
  - output.notify: 发送即时通知（仅用于紧急事项）
  - output.save_report: 保存报告文件

  重要规则：
  1. 使用 state.get_cursor("xxx") 获取上次处理位置，避免重复处理
  2. 只处理新增/更新的内容
  3. 完成后用 state.set_cursor("xxx", value) 更新位置
  4. 高优先级 bug 或 breaking change 需要立即 notify
  5. 生成简洁、结构化的报告

user_prompt: |
  追踪 langchain-ai/langchain 项目：
  - 关注所有 issues（特别是 bug 和 enhancement 标签）
  - 关注所有 releases
  - 每周生成一份报告
  - 紧急 bug（priority:high）立即通知我

schedule:
  type: cron
  cron: "0 9 * * 1"  # 每周一早上 9 点

outputs:
  - type: notification
    config:
      default_priority: normal

  - type: file
    config:
      path: "~/Documents/tracking-reports/langchain/{{ .date }}.md"
```

#### 5.6 AI 执行示例

```
[Scheduler] 唤醒任务: langchain-tracker (2024-01-15 09:00)

[AI] 开始执行 LangChain 项目追踪任务。先获取上次的状态。

[Tool] state.get_cursor("last_issue_updated")
[Result] "2024-01-08T10:00:00Z"

[Tool] state.get_cursor("last_release_tag")
[Result] "v0.1.5"

[AI] 上次处理到 2024-01-08，release v0.1.5。现在获取之后的更新。

[Tool] github.list_issues("langchain-ai/langchain", since="2024-01-08T10:00:00Z", labels=["bug", "enhancement"])
[Result] [
  {number: 12350, title: "Memory leak in ConversationChain", labels: ["bug", "priority:high"], updated: "2024-01-10"},
  {number: 12355, title: "Add support for Claude 3", labels: ["enhancement"], updated: "2024-01-12"},
  {number: 12361, title: "Improve streaming performance", labels: ["enhancement"], updated: "2024-01-14"}
]

[Tool] github.list_releases("langchain-ai/langchain", since="v0.1.5")
[Result] [
  {tag: "v0.1.6", name: "Bug fixes", published: "2024-01-11"},
  {tag: "v0.1.7", name: "Claude 3 support", published: "2024-01-14"}
]

[AI] 发现一个高优先级 bug (#12350)，需要立即通知。

[Tool] output.notify("⚠️ LangChain 紧急 Bug", "Memory leak in ConversationChain #12350\nhttps://github.com/langchain-ai/langchain/issues/12350", priority="high")
[Result] OK

[AI] 现在生成周报。

[AI 生成报告]:
---
# LangChain 周报
**周期**: 2024-01-08 ~ 2024-01-15

## 🚨 紧急问题
- **Memory leak in ConversationChain** [#12350](https://github.com/langchain-ai/langchain/issues/12350)
  - 状态: Open
  - 优先级: High
  - 影响: ConversationChain 长时间运行时内存持续增长

## 📦 新版本发布
| 版本 | 日期 | 主要更新 |
|------|------|---------|
| v0.1.6 | 01-11 | Bug 修复 |
| v0.1.7 | 01-14 | Claude 3 支持 |

## 💡 新功能讨论
- **Add support for Claude 3** [#12355](https://github.com/langchain-ai/langchain/issues/12355) - 已在 v0.1.7 实现
- **Improve streaming performance** [#12361](https://github.com/langchain-ai/langchain/issues/12361) - 讨论中

## 📊 统计
- 新增/更新 Issues: 3
- 新版本: 2
- 紧急通知: 1
---

[Tool] output.save_report("~/Documents/tracking-reports/langchain/2024-01-15.md", <报告内容>)
[Result] OK

[Tool] state.set_cursor("last_issue_updated", "2024-01-15T09:00:00Z")
[Result] OK

[Tool] state.set_cursor("last_release_tag", "v0.1.7")
[Result] OK

[Tool] state.append_history("发现 3 个 issue 更新，2 个新版本。已发送 1 条紧急通知。")
[Result] OK

[AI] ✅ 本周追踪完成。
```

#### 5.7 执行节点选择

**核心问题**: RecurringTask 在哪里执行？

| 方案 | 描述 | 适用场景 |
|------|------|---------|
| **本地执行** | 用户设备上执行 | 需要本地文件访问 |
| **云端执行** | DoorX 云端执行 | 用户设备可能离线 |

**推荐方案**: **云端执行**

```go
func (r *Router) RouteRecurringTask(ctx context.Context, task *RecurringTask) (*Node, error) {
    // RecurringTask 默认云端执行
    // 原因：
    // 1. 用户设备可能关机/离线
    // 2. 定时任务需要可靠触发
    // 3. 云端有稳定的网络访问（GitHub API 等）

    return r.cloudNodes.SelectDefault(), nil
}
```

**本地执行场景**（需要显式指定）：
- 追踪本地文件变化
- 监控本地服务
- 用户明确要求本地执行

#### 5.8 权限需求

| 权限 | 用途 | 风险等级 |
|------|------|----------|
| `system:network` | 访问 GitHub/RSS 等外部 API | Medium |
| `system:file:write` | 保存报告文件 | Low |
| `system:notification:show` | 发送通知 | Low |
| `external:github` | GitHub API 访问（需要 Token） | Medium |

#### 5.9 数据流

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        RecurringTask 数据流                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌─────────────┐                                                       │
│   │  Scheduler  │                                                       │
│   │  (Cloud)    │                                                       │
│   └──────┬──────┘                                                       │
│          │ trigger                                                       │
│          ▼                                                               │
│   ┌─────────────┐     ┌─────────────┐                                   │
│   │   Task      │────▶│   State     │  读取/更新状态                     │
│   │   Runner    │     │   Store     │                                   │
│   └──────┬──────┘     └─────────────┘                                   │
│          │                                                               │
│          │ 构造 prompt + context                                        │
│          ▼                                                               │
│   ┌─────────────┐                                                       │
│   │   AI/LLM    │  ReAct 循环                                           │
│   │   (Eino)    │                                                       │
│   └──────┬──────┘                                                       │
│          │                                                               │
│          │ Tool 调用                                                    │
│          ▼                                                               │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                          Tool Layer                              │   │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │   │
│   │  │ GitHub  │  │  RSS    │  │  State  │  │ Output  │            │   │
│   │  │  Tool   │  │  Tool   │  │  Tool   │  │  Tool   │            │   │
│   │  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘            │   │
│   └───────┼────────────┼────────────┼────────────┼──────────────────┘   │
│           │            │            │            │                       │
│           ▼            ▼            ▼            ▼                       │
│      ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐                │
│      │ GitHub  │  │  RSS    │  │   DB    │  │ Notify  │                │
│      │  API    │  │  Feeds  │  │         │  │ + File  │                │
│      └─────────┘  └─────────┘  └─────────┘  └─────────┘                │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 5.10 管理命令

用户可以通过对话管理 RecurringTask：

| 用户指令 | 操作 |
|---------|------|
| "暂停 langchain 追踪" | `task.Status = paused` |
| "恢复 langchain 追踪" | `task.Status = active` |
| "查看追踪历史" | 展示 `task.State.History` |
| "立即执行一次" | 手动触发执行 |
| "修改为每天检查" | 更新 `task.Schedule` |
| "删除这个追踪任务" | 归档或删除 |

#### 5.11 潜在问题

1. **API 限流**: GitHub API 有请求限制
   - 方案: 合理设置检查频率，使用 Token 提高限额，缓存结果

2. **AI 执行不稳定**: LLM 可能产生不一致的输出
   - 方案: 结构化 prompt，Tool 返回结构化数据，关键操作有重试

3. **状态丢失**: Cursor 丢失导致重复处理
   - 方案: 状态持久化到数据库，有备份机制

4. **成本控制**: 频繁唤醒 AI 产生的 Token 消耗
   - 方案: 显示预估成本，允许用户设置预算上限

5. **通知疲劳**: 过多通知影响用户体验
   - 方案: AI 判断重要性，合并低优先级通知，用户可调整阈值

---

## 总结：架构验证

### 场景覆盖的核心组件

| 组件 | 场景1 | 场景2 | 场景3 | 场景4 | 场景5 |
|------|-------|-------|-------|-------|-------|
| **IntentRecognizer** | ✅ | - | - | ✅ | ✅ |
| **Router** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **LocalNode** | ✅ | ✅ | - | ✅ | - |
| **CloudNode** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **LANNode** | - | - | - | ✅ | - |
| **WorkflowEngine** | - | - | ✅ | - | - |
| **Scheduler** | - | - | - | - | ✅ |
| **RecurringTaskRunner** | - | - | - | - | ✅ |
| **StateStore** | - | - | ✅ | - | ✅ |
| **PermissionChecker** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **CredentialStore** | - | - | ✅ | ✅ | ✅ |
| **EventBus** | ✅ | ✅ | ✅ | ✅ | ✅ |

### DoorX 执行类型总览

| 执行类型 | 触发方式 | 终点 | 状态管理 | 典型场景 |
|---------|---------|------|---------|---------|
| **Action (Simple)** | 用户/事件 | 明确 | 无 | 翻译、查询 |
| **Action (Chain)** | 用户/事件 | 明确 | 无 | 多步骤工具调用 |
| **Workflow (DAG)** | 用户 | 明确 | 检查点 | NAS 视频处理 |
| **RecurringTask** | 定时/Webhook | 无（持续） | Cursor + Memory | 项目追踪 |

### 发现的架构问题

1. **Q2.2 多服务模型**: 场景4 中"工作节点" vs "个人节点"的区别确实存在
   - 建议: 采用 ServiceInstance 模型

2. **Q3.1 分层边界**: WorkflowEngine 的复杂性需要清晰的分层
   - 建议: 采用 Kratos 标准分层 + 领域服务

3. **Q4.1 安全模型**: 多场景都涉及敏感权限
   - 建议: 实施完整的安全模型

4. **Q5.1 同步方案**: 场景3 的长时间工作流需要状态同步
   - 建议: 支持 Logical Replication + 应用层事件

5. **Q7.1 执行协议**: 场景3 的工作流调度需要可靠的任务分发
   - 建议: 采用 LISTEN/NOTIFY 或消息队列

6. **节点发现**: 场景4 的 LAN 节点发现是新需求
   - 建议: 补充 mDNS/Bonjour 支持

7. **定时调度**: 场景5 需要可靠的定时任务调度
   - 建议: 云端 Scheduler 组件，支持 cron/interval/webhook 触发

8. **AI 状态持久化**: 场景5 的 AI 需要跨执行周期保持状态
   - 建议: StateStore 提供 Cursor + Memory 存储，AI 通过 Tool 读写

---

*文档版本: 2026-02-04*
