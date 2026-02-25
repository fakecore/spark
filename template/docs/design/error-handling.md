# 错误处理和重试机制

> **类型**: 功能设计
> **状态**: ✅ 完成
> **最后更新**: 2026-02-04
> **关联文档**: [workflow-schema.md](./workflow-schema.md)

---

## 核心设计

```go
package workflow

// ErrorHandling 错误处理策略
type ErrorHandling string

const (
    ErrorStop     ErrorHandling = "stop"     // 停止工作流
    ErrorContinue ErrorHandling = "continue" // 跳过该步骤，继续执行
    ErrorFallback ErrorHandling = "fallback" // 使用替代方案
)

// RetryPolicy 重试策略
type RetryPolicy struct {
    MaxAttempts  int           // 最大重试次数
    Backoff      BackoffType   // 退避策略
    Delay        time.Duration // 初始延迟
    MaxDelay    time.Duration // 最大延迟
    RetryOn     []string      // 在哪些错误时重试
}

// BackoffType 退避策略
type BackoffType string

const (
    BackoffConstant    BackoffType = "constant"    // 固定延迟
    BackoffLinear      BackoffType = "linear"      // 线性增长
    BackoffExponential BackoffType = "exponential" // 指数增长
)

// StepError 步骤错误
type StepError struct {
    StepID   string
    Err      error
    Attempt  int
    Retries  int
    Output   any
}

func (e *StepError) Error() string {
    return fmt.Sprintf("step %s failed (attempt %d): %v", e.StepID, e.Attempt, e.Err)
}
```

## 重试执行器

```go
type RetryExecutor struct {
    policy *RetryPolicy
}

func (e *RetryExecutor) Execute(ctx context.Context, step *Step, fn func() (any, error)) (any, error) {
    var lastErr error
    var result any

    delay := e.policy.Delay

    for attempt := 1; attempt <= e.policy.MaxAttempts; attempt++ {
        // 执行步骤
        result, err = fn()
        if err == nil {
            return result, nil
        }

        lastErr = err

        // 检查是否应该重试
        if !e.shouldRetry(err) {
            break
        }

        // 计算延迟
        delay = e.calculateDelay(attempt, delay)

        // 等待后重试
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(delay):
            // 重试
        }

        // 发送重试事件
        e.onRetry(step, attempt, err)
    }

    return nil, &StepError{
        StepID:  step.ID,
        Err:    lastErr,
        Attempt: e.policy.MaxAttempts,
    }
}

func (e *RetryExecutor) shouldRetry(err error) bool {
    // 1. 检查错误类型
    if e.policy.RetryOn != nil {
        errType := reflect.TypeOf(err).String()
        for _, retryable := range e.policy.RetryOn {
            if strings.Contains(errType, retryable) {
                return true
            }
        }
    }

    // 2. 默认重试的错误类型
    return isRetryableError(err)
}

func isRetryableError(err error) bool {
    // 网络错误通常可重试
    if isNetworkError(err) {
        return true
    }
    // 超时错误可重试
    if isTimeoutError(err) {
        return true
    }
    return false
}

func (e *RetryExecutor) calculateDelay(attempt int, currentDelay time.Duration) time.Duration {
    switch e.policy.Backoff {
    case BackoffConstant:
        return currentDelay

    case BackoffLinear:
        return time.Duration(attempt) * currentDelay

    case BackoffExponential:
        delay := currentDelay * time.Duration(2) // 每次翻倍
        if e.policy.MaxDelay > 0 && delay > e.policy.MaxDelay {
            delay = e.policy.MaxDelay
        }
        return delay

    default:
        return currentDelay
    }
}

func (e *RetryExecutor) onRetry(step *Step, attempt int, err error) {
    // 发布重试事件
    // log.Printf("step %s retry %d: %v", step.ID, attempt, err)
}
```

## 错误处理器

```go
type ErrorHandler struct {
    fallbackRegistry *FallbackRegistry
}

func (h *ErrorHandler) Handle(ctx context.Context, step *Step, stepErr error, scope *Scope) error {
    switch step.OnError {
    case ErrorStop:
        return stepErr

    case ErrorContinue:
        // 记录错误，继续执行
        scope.Steps[step.ID] = &StepResult{
            Error:  stepErr,
            Status: "failed",
        }
        return nil

    case ErrorFallback:
        // 执行 fallback
        return h.executeFallback(ctx, step, stepErr, scope)

    default:
        return stepErr
    }
}

func (h *ErrorHandler) executeFallback(ctx context.Context, step *Step, stepErr error, scope *Scope) error {
    if step.Fallback == nil {
        return stepErr
    }

    // 执行 fallback 步骤
    fallbackResult, err := h.executeStep(ctx, step.Fallback, scope)
    if err != nil {
        return fmt.Errorf("fallback also failed: %w", err)
    }

    // 更新步骤结果
    scope.Steps[step.ID] = &StepResult{
        Output: fallbackResult,
        Status: "fallback",
    }

    return nil
}

func (h *ErrorHandler) executeStep(ctx context.Context, def *StepDefinition, scope *Scope) (any, error) {
    // 执行 fallback 步骤
    // 根据类型调用 tool 或 workflow
    return nil, nil
}
```

