# Story 5.4: Tool-Calling Agent Loop

Status: done

## Story

As a system,
I want the LLM to receive construction tools and execute multi-step builds,
so that a single "build me a cathedral" request triggers a sequence of WorldEdit commands.

## Acceptance Criteria

1. **Given** a player says "build me a stone wall 20 blocks long"
   **When** the message is sent to the LLM with available tools
   **Then** the LLM returns tool calls (e.g., set selection, `//set`), the system executes them, feeds results back, and the LLM continues until done

2. **Given** a tool call fails (e.g., WorldEdit error)
   **When** the error is fed back to the LLM
   **Then** the LLM adapts (retries with different parameters, reports the issue to the player, or tries an alternative approach)

## Tasks / Subtasks

- [x] Task 1: Create tool executor that wraps BotState (AC: #1, #2)
  - [x] Create `internal/agent/tools.go`
  - [x] `ToolExecutor` struct holds `BotState` reference
  - [x] `ToolDefs() []llm.ToolDef` — returns all 34 tools as LLM tool definitions (name, description, JSON schema)
  - [x] `Execute(ctx context.Context, name string, input json.RawMessage) (string, error)` — dispatches tool calls by name
  - [x] Reuse MCP input/output struct types for JSON marshal/unmarshal (import from `internal/mcp`)
  - [x] Return JSON-encoded results; errors returned as `ToolResult.IsError = true` content
- [x] Task 2: Create agent loop (AC: #1, #2)
  - [x] Create `internal/agent/agent.go`
  - [x] `Agent` struct: `provider llm.Provider`, `executor *ToolExecutor`, `logger *slog.Logger`, `systemPrompt string`
  - [x] `NewAgent(provider llm.Provider, executor *ToolExecutor, logger *slog.Logger, opts ...AgentOption) *Agent`
  - [x] `HandleMessage(ctx context.Context, player string, message string, sendReply func(string) error) error`
  - [x] Loop logic:
    1. Build `[]llm.Message` with user message
    2. Call `provider.Chat(ctx, messages, tools)`
    3. If `StopReason == "end_turn"` → send `Response.Content` via `sendReply`, done
    4. If `StopReason == "tool_use"` → execute each `ToolCall` via executor, build `ToolResult` messages
    5. Append assistant message (with tool calls) + user message (with tool results) to conversation
    6. Loop back to step 2
  - [x] Max iterations guard (default 25) to prevent infinite loops
  - [x] Context cancellation respected between iterations
- [x] Task 3: Wire whisper → agent → reply (AC: #1)
  - [x] In `main.go`, create `ToolExecutor` from `connection.Connection` (which implements `BotState`)
  - [x] Create `Agent` with provider, executor, logger, system prompt
  - [x] Register whisper handler via `conn.OnWhisper(func(sender, msg) { ... })`
  - [x] Handler calls `agent.HandleMessage(ctx, sender, msg, func(reply) { conn.SendWhisper(sender, reply) })`
  - [x] Skip if `llmProvider == nil` (LLM disabled)
- [x] Task 4: System prompt (AC: #1)
  - [x] Define default system prompt in `internal/agent/agent.go`
  - [x] Prompt should describe: bot identity, Minecraft building context, available tool categories
  - [x] `WithSystemPrompt(string)` functional option to override
- [x] Task 5: Write tests (all ACs)
  - [x] Test `ToolDefs()` returns all 34 tools with valid names, descriptions, and JSON schemas
  - [x] Test `Execute()` dispatches known tool by name and returns JSON result
  - [x] Test `Execute()` returns error content for unknown tool name
  - [x] Test `HandleMessage` end-to-end with mock provider: single-turn text response
  - [x] Test `HandleMessage` with mock provider: tool call → result → final text
  - [x] Test `HandleMessage` with mock provider: tool call returns error → LLM adapts
  - [x] Test `HandleMessage` respects max iterations limit
  - [x] Test `HandleMessage` respects context cancellation
  - [x] All existing tests still pass

## Dev Notes

### Architecture: Agent Package

Create `internal/agent/` as a new package. The agent orchestrates: whisper received → LLM conversation → tool execution → whisper reply. It depends on `internal/llm` (Provider interface) and `internal/mcp` (BotState interface + input/output types).

```
whisper in → Agent.HandleMessage() → Provider.Chat() loop → tool execution → whisper out
```

### Tool Executor Design

The `ToolExecutor` bridges LLM tool calls to BotState methods. It needs to:

1. **Expose tool definitions** — convert each tool's name, description, and input struct to `llm.ToolDef` with JSON Schema
2. **Dispatch by name** — unmarshal `json.RawMessage` input to the correct struct, call the BotState method, marshal result to JSON

**Critical: Reuse MCP types.** The input/output structs are already defined in `internal/mcp/server.go` (lines 22-346). Import and reuse them — do NOT duplicate. However, note these types are currently unexported (lowercase). You have two options:
- **Option A (preferred):** Export the input/output types by capitalizing them (e.g., `weSetInput` → `WeSetInput`), since they're simple DTOs
- **Option B:** Keep them unexported and build the tool executor inside `internal/mcp` or use a shared types file

The MCP tool handlers (`handleWESet`, `handleSetblock`, etc.) contain validation and BotState call logic. The tool executor can either:
- Call BotState methods directly (simpler, some validation duplication)
- Call the MCP handler functions (avoids duplication but couples to MCP SDK types)

**Recommended:** Call BotState methods directly for the common path. The MCP middleware checks (connection, tier, selection) should be replicated as simple if-checks in the executor, not reimported.

### Tool Schema Generation

Each tool needs a `json.RawMessage` InputSchema for `llm.ToolDef`. Generate these from the input structs using `encoding/json` reflection or by hardcoding JSON schemas. The MCP SDK generates schemas from struct tags — the `jsonschema` tags on input structs provide field descriptions.

A simple approach: define a `toolRegistry` slice of `{name, description, sampleInput}` and use `reflect` or manual JSON to produce schemas. Don't over-engineer — a static map of tool name → schema JSON is fine.

### Agent Loop Detail

```go
func (a *Agent) HandleMessage(ctx context.Context, player, message string, sendReply func(string) error) error {
    messages := []llm.Message{{Role: llm.RoleUser, Content: message}}
    tools := a.executor.ToolDefs()

    for i := 0; i < a.maxIterations; i++ {
        if ctx.Err() != nil {
            return ctx.Err()
        }

        resp, err := a.provider.Chat(ctx, messages, tools)
        if err != nil {
            return fmt.Errorf("llm chat: %w", err)
        }

        // Append assistant response to conversation
        messages = append(messages, llm.Message{
            Role:      llm.RoleAssistant,
            Content:   resp.Content,
            ToolCalls: resp.ToolCalls,
        })

        if resp.StopReason == llm.StopReasonEndTurn {
            if resp.Content != "" {
                return sendReply(resp.Content)
            }
            return nil
        }

        // Execute tool calls
        var results []llm.ToolResult
        for _, tc := range resp.ToolCalls {
            result, execErr := a.executor.Execute(ctx, tc.Name, tc.Input)
            if execErr != nil {
                results = append(results, llm.ToolResult{
                    ToolCallID: tc.ID,
                    Content:    execErr.Error(),
                    IsError:    true,
                })
            } else {
                results = append(results, llm.ToolResult{
                    ToolCallID: tc.ID,
                    Content:    result,
                    IsError:    false,
                })
            }
        }

        messages = append(messages, llm.Message{
            Role:        llm.RoleUser,
            ToolResults: results,
        })
    }
    return sendReply("I've reached my step limit. Please try a simpler request or continue from where I left off.")
}
```

### Wiring in main.go

```go
// After LLM provider creation (around line 79)
if llmProvider != nil {
    toolExec := agent.NewToolExecutor(conn)
    agentLoop := agent.NewAgent(llmProvider, toolExec, logger)
    conn.OnWhisper(func(sender, msg string) {
        replyFn := func(reply string) error { return conn.SendWhisper(sender, reply) }
        if err := agentLoop.HandleMessage(ctx, sender, msg, replyFn); err != nil {
            logger.Error("agent error", slog.String("player", sender), slog.Any("error", err))
        }
    })
    logger.Info("agent loop wired — whisper to interact")
}
```

Remove the `_ = llmProvider` placeholder.

### BotState Interface

The full interface is in `internal/mcp/middleware.go` (lines 18-39). All 34 tools map to combinations of these methods:

| Tool Category | BotState Methods Used |
|---|---|
| Connection/State (ping, status, get-position, look-at, detect-gamemode, detect-worldedit) | `IsConnected()`, `GetPosition()`, `SendRotation()`, `GetGamemode()`, `GetTier()` |
| World Query (get-block-info, find-block, scan-area, read-sign, find-signs) | `BlockAt()`, `FindBlock()`, `ScanArea()`, `ReadSign()`, `FindSigns()` |
| Selection (set-selection, get-selection) | `SetSelection()`, `GetSelection()`, `HasPos1()`, `HasPos2()` |
| WorldEdit selection-required (we-set, we-replace, we-walls, we-faces, we-hollow, we-generate, we-smooth, we-naturalize, we-overlay, we-copy) | `RunWECommand()` with selection check |
| WorldEdit tier-only (we-sphere, we-cyl, we-pyramid, we-paste, we-rotate, we-flip, we-undo, we-redo) | `RunWECommand()` with tier check |
| Vanilla Commands (setblock, fill, clone) | `RunCommand()`, `RunBulkCommand()` |

### Concurrency Note

`HandleMessage` runs synchronously — one conversation per call. The whisper handler in main.go should run it in a goroutine so the bot's event loop isn't blocked:

```go
conn.OnWhisper(func(sender, msg string) {
    go func() {
        // ... call agentLoop.HandleMessage
    }()
})
```

Story 5.5 will add per-player conversation context and proper concurrency management. For now, each whisper starts a fresh single-turn conversation (no history between messages).

### Testing Strategy

- **ToolExecutor tests:** Create a mock BotState, verify `ToolDefs()` count and schema validity, verify `Execute()` dispatches correctly
- **Agent loop tests:** Create a mock Provider that returns scripted responses. Test: single text response, tool call loop, error handling, max iterations, context cancellation
- **No integration test needed** — the agent is composed of already-tested components (Provider, BotState methods)

### Previous Story Intelligence

- **Story 5.3** established `llm.Provider` interface with `Chat(ctx, []Message, []ToolDef) (*Response, error)`, `StopReason` constants `StopReasonEndTurn` and `StopReasonToolUse`, and `Message` with `Role`, `Content`, `ToolCalls`, `ToolResults` fields
- **Story 5.2** established `chat.Sender` with `SendWhisper(player, message)` — handles splitting and rate limiting
- **Story 5.1** established `chat.Listener` with `OnWhisper(handler)` — created in `connection.New()`, never nil
- Default model is `anthropic.ModelClaudeSonnet4_6` (fast, tool-calling capable)
- `connection.Connection` implements `BotState` interface — use it directly for tool executor
- MCP input struct tags use `jsonschema` for field descriptions — leverage for schema generation

### References

- [Source: internal/llm/provider.go] — Provider interface, Message, ToolDef, ToolCall, ToolResult, Response types
- [Source: internal/llm/claude.go] — ClaudeProvider implementation
- [Source: internal/mcp/server.go#L22-L346] — All 34 tool input/output struct definitions
- [Source: internal/mcp/server.go#L370-L574] — Tool registration with names and descriptions
- [Source: internal/mcp/middleware.go#L18-L39] — BotState interface
- [Source: internal/chat/chat.go] — Listener (OnWhisper) and Sender (SendWhisper)
- [Source: internal/connection/mc.go] — Connection struct implements BotState
- [Source: main.go#L68-L79] — Current LLM provider wiring point
- [Source: docs/epics.md#Story 5.4] — Original story definition
- [Source: docs/epics.md#Story 5.5] — Next story (conversation context) — don't build that yet

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- ToolExecutor dispatches all 34 tools via switch-case on tool name, calling BotState methods directly
- Tool definitions built statically in tooldefs.go with schema builder helpers (prop/schema/require)
- Exported `mcp.ValidPatternRe` to share pattern validation regex with agent package
- Agent loop runs synchronously per call; whisper handler launches goroutine in main.go for non-blocking
- Error results wrapped as JSON `{"error":"..."}` with IsError=true for clean LLM consumption
- WE tools split into weSelectionCmd (connection+tier+selection check) and weTierCmd (connection+tier only)
- copy/paste/rotate/flip/undo/redo use RunWECommand; bulk ops use RunBulkWECommand — matches MCP handlers
- 16 tests total: 11 tool executor (count, names, schemas, dispatch, errors, WE guards) + 5 agent loop (text, tools, error adapt, max iter, ctx cancel)

### File List
- internal/agent/agent.go — Agent struct, NewAgent, HandleMessage loop, system prompt, functional options
- internal/agent/tools.go — ToolExecutor struct, NewToolExecutor, Execute dispatch for all 34 tools, validation helpers
- internal/agent/tooldefs.go — buildToolDefs, schema builder helpers, all 34 tool definitions
- internal/agent/agent_test.go — 5 agent loop tests with mock provider
- internal/agent/tools_test.go — 11 tool executor tests with mock BotState
- internal/mcp/server.go — Exported ValidPatternRe (was validPatternRe)
- main.go — Agent wiring: ToolExecutor + Agent creation, whisper handler registration
