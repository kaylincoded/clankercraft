# Story 3.4: WorldEdit Region Operations

Status: done

## Story

As a construction engine,
I want to execute `//set`, `//replace`, `//walls`, `//faces`, `//hollow`,
so that the bot can fill and modify selected regions.

## Acceptance Criteria

1. **Given** a selection is set and tier is WorldEdit or FAWE
   **When** `//set stone_bricks` is dispatched
   **Then** the region is filled with stone bricks and the engine reports the block count

2. **Given** a `//replace` command with source and target patterns
   **When** dispatched
   **Then** matching blocks in the selection are replaced

3. **Given** tier is Vanilla (no WorldEdit)
   **When** a region operation is requested
   **Then** a clear error is returned indicating WorldEdit is required

4. **Given** no selection is set
   **When** a region operation is requested
   **Then** a clear error is returned indicating a selection is needed

5. **Given** the bot is disconnected
   **When** a region operation tool is called
   **Then** a connection error is returned

## Tasks / Subtasks

- [x] Task 1: Implement RunWECommand helper (AC: #1, #2)
  - [x] `RunWECommand(command string) (string, error)` with configurable timeout (default 5s)
  - [x] Skips wand noise messages via `isWENoiseMessage` helper
  - [x] Returns error on timeout or send failure
- [x] Task 2: Add requireWorldEdit middleware (AC: #3, #4)
  - [x] Checks connection, tier (WorldEdit or FAWE), and selection (both pos set)
  - [x] Specific error messages for each failure mode
- [x] Task 3: Register MCP tools with pattern validation (AC: #1, #2, #3, #4, #5)
  - [x] `we-set`, `we-replace`, `we-walls`, `we-faces`, `we-hollow` — all requireWorldEdit-wrapped
  - [x] `validatePattern()` rejects command injection (newlines, special chars)
- [x] Task 4: Add BotState interface methods (AC: #1)
  - [x] `RunWECommand(command string) (string, error)` on BotState
- [x] Task 5: Write tests (all ACs)
  - [x] RunWECommand: sends command, captures response, skips wand messages, times out, send error
  - [x] requireWorldEdit: allows ready, rejects vanilla, rejects no selection (unit tests)
  - [x] MCP tools: asserts correct command string for all 5 tools + hollow with pattern
  - [x] Error paths: RunWECommand failure, invalid pattern injection, disconnected, vanilla, no selection
  - [x] Existing tests still pass (132 total, 18 new)

## Dev Notes

### RunWECommand Pattern

Reuse the chat listener infrastructure from Story 3.1 (detectTier). Unlike detectTier which filters for specific keywords, RunWECommand should capture the **first** chat response after sending the command (excluding wand selection messages which arrive from the persistent wand listener).

```go
func (c *Connection) RunWECommand(command string) (string, error) {
    ch := c.listenChat()
    defer c.unlistenChat(ch)

    if err := c.SendCommand("/" + command); err != nil {
        return "", err
    }

    timeout := time.After(5 * time.Second)
    for {
        select {
        case msg := <-ch:
            // Skip wand selection messages (handled by wand listener)
            if wandPos1Re.MatchString(msg) || wandPos2Re.MatchString(msg) {
                continue
            }
            return msg, nil
        case <-timeout:
            return "", fmt.Errorf("no response from server within 5s")
        }
    }
}
```

The command string should NOT include the leading `//` — `SendCommand` adds one `/`, making it `//set` on the server. So `RunWECommand("set stone")` sends `//set stone`.

### WorldEdit Response Format

Typical responses:
- `//set stone` → `"42 block(s) have been changed."`
- `//replace stone cobblestone` → `"15 block(s) have been changed."`
- `//walls stone` → `"24 block(s) have been changed."`
- `//faces stone` → `"48 block(s) have been changed."`
- `//hollow` → `"36 block(s) have been changed."`

These come as SystemChat messages. The MCP tools should return the raw response string — no need to parse the block count (the LLM can read it).

### requireWorldEdit Middleware

Stack on top of existing `requireConnection`:

```go
func requireWorldEdit[I, O any](bot BotState, handler ...) ... {
    // 1. Check connected (reuse requireConnection logic)
    // 2. Check tier is WorldEdit or FAWE
    // 3. Check selection is set (both positions)
    // 4. Call handler
}
```

Error messages:
- Not connected: "bot is not connected to a Minecraft server"
- Wrong tier: "WorldEdit is not available on this server (tier: vanilla)"
- No selection: "no selection set — use set-selection or wand to select a region first"

### Command Format Reference

```
//set <pattern>                    — fill selection with pattern
//replace <from> <to>              — replace from-pattern with to-pattern
//walls <pattern>                  — set only the walls (not floor/ceiling)
//faces <pattern>                  — set all 6 faces of the selection
//hollow [pattern]                 — hollow out the selection, optionally filling shell with pattern
```

### References

- [Source: docs/epics.md#Story 3.4] — Original story definition
- [Source: internal/connection/mc.go#SendCommand] — Command sending
- [Source: internal/connection/mc.go#listenChat] — Chat listener infrastructure
- [Source: internal/connection/mc.go#detectTier] — Command+response pattern to reuse
- [Source: internal/mcp/middleware.go#requireConnection] — Middleware pattern to extend
- [Source: internal/connection/mc.go#GetSelection] — Selection check (hasPos1 && hasPos2)
- [Source: internal/connection/mc.go#GetTier] — Tier check

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- `RunWECommand` sends command via `SendCommand("/" + cmd)` (making `//cmd` on server), captures first non-noise SystemChat response
- `isWENoiseMessage` helper skips wand position messages during WE command response capture
- `weTimeout` field on Connection for configurable timeout (tests use 50ms instead of 5s)
- `requireWorldEdit` middleware: connection + tier (WorldEdit/FAWE) + selection (both pos) — stacks all three checks
- `validatePattern` regex allowlist prevents command injection via pattern fields
- 5 new MCP tools: we-set, we-replace, we-walls, we-faces, we-hollow (18 tools total)
- Review fixed: H1 (pattern validation), H2 (noise filter), M2 (command string assertions), M3 (hollow with pattern), M4 (error path test), M6 (configurable timeout)
- 132 total tests (18 new: 4 RunWECommand, 3 middleware, 11 MCP)

### File List

- `internal/connection/mc.go` (modified — RunWECommand, isWENoiseMessage, WECommandTimeout, weTimeout field)
- `internal/connection/mc_test.go` (modified — 4 new RunWECommand tests)
- `internal/mcp/middleware.go` (modified — requireWorldEdit middleware, RunWECommand on BotState)
- `internal/mcp/middleware_test.go` (modified — 3 requireWorldEdit tests, RunWECommand mock)
- `internal/mcp/server.go` (modified — 5 WE tool types/handlers/registration, validatePattern, validPatternRe)
- `internal/mcp/server_test.go` (modified — 11 new WE MCP tool tests)

