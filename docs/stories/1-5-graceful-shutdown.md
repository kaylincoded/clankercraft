# Story 1.5: Graceful Shutdown

Status: done

## Story

As a user,
I want the bot to clean up on SIGINT/SIGTERM,
so that it doesn't leave ghost sessions on the server.

## Acceptance Criteria

1. **Given** the bot is connected
   **When** SIGINT is received
   **Then** the bot disconnects cleanly from the MC server, closes RCON if open, and exits with code 0

2. **Given** the bot is in a reconnect backoff wait
   **When** SIGINT is received
   **Then** the reconnect loop exits immediately and the bot shuts down cleanly

3. **Given** the bot is shutting down
   **When** Close() is called
   **Then** the HandleGame goroutine is drained before the process exits (no goroutine leak)

4. **Given** the shutdown sequence takes longer than expected
   **When** a second SIGINT is received (force quit)
   **Then** the bot exits immediately with non-zero code

## Tasks / Subtasks

- [x] Task 1: Add shutdown timeout and HandleGame goroutine drain to Close() (AC: #3)
  - [x] Add a shutdown timeout constant (5s) to Connection
  - [x] After closing the TCP connection, wait for the HandleGame goroutine to return (with timeout)
  - [x] Log if shutdown times out waiting for goroutine drain
- [x] Task 2: Add force-quit on second signal (AC: #4)
  - [x] In main.go, after first signal cancels context, restore default signal handling so a second SIGINT kills the process
  - [x] Log "press Ctrl+C again to force quit" after first signal
- [x] Task 3: Verify existing shutdown path and add integration-style test (AC: #1, #2)
  - [x] Verify SIGINT → context cancel → RunWithReconnect exits → Close() → exit 0 (existing path works)
  - [x] Test that Close() is safe during reconnect backoff (context cancellation already handles this)
  - [x] Test shutdown timeout behavior: Close() with stuck goroutine returns after timeout
  - [x] Test Close() drains goroutine quickly when doneCh closes
  - [x] Existing tests still pass

## Dev Notes

### Current Shutdown Path (Already Implemented)

Most of the graceful shutdown story is already done across Stories 1.1-1.4:

```
SIGINT/SIGTERM
  → signal.NotifyContext cancels ctx        (main.go:54, Story 1.1)
  → RunWithReconnect returns ctx.Err()      (mc.go:203-205, Story 1.4)
  → conn.Close() closes TCP connection      (main.go:62, Story 1.2)
  → run() returns nil → exit code 0         (main.go:64-67, Story 1.1)
```

**What's missing:**
1. HandleGame goroutine drain — Close() closes TCP but doesn't wait for the goroutine to finish
2. Force-quit on second signal — currently a second Ctrl+C does nothing because the signal is consumed
3. Shutdown logging could be improved (log the "shutting down" before Close, log completion)

### HandleGame Goroutine Drain

Current problem: `HandleGame()` starts a goroutine that calls `client.HandleGame()`. When context is cancelled, `HandleGame()` returns `ctx.Err()` to the caller, but the goroutine is still running until `Close()` closes the TCP socket. There's a race between process exit and goroutine cleanup.

Fix: Add a `doneCh` channel that the goroutine closes when it finishes. `Close()` waits on this channel (with timeout) after closing TCP.

```go
// In HandleGame:
c.doneCh = make(chan struct{})
go func() {
    defer close(c.doneCh)
    errCh <- c.client.HandleGame()
}()

// In Close:
if c.doneCh != nil {
    select {
    case <-c.doneCh:
    case <-time.After(ShutdownTimeout):
        c.logger.Warn("shutdown timeout waiting for game loop")
    }
}
```

### Force-Quit Pattern

Standard Go pattern for double-Ctrl+C:
```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

// ... start work with ctx ...

// After first signal cancels ctx:
// stop() restores default signal behavior, so second signal kills process
```

`stop()` is already called via `defer` — but it runs after `run()` returns. We need to call `stop()` earlier, right after the first signal, so the second signal gets default OS handling (kill).

### RCON Note

AC #1 mentions "closes RCON if open" — RCON doesn't exist yet (Story 4.1). The shutdown path should be extensible for future cleanup hooks, but for now just MC disconnect. No need to add RCON placeholder code.

### Previous Story Learnings (Story 1.4)

- ConnState type: disconnected/connecting/connected with logged transitions
- RunWithReconnect: retry loop with exponential backoff, context-interruptible at every point
- Close(): mutex-protected, sets state to disconnected, closes client.Conn
- Injectable connectAndRun/backoffFn fields for testability
- 36 total tests, all passing
- Review fixes: setState on JoinServer failure, off-by-one in retry count

### What This Story Does NOT Do

- RCON cleanup (Story 4.1 — doesn't exist yet)
- Conversation state persistence across restarts (out of scope — state is in-memory per PRD)
- Shutdown hooks or plugin system
- Graceful drain of in-flight MCP requests (Story 2.1 — MCP doesn't exist yet)

### Project Structure Notes

After this story:

```
internal/connection/
├── auth.go          (unchanged)
├── auth_test.go     (unchanged)
├── mc.go            (modified — doneCh for goroutine drain, shutdown timeout)
└── mc_test.go       (modified — shutdown timeout tests)
main.go              (modified — force-quit on second signal)
```

### References

- [Source: docs/epics.md#Story 1.5] — Original story definition
- [Source: docs/architecture-decision.md#Cross-Cutting Concerns] — Connection lifecycle, error handling
- [Source: docs/stories/1-4-connection-state-machine-auto-reconnect.md] — Previous story learnings
- [Source: docs/prd.md#NFR14] — Graceful shutdown on SIGINT/SIGTERM

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- Added doneCh channel to Connection — HandleGame goroutine closes it on exit
- Close() now waits on doneCh with ShutdownTimeout (5s) after closing TCP
- Logs warning if shutdown times out waiting for goroutine drain
- main.go: call stop() after RunWithReconnect returns to restore default signal handling
- Second SIGINT now gets default OS behavior (immediate kill)
- Log message "press Ctrl+C again to force quit" on first signal
- Shutdown path verified: SIGINT → ctx cancel → RunWithReconnect exits → stop() → Close() (drain) → exit 0
- 39 total tests across all packages, all passing (3 new shutdown tests)

### Code Review Fixes Applied

- **M1 — TestCloseTimesOutOnStuckGoroutine takes 5 real seconds**: Added injectable `shutdownTimeout` field on Connection (defaults to ShutdownTimeout const). Test now uses 50ms timeout — suite dropped from 5s to 0.06s.

### File List

- `internal/connection/mc.go` (modified — doneCh, ShutdownTimeout, Close goroutine drain)
- `internal/connection/mc_test.go` (modified — drain, timeout, constant tests)
- `main.go` (modified — stop() for force-quit, shutdown log message)
