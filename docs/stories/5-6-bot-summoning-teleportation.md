# Story 5.6: Bot Summoning & Teleportation

Status: done

## Story

As a player,
I want to summon the builder to my location with a chat command,
so that it comes to where I'm building.

## Acceptance Criteria

1. **Given** a player whispers "come here" or "summon" (or similar intent)
   **When** the LLM interprets the message
   **Then** the bot teleports to the player's location (via `/tp` or movement)

## Tasks / Subtasks

- [x] Task 1: Add `teleport-to-player` tool to agent (AC: #1)
  - [x] Add tool definition in `tooldefs.go`: `teleport-to-player` with required `player` (string) parameter
  - [x] Add Execute case in `tools.go`: call `te.bot.RunBulkCommand(fmt.Sprintf("tp @s %s", player))` with connection check
  - [x] Sanitize player name input: reject names containing spaces, slashes, or special characters (prevent command injection)
  - [x] Return JSON result with teleport command response
  - [x] Update `buildToolDefs` comment: tool count 34 → 35
- [x] Task 2: Update system prompt for summoning awareness (AC: #1)
  - [x] Add guidance to `DefaultSystemPrompt` in `agent.go` explaining:
    - When a player says "come here", "summon", "teleport to me", or similar, use `teleport-to-player` with the player's name
    - The player name is available from the conversation context (it's the sender)
    - After teleporting, confirm arrival and ask what to build
  - [x] Keep prompt concise — add 2-3 lines max
- [x] Task 3: Pass player name as context to LLM (AC: #1)
  - [x] In `HandleMessage`, prepend the player name to the user message as context: format as `[{player}]: {message}` so the LLM knows who is speaking
  - [x] This allows the LLM to pass the correct player name to `teleport-to-player`
  - [x] Update existing tests that check exact message content to account for the new format
- [x] Task 4: Write tests (AC: #1)
  - [x] Test `teleport-to-player` tool execution with connected bot — verify `RunBulkCommand` called with `tp @s <player>`
  - [x] Test `teleport-to-player` with disconnected bot — returns error
  - [x] Test `teleport-to-player` with invalid player name (spaces, special chars) — returns validation error
  - [x] Test tool count is now 35
  - [x] Test `HandleMessage` includes player name prefix in messages sent to LLM
  - [x] All existing tests still pass

## Dev Notes

### Design Decision: LLM-Interpreted Summoning

The AC says "the LLM interprets the message" — this is NOT a hardcoded command like the reset commands in Story 5.5. The LLM decides whether the player's intent is to summon the bot, then calls the `teleport-to-player` tool.

This means:
- No `isSummonCommand()` function — the LLM handles intent detection
- The system prompt guides the LLM to recognize summoning intent
- The LLM can handle varied phrasing: "come here", "tp to me", "get over here", etc.

### Teleport Mechanism

The bot uses `/tp @s <player>` via `RunBulkCommand` (prefers RCON if available, falls back to chat-based command). The Connection already auto-accepts server teleport packets (`mc.go:240-242` — `Teleported` callback calls `AcceptTeleportation`), so the bot's tracked position updates automatically.

`RunBulkCommand` adds the leading `/` — pass `tp @s <player>` not `/tp @s <player>`.

### Player Name Context

Currently `HandleMessage` receives the player name but only uses it for conversation keying. The LLM never sees WHO is talking. To call `teleport-to-player`, the LLM needs to know the player's name.

Solution: prefix messages with `[PlayerName]: message` before sending to the LLM. This is a minimal change in `HandleMessage` at `agent.go:123`:

```go
// Current:
messages = append(messages, llm.Message{Role: llm.RoleUser, Content: message})

// New:
messages = append(messages, llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf("[%s]: %s", player, message)})
```

### Command Injection Prevention

Player names come from Minecraft whispers (server-validated), but the name is interpolated into a command string. Validate:
- Only alphanumeric + underscore (Minecraft username rules: 3-16 chars, `[a-zA-Z0-9_]`)
- Reject anything else before building the command string

```go
func isValidPlayerName(name string) bool {
    if len(name) < 3 || len(name) > 16 {
        return false
    }
    for _, c := range name {
        if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
            return false
        }
    }
    return true
}
```

### What NOT to Build

- **No player position lookup tool** — `/tp @s <player>` teleports directly to the player by name; no need to resolve coordinates first
- **No pathfinding or walking** — teleport is instantaneous (confirmed in Epic 2 retro: "Movement via teleportation in Story 5-6")
- **No MCP server changes** — this tool is agent-only (LLM tool, not MCP-exposed)

### Previous Story Intelligence

- Story 5.5: `HandleMessage` signature is `(ctx, player, message, sendReply)` — player name already available
- Story 5.5: Conversation history uses `Snapshot`/`Append` — the `[player]: message` prefix will be stored in history, which is fine (LLM sees who said what)
- Story 5.5: `isResetCommand` checks raw message before player prefix is added — move the prefix addition AFTER the reset check
- Story 5.4: `mockBot` in `tools_test.go` implements `BotState` including `RunBulkCommand` — can verify the tp command
- Story 5.4: Tool count test `TestToolDefsCount` checks for 34 — must update to 35

### References

- [Source: internal/agent/agent.go#L20-L31] — DefaultSystemPrompt (needs summoning guidance)
- [Source: internal/agent/agent.go#L106-L123] — HandleMessage (needs player name prefix)
- [Source: internal/agent/tools.go#L34] — Execute switch-case (add teleport-to-player)
- [Source: internal/agent/tooldefs.go#L10-L139] — buildToolDefs (add tool def, update count)
- [Source: internal/agent/tools_test.go] — mockBot, TestToolDefsCount (update to 35)
- [Source: internal/connection/mc.go#L988-L997] — RunBulkCommand (RCON-preferred command execution)
- [Source: internal/connection/mc.go#L240-L242] — Teleported callback (auto-accepts server teleports)
- [Source: internal/mcp/middleware.go#L18-L39] — BotState interface (RunBulkCommand already available)
- [Source: docs/stories/epic-2-retrospective.md#L38] — "No pathfinding — Movement via teleportation in Story 5-6"
- [Source: docs/epics.md#Story 5.6] — Original story definition
- [Source: docs/prd.md#FR11] — "Bot can teleport to the requesting player's location when summoned"
- [Source: docs/prd.md#FR29] — "Bot can teleport via /tp command"

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Added `teleport-to-player` tool (tool #35) using `RunBulkCommand("tp @s <player>")` — prefers RCON, falls back to chat
- `isValidPlayerName` validates Minecraft username rules (`[a-zA-Z0-9_]{3,16}`) to prevent command injection
- System prompt updated with 2-line summoning guidance referencing the `[username]:` prefix
- `HandleMessage` now prefixes user messages as `[PlayerName]: message` — stored in conversation history so LLM always knows who's speaking
- Reset command check remains BEFORE the prefix (operates on raw message)
- 37 tests in agent package (9 conversation + 12 agent + 16 tools), full suite green

### File List
- `internal/agent/tools.go` — MODIFIED: Added `teleport-to-player` Execute case and `isValidPlayerName` validation
- `internal/agent/tooldefs.go` — MODIFIED: Added teleport-to-player tool definition, updated count 34→35
- `internal/agent/agent.go` — MODIFIED: Added player name prefix in HandleMessage, summoning guidance in DefaultSystemPrompt
- `internal/agent/agent_test.go` — MODIFIED: Added TestHandleMessagePlayerNamePrefix, updated content assertions for `[Player]:` format
- `internal/agent/tools_test.go` — MODIFIED: Added 3 teleport tests (connected, disconnected, invalid names), updated tool count to 35
- `docs/stories/sprint-status.yaml` — MODIFIED: Updated 5-6 status
