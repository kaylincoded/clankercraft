# Story 3.3: Wand Selection Sharing

Status: done

## Story

As a builder (Kaylin),
I want the bot to read my WorldEdit wand selections,
so that I can point at things and say "modify this."

## Acceptance Criteria

1. **Given** a player makes a wand selection (left-click pos1)
   **When** WorldEdit outputs "First position set to (x, y, z)" in chat
   **Then** the bot parses the coordinates and updates pos1 of the current selection

2. **Given** a player makes a wand selection (right-click pos2)
   **When** WorldEdit outputs "Second position set to (x, y, z)" in chat
   **Then** the bot parses the coordinates and updates pos2 of the current selection

3. **Given** both positions have been set via wand
   **When** the selection is queried
   **Then** the bot returns both corners from the stored selection

4. **Given** the tier is Vanilla (no WorldEdit)
   **When** no wand messages appear in chat
   **Then** the wand listener does nothing (no crash, no error)

5. **Given** the bot is disconnected
   **When** the bot reconnects
   **Then** the wand listener restarts and prior selection is cleared

## Tasks / Subtasks

- [x] Task 1: Add wand chat parsing functions (AC: #1, #2)
  - [x] Define `parseWandPos1(msg string) (x, y, z int, ok bool)` using regex
  - [x] Define `parseWandPos2(msg string) (x, y, z int, ok bool)` using regex
  - [x] Compile regexes at package level (not per-call)
- [x] Task 2: Implement persistent wand listener (AC: #1, #2, #3, #4, #5)
  - [x] `startWandListener()` — registers chat listener, loops parsing pos1/pos2
  - [x] Runs as goroutine from GameStart alongside detectTier
  - [x] Exits on disconnect via `wandDone` channel (nil guard for edge cases)
  - [x] Updates selection fields individually per position
  - [x] Resets wand state on disconnect (close wandDone, resetSelection)
- [x] Task 3: Update Selection tracking for partial state (AC: #1, #2, #3)
  - [x] Replaced `hasSelection` with `hasPos1` and `hasPos2` on Connection
  - [x] `GetSelection()` returns true only when both flags set
  - [x] `SetSelection()` sets both flags at once
  - [x] Added `HasPos1()` and `HasPos2()` methods
- [x] Task 4: Extend get-selection MCP tool (AC: #3)
  - [x] Extended `get-selection` to report partial state (pos1-only, pos2-only, both, none)
  - [x] Added `HasPos1`/`HasPos2` to BotState interface
- [x] Task 5: Write tests (all ACs)
  - [x] Test parseWandPos1 valid (4 cases) and invalid (4 cases)
  - [x] Test parseWandPos2 valid (3 cases) and invalid (3 cases)
  - [x] Test wand listener updates selection with partial → full state
  - [x] Test wand listener stops on done channel
  - [x] Test full disconnect clears wand state and stops listener
  - [x] Test SetSelection sets both pos flags
  - [x] Test MCP get-selection partial pos1-only state
  - [x] Existing tests still pass (114 total, 9 new)

## Dev Notes

### WorldEdit Wand Chat Messages

WorldEdit 7.4.0 sends these as **SystemChat** packets (not PlayerChatMessage). The existing `dispatchChat` infrastructure already receives them. After `ClearString()`, the text is clean:

```
First position set to (100, 64, -200).
Second position set to (110, 70, -190) (7260).
First position set to (100, 64, -200) (7260).
```

The `(VOLUME)` suffix is optional — it appears only when both positions are set. Regex should not anchor the end.

```go
var wandPos1Re = regexp.MustCompile(`First position set to \((-?\d+), (-?\d+), (-?\d+)\)`)
var wandPos2Re = regexp.MustCompile(`Second position set to \((-?\d+), (-?\d+), (-?\d+)\)`)
```

### Persistent Listener Pattern

Unlike `detectTier()` which is one-shot (send command, wait for response, exit), the wand listener is **long-lived** — it runs for the entire connection session. Use the existing `listenChat()`/`unlistenChat()` infrastructure but with a goroutine that loops until a done signal:

```go
func (c *Connection) startWandListener() {
    ch := c.listenChat()
    defer c.unlistenChat(ch)

    for {
        select {
        case msg := <-ch:
            // parse pos1/pos2
        case <-c.wandDone:
            return
        }
    }
}
```

Create a `wandDone chan struct{}` on Connection, initialized in GameStart, closed on disconnect. This cleanly signals the goroutine to exit.

### Partial Selection State

The current `hasSelection` bool is insufficient — wand selections arrive one corner at a time. Add `hasPos1` and `hasPos2` bools:

- Wand pos1 → sets X1/Y1/Z1, hasPos1=true
- Wand pos2 → sets X2/Y2/Z2, hasPos2=true
- hasSelection = hasPos1 && hasPos2
- `SetSelection()` (programmatic, from Story 3.2) sets both flags at once

### Interface Expansion

BotState does NOT need expansion for this story if we extend `get-selection` to report partial state. If we add a separate `get-wand-selection` tool, no interface change is needed since `GetSelection()` already exists.

Recommended: extend `get-selection` to return partial state info in its message field (e.g., "pos1 set, waiting for pos2") rather than adding a new tool.

### References

- [Source: docs/epics.md#Story 3.3] — Original story definition
- [Source: internal/connection/mc.go#dispatchChat] — Chat listener infrastructure from Story 3.1
- [Source: internal/connection/mc.go#detectTier] — One-shot listener pattern to adapt
- [Source: internal/connection/mc.go#SetSelection] — Programmatic selection from Story 3.2
- [Source: internal/engine/selection.go] — Selection type

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- Package-level compiled regexes `wandPos1Re` and `wandPos2Re` for parsing WorldEdit wand feedback
- `parseWandPos1`/`parseWandPos2` extract integer coords from SystemChat messages
- `startWandListener()` is a persistent goroutine (unlike one-shot `detectTier`), exits via `wandDone` channel
- Replaced single `hasSelection` bool with `hasPos1`/`hasPos2` — wand selections arrive one at a time
- `GetSelection()` returns true only when both positions are set
- `SetSelection()` (programmatic) sets both flags simultaneously
- `get-selection` MCP tool now reports partial state: "pos1 set, waiting for pos2" etc.
- Nil guard on `wandDone` prevents goroutine leak if Disconnect fires before listener starts
- 114 total tests (9 new: 4 parse pos1, 3 parse pos2, 3 wand listener/disconnect, 1 MCP partial, minus shared helpers)

### File List

- `internal/connection/mc.go` (modified — wand regexes, parseWandPos1/Pos2, startWandListener, wandDone, hasPos1/hasPos2, GameStart/Disconnect wiring)
- `internal/connection/mc_test.go` (modified — 9 new tests for wand parsing, listener, disconnect)
- `internal/mcp/middleware.go` (modified — HasPos1/HasPos2 on BotState)
- `internal/mcp/middleware_test.go` (modified — hasPos1/hasPos2 mock fields and methods)
- `internal/mcp/server.go` (modified — handleGetSelection reports partial state)
- `internal/mcp/server_test.go` (modified — partial selection test, hasSelection→hasPos1/hasPos2)

