package llm

import (
	"context"
	"encoding/json"
)

// Role constants for Message.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// StopReason constants returned in Response.
const (
	StopReasonEndTurn = "end_turn"
	StopReasonToolUse = "tool_use"
)

// ToolDef describes a tool the LLM can call. InputSchema is a JSON Schema
// object describing the tool's parameters — kept as raw JSON so the provider
// interface is SDK-agnostic.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolCall represents a single tool invocation requested by the LLM.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult carries the outcome of executing a ToolCall back to the LLM.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// Message is one turn in a conversation. Exactly one of Content, ToolCalls,
// or ToolResults is populated depending on the role and turn type:
//   - User text:      Role=user, Content set
//   - Assistant text:  Role=assistant, Content set
//   - Assistant tools: Role=assistant, ToolCalls set (may also have Content)
//   - Tool results:   Role=user, ToolResults set
type Message struct {
	Role        string       `json:"role"`
	Content     string       `json:"content,omitempty"`
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`
}

// Response is the LLM's reply to a Chat call.
type Response struct {
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	StopReason string     `json:"stop_reason"`
}

// Provider is the interface for LLM backends. Implementations must be safe
// for concurrent use.
type Provider interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDef) (*Response, error)
}
