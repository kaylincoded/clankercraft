# Story 3.5: WorldEdit Generation Commands

Status: done

## Story

As a construction engine,
I want to execute `//sphere`, `//cyl`, `//pyramid`, and `//generate`,
so that the bot can create geometric and mathematical shapes.

## Acceptance Criteria

1. **Given** tier is WorldEdit or FAWE
   **When** `//sphere stone 10` is dispatched at the bot's position
   **Then** a stone sphere of radius 10 is generated

2. **Given** tier is WorldEdit or FAWE
   **When** `//cyl stone 5 10` is dispatched
   **Then** a stone cylinder of radius 5 and height 10 is generated

3. **Given** tier is WorldEdit or FAWE
   **When** `//pyramid stone 5` is dispatched
   **Then** a stone pyramid of size 5 is generated

4. **Given** tier is WorldEdit or FAWE and a selection is set
   **When** `//generate` with a mathematical expression is dispatched
   **Then** the expression is evaluated and blocks are placed in the selection

5. **Given** tier is Vanilla
   **When** a generation command is requested
   **Then** a clear error is returned indicating WorldEdit is required

6. **Given** the bot is disconnected
   **When** a generation tool is called
   **Then** a connection error is returned

## Tasks / Subtasks

- [x] Task 1: Add requireWETier middleware (AC: #5, #6)
  - [x] `requireWETier` — checks connection + tier (WorldEdit or FAWE) but NOT selection
  - [x] For position-based commands (sphere, cyl, pyramid) that don't need a selection
  - [x] `//generate` still uses `requireWorldEdit` since it operates on the selection
- [x] Task 2: Register MCP tools (AC: #1, #2, #3, #4, #5, #6)
  - [x] `we-sphere` — input: {pattern, radius, hollow?}, sends `//sphere <pattern> <radius>` or `//hsphere`
  - [x] `we-cyl` — input: {pattern, radius, height?, hollow?}, sends `//cyl <pattern> <radius> [height]` or `//hcyl`
  - [x] `we-pyramid` — input: {pattern, size, hollow?}, sends `//pyramid <pattern> <size>` or `//hpyramid`
  - [x] `we-generate` — input: {expression, pattern?}, sends `//generate <expression>` (requires selection)
  - [x] sphere/cyl/pyramid wrapped with requireWETier, generate with requireWorldEdit
- [x] Task 3: Write tests (all ACs)
  - [x] Test requireWETier allows WorldEdit tier without selection
  - [x] Test requireWETier rejects vanilla tier
  - [x] Test each tool sends correct command string
  - [x] Test hollow variants (hsphere, hcyl, hpyramid)
  - [x] Test we-generate requires selection
  - [x] Test tools error when disconnected
  - [x] Existing tests still pass

## Dev Notes

### Position-Based vs Selection-Based Commands

**Position-based** (use player position, NO selection needed):
```
//sphere <pattern> <radius>         — solid sphere
//hsphere <pattern> <radius>        — hollow sphere
//cyl <pattern> <radius> [height]   — solid cylinder (height defaults to 1)
//hcyl <pattern> <radius> [height]  — hollow cylinder
//pyramid <pattern> <size>          — solid pyramid
//hpyramid <pattern> <size>         — hollow pyramid
```

**Selection-based** (requires selection):
```
//generate <expression>             — generate blocks from math expression
```

### Hollow Variants

WorldEdit uses the `h` prefix for hollow shapes: `//hsphere`, `//hcyl`, `//hpyramid`. Rather than separate tools, add a `hollow` boolean input to each shape tool and dispatch the `h` variant when true.

### //generate Expression Safety

The `//generate` command takes a mathematical expression like `(x*x + z*z < 100) * stone`. The `validatePattern` regex from Story 3.4 is too restrictive for expressions (which use parentheses, operators, spaces). Use a separate, more permissive validation that still blocks newlines and command injection but allows math characters.

### Reusing Infrastructure

- `RunWECommand` from Story 3.4 handles command dispatch and response capture
- `validatePattern` for block patterns on sphere/cyl/pyramid
- `requireWorldEdit` for //generate (needs selection)
- New `requireWETier` for position-based tools (no selection check)

### References

- [Source: docs/epics.md#Story 3.5] — Original story definition
- [Source: internal/connection/mc.go#RunWECommand] — Command dispatch from Story 3.4
- [Source: internal/mcp/middleware.go#requireWorldEdit] — Selection-requiring middleware
- [Source: internal/mcp/server.go#validatePattern] — Pattern validation from Story 3.4

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Added `requireWETier` middleware for position-based commands (no selection check)
- Added 4 MCP tools: we-sphere, we-cyl, we-pyramid, we-generate (22 total tools)
- Hollow variants use `h` prefix (hsphere, hcyl, hpyramid) via `hollow` boolean input
- Expression validation for //generate blocks newlines, carriage returns, semicolons, and slashes
- Pattern validation reuses `validatePattern` from Story 3.4
- sphere/cyl/pyramid use `requireWETier`, generate uses `requireWorldEdit`
- [Review fix H1] validateExpression now also blocks `;` and `/` to prevent command injection
- [Review fix M1] Added radius/size > 0 validation on sphere, cyl, pyramid handlers

### File List
- internal/mcp/middleware.go — added requireWETier middleware
- internal/mcp/middleware_test.go — added requireWETier unit tests
- internal/mcp/server.go — added 4 generation tool types, handlers, validateExpression, registrations, input validation
- internal/mcp/server_test.go — added 22 tests for generation tools (including review fix tests)
- docs/stories/sprint-status.yaml — status update

