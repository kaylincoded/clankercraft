# Story 3.1: WorldEdit Capability Tier Detection

Status: done

## Story

As a system,
I want to detect whether the server has FAWE, WorldEdit, or neither on connection,
so that the construction engine knows which commands are available.

## Acceptance Criteria

1. **Given** the bot connects to a server with FAWE installed
   **When** capability detection runs
   **Then** tier is set to FAWE and logged

2. **Given** the bot connects to a server with WorldEdit (no FAWE)
   **When** capability detection runs
   **Then** tier is set to WorldEdit and logged

3. **Given** the bot connects to a vanilla server
   **When** capability detection runs
   **Then** tier is set to Vanilla and logged

4. **Given** the bot reconnects after disconnection
   **When** capability detection runs again
   **Then** the tier is re-detected (not cached from previous session)

## Tasks / Subtasks

- [x] Task 1: Define capability tier types in a new `internal/engine/` package (AC: #1, #2, #3)
  - [x] Create `internal/engine/tier.go` with `Tier` type (FAWE, WorldEdit, Vanilla, Unknown)
  - [x] Add `String()` method for logging
- [x] Task 2: Implement chat-command-based detection (AC: #1, #2, #3)
  - [x] Add `SendCommand` method using `msgMgr.SendCommand()`
  - [x] Add chat listener infrastructure (listenChat/unlistenChat/dispatchChat)
  - [x] `detectTier()` sends `//version` via chat, parses response:
    - Response contains "fastasyncworldedit" or "fawe" → FAWE
    - Response contains "worldedit" → WorldEdit
    - No matching response within 3s → Vanilla
  - [x] Detection runs in a goroutine with 3-second timeout for vanilla servers
- [x] Task 3: Wire detection into connection lifecycle (AC: #4)
  - [x] Run detection in goroutine after GameStart event fires
  - [x] Reset tier to TierUnknown on disconnect
  - [x] Re-detect on every reconnect (no caching across sessions)
- [x] Task 4: Expose tier via MCP tool (AC: #1, #2, #3)
  - [x] Add `detect-worldedit` MCP tool that returns current tier string
  - [x] Returns "unknown" if detection hasn't completed yet
  - [x] Wrapped with requireConnection middleware
- [x] Task 5: Write tests (all ACs)
  - [x] Test tier String() for all values
  - [x] Test chat listener dispatch and unlisten
  - [x] Test tier resets on disconnect
  - [x] Test MCP tool returns WorldEdit tier
  - [x] Test MCP tool returns FAWE tier
  - [x] Test MCP tool errors when disconnected
  - [x] Existing tests still pass (92 total, 7 new)

## Dev Notes

### Detection Strategy

The bot sends `//version` as a chat message after spawning. WorldEdit responds with a system chat message like:

```
WorldEdit version 7.4.0
```

or for FAWE:

```
FastAsyncWorldEdit version 2.x.x
```

On vanilla servers, the command is unknown and either produces no response or an "Unknown command" message. Use a timeout to handle this case.

### Chat Send/Receive Infrastructure

This story needs two new Connection capabilities:
1. **SendChat** — send a chat message (the bot typing in chat). Use `ServerboundChat` packet or the `msg.Manager` that already exists on the Connection.
2. **Chat response capture** — ability to wait for a system chat message matching a pattern, with timeout. The Connection already has a `SystemChat` handler in `msg.EventsHandler` that logs messages. We need to add a way to register a one-shot listener that captures a specific response.

**Important**: The `msg.Manager` in go-mc handles signed chat. For sending chat commands (starting with `/`), use the chat command packet (`ServerboundChatCommand`) rather than regular chat, since Paper servers enforce signed chat differently for commands vs messages.

### Package Structure

Create `internal/engine/` as the home for the construction engine. Story 3.1 only adds tier detection, but future stories (3.2-3.9) will add command dispatch, selection management, etc.

```
internal/engine/
├── tier.go        # Tier type, Detector struct
└── tier_test.go   # Detection tests
```

### Connection Interface Expansion

BotState interface needs:
- `SendChat(msg string) error` — for sending `//version`
- `GetTier() string` — returns detected WorldEdit tier

OR: the engine package could take the Connection directly rather than going through BotState. Consider which is cleaner for the construction engine's future needs.

### Server Context

The creative server at `mcc.goonies.gg` has WorldEdit 7.4.0 (no FAWE). This is the primary test target.

### go-mc Chat Command Packet

For sending commands (messages starting with `/`), check if go-mc has a `ServerboundChatCommand` packet handler in the `msg` package. If not, construct the packet manually:
- Packet ID: `ServerboundChatCommand` (check `data/packetid/`)
- Fields: command string (without leading `/`), timestamp, salt, signatures

### References

- [Source: docs/epics.md#Story 3.1] — Original story definition
- [Source: docs/epics.md#Epic 3] — Full epic context for future stories
- [Source: internal/connection/mc.go#Connect] — GameStart event handler where detection should trigger
- [Source: internal/connection/mc.go#msg.EventsHandler] — Existing chat handlers
- [Source: internal/mcp/middleware.go#BotState] — Interface to expand

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- `engine.Tier` is a simple int enum: Unknown, Vanilla, WorldEdit, FAWE with String() method
- Chat listener infrastructure: `listenChat()` returns buffered channel, `dispatchChat()` broadcasts to all, `unlistenChat()` removes by channel identity
- `detectTier()` runs as goroutine from GameStart, sends `//version` via `msgMgr.SendCommand()`, loops on chat listener for 3s timeout
- Detection is case-insensitive, checks "fastasyncworldedit"/"fawe" first (FAWE includes WorldEdit in its response)
- Tier resets to TierUnknown on disconnect, re-detected on every reconnect
- `SendCommand` wraps `msgMgr.SendCommand()` which sends `ServerboundChatCommand` packet (no leading /)
- SystemChat handler now calls `ClearString()` for clean text and dispatches to listeners
- 92 total tests (7 new: 1 tier string, 2 chat listener, 1 tier reset, 3 MCP tool)

### File List

- `internal/engine/tier.go` (new — Tier type, constants, String())
- `internal/engine/tier_test.go` (new — tier string test)
- `internal/connection/mc.go` (modified — chat listeners, SendCommand, detectTier, resetTier, GetTier, GameStart wiring)
- `internal/connection/mc_test.go` (modified — 3 new tests for chat listeners and tier reset)
- `internal/mcp/middleware.go` (modified — BotState interface expanded with GetTier)
- `internal/mcp/middleware_test.go` (modified — mockBotState expanded with tier field)
- `internal/mcp/server.go` (modified — detect-worldedit tool types, handler, registration)
- `internal/mcp/server_test.go` (modified — 3 new MCP tool tests)
