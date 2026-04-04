# Story 3.9: Vanilla Fallback Construction

Status: done

## Story

As a construction engine,
I want to use `/fill`, `/setblock`, `/clone` when WorldEdit is unavailable,
so that the bot works on vanilla servers.

## Acceptance Criteria

1. **Given** tier is Vanilla and a selection is set
   **When** a region fill is requested
   **Then** it's decomposed into `/fill` commands respecting the 32,768 block limit per command

2. **Given** tier is any (Vanilla, WorldEdit, or FAWE)
   **When** a single block placement is requested
   **Then** `/setblock x y z block_type` is dispatched

3. **Given** tier is any and a selection is set
   **When** a clone operation is requested with destination coordinates
   **Then** `/clone` is used with appropriate source/destination coordinates

4. **Given** the bot is disconnected
   **When** a vanilla construction tool is called
   **Then** a connection error is returned

5. **Given** a fill region exceeds 32,768 blocks
   **When** the fill is requested
   **Then** the region is automatically decomposed into multiple `/fill` commands that each fit within the limit

## Tasks / Subtasks

- [x] Task 1: Register MCP tools (AC: #1, #2, #3, #4)
  - [x] `setblock` — input: {x, y, z, block}, sends `/setblock x y z block`, uses `requireConnection`
  - [x] `fill` — input: {x1, y1, z1, x2, y2, z2, block}, decomposes into `/fill` commands, uses `requireConnection`
  - [x] `clone` — input: {x1, y1, z1, x2, y2, z2, dx, dy, dz}, sends `/clone`, uses `requireConnection`
- [x] Task 2: Implement fill decomposition (AC: #1, #5)
  - [x] Calculate volume of region
  - [x] If volume <= 32,768, send single `/fill` command
  - [x] If volume > 32,768, split along the longest axis into sub-regions
  - [x] Recursively decompose until each sub-region fits
  - [x] Execute all sub-fill commands sequentially
  - [x] Report total blocks affected across all commands
- [x] Task 3: Validate block pattern for vanilla commands
  - [x] Use existing `validatePattern()` for block input on setblock and fill
  - [x] Vanilla block names use same format as WorldEdit patterns (e.g., `minecraft:stone`)
- [x] Task 4: Write tests (all ACs)
  - [x] Test setblock sends correct `/setblock x y z block` command
  - [x] Test fill sends single `/fill` for small region
  - [x] Test fill decomposes large region into multiple commands
  - [x] Test fill decomposition respects 32,768 block limit per chunk
  - [x] Test clone sends correct `/clone` command with coordinates
  - [x] Test all three tools work when disconnected (error)
  - [x] Test all three tools work on any tier (not restricted to vanilla)
  - [x] Test validatePattern rejects invalid block names
  - [x] Existing tests still pass

## Dev Notes

### Vanilla Commands Are Tier-Agnostic

These tools use `requireConnection` (not `requireWETier`) — they work on ALL server tiers. On a WorldEdit server, `/fill` and `/setblock` still work as vanilla server commands. The LLM decides when to use vanilla commands vs WorldEdit based on context.

### Command Syntax

```
/setblock <x> <y> <z> <block>
/fill <x1> <y1> <z1> <x2> <y2> <z2> <block>
/clone <x1> <y1> <z1> <x2> <y2> <z2> <dx> <dy> <dz>
```

### Fill Decomposition Algorithm

Vanilla `/fill` has a 32,768 block limit per command. For large regions:

1. Calculate volume: `(x2-x1+1) * (y2-y1+1) * (z2-z1+1)`
2. If volume <= 32,768, send single `/fill`
3. If volume > limit, split the region along the longest axis into two halves
4. Recurse on each half until all sub-regions fit within the limit
5. Execute commands sequentially, collecting responses

The decomposition function should be a pure function (testable without connection):
```go
func decomposeFill(x1, y1, z1, x2, y2, z2 int) [][6]int
```

### SendCommand for Vanilla

Vanilla commands use `Connection.SendCommand(command)` directly (no `RunWECommand` — that adds a leading `/` for `//` prefix). For vanilla commands like `/fill`, call `SendCommand("fill ...")` which sends `/fill ...` to the server.

Actually — `SendCommand` sends the raw command. Looking at the code: `mgr.SendCommand(command)` sends the command as a chat message. For vanilla commands, pass the command without the leading `/` since the message manager handles it. Check how `RunWECommand` calls it: `c.SendCommand("/" + command)` which becomes `//command`. For vanilla, just call `c.SendCommand("setblock ...")` which sends `/setblock ...`.

**Wait — verify this.** `RunWECommand` does `c.SendCommand("/" + command)` for WorldEdit (becomes `//cmd`). For vanilla, we want `/cmd`, so call `c.SendCommand("setblock ...")` and the msg manager sends `/setblock ...`. Need to verify `SendCommand` behavior — does it add a `/` or does the caller need to include it?

### Response Capture

Unlike WorldEdit which sends chat responses, vanilla commands may or may not produce visible output. Consider:
- `/setblock` — silent on success, error message on failure
- `/fill` — reports "X blocks have been filled"
- `/clone` — reports "X blocks have been cloned"

For now, use `RunWECommand`-style response capture with a shorter timeout, or fire-and-forget with `SendCommand`. The safest approach: create a `RunVanillaCommand` helper similar to `RunWECommand` but without the `//` prefix handling.

### References

- [Source: docs/epics.md#Story 3.9] — Original story definition
- [Source: internal/connection/mc.go#SendCommand] — Command dispatch method
- [Source: internal/connection/mc.go#RunWECommand] — Response capture pattern
- [Source: internal/mcp/server.go#validatePattern] — Block name validation
- [Source: internal/mcp/middleware.go#requireConnection] — Connection-only middleware

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Added `RunCommand` to `Connection` for vanilla command dispatch (no `//` prefix)
- Added `RunCommand` to `BotState` interface and mock
- Implemented `decomposeFill` as pure recursive function splitting along longest axis
- 3 new tools (setblock, fill, clone) all use `requireConnection` — tier-agnostic
- 17 new tests: 5 decomposeFill unit tests, 4 setblock, 5 fill, 3 clone
- Total tool count: 34

### Code Review Fixes (Claude Opus 4.6, 2026-04-04)
- **H1**: Removed `isWENoiseMessage` filter from `RunCommand` (vanilla commands don't produce WE noise); fixed timeout error to use resolved `dur` instead of hardcoded `WECommandTimeout`
- **M1**: Added overlap warning to clone tool description

### File List
- internal/connection/mc.go — Added `RunCommand` method
- internal/mcp/middleware.go — Added `RunCommand` to `BotState` interface
- internal/mcp/middleware_test.go — Added `runCommandFn` and `RunCommand` to mock
- internal/mcp/server.go — Added input/output types, `decomposeFill`, 3 handlers, 3 tool registrations
- internal/mcp/server_test.go — Added 17 tests for vanilla tools

