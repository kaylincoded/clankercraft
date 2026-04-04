# Story 1.2: Minecraft Connection via go-mc

Status: done

## Story

As a developer,
I want the bot to connect to a Minecraft Java Edition server using go-mc,
so that we have a live protocol connection to build on.

## Acceptance Criteria

1. **Given** a running Minecraft 1.21.x server in offline mode
   **When** the binary is started with `--offline --host localhost --port 25565`
   **Then** the bot connects, joins the server, and appears in the player list

2. **Given** the bot is connected
   **When** the server sends a chat message
   **Then** the bot receives and logs the message to stderr

3. **Given** the bot is connected and receives SIGINT
   **When** shutdown is triggered
   **Then** the bot disconnects cleanly from the server before exiting

4. **Given** the bot fails to connect (server offline, wrong port, auth failure)
   **When** the connection attempt fails
   **Then** an error is logged and the process exits with a non-zero code

5. **Given** the bot is connected
   **When** the server kicks the bot or the connection drops
   **Then** the disconnection is logged with the reason (this story does NOT auto-reconnect — that's Story 1.4)

## Tasks / Subtasks

- [x] Task 1: Create `internal/connection/mc.go` — Minecraft connection manager (AC: #1, #3, #4, #5)
  - [x] Define `Connection` struct wrapping `*bot.Client`, `*basic.Player`, `*msg.Manager`, `*playerlist.PlayerList`
  - [x] `New(cfg *config.Config, logger *slog.Logger) *Connection`
  - [x] `Connect(ctx context.Context) error` — calls `client.JoinServer()`, sets up `basic.Player` and `msg.Manager`
  - [x] `HandleGame(ctx context.Context) error` — runs `client.HandleGame()` in a goroutine, returns on context cancellation or disconnect
  - [x] `Close() error` — clean disconnect
  - [x] Offline mode: set `Auth.Name` from config, compute UUID via `offline.NameToUUID`, skip access token
  - [x] Register `GameStart` callback to log spawn confirmation
  - [x] Register `Disconnect` callback to log kick reason
- [x] Task 2: Create chat message logging (AC: #2)
  - [x] Register `SystemChat` handler to log system messages
  - [x] Register `PlayerChatMessage` handler to log player messages with sender info
  - [x] Register `DisguisedChat` handler to log disguised messages
  - [x] All chat logged to stderr via slog at info level
- [x] Task 3: Wire connection into main.go (AC: #1, #3, #4)
  - [x] Create `Connection` after config/logger setup
  - [x] Call `Connect()` — exit on failure
  - [x] Run `HandleGame()` — blocks until disconnect or context cancellation
  - [x] On SIGINT/SIGTERM: cancel context, `Close()`, then exit
- [x] Task 4: Write tests (all ACs)
  - [x] Test `New()` returns properly configured Connection
  - [x] Test offline mode sets Auth.Name and UUID correctly, leaves AsTk empty
  - [x] Test `Close()` is safe to call when not connected
  - [x] Test HandleGame errors without Connect, IsConnected default, multiple Close safety

## Dev Notes

### go-mc API Reference

**Connection pattern** (from research — verified against go-mc source):

```go
import (
    "github.com/Tnze/go-mc/bot"
    "github.com/Tnze/go-mc/bot/basic"
    "github.com/Tnze/go-mc/bot/msg"
    "github.com/Tnze/go-mc/bot/playerlist"
    "github.com/Tnze/go-mc/chat"
    "github.com/Tnze/go-mc/offline"
    "encoding/hex"
)

c := bot.NewClient()
c.Auth.Name = "BotName"
id := offline.NameToUUID(c.Auth.Name)
c.Auth.UUID = hex.EncodeToString(id[:])
// Leave c.Auth.AsTk empty for offline mode

err := c.JoinServer("localhost:25565")  // handles full login sequence
// JoinServer does: TCP connect → handshake → login → configuration → ready

player := basic.NewPlayer(c, basic.Settings{}, basic.EventsListener{
    GameStart:  func() error { /* spawned */ },
    Disconnect: func(reason chat.Message) error { /* kicked */ },
})

pl := playerlist.New(c)

msgMgr := msg.New(c, player, pl, msg.EventsHandler{
    SystemChat:        func(m chat.Message, overlay bool) error { /* system msg */ },
    PlayerChatMessage: func(m chat.Message, validated bool) error { /* player msg */ },
    DisguisedChat:     func(m chat.Message) error { /* /say etc */ },
})

err = c.HandleGame()  // blocking loop — reads packets, dispatches to handlers
```

**Key facts:**
- `JoinServer` is synchronous — returns when login+configuration complete, or errors
- `HandleGame` is blocking — runs until connection drops or error
- `basic.Player` MUST be created — it handles keepalive responses (otherwise server kicks after 20s)
- `basic.Player` auto-handles teleport acceptance, keepalive echo, respawn
- Address format: `"host:port"` — port defaults to 25565 if omitted
- Chat messages have three types: SystemChat, PlayerChatMessage, DisguisedChat
- `msg.Manager` also provides `SendMessage(string)` and `SendCommand(string)` — needed in later stories

### go-mc Version Warning

go-mc master targets MC 1.21 (protocol 767). The latest tagged release is v1.20.2. Use `@master` or `@latest` to get 1.21 support — **do NOT use v1.20.2 tag** as it targets MC 1.20.2 protocol.

```
go get github.com/Tnze/go-mc@master
```

If `@master` has issues, fall back to `@v1.20.2` and test against a 1.20.2 server. The API shape is the same — only packet IDs and protocol details differ. Document which version was used.

### Architecture Compliance

- **`internal/connection/mc.go`** — new package. Provides MC protocol and chat I/O to the engine (future). No knowledge of MCP or chat semantics.
- **`internal/connection/` boundary rule:** provides connection primitives. Does NOT import `engine`, `mcp`, or `chat` packages.
- **`main.go`** wires connection into the startup sequence after config+logger, before the MCP server (Story 2.1).
- **Logging:** all output to stderr via slog. stdout is reserved for MCP (Story 2.1).
- **Context propagation:** `Connect()` and `HandleGame()` accept `context.Context` so SIGINT cancellation propagates cleanly.

### Connection Lifecycle

```
main.go:
  1. Load config            (Story 1.1 ✅)
  2. Setup logger           (Story 1.1 ✅)
  3. Create Connection      (this story)
  4. Connect()              (this story)
  5. HandleGame() in goroutine (this story)
  6. Wait for signal        (Story 1.1 ✅)
  7. Close() + exit         (this story)
```

### What This Story Does NOT Do

- Auto-reconnect (Story 1.4)
- MSA authentication (Story 1.3) — this story is offline-only
- MCP stdio transport (Story 2.1)
- Any construction/WorldEdit commands (Epic 3)
- Sending chat messages (used in later stories via `msg.Manager.SendMessage`)

### Previous Story Learnings (Story 1.1)

- cobra v1.10.2, viper v1.21.0 already in go.mod
- Config struct at `internal/config/config.go` — use `cfg.Host`, `cfg.Port`, `cfg.Username`, `cfg.Offline`
- Logger setup at `internal/log/log.go` — returns `*slog.Logger`
- `main.go` uses `signal.NotifyContext` for graceful shutdown — wire Connection.Close() into the shutdown path
- All logging to stderr via slog JSON handler
- Malformed config files now properly error (M1 fix from code review)

### Testing Strategy

Unit tests mock the go-mc client. **Do NOT require a live Minecraft server for tests** — that's integration testing.

- Test `New()` config mapping (Auth.Name, UUID, address formatting)
- Test offline mode flag correctly leaves AsTk empty
- Test `Close()` is nil-safe (called before Connect)
- Test chat log handlers by capturing slog output to a buffer
- go-mc's `bot.Client` is a struct (not interface) — test at the Connection wrapper level, verifying configuration and state, not the actual TCP connection

### Spike Gate

This story IS the spike gate from the PRD. If go-mc cannot connect to a 1.21.x server, we discover it here on day 1. The fallback is `--offline` mode against a 1.20.2 server while the 1.21 protocol support matures.

### Project Structure Notes

After this story:

```
internal/
├── config/
│   ├── config.go
│   └── config_test.go
├── connection/
│   ├── mc.go              (new — this story)
│   └── mc_test.go         (new — this story)
└── log/
    ├── log.go
    └── log_test.go
```

### References

- [Source: docs/architecture-decision.md#Component Architecture] — Connection layer provides MC protocol to engine
- [Source: docs/architecture-decision.md#Connection State Machine] — States: disconnected → connecting → connected
- [Source: docs/architecture-decision.md#Architectural Boundaries] — connection never imports mcp/chat/engine
- [Source: docs/prd.md#FR1] — Connect to MC Java Edition 1.21.x
- [Source: docs/prd.md#FR3] — Offline mode with --offline flag
- [Source: docs/prd.md#NFR13] — MC protocol via go-mc
- [Source: docs/epics.md#Story 1.2] — Original story definition
- [Source: docs/stories/1-1-project-initialization-cli-scaffolding.md] — Previous story learnings

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- go-mc installed from master: v1.20.3-0.20241224032005 (targets MC 1.21, protocol 767)
- Connection struct wraps bot.Client, basic.Player, msg.Manager, playerlist.PlayerList
- Offline mode: sets Auth.Name + offline UUID, leaves AsTk empty (no MSA)
- HandleGame runs client.HandleGame() in goroutine, select on context cancellation or game loop exit
- Context cancellation calls Close() to unblock the game loop, then drains the error channel
- Chat logging: SystemChat, PlayerChatMessage, DisguisedChat all logged to stderr via slog
- GameStart callback sets connected=true, Disconnect callback sets connected=false + logs reason
- main.go wired: config → logger → connection.New → Connect → HandleGame → signal → Close
- 8 new connection tests, 24 total tests across all packages, all passing
- Note: playerlist type is `PlayerList` not `List` (corrected from story spec)

### Code Review Fixes Applied

- **M1 — TestOfflineModeAuthSetup was testing config storage, not auth setup**: Extracted `setupAuth()` method from `Connect()`. Test now creates a `bot.Client`, calls `setupAuth`, and asserts `Auth.Name`, `Auth.UUID` (offline UUID computation), and empty `Auth.AsTk`. Added `TestOnlineModeAuthSetup` for the non-offline path.
- **M2 — HandleGame() owned Close() internally, unclear contract**: Removed `Close()` call from inside `HandleGame()`. Caller owns cleanup — `main.go` calls `Close()` after `HandleGame()` returns, which unblocks the goroutine via TCP close. Buffered errCh prevents goroutine leak.

### File List

- `internal/connection/mc.go` (new)
- `internal/connection/mc_test.go` (new)
- `main.go` (modified)
- `go.mod` (modified — added go-mc dependency)
- `go.sum` (modified)
