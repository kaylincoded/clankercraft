# Story 5.1: Whisper Listener

Status: done

## Story

As the bot,
I want to detect and parse whispered messages (`/msg`) from players,
so that players can talk to me in-game.

## Acceptance Criteria

1. **Given** a player sends `/msg <BotName> build me a house`
   **When** the bot receives the message
   **Then** it identifies the sender username and message content, and emits a whisper event

2. **Given** a public chat message (not a whisper)
   **When** the bot receives it
   **Then** it does not treat it as a command (ignores non-whisper messages)

3. **Given** a whisper is received
   **When** the event is emitted
   **Then** consumers (future LLM integration) can subscribe and receive structured `{Sender, Message}` data

## Tasks / Subtasks

- [x] Task 1: Define whisper types and listener interface (AC: #1, #3)
  - [x] Create `internal/chat/chat.go` with `Whisper` struct: `{Sender string, Message string}`
  - [x] Define `WhisperHandler func(Whisper)` callback type
  - [x] Create `Listener` struct that holds registered handlers
  - [x] `NewListener() *Listener` constructor
  - [x] `OnWhisper(handler WhisperHandler)` — registers a callback
  - [x] `HandleSystemChat(text string)` — parses system chat, emits whisper events to handlers
- [x] Task 2: Parse whisper format from system chat (AC: #1, #2)
  - [x] Vanilla whisper format: `<sender> whispers to you: <message>` (from `ClearString()`)
  - [x] Alternative format: `[<sender> -> <botname>] <message>` (some server plugins)
  - [x] Use regex to extract sender and message from known formats
  - [x] Return `nil` for non-whisper messages (public chat, system messages)
  - [x] `parseWhisper(text string) *Whisper` — pure function for testability
- [x] Task 3: Wire listener into Connection (AC: #1, #2, #3)
  - [x] Add `SetWhisperListener(*chat.Listener)` to `Connection`
  - [x] In `SystemChat` handler, call `listener.HandleSystemChat(text)` after existing `dispatchChat`
  - [x] Listener is optional — if nil, whispers are silently ignored (backward compatible)
- [x] Task 4: Expose whisper subscription on BotState (AC: #3)
  - [x] Add `OnWhisper(func(sender, message string))` to `BotState` interface
  - [x] `Connection.OnWhisper` delegates to the chat `Listener`
  - [x] MCP server or future LLM can subscribe via this interface
- [x] Task 5: Write tests (all ACs)
  - [x] Test `parseWhisper` extracts sender and message from vanilla format
  - [x] Test `parseWhisper` extracts sender and message from plugin format
  - [x] Test `parseWhisper` returns nil for public chat messages
  - [x] Test `parseWhisper` returns nil for system messages (WorldEdit output, etc.)
  - [x] Test `Listener.HandleSystemChat` dispatches to registered handlers
  - [x] Test `Listener.HandleSystemChat` ignores non-whisper messages
  - [x] Test multiple handlers receive the same whisper
  - [x] Test no handlers registered = no panic
  - [x] All existing tests still pass

## Dev Notes

### How Minecraft Whispers Work

When a player sends `/msg BotName hello`, the **bot** receives a **SystemChat** message (not PlayerChatMessage). The format after `ClearString()` is typically:

**Vanilla server (1.21+):**
```
PlayerName whispers to you: hello
```

**Some plugin servers (Essentials, etc.):**
```
[PlayerName -> BotName] hello
```

The `PlayerChatMessage` event is for **public** chat messages typed in chat (not commands). Whispers are commands (`/msg`, `/tell`, `/w`) and their output comes through SystemChat.

### Current Chat Architecture

`Connection` already has:
- `SystemChat` callback in `msg.EventsHandler` (mc.go:246) — calls `dispatchChat(text)` which broadcasts to `chatListeners`
- `chatListeners` are used by `RunWECommand`/`RunCommand` for response capture
- `PlayerChatMessage` callback — currently only logs

The whisper listener hooks into the **same** `SystemChat` path but with different parsing. It runs alongside `dispatchChat` — no conflict.

### Why a Separate Package

`internal/chat/` is cleaner than adding whisper parsing to `internal/connection/mc.go` (already ~1200 lines). The chat package handles message parsing/formatting, while connection handles transport.

### Regex Patterns

```go
// Vanilla: "PlayerName whispers to you: message content"
var vanillaWhisperRe = regexp.MustCompile(`^(\w+) whispers to you: (.+)$`)

// Plugin: "[PlayerName -> BotName] message content"
var pluginWhisperRe = regexp.MustCompile(`^\[(\w+) -> \w+\] (.+)$`)
```

### Future Integration (Story 5.3+)

The `OnWhisper` subscription will be used by the LLM agent loop (Story 5.4) to receive player requests. For now, just emit events — no LLM processing.

### References

- [Source: internal/connection/mc.go:244-260] — Current chat handler setup
- [Source: internal/connection/mc.go:884-920] — dispatchChat and listenChat/unlistenChat
- [Source: github.com/Tnze/go-mc/bot/msg/chat.go] — SystemChat/PlayerChatMessage packet handling
- [Source: github.com/Tnze/go-mc/chat/message.go] — chat.Message with Translate field
- [Source: docs/epics.md#Story 5.1] — Original story definition

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Tasks 1 & 2 implemented as pure `internal/chat` package — no connection dependency
- parseWhisper is a pure function with two regex patterns (vanilla + plugin format)
- Listener uses RWMutex for concurrent handler registration and dispatch
- Listener created by default in connection.New() — no external SetWhisperListener needed
- SystemChat handler calls whisperListener.HandleSystemChat unconditionally (never nil)
- Task 4 added OnWhisper to BotState interface; Connection.OnWhisper adapts chat.Whisper to (sender, message) callback
- 10 tests in chat package covering all acceptance criteria
- mockBotState updated with no-op OnWhisper

### File List
- internal/chat/chat.go — Whisper struct, WhisperHandler, Listener, parseWhisper
- internal/chat/chat_test.go — 10 tests for parsing and listener behavior
- internal/connection/mc.go — whisperListener initialized in New(), OnWhisper, SystemChat handler integration
- internal/mcp/middleware.go — Added OnWhisper to BotState interface
- internal/mcp/middleware_test.go — Added OnWhisper to mockBotState

