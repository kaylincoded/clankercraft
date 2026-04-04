# Story 2.7: Sign & Game State Tools

Status: done

## Story

As an MCP client,
I want `read-sign`, `find-signs`, and `detect-gamemode` tools,
so that the bot can read environmental information and game state.

## Acceptance Criteria

1. **Given** a sign at the specified coordinates
   **When** `read-sign` is called
   **Then** it returns front and back text of the sign

2. **Given** a max distance
   **When** `find-signs` is called
   **Then** it returns all signs within range (max 50) with their text and positions

3. **Given** the block at the specified coordinates is not a sign
   **When** `read-sign` is called
   **Then** it returns an error indicating no sign at that position

4. **Given** the bot is connected
   **When** `detect-gamemode` is called
   **Then** it returns the current game mode (survival, creative, adventure, spectator)

5. **Given** the bot is disconnected
   **When** any of these tools are called
   **Then** a connection error is returned (via requireConnection middleware)

## Tasks / Subtasks

- [x] Task 1: Add sign reading methods to Connection (AC: #1, #2, #3)
  - [x] Add `SignText` struct with FrontLines/BackLines
  - [x] Add `ReadSign(x, y, z int) (SignText, error)` — reads block entity NBT at coords
  - [x] Add `FindSigns(maxDist int) ([]SignInfo, error)` — iterates loaded chunks for sign entities
  - [x] Define `SignInfo` struct (SignText + position)
- [x] Task 2: Add gamemode method to Connection (AC: #4)
  - [x] Add `GetGamemode() string` — returns "survival"/"creative"/"adventure"/"spectator"
- [x] Task 3: Expand BotState interface (AC: #1, #2, #4)
  - [x] Add `ReadSign`, `FindSigns`, `GetGamemode` to BotState
- [x] Task 4: Register MCP tools (AC: #1, #2, #3, #4, #5)
  - [x] `read-sign` — input: {x, y, z}, output: {front, back, block, message}
  - [x] `find-signs` — input: {maxDistance?}, output: {signs: [...], count, message}
  - [x] `detect-gamemode` — input: {}, output: {gamemode}
  - [x] All wrapped with requireConnection middleware
- [x] Task 5: Write tests (all ACs)
  - [x] Test read-sign returns sign text
  - [x] Test read-sign errors on non-sign block
  - [x] Test find-signs returns signs within range
  - [x] Test find-signs returns empty when none found
  - [x] Test detect-gamemode returns mode string
  - [x] Test all tools error when disconnected
  - [x] Existing tests still pass (85 total: 8 MCP-level + 6 unit tests for parseSignMessages/GetGamemode)

## Dev Notes

### Sign NBT Structure (1.20+)

```nbt
{
  front_text: {
    messages: ['{"text":"line1"}', '{"text":"line2"}', '{"text":""}', '{"text":""}'],
    color: "black",
    has_glowing_text: 0b
  },
  back_text: { ... same structure ... },
  is_waxed: 0b
}
```

Messages are JSON text components — parsed via `chat.Message` and `ClearString()`.

### Block Entity Access

`level.Chunk.BlockEntity` stores `[]level.BlockEntity` with:
- `XZ` (packed byte), `Y` (int16), `Type` (block.EntityType), `Data` (nbt.RawMessage)
- Sign type: `block.EntityTypes["minecraft:sign"]` (index 7)
- Hanging sign: `block.EntityTypes["minecraft:hanging_sign"]` (index 8)
- `UnpackXZ()` returns (X, Z int) within 0-15 chunk range

### Gamemode

`player.Gamemode` byte: 0=Survival, 1=Creative, 2=Adventure, 3=Spectator

### References

- [Source: docs/epics.md#Story 2.7] — Original story definition
- [Source: internal/connection/mc.go#FindBlock] — Chunk iteration pattern reused for FindSigns
- [Source: level/block/blockentities.go] — Sign entity type definitions

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- `ReadSign` looks up chunk block entities by matching sign/hanging_sign entity types and coordinates, then unmarshals NBT into `signNBT` struct. Returns block name alongside text (single chunk lookup, no redundant BlockAt call)
- `parseSignMessages` uses `chat.Message` to parse JSON text components from sign lines. Falls back to plain JSON string unmarshal. Returns empty string on unparseable data (never exposes raw JSON)
- `FindSigns` iterates loaded chunks within distance, caps at 50 results, **sorted nearest first** by Euclidean distance
- `GetGamemode` reads `player.Gamemode` byte and maps to string via `gamemodeNames` array
- Both sign entity types handled: `minecraft:sign` and `minecraft:hanging_sign`
- 85 total tests (8 MCP-level + 6 unit tests: 5 for parseSignMessages, 1 for GetGamemode)

### File List

- `internal/connection/mc.go` (modified — SignText, SignInfo, ReadSign, FindSigns, GetGamemode, parseSignMessages, distance-sorted results)
- `internal/connection/mc_test.go` (modified — 6 new unit tests for parseSignMessages and GetGamemode)
- `internal/mcp/middleware.go` (modified — BotState interface expanded with ReadSign, FindSigns, GetGamemode)
- `internal/mcp/middleware_test.go` (modified — mockBotState expanded with readSignFn, findSignsFn, gamemode)
- `internal/mcp/server.go` (modified — read-sign, find-signs, detect-gamemode tool types, handlers, registration)
- `internal/mcp/server_test.go` (modified — 8 new MCP-level tests)
- `docs/stories/2-7-sign-game-state-tools.md` (new)
