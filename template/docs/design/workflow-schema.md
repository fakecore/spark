# Workflow Schema 设计

> **类型**: 功能设计
> **状态**: ✅ 完成
> **最后更新**: 2026-02-04
> **关联文档**: [error-handling.md](./error-handling.md)

---

## 完整 Schema 定义

```yaml
# 工作流元数据
id: workflow.id                      # 唯一标识符 (必需)
name: 工作流名称                      # 人类可读名称 (必需)
description: 工作流描述               # 详细说明 (可选)
version: "1.0"                        # 版本号 (可选)
author: user                         # 作者 (可选)
tags: [video, nas, subtitle]         # 标签 (可选)

# SSH 目标配置 (可选，也可引用全局配置)
ssh_targets:
  nas:                               # 目标名称
    host: "192.168.1.100"            # 主机地址
    port: 22                          # 端口
    user: "admin"                     # 用户名
    auth_type: "key"                  # 认证方式: key | password
    # 密钥/密码从安全存储读取，不在 YAML 中明文

# 输入参数定义
input:
  properties:
    video_url:
      type: string
      description: 视频 URL 或路径
    target_language:
      type: string
      default: "zh"
      description: 目标语言
  required: [video_url]

# 输出定义
output:
  properties:
    subtitle_path:
      type: string
      description: 生成的字幕文件路径
    transcript:
      type: string
      description: 转写文本

# 变量定义（内部状态）
variables:
  temp_dir: "/tmp/workflows"
  video_name: "{{.input.video_url | basename}}"
  docker_image: "ffmpeg:latest"

# 步骤定义
steps:
  - id: step_id                       # 步骤唯一标识 (必需)
    name: 步骤名称                     # 人类可读名称 (必需)
    description: 步骤描述               # 详细说明 (可选)

    # 工具调用
    tool: tool.name                    # 工具名称 (必需)

    # 执行模式
    execution: local                   # local | ssh (默认: local)
    target: nas                        # SSH 目标 (execution=ssh 时必需)

    # 输入参数 (支持变量引用)
    input:
      param1: "{{.variables.video_name}}"
      param2: "{{.steps.previous_step.output}}"

    # 输出变量 (可选)
    output: step_output                # 输出变量名

    # 依赖关系 (可选)
    depends_on: [step1, step2]        # 依赖的步骤 ID

    # 并行执行 (可选)
    parallel: false                    # true | false (默认: false)

    # 条件执行 (可选)
    condition: "{{.variables.enable_step == true}}"

    # 重试配置 (可选)
    retry:
      max_attempts: 3                 # 最大重试次数
      backoff: exponential            # 重试策略: constant | linear | exponential
      delay: 1000                     # 延迟 (毫秒)

    # 超时配置 (可选)
    timeout: 30000                     # 超时时间 (毫秒)

    # 错误处理 (可选)
    on_error: continue                 # continue | stop | fallback
    fallback:
      tool: alternative.tool
      input: {...}

# 完成后的操作 (可选)
on_completion:
  - type: notify
    message: "工作流完成！字幕已保存到 {{.output.subtitle_path}}"
  - type: cleanup
    delete:
      - "{{.variables.temp_dir}}"

# 错误处理 (可选)
on_error:
  - type: notify
    message: "工作流失败: {{.error}}"
  - type: cleanup
    delete:
      - "{{.variables.temp_dir}}"
```

## 核心概念

### 1. 变量引用语法

| 语法 | 说明 | 示例 |
|------|------|------|
| `{{.input.xxx}}` | 引用输入参数 | `{{.input.video_url}}` |
| `{{.variables.xxx}}` | 引用内部变量 | `{{.variables.video_name}}` |
| `{{.steps.xxx.output}}` | 引用步骤输出 | `{{.steps.transcribe.output}}` |
| `{{.steps.xxx.error}}` | 引用步骤错误 | `{{.steps.extract.error}}` |
| `{{.output.xxx}}` | 引用最终输出 | `{{.output.subtitle_path}}` |

### 2. 内置函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `basename` | 提取文件名 | `{{.input.path \| basename}}` |
| `dirname` | 提取目录 | `{{.input.path \| dirname}}` |
| `ext` | 提取扩展名 | `{{.input.path \| ext}}` |
| `timestamp` | 时间戳 | `{{.timestamp}}` |
| `env` | 环境变量 | `{{.env.HOME}}` |
| `json_path` | JSON 提取 | `{{.input.data \| json_path "$.key"}}` |

