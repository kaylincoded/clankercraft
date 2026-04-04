# Story 3.7: WorldEdit Clipboard Operations

Status: done

## Story

As a construction engine,
I want to execute `//copy`, `//paste`, `//rotate`, and `//flip`,
so that the bot can duplicate and transform structures.

## Acceptance Criteria

1. **Given** tier is WorldEdit or FAWE and a selection is set
   **When** `//copy` is dispatched
   **Then** the selection is copied to the clipboard

2. **Given** a copied structure in the clipboard
   **When** `//paste` is dispatched at a new position
   **Then** the structure is pasted at the bot's position

3. **Given** a copied structure in the clipboard
   **When** `//rotate 90` then `//paste` is dispatched
   **Then** the structure is pasted rotated 90 degrees

4. **Given** a copied structure in the clipboard
   **When** `//flip north` is dispatched then `//paste`
   **Then** the structure is pasted flipped along the north-south axis

5. **Given** tier is Vanilla
   **When** a clipboard operation is requested
   **Then** a clear error is returned indicating WorldEdit is required

6. **Given** the bot is disconnected
   **When** a clipboard tool is called
   **Then** a connection error is returned

## Tasks / Subtasks

- [x] Task 1: Register MCP tools (AC: #1, #2, #3, #4, #5, #6)
  - [x] `we-copy` — input: {} (no params), sends `//copy`, requires selection (`requireWorldEdit`)
  - [x] `we-paste` — input: {skipAir?}, sends `//paste` or `//paste -a`, no selection needed (`requireWETier`)
  - [x] `we-rotate` — input: {degrees}, sends `//rotate <degrees>`, no selection needed (`requireWETier`)
  - [x] `we-flip` — input: {direction?}, sends `//flip [direction]`, no selection needed (`requireWETier`)
  - [x] Validate rotate degrees: must be 90, 180, or 270
  - [x] Validate flip direction if provided: north/south/east/west/up/down
- [x] Task 2: Write tests (all ACs)
  - [x] Test we-copy sends `copy`
  - [x] Test we-paste sends `paste`
  - [x] Test we-paste with skipAir sends `paste -a`
  - [x] Test we-rotate sends `rotate 90`, `rotate 180`, `rotate 270`
  - [x] Test we-rotate rejects invalid degrees (e.g., 45, 0, 360)
  - [x] Test we-flip sends `flip` (no direction), `flip north`, `flip up`
  - [x] Test we-flip rejects invalid direction
  - [x] Test we-copy requires selection (uses requireWorldEdit)
  - [x] Test we-paste/rotate/flip work without selection (uses requireWETier)
  - [x] Test all four tools reject vanilla tier
  - [x] Test all four tools reject disconnected
  - [x] Existing tests still pass

## Dev Notes

### Middleware Selection

**Selection-based** (requires `requireWorldEdit`):
- `//copy` — needs a selection to know what to copy

**Clipboard-based** (requires `requireWETier` only — no selection needed):
- `//paste` — pastes from clipboard at player's position
- `//rotate <degrees>` — rotates the clipboard contents
- `//flip [direction]` — flips the clipboard contents

This follows the same split as Story 3.5 where sphere/cyl/pyramid use `requireWETier` (position-based) while generate uses `requireWorldEdit` (selection-based).

### Command Syntax

```
//copy                  — copy selection to clipboard
//paste                 — paste clipboard at player position
//paste -a              — paste clipboard, skipping air blocks
//rotate <degrees>      — rotate clipboard (90, 180, 270 only)
//flip [direction]      — flip clipboard (north/south/east/west/up/down, default: player facing)
```

### Rotate Degrees Validation

WorldEdit only supports 90-degree increments: 90, 180, 270. Reject other values with a clear error. Do NOT use `validatePattern` — degrees are integers, not block patterns.

### Flip Direction Validation

Valid directions: `north`, `south`, `east`, `west`, `up`, `down`. Direction is optional — if omitted, WorldEdit flips relative to the player's facing direction. Validate via a simple string set lookup.

### Paste -a Flag

The `-a` flag on `//paste` skips air blocks, useful for overlaying structures without clearing surrounding blocks. Expose as a `skipAir` boolean input.

### Reusing Infrastructure

- `RunWECommand` from Story 3.4 handles command dispatch
- `requireWorldEdit` for copy (needs selection)
- `requireWETier` for paste/rotate/flip (clipboard-based, no selection)
- `weCommandOutput` shared output type
- `toolError` for error responses

### References

- [Source: docs/epics.md#Story 3.7] — Original story definition
- [Source: internal/mcp/server.go#handleWENaturalize] — No-param tool example
- [Source: internal/mcp/server.go#handleWESmooth] — Optional int param example
- [Source: internal/mcp/middleware.go#requireWETier] — Tier-only middleware
- [Source: internal/mcp/middleware.go#requireWorldEdit] — Selection-requiring middleware

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Added 4 MCP tools: we-copy, we-paste, we-rotate, we-flip (29 total tools)
- Copy uses requireWorldEdit (needs selection), paste/rotate/flip use requireWETier (clipboard-based)
- Rotate validates 90/180/270 only, flip validates direction via set lookup
- Paste supports skipAir flag for //paste -a
- Flip direction is optional (defaults to player facing)

### File List
- internal/mcp/server.go — added 4 clipboard tool types, handlers, validFlipDirections, registrations
- internal/mcp/server_test.go — added 20 tests for clipboard tools (with subtests for rotate/flip variants)

