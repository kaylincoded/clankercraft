# Story 3.6: WorldEdit Terrain Operations & Patterns

Status: done

## Story

As a construction engine,
I want to execute `//smooth`, `//naturalize`, `//overlay` and use WorldEdit pattern syntax,
so that the bot can create organic, natural-looking terrain and mixed materials.

## Acceptance Criteria

1. **Given** tier is WorldEdit or FAWE and a selection is set
   **When** `//smooth 5` is dispatched
   **Then** the terrain is smoothed with 5 iterations

2. **Given** tier is WorldEdit or FAWE and a selection is set
   **When** a pattern like `50%stone,30%cobblestone,20%mossy_stone_bricks` is used in a `//set` command
   **Then** the region is filled with the specified material distribution

3. **Given** tier is WorldEdit or FAWE and a selection is set
   **When** `//naturalize` is dispatched
   **Then** the terrain is naturalized (top layer becomes grass/dirt)

4. **Given** tier is WorldEdit or FAWE and a selection is set
   **When** `//overlay stone` is dispatched
   **Then** the pattern is placed on top of existing blocks in the selection

5. **Given** tier is Vanilla
   **When** a terrain operation is requested
   **Then** a clear error is returned indicating WorldEdit is required

6. **Given** the bot is disconnected
   **When** a terrain tool is called
   **Then** a connection error is returned

## Tasks / Subtasks

- [x] Task 1: Register MCP tools (AC: #1, #3, #4, #5, #6)
  - [x] `we-smooth` — input: {iterations?}, sends `//smooth [iterations]` (default 1)
  - [x] `we-naturalize` — input: {} (no params), sends `//naturalize`
  - [x] `we-overlay` — input: {pattern}, sends `//overlay <pattern>`
  - [x] All three wrapped with `requireWorldEdit` (connection + tier + selection)
  - [x] Validate iterations > 0 on smooth (reject zero/negative)
  - [x] Validate pattern on overlay via existing `validatePattern()`
- [x] Task 2: Test pattern syntax end-to-end (AC: #2)
  - [x] Test `we-set` with weighted pattern `50%stone,50%cobblestone` sends correct command string
  - [x] Test `validatePattern` accepts `50%stone,30%cobblestone,20%mossy_stone_bricks`
  - [x] This validates the existing pattern regex covers weighted distribution syntax
- [x] Task 3: Write tests for new tools (all ACs)
  - [x] Test we-smooth sends `smooth 5` with iterations=5
  - [x] Test we-smooth defaults to `smooth 1` with no iterations
  - [x] Test we-smooth rejects iterations <= 0
  - [x] Test we-naturalize sends `naturalize`
  - [x] Test we-overlay sends `overlay stone` with pattern
  - [x] Test we-overlay rejects invalid pattern (command injection)
  - [x] Test all three tools reject vanilla tier
  - [x] Test all three tools reject no selection
  - [x] Test all three tools reject disconnected
  - [x] Existing tests still pass

## Dev Notes

### WorldEdit Command Syntax

```
//smooth [iterations]    — smooths terrain (default 1 iteration)
//naturalize             — naturalizes terrain (grass top, dirt below, stone deep)
//overlay <pattern>      — places pattern on top of existing blocks in selection
```

All three are **selection-based** — require both pos1 and pos2 set. Use `requireWorldEdit` middleware (same as `we-set`, `we-replace`, etc.).

### Pattern System Already Works

The `validatePattern` regex (`^[a-zA-Z0-9_:,%!^.\[\]]+$`) already supports weighted patterns:
- `50%stone,50%cobblestone` — commas separate entries, `%` for weights
- `50%stone,30%cobblestone,20%mossy_stone_bricks` — multi-material mix

AC #2 only requires proving this works end-to-end — no new pattern parsing code needed. A test that calls `we-set` with a weighted pattern and asserts the correct command string is sufficient.

### Smooth Iterations Validation

`//smooth` accepts an optional iteration count (default 1). Validate `iterations > 0` like the radius/size validation added to sphere/cyl/pyramid in Story 3.5 review. When iterations is 0 or omitted in the input, default to 1.

### Reusing Infrastructure

- `RunWECommand` from Story 3.4 handles command dispatch
- `validatePattern` for overlay's block pattern
- `requireWorldEdit` middleware for all three tools (selection required)
- `weCommandOutput` shared output type
- `toolError` for error responses

### Implementation Pattern

Follow exact same structure as existing WE tools (e.g., `handleWESet`, `handleWEHollow`):
1. Input type with JSON schema tags
2. Handler validates input, builds command string, calls `RunWECommand`
3. Returns `weCommandOutput{Response, Message}`
4. Register with `requireWorldEdit` middleware

### References

- [Source: docs/epics.md#Story 3.6] — Original story definition
- [Source: internal/mcp/server.go#validatePattern] — Pattern validation regex
- [Source: internal/mcp/server.go#handleWESet] — Example WE tool handler
- [Source: internal/mcp/middleware.go#requireWorldEdit] — Selection-requiring middleware
- [Source: internal/connection/mc.go#RunWECommand] — Command dispatch

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Added 3 MCP tools: we-smooth, we-naturalize, we-overlay (25 total tools)
- Smooth defaults to 1 iteration when omitted or <= 0
- Overlay validates pattern via existing validatePattern()
- Naturalize has no parameters
- All three use requireWorldEdit middleware (selection required)
- Pattern syntax end-to-end validated with weighted distribution test

### File List
- internal/mcp/server.go — added 3 terrain tool types, handlers, registrations
- internal/mcp/server_test.go — added 14 tests for terrain tools + pattern syntax

