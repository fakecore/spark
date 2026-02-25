package llm

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Role is aligned with OpenAI-compatible chat roles.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role Role `json:"role"`
	// Content is used for system/user/assistant messages.
	Content string `json:"content,omitempty"`

	// Name is optional and provider-specific (e.g. function name or persona name).
	Name string `json:"name,omitempty"`

	// ToolCallID is used for tool role messages to correlate tool results.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	JSONSchema  map[string]any `json:"json_schema,omitempty"`
}

// ToolCall is a normalized tool call.
type ToolCall struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name"`
	ArgumentsJSON string `json:"arguments_json,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type Request struct {
	Provider string `json:"provider,omitempty"` // optional: used by router/registry
	Model    string `json:"model"`

	Messages []Message `json:"messages"`

	Tools       []Tool   `json:"tools,omitempty"`
	ToolChoice  string   `json:"tool_choice,omitempty"` // "auto"|"none"|tool name
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxOutputTokens is the maximum tokens in the completion.
	MaxOutputTokens *int `json:"max_output_tokens,omitempty"`

	// Metadata is for observability/audit, never sent to the model unless provider supports it.
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Response struct {
	ID    string `json:"id,omitempty"`
	Model string `json:"model,omitempty"`

	Message   Message    `json:"message"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage,omitempty"`

	// Raw can hold the provider's original JSON response (optional).
	Raw map[string]any `json:"raw,omitempty"`
}

// StreamEvent is a normalized streaming event.
type StreamEvent struct {
	DeltaText     string    `json:"delta_text,omitempty"`
	ToolCallDelta *ToolCall `json:"tool_call_delta,omitempty"`
	Usage         *Usage    `json:"usage,omitempty"`
	Done          bool      `json:"done,omitempty"`
}

// Stream is a pull-based streaming interface (similar to gRPC streaming semantics).
type Stream interface {
	Recv() (*StreamEvent, error)
	Close() error
}

// Provider executes LLM requests. Implementations live under infrastructure.
type Provider interface {
	Chat(ctx context.Context, req *Request) (*Response, error)
	ChatStream(ctx context.Context, req *Request) (Stream, error)
}

type ErrorKind string

const (
	ErrorKindUnknown        ErrorKind = "unknown"
	ErrorKindInvalidRequest ErrorKind = "invalid_request"
	ErrorKindUnauthorized   ErrorKind = "unauthorized"
	ErrorKindRateLimited    ErrorKind = "rate_limited"
	ErrorKindTransient      ErrorKind = "transient"
	ErrorKindTimeout        ErrorKind = "timeout"
)

// ProviderError normalizes provider failures for retry/UX logic.
type ProviderError struct {
	Kind       ErrorKind
	StatusCode int
	Message    string
	RetryAfter time.Duration
	Cause      error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "<nil>"
	}
	base := fmt.Sprintf("llm provider error kind=%s status=%d", e.Kind, e.StatusCode)
	if e.Message != "" {
		base += ": " + e.Message
	}
	if e.Cause != nil {
		base += ": " + e.Cause.Error()
	}
	return base
}

func (e *ProviderError) Unwrap() error { return e.Cause }

func IsKind(err error, kind ErrorKind) bool {
	var pe *ProviderError
	if !errors.As(err, &pe) {
		return false
	}
	return pe.Kind == kind
}
