# Story 4.1: RCON Client Connection

Status: done

## Story

As a system,
I want to connect to the Minecraft server's RCON port,
so that I can send commands directly to the server console.

## Acceptance Criteria

1. **Given** RCON is configured (`--rcon-port`, `--rcon-password`)
   **When** the bot starts
   **Then** it establishes an RCON connection and logs success

2. **Given** RCON is not configured (no `--rcon-password` provided)
   **When** the bot starts
   **Then** RCON is marked as unavailable and the engine routes all commands through chat

3. **Given** RCON connection fails (wrong password, port unreachable)
   **When** the bot starts
   **Then** it logs a warning and falls back to chat-only mode (does not crash)

## Tasks / Subtasks

- [x] Task 1: Create RCON client wrapper (AC: #1, #2, #3)
  - [x] Create `internal/rcon/rcon.go` with `Client` struct wrapping `go-mc/net.RCONClientConn`
  - [x] `New(cfg *config.Config, logger *slog.Logger) *Client` constructor
  - [x] `Connect(ctx context.Context) error` — dials RCON using `mcnet.DialRCON(addr, password)`
  - [x] `Execute(command string) (string, error)` — sends command via `Cmd()`, reads via `Resp()`
  - [x] `Close() error` — closes the underlying connection
  - [x] `IsAvailable() bool` — returns true if connected successfully
  - [x] Skip connection entirely if `RCONPassword` is empty (AC #2)
- [x] Task 2: Integrate RCON into startup (AC: #1, #2, #3)
  - [x] In `main.go`, after creating `conn`, create RCON client: `rconClient := rcon.New(cfg, logger)`
  - [x] Call `rconClient.Connect(ctx)` — log success or warning on failure
  - [x] On failure: log warning, continue with chat-only mode (do NOT return error)
  - [x] Pass `rconClient` to `mcp.New()` or store on connection for later use by Story 4.2
  - [x] Add `rconClient.Close()` to shutdown sequence
- [x] Task 3: Write tests (all ACs)
  - [x] Test `New` creates client without connecting
  - [x] Test `Connect` with no password returns immediately, `IsAvailable()` = false
  - [x] Test `Connect` with invalid address fails gracefully, `IsAvailable()` = false
  - [x] Test `Execute` on connected client sends command and returns response
  - [x] Test `Execute` on unavailable client returns error
  - [x] Test `Close` is safe to call on unconnected client
  - [x] Use `go-mc/net.ListenRCON` to create a test RCON server for integration tests

## Dev Notes

### go-mc Has Built-In RCON

The existing dependency `github.com/Tnze/go-mc` (via the `mj41/go-mc` fork in go.mod) includes a full RCON implementation at `github.com/Tnze/go-mc/net`. No new dependencies needed.

**Client API:**
```go
import mcnet "github.com/Tnze/go-mc/net"

// Connect and authenticate in one call
client, err := mcnet.DialRCON("localhost:25575", "password")

// Send command
err = client.Cmd("say hello")

// Read response
resp, err := client.Resp()

// Close
client.Close()
```

**Test Server API (for integration tests):**
```go
listener, _ := mcnet.ListenRCON("127.0.0.1:0") // random port
conn, _ := listener.Accept()
conn.AcceptLogin("testpass")
cmd, _ := conn.AcceptCmd()
conn.RespCmd("Done")
```

### Config Already Supports RCON

`internal/config/config.go` already defines:
- `RCONPort int` (default: 25575, flag: `--rcon-port`)
- `RCONPassword string` (default: "", flag: `--rcon-password`)
- `MaskedRCONPassword()` helper for safe logging
- Env vars: `CLANKERCRAFT_RCON_PORT`, `CLANKERCRAFT_RCON_PASSWORD`

The config is already logged in `main.go:52-53`. No config changes needed.

### RCON Is a Separate Package

Create `internal/rcon/` as a new package — RCON is independent of the Minecraft protocol connection. The `Connection` struct in `internal/connection/mc.go` manages the game client (go-mc bot); RCON is a separate TCP connection to the server console.

### RCON Address Format

RCON connects to the same host but different port: `cfg.Host + ":" + strconv.Itoa(cfg.RCONPort)`. The default RCON port is 25575.

### Graceful Degradation Pattern

This story establishes the pattern: try RCON, log result, continue regardless. Story 4.2 will add command routing that checks `rconClient.IsAvailable()` to decide chat vs RCON dispatch.

### Concurrency Note

`DialRCON` is a blocking TCP dial. Call it outside the errgroup goroutines — do it synchronously in `run()` before starting the errgroup, since RCON availability is known immediately. If it fails, log and continue. The RCON connection itself is not long-lived in the same way as the MC bot connection — it's a simple request/response TCP socket.

### References

- [Source: github.com/Tnze/go-mc/net/rcon.go] — RCON client/server implementation
- [Source: internal/config/config.go#L20-21] — RCONPort, RCONPassword config fields
- [Source: internal/config/config.go#L41-42] — CLI flag bindings
- [Source: main.go#L52-53] — RCON config already logged at startup
- [Source: docs/epics.md#Story 4.1] — Original story definition
- [Source: docs/epics.md#Story 4.2] — Next story (command routing) depends on this

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Created `internal/rcon/` package with `Client` wrapping `go-mc/net.RCONClientConn`
- Injectable `DialFunc` for testability — unit tests use mock, integration tests use real `ListenRCON`
- Graceful degradation: no password = skip, dial failure = warn + continue, never crash
- `sync.Mutex` protects conn/available for thread safety
- Integrated into `main.go` startup (before errgroup) and shutdown sequence
- 13 tests: 11 unit tests + 2 integration tests using go-mc's built-in RCON server
- No new dependencies — uses existing `go-mc/net` RCON support

### Code Review Fixes (Claude Opus 4.6, 2026-04-04)
- **H1**: `Connect()` now runs `dialFn` in a goroutine with context select — cancellation interrupts blocking dials. Added `TestConnectRespectsContextCancellation`.

### File List
- internal/rcon/rcon.go — New RCON client package
- internal/rcon/rcon_test.go — 12 tests (unit + integration)
- main.go — RCON client creation, connect, and close in startup/shutdown