## 常见可重试错误

```go
// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
    switch e := err.(type) {
    case interface{ Temporary() bool }:
        return e.Temporary()

    case interface{ Timeout() bool }:
        return e.Timeout()

    case *net.OpError:
        return true // 网络错误通常可重试

    case interface{ Network() bool }:
        return e.Network()

    default:
        // 检查错误消息
        if strings.Contains(err.Error(), "connection refused") {
            return true
        }
        if strings.Contains(err.Error(), "timeout") {
            return true
        }
        if strings.Contains(err.Error(), "temporary") {
            return true
        }
    }
    return false
}
```

## 完整执行流程

```go
type WorkflowExecutor struct {
    engine         *WorkflowEngine
    retryExecutor  *RetryExecutor
    errorHandler  *ErrorHandler
}

func (e *WorkflowExecutor) ExecuteStep(ctx context.Context, step *Step, scope *Scope) error {
    var result any
    var err error

    // 带重试的执行
    if step.Retry != nil {
        result, err = e.retryExecutor.Execute(ctx, step, func() (any, error) {
            return e.engine.ExecuteStep(ctx, step, scope)
        })
    } else {
        result, err = e.engine.ExecuteStep(ctx, step, scope)
    }

    // 处理结果
    if err != nil {
        // 记录错误
        scope.Steps[step.ID] = &StepResult{
            Error:  err,
            Status: "failed",
        }

        // 错误处理
        return e.errorHandler.Handle(ctx, step, err, scope)
    }

    // 成功
    scope.Steps[step.ID] = &StepResult{
        Output: result,
        Status: "success",
    }

    return nil
}
```

## YAML 配置示例

### 基础重试

```yaml
steps:
  - id: download
    tool: ssh.download
    retry:
      max_attempts: 3
      backoff: exponential
      delay: 1000
    input:
      target: nas
      remote_path: /data/video.mp4
      local_path: /tmp/video.mp4
```

### 条件重试

```yaml
steps:
  - id: api_call
    tool: http.post
    retry:
      max_attempts: 5
      retry_on: [timeout, network, 5xx]
    input:
      url: https://api.example.com/process
```

### Fallback 机制

```yaml
steps:
  - id: primary_method
    tool: ocr.paddle
    retry:
      max_attempts: 2
    on_error: fallback
    fallback:
      tool: ocr.tesseract  # 失败时使用备用方案
    input:
      image_path: "{{.input.image}}"
```

### 组合使用

```yaml
steps:
  - id: risky_operation
    name: 风险操作
    tool: ssh.exec
    target: nas
    retry:
      max_attempts: 3
      backoff: exponential
      delay: 1000
      max_delay: 10000
    on_error: fallback
    fallback:
      tool: ssh.exec
      target: nas_backup  # 使用备用 NAS
    input:
      command: "{{.variables.command}}"
    timeout: 30000
```

## 事件回调

```go
// WorkflowEvent 工作流事件
type WorkflowEvent struct {
    WorkflowID string
    StepID     string
    EventType  string  // start | success | error | retry | fallback
    Timestamp  int64
    Data       map[string]any
}

// EventCallback 事件回调
type EventCallback interface {
    OnStart(ctx context.Context, event *WorkflowEvent)
    OnSuccess(ctx context.Context, event *WorkflowEvent)
    OnError(ctx context.Context, event *WorkflowEvent)
    OnRetry(ctx context.Context, event *WorkflowEvent)
    OnFallback(ctx context.Context, event *WorkflowEvent)
}

type EventEmitter struct {
    callbacks []EventCallback
}

func (e *EventEmitter) Emit(ctx context.Context, event *WorkflowEvent) {
    for _, cb := range e.callbacks {
        switch event.EventType {
        case "start":
            cb.OnStart(ctx, event)
        case "success":
            cb.OnSuccess(ctx, event)
        case "error":
            cb.OnError(ctx, event)
        case "retry":
            cb.OnRetry(ctx, event)
        case "fallback":
            cb.OnFallback(ctx, event)
        }
    }
}
```

## 完整示例

```go
package workflow_test

func TestRetryWithFallback() {
    executor := NewWorkflowExecutor()

    workflow := &Workflow{
        ID:   "test.retry",
        Name: "重试测试",
        Steps: []*Step{
            {
                ID:   "flaky_step",
                Tool: "ssh.exec",
                Retry: &RetryPolicy{
                    MaxAttempts: 3,
                    Backoff:      BackoffExponential,
                    Delay:        1000,
                    MaxDelay:     5000,
                },
                OnError: ErrorFallback,
                Fallback: &StepDefinition{
                    Tool: "alternative.method",
                },
            },
        },
    }

    scope := &Scope{}

    err := executor.Execute(context.Background(), workflow, scope)
    fmt.Printf("Workflow result: %v\n", err)
}
```
