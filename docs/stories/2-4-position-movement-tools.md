# Story 2.4: Position & Look-At Tools

Status: done

## Story

As an MCP client,
I want `get-position` and `look-at` tools,
so that the bot can report its location and face a target.

## Acceptance Criteria

1. **Given** the bot is connected and has received a position from the server
   **When** `get-position` is called
   **Then** it returns the bot's current x, y, z coordinates and yaw/pitch facing direction

2. **Given** the bot is connected
   **When** the server sends a `ClientboundPlayerPosition` packet (teleport/spawn)
   **Then** the bot's tracked position and rotation update accordingly

3. **Given** valid target coordinates
   **When** `look-at` is called
   **Then** the bot rotates to face the target and sends the rotation packet to the server

4. **Given** the bot is disconnected
   **When** `get-position` or `look-at` is called
   **Then** a connection error is returned (via requireConnection middleware)

5. **Given** the bot has not yet received a position from the server
   **When** `get-position` is called
   **Then** an error is returned indicating position is not yet known

## Tasks / Subtasks

- [x] Task 1: Add position tracking to Connection (AC: #1, #2)
  - [x] Add `Position` struct with X, Y, Z (float64) and Yaw, Pitch (float32) fields to connection package
  - [x] Store position atomically on Connection (mutex-protected)
  - [x] Capture position from `Teleported` event in `Connect()` → `basic.EventsListener`
  - [x] Call `AcceptTeleportation` to acknowledge server position updates
  - [x] Add `GetPosition() (Position, bool)` method — returns position and whether it's been set
  - [x] Reset position on disconnect (stale data from previous session)
- [x] Task 2: Add `SendRotation` method to Connection (AC: #3)
  - [x] Send `ServerboundMovePlayerRot` packet via `client.Conn.WritePacket(pk.Marshal(...))`
  - [x] Update tracked yaw/pitch after sending
- [x] Task 3: Expand ConnChecker interface for position tools (AC: #1, #3, #4)
  - [x] Add `GetPosition() (Position, bool)` to a new interface (or expand ConnChecker)
  - [x] Add `SendRotation(yaw, pitch float32) error` to the interface
  - [x] Keep ConnChecker minimal — consider a broader `BotState` interface for tools that need more than IsConnected
- [x] Task 4: Register `get-position` MCP tool (AC: #1, #5)
  - [x] Input: none
  - [x] Output: `{x, y, z, yaw, pitch}` — floored integers for x/y/z (matches v2 behavior)
  - [x] Wrap with requireConnection middleware
  - [x] Return error if position not yet known
- [x] Task 5: Register `look-at` MCP tool (AC: #3)
  - [x] Input: `{x, y, z}` (float64 coordinates)
  - [x] Calculate yaw and pitch from bot's current position to target
  - [x] Send rotation packet via Connection.SendRotation
  - [x] Wrap with requireConnection middleware
- [x] Task 6: Write tests (all ACs)
  - [x] Test position tracking updates from Teleported event
  - [x] Test position resets on disconnect
  - [x] Test GetPosition returns false before first position update
  - [x] Test yaw/pitch calculation for look-at (unit test the math)
  - [x] Test get-position tool returns position via MCP transport
  - [x] Test look-at tool sends rotation via MCP transport
  - [x] Test both tools return error when disconnected
  - [x] Existing tests still pass (50 total)

## Dev Notes

### Position Tracking

go-mc doesn't track position — we must capture it ourselves. The `Teleported` event fires on spawn and server-initiated teleports:

```go
Teleported: func(x, y, z float64, yaw, pitch float32, flags byte, teleportID int32) error {
    c.setPosition(x, y, z, yaw, pitch, flags)
    return c.player.AcceptTeleportation(pk.VarInt(teleportID))
},
```

The `flags` byte is a bitfield — if bit N is set, that coordinate is relative to current position rather than absolute. Bits: 0=X, 1=Y, 2=Z, 3=Yaw, 4=Pitch. Handle both absolute and relative updates.

```go
type Position struct {
    X, Y, Z    float64
    Yaw, Pitch float32
}
```

### Yaw/Pitch Calculation for look-at

Minecraft rotation conventions:
- **Yaw:** degrees, 0 = +Z (south), 90 = -X (west), 180 = -Z (north), 270 = +X (east)
- **Pitch:** degrees, -90 = up, 0 = horizontal, 90 = down

```go
func calcYawPitch(from, to Position) (yaw float32, pitch float32) {
    dx := to.X - from.X
    dy := to.Y - from.Y
    dz := to.Z - from.Z
    dist := math.Sqrt(dx*dx + dz*dz)
    yaw = float32(-math.Atan2(dx, dz) * 180 / math.Pi)
    pitch = float32(-math.Atan2(dy, dist) * 180 / math.Pi)
    return yaw, pitch
}
```

### Sending Rotation Packet

```go
func (c *Connection) SendRotation(yaw, pitch float32) error {
    if c.client == nil || c.client.Conn == nil {
        return fmt.Errorf("not connected")
    }
    return c.client.Conn.WritePacket(pk.Marshal(
        packetid.ServerboundMovePlayerRot,
        pk.Float(yaw),
        pk.Float(pitch),
        pk.Boolean(true), // onGround
    ))
}
```

### Interface Design

The ConnChecker interface is intentionally minimal (just `IsConnected()`). For position tools, we need more. Options:

**Option A — Expand ConnChecker:**
```go
type ConnChecker interface {
    IsConnected() bool
    GetPosition() (Position, bool)
    SendRotation(yaw, pitch float32) error
}
```

**Option B — Separate interface per capability:**
```go
type BotState interface {
    ConnChecker
    GetPosition() (Position, bool)
    SendRotation(yaw, pitch float32) error
}
```

Prefer Option B — keeps ConnChecker stable for tools that only need connection check, while BotState extends it for tools needing game state access. The Server struct takes BotState.

### v2 Behavior Reference

From `src/tools/position-tools.ts`:
- `get-position` returns floored integer coordinates: `Math.floor(position.x)`
- `look-at` uses `bot.lookAt(new Vec3(x, y, z), true)` — force=true means instant rotation
- `move-to-position` used pathfinder — **not implementing** (raw position packets or `/tp` for summoning in Story 5.6)
- `jump` used `setControlState` — **not implementing** (no physics engine)

### Scope Reduction from Epic Definition

The epic defined `get-position`, `move-to-position`, `look-at`, `jump`. This story implements only:
- `get-position` — position tracking + MCP tool
- `look-at` — rotation calculation + packet sending + MCP tool

**Deferred:**
- `move-to-position` → Story 5.6 (bot summoning via teleport, 2 blocks in front of player)
- `jump` → not needed for a building bot without physics simulation

### Package Boundary

- `Position` struct defined in `internal/connection/` (it's connection state)
- `BotState` interface defined in `internal/mcp/` (it's the mcp package's view of what it needs)
- `*Connection` satisfies `BotState` implicitly

### Previous Story Learnings

- requireConnection middleware wraps tools with connection check (Story 2.2)
- toolError() for consistent MCP error responses (Story 2.2)
- InMemoryTransports for integration tests (Story 2.1)
- Injectable fields on Connection for testability (Stories 1.3-1.5)

### What This Story Does NOT Do

- Pathfinding or walking movement (future epic if needed)
- Jump (no physics engine)
- Bot summoning/teleportation (Story 5.6)
- Block query tools (Story 2.5)
- Any construction/WorldEdit tools (Epic 3)

### Project Structure After This Story

```
internal/
├── connection/
│   ├── mc.go            (modified — Position struct, tracking, SendRotation)
│   ├── mc_test.go       (modified — position tracking tests)
│   └── auth.go          (unchanged)
└── mcp/
    ├── server.go        (modified — BotState interface, get-position + look-at tools)
    ├── server_test.go   (modified — new tool integration tests)
    ├── middleware.go     (unchanged or minor interface update)
    ├── middleware_test.go (unchanged)
    └── errors.go        (unchanged)
```

### References

- [Source: docs/epics.md#Story 2.4] — Original story definition
- [Source: src/tools/position-tools.ts] — v2 mineflayer implementation
- [Source: internal/connection/mc.go] — Connection struct, event handlers
- [Source: internal/mcp/middleware.go] — ConnChecker interface, requireConnection
- [External: go-mc bot/basic/events.go] — Teleported event, AcceptTeleportation
- [External: go-mc data/packetid] — ServerboundMovePlayerRot packet ID
- [External: wiki.vg/Protocol#Synchronize_Player_Position] — Position flags bitfield

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- Position tracked via `Teleported` event in `basic.EventsListener` — handles both absolute and relative flag bits
- `AcceptTeleportation` called on every position update to acknowledge server
- Position reset on disconnect to prevent stale data from previous session
- `BotState` interface extends `ConnChecker` — keeps middleware stable, adds position/rotation methods
- `mcp` package now imports `connection` for `Position` type only (data struct, not behavior dependency)
- `get-position` returns floored integers matching v2 behavior (`Math.floor`)
- `look-at` calculates yaw/pitch via `atan2` — Minecraft convention: yaw 0=south, pitch -90=up
- `calcYawPitch` is a package-level function (not method) for easy unit testing
- `jsonschema` tag format is bare description text, NOT `key=value` (SDK panics on `description=...`)
- `SendRotation` acquires mutex to read client, sends packet, then re-acquires to update tracked rotation
- 62 total tests across all packages (12 new: 5 connection position + 7 MCP tool integration)

### File List

- `internal/connection/mc.go` (modified — Position struct, updatePosition, resetPosition, GetPosition, SendRotation)
- `internal/connection/mc_test.go` (modified — 5 new position tracking tests)
- `internal/mcp/middleware.go` (modified — BotState interface, connection.Position import)
- `internal/mcp/middleware_test.go` (modified — mockBotState replaces mockConnChecker)
- `internal/mcp/server.go` (modified — get-position + look-at tools, calcYawPitch, BotState parameter)
- `internal/mcp/server_test.go` (modified — 7 new tests: position, look-at, yaw/pitch math)
