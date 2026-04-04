# Story 4.2: Command Routing (Chat vs RCON)

Status: done

## Story

As a construction engine,
I want to route commands to chat or RCON based on operation type and availability,
so that bulk operations aren't throttled by chat rate limiting.

## Acceptance Criteria

1. **Given** a bulk operation (large `//set`, `//replace`, schematic paste) and RCON is available
   **When** the command is dispatched
   **Then** it goes through RCON

2. **Given** a player-session operation (wand selection, `//undo`) and RCON is available
   **When** the command is dispatched
   **Then** it goes through chat (because RCON runs as console, not as the bot player)

3. **Given** any operation and RCON is unavailable
   **When** the command is dispatched
   **Then** it goes through chat

## Tasks / Subtasks

- [x] Task 1: Define CommandRouter interface and implementation (AC: #1, #2, #3)
  - [x] Added `RCONExecutor` interface and `SetRCON` method to `Connection`
  - [x] `RunBulkWECommand` tries RCON (`//" + command`) first, falls back to `RunWECommand`
  - [x] `RunBulkCommand` tries RCON first, falls back to `RunCommand`
  - [x] Routing integrated directly into Connection — no separate Router struct needed
- [x] Task 2: Wire Router into MCP Server (AC: #1, #2, #3)
  - [x] Added `RunBulkWECommand` and `RunBulkCommand` to `BotState` interface
  - [x] `conn.SetRCON(rconClient)` called in `main.go` before `mcp.New()`
  - [x] Bulk WE tools (`we-set`, `we-replace`, `we-walls`, `we-faces`, `we-hollow`, `we-smooth`, `we-naturalize`, `we-overlay`) use `RunBulkWECommand`
  - [x] Player-session tools (`we-undo`, `we-redo`, `we-copy`, `we-paste`, `we-rotate`, `we-flip`) always use `RunWECommand`
  - [x] Generation tools (`we-sphere`, `we-cyl`, `we-pyramid`, `we-generate`) use `RunBulkWECommand`
  - [x] Vanilla tools (`setblock`, `fill`, `clone`) use `RunBulkCommand`
- [x] Task 3: Write tests (all ACs)
  - [x] Test RunBulkWECommand uses RCON when available
  - [x] Test RunBulkWECommand falls back to chat when RCON unavailable
  - [x] Test RunBulkCommand uses RCON when available
  - [x] Test RunBulkCommand falls back to chat when RCON unavailable
  - [x] Test player-session tools always use chat (never RunBulkWECommand)
  - [x] Test bulk tools route through RunBulkWECommand
  - [x] Test vanilla tools route through RunBulkCommand
  - [x] Updated mockBotState with RunBulkWECommand/RunBulkCommand
  - [x] All existing tests still pass

## Dev Notes

### RCON vs Chat: Why It Matters

RCON runs commands as the **server console**, not as the bot player. This means:
- `//set stone` via RCON → changes blocks as console (no player context)
- `//undo` via RCON → tries to undo console's last action, NOT the bot's
- Wand selections are per-player → RCON can't see/use them

**Rule:** Only stateless, bulk operations go through RCON. Anything that depends on the bot's player session (undo history, clipboard, wand selection) MUST use chat.

### Bulk vs Player-Session Classification

**RCON-eligible (bulk):** `//set`, `//replace`, `//walls`, `//faces`, `//hollow`, `//smooth`, `//naturalize`, `//overlay`, `//sphere`, `//cyl`, `//pyramid`, `//generate`, `/setblock`, `/fill`, `/clone`

**Chat-only (player-session):** `//undo`, `//redo`, `//copy`, `//paste`, `//rotate`, `//flip`

### Important RCON Command Format

RCON commands are sent **without** the leading `/`. The RCON protocol implicitly dispatches as server commands. So:
- WorldEdit via RCON: send `//set stone` (the `//` is part of the WE command, not the chat prefix)
- Vanilla via RCON: send `fill 0 64 0 10 74 10 stone` (no `/` prefix)

Verify this against `RunWECommand` which does `SendCommand("/" + command)` — for RCON, the equivalent is `Execute("//" + command)` for WE or `Execute(command)` for vanilla.

Wait — actually check what RCON expects. In Minecraft, RCON receives commands exactly as typed without the leading `/`. So `//set stone` works for WorldEdit, and `fill 0 0 0 10 10 10 stone` works for vanilla. The chat `SendCommand` adds `/` automatically, but RCON does not. So:
- WE via RCON: `rcon.Execute("/" + command)` where command is `"set stone"` → sends `//set stone` ✗
- Actually, `RunWECommand("set stone")` calls `SendCommand("/" + "set stone")` which sends `//set stone` as chat. For RCON equivalence: `rcon.Execute("/" + command)` sends `/set stone` which is wrong. Need `rcon.Execute("//" + command)` or just `rcon.Execute("/" + command)`.

**Verify with Minecraft RCON behavior:** RCON input is processed the same as console input. Console does NOT add a `/` prefix. So to run WorldEdit's `//set stone`, you send exactly `//set stone` to RCON. For vanilla `fill`, you send `fill 0 0 0 10 10 10 stone` (no `/`).

**Conclusion:**
- WE commands via RCON: `rcon.Execute("//" + command)` where `command` = `"set stone"`
- Vanilla commands via RCON: `rcon.Execute(command)` where `command` = `"fill 0 0 0 10 10 10 stone"`

### Minimal Approach: Router on Connection

Rather than refactoring all tool handlers, the simplest approach is:
1. Add a `SetRCON(*rcon.Client)` method on `Connection`
2. Modify `RunWECommand` and `RunCommand` to check RCON availability internally
3. Add a `rconEligible` bool parameter or create `RunBulkWECommand`/`RunBulkCommand` variants

Actually, the cleanest approach: add `RunBulkWECommand` and `RunBulkCommand` to `Connection` that try RCON first, fall back to chat. Then update the bulk tool handlers to call the bulk variants. Player-session tools keep using `RunWECommand`.

### Wire-Up in main.go

Currently: `mcp.New(version, logger, conn)` — `conn` is `*connection.Connection` which satisfies `BotState`.

Option A: Add `conn.SetRCON(rconClient)` before `mcp.New` — Connection holds the RCON client.
Option B: Pass RCON client separately: `mcp.New(version, logger, conn, rconClient)`.

Option A is simpler — Connection already has all the command dispatch methods, and RCON is just another dispatch channel. The BotState interface grows with `RunBulkWECommand` and `RunBulkCommand`.

### References

- [Source: internal/rcon/rcon.go] — RCON client (Story 4.1)
- [Source: internal/connection/mc.go#RunWECommand] — Chat WE dispatch
- [Source: internal/connection/mc.go#RunCommand] — Chat vanilla dispatch
- [Source: internal/mcp/middleware.go#BotState] — Interface tools call through
- [Source: internal/mcp/server.go] — All tool handlers calling RunWECommand/RunCommand
- [Source: main.go] — Wiring: conn, rconClient, mcpServer
- [Source: docs/epics.md#Story 4.2] — Original story definition

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Used Option A from dev notes: routing integrated into Connection via `SetRCON`, `RunBulkWECommand`, `RunBulkCommand`
- No separate Router struct needed — Connection already owns command dispatch
- `RCONExecutor` interface avoids circular import (connection doesn't import rcon)
- WE bulk commands send `"//" + command` to RCON; vanilla sends raw command
- 6 player-session tools unchanged (copy, paste, rotate, flip, undo, redo)
- 15 bulk tools updated to use bulk variants
- 10 new tests: 6 connection-level routing + 4 MCP-level routing verification

### File List
- internal/connection/mc.go — Added `RCONExecutor` interface, `SetRCON`, `RunBulkWECommand`, `RunBulkCommand`
- internal/connection/mc_test.go — Added 6 routing tests
- internal/mcp/middleware.go — Added `RunBulkWECommand`, `RunBulkCommand` to `BotState`
- internal/mcp/middleware_test.go — Updated mock with bulk methods
- internal/mcp/server.go — 15 handlers updated to use bulk dispatch
- internal/mcp/server_test.go — Added 4 routing verification tests
- main.go — Added `conn.SetRCON(rconClient)` wiring

