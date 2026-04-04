# Story 2.6: Area Scanning Tool

Status: done

## Story

As an MCP client,
I want `scan-area` to scan a rectangular region and return block data,
so that the bot can understand terrain before building.

## Acceptance Criteria

1. **Given** two corner coordinates defining a region ≤ 10,000 blocks
   **When** `scan-area` is called
   **Then** it returns all non-air blocks with their types and positions

2. **Given** a region > 10,000 blocks
   **When** `scan-area` is called
   **Then** it returns an error with the actual block count and the 10K limit

3. **Given** the bot is disconnected
   **When** `scan-area` is called
   **Then** a connection error is returned (via requireConnection middleware)

## Tasks / Subtasks

- [x] Task 1: Add `ScanArea` method to Connection (AC: #1, #2)
  - [x] Accept two corner coordinates (x1,y1,z1) and (x2,y2,z2)
  - [x] Validate region size ≤ 10,000 blocks before scanning
  - [x] Iterate every position in the box, call BlockAt for each
  - [x] Skip air blocks, collect non-air blocks with position and type
  - [x] Return collected blocks as a slice of {block, x, y, z}
- [x] Task 2: Expand BotState interface (AC: #1)
  - [x] Add `ScanArea(x1,y1,z1,x2,y2,z2 int) ([]BlockInfo, error)` to BotState
  - [x] Define BlockInfo struct in mcp package (or connection package)
- [x] Task 3: Register `scan-area` MCP tool (AC: #1, #2, #3)
  - [x] Input: `{x1, y1, z1, x2, y2, z2}`
  - [x] Output: `{blocks: [{block, x, y, z}...], count, totalScanned}`
  - [x] Wrap with requireConnection middleware
- [x] Task 4: Write tests (all ACs)
  - [x] Test scan-area returns non-air blocks
  - [x] Test scan-area rejects region > 10,000 blocks
  - [x] Test scan-area returns error when disconnected
  - [x] Existing tests still pass (68 total)

## Dev Notes

### BlockInfo Type

```go
type BlockInfo struct {
    Block string
    X, Y, Z int
}
```

Define in `internal/connection/` since it's connection-level data returned by ScanArea.

### Size Validation

```go
dx := abs(x2-x1) + 1
dy := abs(y2-y1) + 1
dz := abs(z2-z1) + 1
volume := dx * dy * dz
if volume > MaxScanVolume {
    return nil, fmt.Errorf("region too large: %d blocks (max %d)", volume, MaxScanVolume)
}
```

### ScanArea Implementation

Reuses BlockAt internally. For chunks that aren't loaded, skip those blocks (don't error — partial results are better than failing entirely for a large scan).

### References

- [Source: docs/epics.md#Story 2.6] — Original story definition
- [Source: internal/connection/mc.go#BlockAt] — Block lookup method from Story 2.5

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- `ScanArea` normalizes corners (swaps if x1>x2 etc.), validates volume ≤ 10,000, then iterates calling `BlockAt` per position
- Skips air (minecraft:air, cave_air, void_air) and unloaded chunks (BlockAt errors silently skipped)
- `BlockInfo` struct defined in connection package — reusable by future tools
- MCP tool converts `[]connection.BlockInfo` to `[]scanAreaBlock` (typed output for JSON schema)
- 71 total tests (3 new scan-area integration tests)

### File List

- `internal/connection/mc.go` (modified — BlockInfo struct, MaxScanVolume, ScanArea method)
- `internal/mcp/middleware.go` (modified — BotState interface expanded with ScanArea)
- `internal/mcp/middleware_test.go` (modified — mockBotState expanded with scanAreaFn)
- `internal/mcp/server.go` (modified — scan-area tool types, handler, registration)
- `internal/mcp/server_test.go` (modified — 3 new scan-area tests)
- `docs/stories/2-6-area-scanning-tool.md` (new)
