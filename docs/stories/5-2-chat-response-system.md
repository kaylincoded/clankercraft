# Story 5.2: Chat Response System

Status: done

## Story

As the bot,
I want to send chat responses back to players (whisper or public),
so that players see my replies in-game.

## Acceptance Criteria

1. **Given** the bot needs to respond to a player
   **When** it calls `SendWhisper(player, message)`
   **Then** the bot sends `/msg <player> <message>` as a chat command, and the player sees the whisper in-game

2. **Given** a response exceeds 256 characters (Minecraft chat limit)
   **When** formatting for chat
   **Then** it's split across multiple messages at word boundaries with appropriate delays between sends

3. **Given** the chat response system
   **When** a caller sends a message
   **Then** it respects Minecraft's chat rate limit (~1 message per 200ms) to avoid being kicked for spam

## Tasks / Subtasks

- [x] Task 1: Add SendWhisper to chat package (AC: #1)
  - [x] Create `Sender` struct in `internal/chat/chat.go` with `sendCommandFn func(string) error`
  - [x] `NewSender(sendCommandFn func(string) error) *Sender` constructor
  - [x] `SendWhisper(player, message string) error` — calls `sendCommandFn("msg " + player + " " + message)`
  - [x] Validate player name is non-empty, message is non-empty
- [x] Task 2: Message splitting for long messages (AC: #2)
  - [x] `splitMessage(text string, maxLen int) []string` — pure function
  - [x] Split at word boundaries (space); if a single word exceeds maxLen, hard-split it
  - [x] Minecraft chat command limit: 256 chars total for `/msg Player ...`, so maxLen = 256 - len("msg ") - len(player) - len(" ") = `251 - len(player)`
  - [x] Each chunk becomes a separate `/msg` command
- [x] Task 3: Rate-limited sending (AC: #3)
  - [x] `SendWhisper` sends multi-part messages with a configurable delay between each (default 200ms)
  - [x] Use `time.Sleep` between sends (simple; no need for a token bucket yet)
  - [x] Expose delay as a field on Sender for testability: `MessageDelay time.Duration`
- [x] Task 4: Wire Sender into Connection and BotState (AC: #1, #2, #3)
  - [x] Add `chatSender *chatpkg.Sender` field to Connection, initialized in `New()`
  - [x] `Connection.SendWhisper(player, message string) error` delegates to `chatSender.SendWhisper`
  - [x] Add `SendWhisper(player, message string) error` to BotState interface
  - [x] Sender wired via closure to `c.SendCommand()` in New() — works because SendCommand handles nil msgMgr
- [x] Task 5: Write tests (all ACs)
  - [x] Test `splitMessage` splits at word boundaries
  - [x] Test `splitMessage` handles single word exceeding maxLen (hard split)
  - [x] Test `splitMessage` message under limit returns single chunk
  - [x] Test `splitMessage` accounts for player name length in limit
  - [x] Test `SendWhisper` sends correct `/msg player message` command
  - [x] Test `SendWhisper` splits long messages into multiple commands
  - [x] Test `SendWhisper` validates empty player/message
  - [x] Test rate limiting: multiple sends have delay between them
  - [x] All existing tests still pass

## Dev Notes

### How to Send Whispers in Minecraft

The bot sends whispers using the `/msg` command:
```
/msg PlayerName your response here
```

This is sent via `Connection.SendCommand("msg PlayerName your response here")`. The `SendCommand` method uses `ServerboundChatCommand` which adds an implicit `/` prefix — so we pass `"msg ..."` not `"/msg ..."`.

### Minecraft Chat Limits

- **Max command length:** 256 characters (enforced by `ServerboundChatCommand` packet)
- **Rate limit:** Servers typically kick for spam if >20 messages in a short window. Safe rate: ~1 message per 200ms.
- The 256 char limit includes the full command: `msg PlayerName message_content`
- Player names are max 16 chars, so worst case: `msg ` (4) + name (16) + ` ` (1) = 21 chars overhead, leaving 235 chars for message content

### Message Splitting Strategy

For a message that exceeds the per-command content limit:
1. Calculate `maxContentLen = 256 - len("msg ") - len(player) - 1` (the -1 is for the space before content)
2. Split `message` into chunks of at most `maxContentLen` characters, preferring word boundaries
3. Send each chunk as a separate `/msg player chunk` command with a delay between

### Sender Design

The `Sender` lives in `internal/chat/` alongside the `Listener`. It takes a `sendCommandFn` to avoid importing the connection package (same decoupling pattern as `Listener`).

```go
type Sender struct {
    sendCommandFn func(string) error
    MessageDelay  time.Duration // default 200ms, configurable for tests
}
```

### Wiring in Connection

The `Sender` needs `Connection.SendCommand` as its `sendCommandFn`. Since `SendCommand` requires `msgMgr` to be initialized (which happens in `Connect()`), the sender should be created lazily or wired during `Connect()`. Simplest approach: create Sender in `New()` with a closure that calls `c.SendCommand()` — since `SendCommand` already handles the nil-mgr case with an error.

```go
// In New():
c.chatSender = chatpkg.NewSender(func(cmd string) error {
    return c.SendCommand(cmd)
})
```

### Existing Patterns

- Story 5.1 established `internal/chat/` package with `Listener` — `Sender` follows the same pattern
- `Connection.OnWhisper` wraps `chatpkg.Listener` — `Connection.SendWhisper` wraps `chatpkg.Sender`
- `BotState` interface grows with `SendWhisper` — same pattern as `RunBulkWECommand` addition in Story 4.2

### References

- [Source: internal/connection/mc.go:926-943] — SendCommand method (uses ServerboundChatCommand)
- [Source: internal/chat/chat.go] — Existing chat package with Listener, Whisper types
- [Source: github.com/Tnze/go-mc/bot/msg/chat.go:181] — SendCommand packet, 256 char limit
- [Source: github.com/Tnze/go-mc/bot/msg/chat.go:156] — SendMessage (regular chat), 256 char limit
- [Source: docs/epics.md#Story 5.2] — Original story definition
- [Source: docs/stories/5-1-whisper-listener.md] — Previous story, chat package patterns

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- Sender struct added to internal/chat alongside existing Listener — same decoupling pattern
- splitMessage pure function splits at word boundaries, hard-splits single long words
- SendWhisper computes maxContent = 256 - len("msg " + player + " "), splits, sends with delay
- DefaultMessageDelay = 200ms, configurable via Sender.MessageDelay for tests
- Connection.New() wires Sender via closure to SendCommand (no lazy init needed)
- 11 new tests: 4 splitMessage + 6 SendWhisper + 1 command error propagation

### File List
- internal/chat/chat.go — Added Sender, NewSender, SendWhisper, splitMessage, lastSpace, DefaultMessageDelay
- internal/chat/chat_test.go — 11 new tests for Sender and splitMessage
- internal/connection/mc.go — Added chatSender field, initialized in New(), SendWhisper method
- internal/mcp/middleware.go — Added SendWhisper to BotState interface
- internal/mcp/middleware_test.go — Added SendWhisper to mockBotState

