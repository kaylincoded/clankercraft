# Epic 2 Retrospective: MCP Stdio Interface & Observation Tools

## Goal

Implement the MCP JSON-RPC stdio transport and observation tools, achieving basic tool parity with v2 for Claude Code/Desktop integration.

## Stories Completed

| Story | Title | Notes |
|-------|-------|-------|
| 2-1 | MCP Stdio Transport | Official Go SDK replaced custom JSON-RPC plan |
| 2-2 | MCP Tool Registration Framework | Generic `requireConnection[I,O]` middleware pattern |
| 2-3 | Tool Listing & Discovery | Zero-code — SDK handles tools/list natively |
| 2-4 | Position & Movement Tools | get-position, look-at; movement deferred to 5-6 |
| 2-5 | Block Query Tools | get-block-info, find-block (loaded chunks only) |
| 2-6 | Area Scanning Tool | scan-area with 10K block volume cap |
| 2-7 | Sign & Game State Tools | read-sign, find-signs, detect-gamemode |

## What Went Well

- **Official MCP Go SDK** was a major win. Replaced the architecture doc's "custom JSON-RPC" approach with `gomcp.AddTool` + typed handlers. Eliminated an entire category of transport/serialization bugs.
- **Generic middleware pattern** (`requireConnection[I, O]`) scaled cleanly across all 9 connection-gated tools with zero duplication.
- **Incremental interface expansion** worked well. `ConnChecker` → `BotState` with each story adding methods. Mock stayed manageable with function fields.
- **go-mc chunk architecture** gave us block queries, sign reading, and area scanning without any custom packet handling — just reading the world the library already maintains.
- **First live connection** succeeded at end of epic. MSA auth → Paper 1.21.11 server. Bot spawned and was visible in-game.

## What Could Be Improved

- **go-mc version lag** was a blocker at test time. The library doesn't track Minecraft releases closely. We had to switch to the mj41 fork for 1.21.11 support. This dependency is a risk going forward — if the fork goes stale, we're stuck.
- **MSA auth library logging** bypasses our logger, printing directly to stderr. The `go-mc-ms-auth` library uses its own log calls. Not critical but messy in terminal output.
- **No live integration tests.** All tests mock the connection layer. We verified tool behavior against mocks but only did manual testing against a real server at the very end. Consider a test server fixture for Epic 3+.
- **Story 2-3 was zero-code.** The SDK handled tool listing natively. In hindsight, this story could have been folded into 2-1 or 2-2 during planning.
- **Movement was descoped from 2-4.** Pathfinding and move-to-position were deferred to 5-6 (teleport-based summoning). The right call for a building bot, but worth noting the scope change.

## Key Decisions

1. **MCP Go SDK over custom JSON-RPC** — Reduced transport code to near zero. Trade-off: coupled to SDK's handler signature conventions.
2. **No pathfinding** — Bot is a builder, not an explorer. Movement via teleportation in Story 5-6.
3. **Silent operations** — Raw packets over chat commands. No visible chat spam for other players.
4. **BotState interface over direct Connection access** — Keeps MCP package testable without real Minecraft connections.
5. **mj41/go-mc fork** — Pragmatic choice for 1.21.11 support. Replace directive keeps our import paths clean if upstream merges the PRs.

## Metrics

- **10 MCP tools registered:** ping, status, get-position, look-at, get-block-info, find-block, scan-area, read-sign, find-signs, detect-gamemode
- **85 tests** across config, connection, log, and MCP packages
- **7 stories** completed (1 zero-code)
- **0 bugs found in production** (limited live testing, but clean first connection)

## Carry-Forward Items

- Monitor mj41/go-mc fork — upstream PRs #294-296 are open. If merged, switch back to upstream.
- MSA auth library log noise — consider wrapping or redirecting in a future story.
- Live integration test infrastructure — would catch protocol-level issues earlier.