### 3. 执行模式

```yaml
# 本地执行（默认）
- tool: file.read
  execution: local
  input:
    path: "/tmp/file.txt"

# SSH 执行
- tool: ssh.exec
  execution: ssh
  target: nas
  input:
    command: "ls /data"

# 也可以用工具前缀区分（替代方案）
- tool: ssh.exec@nas
  input:
    command: "ls /data"
```

### 4. 并行执行

```yaml
steps:
  # 步骤 A 和 B 并行执行，都完成后执行 C
  - id: step_a
    tool: asr.audio1
    parallel: true

  - id: step_b
    tool: asr.audio2
    parallel: true

  - id: step_c
    tool: llm.translate
    depends_on: [step_a, step_b]
```

### 5. 条件执行

```yaml
steps:
  - id: conditional
    tool: ssh.exec
    condition: "{{.input.enable_remote == true}}"
    input:
      target: nas
      command: "ls"
```

### 6. 错误处理

```yaml
steps:
  - id: risky_step
    tool: ssh.exec
    retry:
      max_attempts: 3
      delay: 1000
    on_error: fallback
    fallback:
      tool: alternative.method
```

## 完整示例

```yaml
id: nas.video.subtitle
name: NAS 视频字幕生成
description: 从 NAS 视频提取音频，转写并生成双语字幕

input:
  properties:
    video_path:
      type: string
      description: NAS 上的视频路径
    languages:
      type: array
      default: ["en", "zh"]
  required: [video_path]

ssh_targets:
  nas:
    host: "192.168.1.100"
    port: 22
    user: "admin"
    auth_type: "key"

variables:
  work_dir: "/tmp/video-work"
  video_name: "{{.input.video_path | basename}}"
  audio_name: "{{.variables.video_name}}.wav"

steps:
  # 1. 在 NAS 上提取音频
  - id: extract_audio
    name: 提取音频
    tool: ssh.docker_run
    execution: ssh
    target: nas
    input:
      image: "ffmpeg:latest"
      volumes: ["/data/videos:/work"]
      command: |
        ffmpeg -i "/work/{{.input.video_path}}" \
               -vn -acodec pcm_s16le -ar 16000 -ac 1 \
               -y "/work/{{.variables.audio_name}}"
    output: audio_path

  # 2. 下载音频到本地
  - id: download_audio
    name: 下载音频
    tool: ssh.download
    input:
      target: nas
      remote_path: "/data/videos/{{.variables.audio_name}}"
      local_path: "{{.variables.work_dir}}/audio.wav"
    depends_on: [extract_audio]

  # 3. 语音转写
  - id: transcribe
    name: 语音转写
    tool: asr.whisper
    input:
      audio_path: "{{.variables.work_dir}}/audio.wav"
      language: "auto"
    output: transcript

  # 4. 并行翻译
  - id: translate_en
    name: 翻译英文
    tool: llm.translate
    input:
      text: "{{.steps.transcribe.output}}"
      to: "en"
    parallel: true
    depends_on: [transcribe]
    output: transcript_en

  - id: translate_zh
    name: 翻译中文
    tool: llm.translate
    input:
      text: "{{.steps.transcribe.output}}"
      to: "zh"
    parallel: true
    depends_on: [transcribe]
    output: transcript_zh

  # 5. 生成 SRT
  - id: generate_srt
    name: 生成字幕文件
    tool: file.write
    input:
      path: "{{.variables.work_dir}}/{{.variables.video_name}}.srt"
      content: "{{.steps.translate_zh.output}}"

  # 6. 上传回 NAS
  - id: upload_srt
    name: 上传字幕
    tool: ssh.upload
    input:
      target: nas
      local_path: "{{.variables.work_dir}}/{{.variables.video_name}}.srt"
      remote_path: "/data/videos/subtitles/{{.variables.video_name}}.srt"

  # 7. 清理
  - id: cleanup
    name: 清理临时文件
    tool: file.delete
    input:
      paths:
        - "{{.variables.work_dir}}"
```
