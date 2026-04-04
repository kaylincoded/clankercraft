# Story 3.8: WorldEdit Undo/Redo

Status: done

## Story

As a user,
I want the bot to undo and redo its WorldEdit operations,
so that mistakes can be reversed.

## Acceptance Criteria

1. **Given** the bot has executed a WorldEdit operation
   **When** `//undo` is dispatched
   **Then** the last operation is reversed

2. **Given** the bot has undone an operation
   **When** `//redo` is dispatched
   **Then** the undone operation is reapplied

3. **Given** tier is Vanilla
   **When** undo or redo is requested
   **Then** a clear error is returned indicating WorldEdit is required

4. **Given** the bot is disconnected
   **When** undo or redo is called
   **Then** a connection error is returned

## Tasks / Subtasks

- [x] Task 1: Register MCP tools (AC: #1, #2, #3, #4)
  - [x] `we-undo` — input: {} (no params), sends `//undo`, no selection needed (`requireWETier`)
  - [x] `we-redo` — input: {} (no params), sends `//redo`, no selection needed (`requireWETier`)
- [x] Task 2: Write tests (all ACs)
  - [x] Test we-undo sends `undo`
  - [x] Test we-redo sends `redo`
  - [x] Test both work without selection (uses requireWETier)
  - [x] Test both reject vanilla tier
  - [x] Test both reject disconnected
  - [x] Existing tests still pass

## Dev Notes

### Command Syntax

```
//undo    — reverses the last WorldEdit operation
//redo    — reapplies the last undone operation
```

Both are **clipboard/history-based** — they operate on the bot's WorldEdit session history, not on a selection. Use `requireWETier` middleware (same as paste/rotate/flip from Story 3.7).

### Implementation Pattern

Follow `handleWECopy` / `handleWENaturalize` — empty input struct, handler sends command via `RunWECommand`, returns `weCommandOutput`. This is the simplest tool pattern in the codebase.

### Reusing Infrastructure

- `RunWECommand` from Story 3.4 handles command dispatch
- `requireWETier` for both tools (no selection needed)
- `weCommandOutput` shared output type
- `toolError` for error responses

### References

- [Source: docs/epics.md#Story 3.8] — Original story definition
- [Source: internal/mcp/server.go#handleWECopy] — No-param WE tool example
- [Source: internal/mcp/middleware.go#requireWETier] — Tier-only middleware

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Added 2 MCP tools: we-undo, we-redo (31 total tools)
- Both use requireWETier middleware (history-based, no selection needed)
- Simplest tool pattern — no-param input, single command dispatch

### File List
- internal/mcp/server.go — added 2 undo/redo tool types, handlers, registrations
- internal/mcp/server_test.go — added 7 tests for undo/redo tools

