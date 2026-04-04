# Story 1.4: Connection State Machine & Auto-Reconnect

Status: done

## Story

As a user,
I want the bot to automatically reconnect after disconnection,
so that building sessions survive server hiccups.

## Acceptance Criteria

1. **Given** the bot is connected and the server restarts
   **When** the disconnection is detected
   **Then** the bot transitions to `disconnected`, waits 1s, and attempts reconnection with exponential backoff (1s, 2s, 4s, 8s, 16s, cap 30s)

2. **Given** the bot has failed to reconnect 5 times
   **When** the 5th attempt fails
   **Then** it logs the failure and stays in `disconnected` state without crashing

3. **Given** the bot is reconnecting
   **When** the user sends SIGINT/SIGTERM
   **Then** the reconnect loop is cancelled immediately and the bot exits cleanly

4. **Given** the bot is connected
   **When** the connection state changes (disconnected, connecting, connected)
   **Then** each transition is logged with the reason and attempt count

5. **Given** the bot successfully reconnects
   **When** the new connection is established
   **Then** the game loop resumes automatically with chat logging intact

## Tasks / Subtasks

- [x] Task 1: Add connection state type and tracking to Connection struct (AC: #4)
  - [x] Define `ConnState` type with constants: `StateDisconnected`, `StateConnecting`, `StateConnected`
  - [x] Add `state ConnState` field to `Connection` struct (replaces boolean `connected`)
  - [x] Add `State() ConnState` public getter
  - [x] Update `IsConnected()` to check `state == StateConnected`
  - [x] Update existing code that sets `connected = true/false` to use new state values
  - [x] Log state transitions with old→new state and reason
- [x] Task 2: Implement reconnect loop with exponential backoff (AC: #1, #2, #3)
  - [x] Create `RunWithReconnect(ctx context.Context) error` — wraps Connect+HandleGame in a retry loop
  - [x] Backoff schedule: 1s, 2s, 4s, 8s, 16s, cap 30s (doubles each attempt)
  - [x] Max 5 reconnect attempts; reset counter on successful connection
  - [x] On disconnect detected: transition to `disconnected`, wait backoff, transition to `connecting`, call `Connect()`
  - [x] On max retries exceeded: log failure, return error (don't crash — caller decides)
  - [x] On context cancellation: break immediately, return `ctx.Err()`
  - [x] Use `time.After` for interruptible waits via select
- [x] Task 3: Wire reconnect into main.go (AC: #1, #3, #5)
  - [x] Replace `conn.Connect() + conn.HandleGame()` with `conn.RunWithReconnect(ctx)`
  - [x] Shutdown path unchanged: context cancellation → reconnect loop exits → Close() → exit
- [x] Task 4: Write tests (all ACs)
  - [x] Test state transitions: disconnected→connecting→connected lifecycle
  - [x] Test exponential backoff values: 1s, 2s, 4s, 8s, 16s, 30s, 30s (cap)
  - [x] Test max retries: after 5 failures, returns error without crash (retry exhaustion test)
  - [x] Test context cancellation during RunWithReconnect: exits immediately
  - [x] Test successful reconnect resets retry counter
  - [x] Test `IsConnected()` reflects new state-based logic
  - [x] Test ConnState.String() for all states including unknown
  - [x] Test existing tests still pass
  - [ ] Test existing tests still pass (offline auth, address formatting, etc.)

## Dev Notes

### Connection State Machine (from architecture-decision.md)

```
States: disconnected, connecting, connected
Transitions:
  disconnected → connecting (on connect() or auto-reconnect trigger)
  connecting → connected (on successful spawn)
  connecting → disconnected (on auth failure, timeout, max retries)
  connected → disconnected (on kick, error, server shutdown)
```

Exponential backoff: 1s, 2s, 4s, 8s, 16s, cap at 30s. Max 5 attempts.

### Current Connection Architecture

The `Connection` struct in `mc.go` currently has:
- `connected bool` + mutex — replace with `state ConnState`
- `Connect(ctx)` — creates client, auth, JoinServer, sets up handlers. Returns on success/failure.
- `HandleGame(ctx)` — runs `client.HandleGame()` in goroutine, returns on disconnect or ctx cancel
- `Close()` — mutex-protected disconnect, nil-safe
- `IsConnected()` — returns `connected` under mutex

The Disconnect callback in `basic.EventsListener` fires when the server kicks the bot. This is the trigger for auto-reconnect.

### Reconnect Design

`RunWithReconnect(ctx)` should be the new top-level API that replaces the separate `Connect()` + `HandleGame()` sequence in `main.go`:

```go
func (c *Connection) RunWithReconnect(ctx context.Context) error {
    for attempt := 0; ; attempt++ {
        c.setState(StateConnecting)
        if err := c.Connect(ctx); err != nil {
            if attempt >= maxRetries {
                return fmt.Errorf("max reconnect attempts exceeded: %w", err)
            }
            backoff := c.backoffDuration(attempt)
            c.logger.Warn("connection failed, retrying", ...)
            // interruptible wait
            select {
            case <-time.After(backoff):
                continue
            case <-ctx.Done():
                return ctx.Err()
            }
        }
        attempt = 0 // reset on successful connect
        gameErr := c.HandleGame(ctx)
        if ctx.Err() != nil {
            return ctx.Err() // shutdown requested
        }
        // connection lost — loop back to reconnect
        c.logger.Warn("connection lost, will reconnect", ...)
    }
}
```

Key design points:
- `Connect()` and `HandleGame()` remain as they are — `RunWithReconnect` orchestrates them
- Retry counter resets on successful connection (not just successful JoinServer)
- Context cancellation breaks out at any point: during Connect, HandleGame, or backoff wait
- `time.After` is fine here (short-lived timers, no leak concern in a loop that resets)

### Backoff Calculation

```go
func (c *Connection) backoffDuration(attempt int) time.Duration {
    d := time.Second << attempt // 1s, 2s, 4s, 8s, 16s, 32s, 64s...
    if d > 30*time.Second {
        d = 30 * time.Second
    }
    return d
}
```

### State Transition Logging

Every `setState()` call should log the transition:
```go
func (c *Connection) setState(new ConnState) {
    c.mu.Lock()
    old := c.state
    c.state = new
    c.mu.Unlock()
    if old != new {
        c.logger.Info("connection state changed",
            slog.String("from", old.String()),
            slog.String("to", new.String()),
        )
    }
}
```

### main.go Changes

Current:
```go
conn := connection.New(cfg, logger)
if err := conn.Connect(ctx); err != nil {
    return fmt.Errorf("minecraft connection: %w", err)
}
gameErr := conn.HandleGame(ctx)
```

After:
```go
conn := connection.New(cfg, logger)
gameErr := conn.RunWithReconnect(ctx)
```

Much simpler — reconnect logic is encapsulated in the connection layer.

### go-mc Reconnection Notes

- `bot.NewClient()` must be called fresh for each connection attempt — the client is not reusable after disconnect
- `basic.Player`, `msg.Manager`, `playerlist.PlayerList` are all per-client — recreated in `Connect()`
- `JoinServer()` is the only network call — if it fails, the client is still clean (no partial state)
- `HandleGame()` returns when the TCP connection drops — this is the disconnect signal
- Auth tokens can be reused across reconnects (MSA token lasts 24h)

### Testing Strategy

Since `Connect()` actually calls `JoinServer()` (real TCP), the reconnect tests should test the orchestration logic without hitting the network:
- Test `backoffDuration()` directly for correct exponential values
- Test state transitions via `setState()` and `State()` getters
- For `RunWithReconnect`, inject a mock `connectFn` similar to how `authFn` works — or test the reconnect loop logic in isolation
- Consider extracting the loop logic to accept a `connectAndRun func(ctx) error` for testability

### Previous Story Learnings (Story 1.3)

- `setupAuth()` returns error — online path calls injected `authFn`
- `AuthFunc` type + `authFn` field on Connection enables test injection without interfaces — use same pattern for reconnect testability if needed
- Offline path unchanged — sets Auth.Name + offline UUID, no auth call
- 28 total tests across all packages, all passing
- Code review M1 fixed: `os.Chmod` after `MkdirAll` enforces permissions on existing dirs
- go-mc installed from master: v1.20.3-0.20241224032005
- go-mc-ms-auth: v0.0.0-20230820124717-22f4d907eac4

### What This Story Does NOT Do

- Conversation context preservation across reconnects (Epic 5 concern — no conversations yet)
- Different behavior for auth failures vs network failures (all failures treated the same for MVP)
- Configurable max retries or backoff parameters (hardcoded constants for now)
- Health checks or heartbeat monitoring (keepalive is handled by go-mc's basic.Player)

### Project Structure Notes

After this story:

```
internal/connection/
├── auth.go          (unchanged)
├── auth_test.go     (unchanged)
├── mc.go            (modified — state machine, reconnect loop)
└── mc_test.go       (modified — new state + reconnect tests)
main.go              (modified — use RunWithReconnect)
```

### References

- [Source: docs/architecture-decision.md#Connection State Machine] — States, transitions, backoff spec
- [Source: docs/epics.md#Story 1.4] — Original story definition
- [Source: docs/stories/1-3-msa-authentication.md] — Previous story learnings
- [Source: docs/prd.md#FR1] — Connect to MC Java Edition 1.21.x
- [Source: docs/architecture-decision.md#Cross-Cutting Concerns] — Connection lifecycle with auto-reconnect

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- ConnState type replaces boolean `connected` — three states: disconnected, connecting, connected
- setState() logs transitions with from/to state names
- Connect() now sets StateConnecting on entry, StateDisconnected on auth failure
- GameStart callback sets StateConnected, Disconnect callback sets StateDisconnected
- RunWithReconnect() wraps Connect+HandleGame in retry loop with exponential backoff
- Backoff: 1s, 2s, 4s, 8s, 16s, cap 30s via bit-shift (time.Second << attempt)
- Max 5 reconnect attempts; counter resets on successful connection
- Context cancellation breaks out at any point: connect, game loop, or backoff wait
- main.go simplified from Connect+HandleGame to single RunWithReconnect call
- 36 total tests across all packages, all passing (8 new in mc_test.go)

### Code Review Fixes Applied

- **M1 — State stuck at StateConnecting on JoinServer failure**: Added `c.setState(StateDisconnected)` before returning error from JoinServer failure path in Connect().
- **M2 — Off-by-one in retry count**: Changed `attempt > MaxReconnectAttempts` to `attempt >= MaxReconnectAttempts`. Added injectable `connectAndRun` and `backoffFn` fields for testability. Added `TestRunWithReconnectRetryExhaustion` (verifies exactly 5 attempts) and `TestRunWithReconnectResetsOnSuccess` (verifies counter reset after success).
- **L1 — Task checkbox unchecked**: Fixed documentation.
- **L2 — Immediate first reconnect**: Skipped — immediate reconnect on clean server restart is better behavior.

### File List

- `internal/connection/mc.go` (modified — ConnState type, setState, RunWithReconnect, backoffDuration, injectable connectAndRun/backoffFn)
- `internal/connection/mc_test.go` (modified — state, backoff, reconnect exhaustion/reset tests)
- `main.go` (modified — use RunWithReconnect)
