# Story 3.2: WorldEdit Selection Management

Status: done

## Story

As a construction engine,
I want to set WorldEdit positions (`//pos1`, `//pos2`) and manage selections,
so that region operations have targets.

## Acceptance Criteria

1. **Given** coordinates (x1, y1, z1) and (x2, y2, z2)
   **When** a selection is set
   **Then** the bot sends `//pos1 x1,y1,z1` and `//pos2 x2,y2,z2` via chat

2. **Given** a selection is set
   **When** queried
   **Then** the engine tracks the current selection coordinates in memory

3. **Given** the tier is Vanilla (no WorldEdit)
   **When** a selection is set
   **Then** the coordinates are still tracked in memory (for vanilla /fill commands later)

4. **Given** the bot is disconnected
   **When** selection tools are called
   **Then** a connection error is returned

## Tasks / Subtasks

- [x] Task 1: Add Selection type to engine package (AC: #2, #3)
  - [x] Define `Selection` struct with X1,Y1,Z1,X2,Y2,Z2 ints and String() method
  - [x] Add selection state to Connection (selection + hasSelection fields)
- [x] Task 2: Implement set-selection logic (AC: #1, #3)
  - [x] `SetSelection(x1,y1,z1, x2,y2,z2)` stores coords in memory
  - [x] If tier is WorldEdit or FAWE, also send `//pos1 x1,y1,z1` and `//pos2 x2,y2,z2` via SendCommand
  - [x] If tier is Vanilla, only store in memory (no chat commands)
  - [x] Selection resets on disconnect alongside tier and position
- [x] Task 3: Register MCP tools (AC: #1, #2, #3, #4)
  - [x] `set-selection` — input: {x1,y1,z1,x2,y2,z2}, sets both positions
  - [x] `get-selection` — returns current selection or "no selection set"
  - [x] Both wrapped with requireConnection middleware
- [x] Task 4: Write tests (all ACs)
  - [x] Test set-selection stores coordinates
  - [x] Test get-selection returns stored coordinates
  - [x] Test set-selection stores via MCP tool
  - [x] Test get-selection returns via MCP tool / errors when not set
  - [x] Test tools error when disconnected
  - [x] Test selection resets on disconnect
  - [x] Test set-selection sends WE commands when tier is WorldEdit
  - [x] Test set-selection sends WE commands when tier is FAWE
  - [x] Test set-selection skips WE commands when tier is Vanilla
  - [x] Test set-selection returns error on command failure
  - [x] Existing tests still pass (105 total, 12 new)

## Dev Notes

### WorldEdit Position Commands

```
//pos1 x,y,z    — sets first corner of selection
//pos2 x,y,z    — sets second corner of selection
```

These are sent via `SendCommand("/pos1 x,y,z")` which becomes `//pos1 x,y,z` on the server.

### Selection State

Store selection on Connection alongside tier. The selection is reset on disconnect (same as tier/position). Future stories (3.3) will also update selection from wand chat parsing.

### Interface Expansion

BotState needs:
- `SetSelection(x1,y1,z1,x2,y2,z2 int) error`
- `GetSelection() (Selection, bool)` — returns selection and whether one is set

### References

- [Source: docs/epics.md#Story 3.2] — Original story definition
- [Source: internal/connection/mc.go#SendCommand] — Chat command sending from Story 3.1
- [Source: internal/engine/tier.go] — Tier type for conditional WE command dispatch

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- `engine.Selection` struct with X1,Y1,Z1,X2,Y2,Z2 and String() method in `selection.go`
- Connection gains `selection` and `hasSelection` fields, reset on disconnect
- `SetSelection()` stores coords and conditionally sends `//pos1` / `//pos2` via SendCommand when tier is WorldEdit or FAWE
- `GetSelection()` returns selection and whether one is set
- BotState interface expanded with `SetSelection` and `GetSelection`
- Two new MCP tools: `set-selection` (13th tool) and `get-selection` (14th tool), both requireConnection-wrapped
- Added `sendCommandFn` injectable on Connection for testing command dispatch (same pattern as `connectAndRun`, `backoffFn`)
- 105 total tests (12 new: 1 engine, 6 connection, 5 MCP)

### File List

- `internal/engine/selection.go` (new — Selection struct, String())
- `internal/engine/selection_test.go` (new — selection string test)
- `internal/connection/mc.go` (modified — selection fields, SetSelection, GetSelection, resetSelection, disconnect wiring)
- `internal/connection/mc_test.go` (modified — 2 new tests for selection store and reset)
- `internal/mcp/middleware.go` (modified — BotState interface expanded with SetSelection, GetSelection)
- `internal/mcp/middleware_test.go` (modified — mockBotState expanded with selection fields and methods)
- `internal/mcp/server.go` (modified — set-selection and get-selection tool types, handlers, registration)
- `internal/mcp/server_test.go` (modified — 5 new MCP tool tests)
