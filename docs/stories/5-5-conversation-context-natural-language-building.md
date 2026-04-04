# Story 5.5: Conversation Context & Natural Language Building

Status: done

## Story

As a builder,
I want the bot to maintain conversation context and understand iterative requests,
so that I can say "make it taller" without repeating the full context.

## Acceptance Criteria

1. **Given** the bot built a wall and the player says "make it taller"
   **When** the LLM receives the message with conversation history
   **Then** it understands "it" refers to the wall and modifies accordingly

2. **Given** a conversation has been ongoing for 20+ messages
   **When** a new message arrives
   **Then** the full conversation context is available to the LLM (within token limits)

## Tasks / Subtasks

- [x] Task 1: Per-player conversation store (AC: #1, #2)
  - [x] Create `internal/agent/conversation.go`
  - [x] `ConversationStore` struct with `sync.RWMutex` protecting `map[string]*Conversation`
  - [x] `Conversation` struct: `Messages []llm.Message`, `LastActive time.Time`
  - [x] `NewConversationStore() *ConversationStore`
  - [x] `Get(player string) *Conversation` — returns existing or creates new
  - [x] `Append(player string, msgs ...llm.Message)` — appends messages to player's conversation
  - [x] `Reset(player string)` — clears a player's conversation history
  - [x] Thread-safe: all methods acquire appropriate locks
- [x] Task 2: Integrate conversation store into Agent (AC: #1, #2)
  - [x] Add `conversations *ConversationStore` field to `Agent` struct
  - [x] Create store in `NewAgent`
  - [x] Modify `HandleMessage` to:
    1. Load existing conversation history for the player
    2. Prepend history before the new user message
    3. After the agent loop completes, persist the full conversation (user msg + all tool turns + final assistant reply)
  - [x] Conversation persists across separate whisper messages from the same player
- [x] Task 3: Conversation management commands (AC: #1)
  - [x] Detect "reset", "clear", "new conversation", or "forget" as special commands
  - [x] On detection: reset the player's conversation store, reply with confirmation, skip LLM call
  - [x] Keep detection simple — exact prefix match, not LLM-interpreted
- [x] Task 4: Conversation size limit (AC: #2)
  - [x] Add `maxConversationMessages int` to Agent (default 50 messages)
  - [x] `WithMaxConversationMessages(n int)` functional option
  - [x] When conversation exceeds limit, trim oldest messages (keep most recent N)
  - [x] Always preserve the first user message (original request context) when trimming
- [x] Task 5: Idle conversation cleanup (AC: #2)
  - [x] Add cleanup goroutine that runs periodically (every 5 minutes)
  - [x] Remove conversations idle for more than 30 minutes
  - [x] `Start(ctx context.Context)` method on Agent that launches the cleanup goroutine
  - [x] Cleanup respects context cancellation
- [x] Task 6: Write tests (all ACs)
  - [x] Test `ConversationStore.Get` creates new conversation for unknown player
  - [x] Test `ConversationStore.Get` returns existing conversation for known player
  - [x] Test `ConversationStore.Append` adds messages to correct player
  - [x] Test `ConversationStore.Reset` clears player history
  - [x] Test `ConversationStore` is safe for concurrent access (parallel goroutines)
  - [x] Test `HandleMessage` preserves history across two calls for same player
  - [x] Test `HandleMessage` isolates conversations between different players
  - [x] Test reset command ("reset") clears history and replies without LLM
  - [x] Test conversation trimming when exceeding max messages
  - [x] Test trimming preserves first user message
  - [x] Test idle cleanup removes stale conversations
  - [x] All existing tests still pass

## Dev Notes

### Current State (Story 5.4)

`Agent.HandleMessage` currently creates a fresh `[]llm.Message` each call (`agent.go:68`). No state persists between whisper messages. Each whisper is an independent single-turn conversation.

The whisper handler in `main.go:94-99` runs `HandleMessage` in a goroutine per whisper. Multiple concurrent whispers from the same player could race — the conversation store must be thread-safe.

### Design: ConversationStore

```go
type Conversation struct {
    Messages   []llm.Message
    LastActive time.Time
    mu         sync.Mutex // per-conversation lock for append safety
}

type ConversationStore struct {
    mu    sync.RWMutex
    convs map[string]*Conversation
}
```

Use a per-conversation mutex for append operations (so two concurrent messages from the same player don't interleave). The store-level RWMutex protects the map itself.

### HandleMessage Changes

```go
func (a *Agent) HandleMessage(ctx context.Context, player, message string, sendReply func(string) error) error {
    conv := a.conversations.Get(player)

    // Check for reset command.
    if isResetCommand(message) {
        a.conversations.Reset(player)
        return sendReply("Conversation cleared. What would you like to build?")
    }

    // Build message list: history + new user message.
    conv.mu.Lock()
    history := make([]llm.Message, len(conv.Messages))
    copy(history, conv.Messages)
    conv.mu.Unlock()

    messages := append(history, llm.Message{Role: llm.RoleUser, Content: message})
    // ... existing agent loop ...

    // After loop: persist all new messages to conversation.
    // newMsgs = messages[len(history):]  (user msg + tool turns + assistant reply)
    a.conversations.Append(player, newMsgs...)
    a.trimConversation(player)
}
```

### Reset Command Detection

Simple prefix matching — no LLM involved:

```go
func isResetCommand(msg string) bool {
    lower := strings.ToLower(strings.TrimSpace(msg))
    return lower == "reset" || lower == "clear" || lower == "new conversation" || lower == "forget"
}
```

### Conversation Trimming

When messages exceed `maxConversationMessages`:
1. Keep messages[0] (first user message — provides original context)
2. Keep the most recent N-1 messages
3. Drop everything in between

This preserves the "what we're building" context while keeping the conversation within token limits.

### Idle Cleanup

Launch a goroutine in `Agent.Start(ctx)` that ticks every 5 minutes and removes conversations where `time.Since(conv.LastActive) > 30*time.Minute`. The `Start` method should be called from `main.go` after agent creation.

### Wiring Changes in main.go

```go
agentLoop := agent.NewAgent(llmProvider, toolExec, logger)
agentLoop.Start(ctx)  // launches cleanup goroutine
```

### Concurrency Safety

- **ConversationStore.mu (RWMutex):** Protects the `convs` map. Read lock for Get (when conversation exists), Write lock for creating new entries or deleting during cleanup.
- **Conversation.mu (Mutex):** Protects the Messages slice within a single conversation. Acquired during HandleMessage to snapshot history and during Append to add new messages.
- The whisper handler already runs each HandleMessage in a goroutine — concurrent whispers from the same player will contend on Conversation.mu but won't corrupt state.

### Previous Story Intelligence

- Story 5.4: `HandleMessage` signature is `(ctx, player, message string, sendReply func(string) error) error` — player is already passed, perfect for per-player keying
- Story 5.4: Agent loop builds conversation within HandleMessage: user msg → (tool call → tool result)* → assistant reply. All these messages need to be persisted.
- Story 5.3: `llm.Message` has `Role`, `Content`, `ToolCalls`, `ToolResults` — all serializable, safe to store
- Story 5.2: `SendWhisper` handles message splitting — long LLM responses are already handled
- Story 5.1: `OnWhisper` callback signature is `func(sender, message string)` — sender is the player name

### References

- [Source: internal/agent/agent.go] — Current Agent struct and HandleMessage (no conversation state)
- [Source: internal/agent/tools.go] — ToolExecutor (unchanged by this story)
- [Source: internal/llm/provider.go] — Message, ToolCall, ToolResult types
- [Source: main.go#L91-L101] — Whisper handler wiring (needs Start() call added)
- [Source: docs/epics.md#Story 5.5] — Original story definition
- [Source: docs/epics.md#Story 5.6] — Next story (bot summoning) — don't build that yet

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Implemented `ConversationStore` with double-checked locking for thread-safe per-player conversation management
- `Conversation` struct uses per-conversation mutex to allow concurrent whispers from same player without corruption
- `HandleMessage` loads history via `Snapshot` (copy), runs agent loop without holding locks, then persists new messages via `Append`
- Trim strategy preserves first message (original build context) + most recent N-1 messages
- Reset commands detected via exact string match — no LLM round-trip for "reset", "clear", "new conversation", "forget"
- Idle cleanup goroutine launched via `Agent.Start(ctx)`, ticks every 5 minutes, removes conversations idle > 30 minutes
- `main.go` updated to call `agentLoop.Start(ctx)` after agent creation
- All 30 agent package tests pass (9 conversation + 10 agent + 11 tools)

### File List
- `internal/agent/conversation.go` — NEW: ConversationStore and Conversation types with thread-safe operations
- `internal/agent/conversation_test.go` — NEW: 9 tests for conversation store (CRUD, concurrency, trim, cleanup)
- `internal/agent/agent.go` — MODIFIED: Added conversations field, Start(), isResetCommand(), history loading/persisting/trimming in HandleMessage
- `internal/agent/agent_test.go` — MODIFIED: Added 5 tests (PreservesHistory, IsolatesPlayers, ResetCommand, Trimming, trackingProvider helper)
- `main.go` — MODIFIED: Added `agentLoop.Start(ctx)` call
- `docs/stories/sprint-status.yaml` — MODIFIED: Updated 5-5 status
