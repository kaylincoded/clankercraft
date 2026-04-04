# Story 5.3: LLM Provider Interface & Claude Implementation

Status: done

## Story

As a developer,
I want a pluggable LLM provider interface with a Claude implementation,
so that the chat interface can use Claude API for natural language understanding.

## Acceptance Criteria

1. **Given** `ANTHROPIC_API_KEY` is set in environment
   **When** the Claude provider is initialized
   **Then** it can send messages to the Claude API with tool definitions and receive tool-call responses

2. **Given** the LLM provider interface
   **When** a new provider is implemented (e.g., OpenAI)
   **Then** it can be swapped in without changing chat or engine code

3. **Given** the provider receives a message with tools
   **When** Claude responds with tool_use blocks
   **Then** the provider returns structured tool calls (name, ID, input JSON) that the caller can execute and feed back

4. **Given** the provider is used in a conversation
   **When** multiple turns occur (user message → tool calls → tool results → assistant response)
   **Then** the full conversation history is maintained and sent with each request

## Tasks / Subtasks

- [x] Task 1: Define LLM provider interface (AC: #2)
  - [x] Create `internal/llm/provider.go`
  - [x] `ToolDef` struct: `{Name string, Description string, InputSchema json.RawMessage}`
  - [x] `ToolCall` struct: `{ID string, Name string, Input json.RawMessage}`
  - [x] `ToolResult` struct: `{ToolCallID string, Content string, IsError bool}`
  - [x] `Message` struct: `{Role string, Content string, ToolCalls []ToolCall, ToolResults []ToolResult}`
  - [x] `Response` struct: `{Content string, ToolCalls []ToolCall, StopReason string}`
  - [x] `Provider` interface: `Chat(ctx context.Context, messages []Message, tools []ToolDef) (*Response, error)`
- [x] Task 2: Implement Claude provider (AC: #1, #3)
  - [x] Create `internal/llm/claude.go`
  - [x] `ClaudeProvider` struct with `anthropic.Client`, model string, system prompt
  - [x] `NewClaudeProvider(apiKey string, opts ...ClaudeOption) *ClaudeProvider`
  - [x] Functional options: `WithModel(string)`, `WithMaxTokens(int64)`, `WithSystemPrompt(string)`
  - [x] Default model: `anthropic.ModelClaudeSonnet4_6` (fast, capable, cost-effective for tool calling)
  - [x] Default max tokens: 4096
  - [x] Implement `Chat()`: convert `[]Message` → `[]anthropic.MessageParam`, `[]ToolDef` → `[]anthropic.ToolUnionParam`
  - [x] Parse response: extract text content and `ToolUseBlock`s into `Response`
- [x] Task 3: Message conversion helpers (AC: #3, #4)
  - [x] `toAnthropicMessages(msgs []Message) []anthropic.MessageParam` — handles user text, assistant text+tool_use, user tool_results
  - [x] `toAnthropicTools(tools []ToolDef) []anthropic.ToolUnionParam` — converts ToolDef to SDK tool params
  - [x] `parseResponse(msg *anthropic.Message) *Response` — extracts text and tool calls from API response
  - [x] Handle `StopReason`: "end_turn" (done), "tool_use" (needs tool execution)
- [x] Task 4: Add config and wiring (AC: #1)
  - [x] Add `AnthropicAPIKey string` to config.Config (env var: `ANTHROPIC_API_KEY`)
  - [x] Add `LLMModel string` to config.Config (optional override, env var: `LLM_MODEL`)
  - [x] Create provider in `main.go` if API key is set, log warning if not
  - [x] Provider is optional — nil means no LLM features (graceful degradation)
- [x] Task 5: Write tests (all ACs)
  - [x] Test `toAnthropicMessages` converts user text messages
  - [x] Test `toAnthropicMessages` converts assistant messages with tool calls
  - [x] Test `toAnthropicMessages` converts user messages with tool results
  - [x] Test `toAnthropicMessages` handles multi-turn conversation
  - [x] Test `toAnthropicTools` converts ToolDef to SDK format
  - [x] Test `parseResponse` extracts text content
  - [x] Test `parseResponse` extracts tool calls with ID, name, input
  - [x] Test `parseResponse` handles mixed text + tool_use content
  - [x] Test `NewClaudeProvider` applies functional options
  - [x] Test `NewClaudeProvider` uses defaults when no options given
  - [x] Test `ClaudeProvider.Chat` with mock HTTP server (text response)
  - [x] All existing tests still pass

## Dev Notes

### Anthropic Go SDK

Package: `github.com/anthropics/anthropic-sdk-go`

```bash
go get -u 'github.com/anthropics/anthropic-sdk-go'
```

**Client creation:**
```go
import (
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

client := anthropic.NewClient(option.WithAPIKey(apiKey))
```

**Sending messages with tools:**
```go
message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
    Model:     anthropic.ModelClaude4Sonnet,
    MaxTokens: 4096,
    Messages:  messages,
    Tools:     tools,
})
```

**Tool definitions:**
```go
anthropic.ToolUnionParam{
    OfTool: &anthropic.ToolParam{
        Name:        "tool_name",
        Description: anthropic.String("description"),
        InputSchema: anthropic.ToolInputSchemaParam{
            Properties: map[string]any{ ... },
        },
    },
}
```

**Processing tool use responses:**
```go
for _, block := range message.Content {
    switch v := block.AsAny().(type) {
    case anthropic.TextBlock:
        // v.Text
    case anthropic.ToolUseBlock:
        // v.ID, v.Name, v.Input
    }
}
```

**Sending tool results back:**
```go
anthropic.NewToolResultBlock(toolUseID, resultJSON, isError)
```

**Conversation history:**
```go
messages = append(messages, message.ToParam()) // assistant response
messages = append(messages, anthropic.NewUserMessage(toolResults...)) // tool results
```

### Interface Design

The `Provider` interface is intentionally minimal — one method `Chat()`. This keeps the contract simple:

```go
type Provider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDef) (*Response, error)
}
```

The caller (Story 5.4's agent loop) builds the conversation, feeds tool results, and calls `Chat()` in a loop until `StopReason == "end_turn"`.

### Why json.RawMessage for Schemas

`ToolDef.InputSchema` uses `json.RawMessage` so the provider interface doesn't depend on any specific LLM SDK types. The Claude implementation converts this to `anthropic.ToolInputSchemaParam`. A future OpenAI implementation would convert to its own schema format.

### System Prompt

The Claude provider accepts a system prompt via `WithSystemPrompt()`. This will be used by the agent loop (Story 5.4) to inject the bot's personality and available tools context. The system prompt is sent as a top-level parameter, not as a message.

### Config

The API key comes from `ANTHROPIC_API_KEY` environment variable (standard for Anthropic SDK). Adding it to Config struct keeps it consistent with other config — env vars are already supported by viper with the `CLANKERCRAFT_` prefix, but `ANTHROPIC_API_KEY` is the SDK's default env var name, so we should support both.

### Testing Strategy

Most tests are unit tests for conversion functions (no API calls). The `ClaudeProvider.Chat` method can be tested with a mock HTTP server using `option.WithBaseURL()` from the SDK, or by testing the conversion layer separately and trusting the SDK for HTTP.

### References

- [Source: github.com/anthropics/anthropic-sdk-go] — Official Go SDK
- [Source: internal/chat/chat.go] — Whisper listener/sender (consumer of LLM responses)
- [Source: internal/mcp/server.go] — MCP tools that will become LLM tool definitions
- [Source: internal/config/config.go] — Config struct and viper bindings
- [Source: main.go] — Wiring point for provider creation
- [Source: docs/epics.md#Story 5.3] — Original story definition
- [Source: docs/epics.md#Story 5.4] — Agent loop that will consume this provider

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Provider interface uses json.RawMessage for InputSchema — SDK-agnostic
- ClaudeProvider uses anthropic-sdk-go v1.30.0 with functional options pattern
- Default model is ModelClaudeSonnet4_6 (not ModelClaude4Sonnet as originally noted in story — corrected to actual SDK constant)
- SDK returns value type `anthropic.Client` not pointer — adjusted struct field
- System prompt sent as `[]TextBlockParam` per SDK API
- toAnthropicMessages handles 4 message types: user text, assistant text, assistant tool_use, user tool_results
- Config supports both `ANTHROPIC_API_KEY` (SDK standard) and `CLANKERCRAFT_ANTHROPIC_API_KEY` (viper prefix)
- Mock HTTP server test validates full round-trip: request → mock API → parsed response
- 11 tests total: 4 message conversion, 1 tool conversion, 3 response parsing, 2 provider construction, 1 integration

### File List
- internal/llm/provider.go — Provider interface, ToolDef, ToolCall, ToolResult, Message, Response types
- internal/llm/claude.go — ClaudeProvider, NewClaudeProvider, functional options, Chat, conversion helpers
- internal/llm/claude_test.go — 11 tests
- internal/config/config.go — Added AnthropicAPIKey, LLMModel fields with ANTHROPIC_API_KEY fallback
- main.go — LLM provider creation and wiring
- go.mod, go.sum — Added anthropic-sdk-go dependency

