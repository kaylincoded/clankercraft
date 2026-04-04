---
stepsCompleted: ['step-01-validate', 'step-02-design', 'step-03-stories', 'step-04-validation']
inputDocuments:
  - docs/prd.md
  - docs/architecture-decision.md
---

# Clankercraft v3 — Epic Breakdown

## Overview

This document decomposes the PRD's 46 functional requirements and 21 non-functional requirements into implementable epics and stories, organized by the phased development roadmap.

## Requirements Inventory

### Functional Requirements

- FR1-FR5: Connection & Authentication
- FR6-FR11: In-Game Chat Interface
- FR12-FR15: MCP Stdio Interface
- FR16-FR25: WorldEdit Construction
- FR26-FR29: Vanilla Fallback Construction
- FR30-FR32: RCON Operations
- FR33-FR39: World Observation
- FR40-FR43: LLM Integration
- FR44-FR46: Configuration & Distribution

### Non-Functional Requirements

- NFR1-NFR6: Performance
- NFR7-NFR12: Reliability
- NFR13-NFR17: Integration
- NFR18-NFR21: Security

## FR Coverage Map

| FR | Epic | Story |
|---|---|---|
| FR1-FR3 | Epic 1 | 1.1, 1.2, 1.3 |
| FR4-FR5 | Epic 1 | 1.4, 1.5 |
| FR44-FR45 | Epic 1 | 1.6, 1.7 |
| FR12-FR15 | Epic 2 | 2.1, 2.2, 2.3 |
| FR33-FR39 | Epic 2 | 2.4, 2.5, 2.6, 2.7 |
| FR16-FR17 | Epic 3 | 3.1, 3.2 |
| FR18 | Epic 3 | 3.3 |
| FR19-FR22 | Epic 3 | 3.4, 3.5, 3.6 |
| FR23-FR25 | Epic 3 | 3.7, 3.8 |
| FR26-FR29 | Epic 3 | 3.9 |
| FR30-FR32 | Epic 4 | 4.1, 4.2 |
| FR6-FR7 | Epic 5 | 5.1, 5.2 |
| FR40-FR43 | Epic 5 | 5.3, 5.4 |
| FR8-FR11 | Epic 5 | 5.5, 5.6 |
| FR46 | Epic 6 | 6.1 |
| FR9-FR10 | Epic 5 | 5.5 |

## Epic List

| # | Epic | Phase | Dependencies | Stories |
|---|---|---|---|---|
| 1 | Foundation — Connection, Config, Binary | MVP | None | 7 |
| 2 | MCP Stdio Interface & Observation Tools | MVP | Epic 1 | 7 |
| 3 | Construction Engine & WorldEdit Integration | MVP | Epic 1 | 9 |
| 4 | RCON Channel | MVP | Epic 1, 3 | 2 |
| 5 | In-Game Chat Interface & LLM Integration | MVP | Epic 1, 3 | 6 |
| 6 | Distribution & Release | MVP | Epic 1-5 | 3 |

---

## Epic 1: Foundation — Connection, Config, Binary

**Goal:** Establish the Go project, Minecraft connection, authentication, configuration system, and binary build pipeline. This is the spike gate — validates day 1 that go-mc works for our use case.

### Story 1.1: Project Initialization & CLI Scaffolding

As a developer,
I want a Go module with cobra/viper CLI scaffolding,
So that I have a working binary that parses flags and loads config.

**Acceptance Criteria:**

**Given** the binary is invoked with `--host`, `--port`, `--username` flags
**When** it starts
**Then** it parses all flags, merges with env vars and config file (`~/.config/clankercraft/config.yaml`), and logs the resolved configuration to stderr

**Given** no flags or config are provided
**When** the binary starts
**Then** it uses defaults: host=localhost, port=25565, username=LLMBot

### Story 1.2: Minecraft Connection via go-mc

As a developer,
I want the bot to connect to a Minecraft Java Edition server using go-mc,
So that we have a live protocol connection to build on.

**Acceptance Criteria:**

**Given** a running Minecraft 1.21.x server in offline mode
**When** the binary is started with `--offline --host localhost --port 25565`
**Then** the bot connects, joins the server, and appears in the player list

**Given** the bot is connected
**When** the server sends a chat message
**Then** the bot receives and logs the message to stderr

### Story 1.3: MSA Authentication

As a user,
I want the bot to authenticate with my Microsoft account,
So that it can join online-mode servers.

**Acceptance Criteria:**

**Given** no cached auth token exists
**When** the binary starts without `--offline`
**Then** it initiates MSA device code flow, displays the code/URL, and waits for user to authenticate

**Given** a valid cached token exists at `~/.config/clankercraft/tokens/`
**When** the binary starts
**Then** it uses the cached token without prompting

**Given** the cached token is expired
**When** the binary starts
**Then** it refreshes the token automatically

### Story 1.4: Connection State Machine & Auto-Reconnect

As a user,
I want the bot to automatically reconnect after disconnection,
So that building sessions survive server hiccups.

**Acceptance Criteria:**

**Given** the bot is connected and the server restarts
**When** the disconnection is detected
**Then** the bot transitions to `disconnected`, waits 1s, and attempts reconnection with exponential backoff (1s, 2s, 4s, 8s, 16s, cap 30s)

**Given** the bot has failed to reconnect 5 times
**When** the 5th attempt fails
**Then** it logs the failure and stays in `disconnected` state without crashing

### Story 1.5: Graceful Shutdown

As a user,
I want the bot to clean up on SIGINT/SIGTERM,
So that it doesn't leave ghost sessions on the server.

**Acceptance Criteria:**

**Given** the bot is connected
**When** SIGINT is received
**Then** the bot disconnects cleanly from the MC server, closes RCON if open, and exits with code 0

### Story 1.6: Configuration Layering

As a server admin,
I want to configure the bot via config file, env vars, and CLI flags with clear priority,
So that I can deploy it flexibly.

**Acceptance Criteria:**

**Given** `config.yaml` sets `host: server1` and CLI flag sets `--host server2`
**When** config is resolved
**Then** CLI flag wins: host = server2

**Given** `CLANKERCRAFT_RCON_PASSWORD=secret` is set and config.yaml has `rcon_password: other`
**When** config is resolved
**Then** env var wins: rcon_password = secret

### Story 1.7: Structured Logging

As a developer,
I want structured JSON logging to stderr with configurable level,
So that logs don't corrupt the MCP stdout stream and are parseable.

**Acceptance Criteria:**

**Given** the binary is running
**When** any component logs a message
**Then** it appears on stderr as a JSON line with timestamp, level, message, and structured fields

**Given** `--log-level debug` is set
**When** debug messages are logged
**Then** they appear in output

**Given** `--log-level info` is set (default)
**When** debug messages are logged
**Then** they are suppressed

---

## Epic 2: MCP Stdio Interface & Observation Tools

**Goal:** Implement the MCP JSON-RPC stdio transport and observation tools, achieving basic tool parity with v2 for Claude Code/Desktop integration.

### Story 2.1: MCP Stdio Transport

As an MCP client (Claude Code/Desktop),
I want to communicate with clankercraft via JSON-RPC over stdin/stdout,
So that it works as a standard MCP tool server.

**Acceptance Criteria:**

**Given** the binary is started in MCP mode
**When** a valid JSON-RPC tool call arrives on stdin
**Then** the server parses it, dispatches to the registered tool, and writes the JSON-RPC response to stdout

**Given** stdout receives non-JSON-RPC output (from go-mc logging, etc.)
**When** the MCP transport is active
**Then** non-JSON-RPC output is filtered and redirected to stderr

### Story 2.2: MCP Tool Registration Framework

As a developer,
I want a tool registration pattern with connection-check middleware and argument validation,
So that all tools share consistent error handling.

**Acceptance Criteria:**

**Given** a tool is registered with a JSON schema
**When** a tool call arrives with invalid arguments
**Then** a structured error response is returned without crashing

**Given** a tool is called while the bot is disconnected
**When** the connection check runs
**Then** it attempts reconnection. If reconnect succeeds, the tool executes. If it fails, an error response is returned.

### Story 2.3: Tool Listing & Discovery

As an MCP client,
I want to discover available tools via the MCP `tools/list` method,
So that the LLM knows what tools are available.

**Acceptance Criteria:**

**Given** the MCP server is running
**When** a `tools/list` request arrives
**Then** all registered tools are returned with their names, descriptions, and parameter schemas

### Story 2.4: Position & Movement Tools

As an MCP client,
I want `get-position`, `move-to-position`, `look-at`, `jump` tools,
So that the bot can navigate and report its location.

**Acceptance Criteria:**

**Given** the bot is connected and spawned
**When** `get-position` is called
**Then** it returns the bot's current x, y, z coordinates and facing direction

**Given** valid coordinates
**When** `move-to-position` is called
**Then** the bot moves to the target (or within `range`) and reports success/failure

### Story 2.5: Block Query Tools

As an MCP client,
I want `get-block-info` and `find-block` tools,
So that the bot can inspect the world.

**Acceptance Criteria:**

**Given** valid coordinates
**When** `get-block-info` is called
**Then** it returns the block type, state properties, and metadata at that position

**Given** a block type and max distance
**When** `find-block` is called
**Then** it returns the coordinates of the nearest matching block, or a not-found message

### Story 2.6: Area Scanning Tool

As an MCP client,
I want `scan-area` to scan a rectangular region and return block data,
So that the bot can understand terrain before building.

**Acceptance Criteria:**

**Given** two corner coordinates defining a region ≤ 10,000 blocks
**When** `scan-area` is called
**Then** it returns all non-air blocks with their types and positions

**Given** a region > 10,000 blocks
**When** `scan-area` is called
**Then** it returns an error with the actual block count and the 10K limit

### Story 2.7: Sign & Game State Tools

As an MCP client,
I want `read-sign`, `find-signs`, and `detect-gamemode` tools,
So that the bot can read environmental information.

**Acceptance Criteria:**

**Given** a sign at the specified coordinates
**When** `read-sign` is called
**Then** it returns front and back text of the sign

**Given** a max distance
**When** `find-signs` is called
**Then** it returns all signs within range (max 50) with their text and positions

---

## Epic 3: Construction Engine & WorldEdit Integration

**Goal:** Build the shared construction engine that composes and executes WorldEdit commands, with vanilla fallback. This is the core value — translating tool calls into world modifications.

### Story 3.1: WorldEdit Capability Tier Detection

As a system,
I want to detect whether the server has FAWE, WorldEdit, or neither on connection,
So that the construction engine knows which commands are available.

**Acceptance Criteria:**

**Given** the bot connects to a server with FAWE installed
**When** capability detection runs
**Then** tier is set to FAWE and logged

**Given** the bot connects to a server with WorldEdit (no FAWE)
**When** capability detection runs
**Then** tier is set to WorldEdit and logged

**Given** the bot connects to a vanilla server
**When** capability detection runs
**Then** tier is set to Vanilla and logged

**Given** the bot reconnects after disconnection
**When** capability detection runs again
**Then** the tier is re-detected (not cached from previous session)

### Story 3.2: WorldEdit Selection Management

As a construction engine,
I want to set WorldEdit positions (`//pos1`, `//pos2`) and manage selections,
So that region operations have targets.

**Acceptance Criteria:**

**Given** coordinates (x1, y1, z1) and (x2, y2, z2)
**When** a selection is set
**Then** the bot sends `//pos1 x1,y1,z1` and `//pos2 x2,y2,z2` via chat

**Given** a selection is set
**When** queried
**Then** the engine tracks the current selection coordinates in memory

### Story 3.3: Wand Selection Sharing

As a builder (Kaylin),
I want the bot to read my WorldEdit wand selections,
So that I can point at things and say "modify this."

**Acceptance Criteria:**

**Given** a player makes a wand selection (left-click pos1, right-click pos2)
**When** WorldEdit outputs the coordinate feedback in chat
**Then** the bot parses the coordinates and stores them as the player's current selection

**Given** a player asks the bot to "modify this area"
**When** the bot processes the request
**Then** it uses the player's most recent wand selection coordinates

### Story 3.4: WorldEdit Region Operations

As a construction engine,
I want to execute `//set`, `//replace`, `//walls`, `//faces`, `//hollow`,
So that the bot can fill and modify selected regions.

**Acceptance Criteria:**

**Given** a selection is set and tier is WorldEdit or FAWE
**When** `//set stone_bricks` is dispatched
**Then** the region is filled with stone bricks and the engine reports the block count

**Given** a `//replace` command with source and target patterns
**When** dispatched
**Then** matching blocks in the selection are replaced

### Story 3.5: WorldEdit Generation Commands

As a construction engine,
I want to execute `//generate`, `//sphere`, `//cyl`, `//pyramid`,
So that the bot can create geometric and mathematical shapes.

**Acceptance Criteria:**

**Given** tier is WorldEdit or FAWE
**When** `//sphere stone 10` is dispatched at the bot's position
**Then** a stone sphere of radius 10 is generated

**Given** tier is WorldEdit or FAWE
**When** `//generate` with a mathematical expression is dispatched
**Then** the expression is evaluated and blocks are placed accordingly

### Story 3.6: WorldEdit Terrain Operations & Patterns

As a construction engine,
I want to execute `//smooth`, `//naturalize`, `//overlay` and use WorldEdit pattern syntax,
So that the bot can create organic, natural-looking terrain and mixed materials.

**Acceptance Criteria:**

**Given** a selection over terrain
**When** `//smooth 5` is dispatched
**Then** the terrain is smoothed with 5 iterations

**Given** a pattern like `50%stone,30%cobblestone,20%mossy_stone_bricks`
**When** used in a `//set` command
**Then** the region is filled with the specified material distribution

### Story 3.7: WorldEdit Clipboard Operations

As a construction engine,
I want `//copy`, `//paste`, `//rotate`, `//flip`,
So that the bot can duplicate and transform structures.

**Acceptance Criteria:**

**Given** a selection containing a structure
**When** `//copy` then `//paste` is executed at a new location
**Then** the structure is duplicated at the target position

**Given** a copied structure
**When** `//rotate 90` then `//paste` is executed
**Then** the structure is pasted rotated 90 degrees

### Story 3.8: WorldEdit Undo/Redo

As a user,
I want the bot to undo and redo its WorldEdit operations,
So that mistakes can be reversed.

**Acceptance Criteria:**

**Given** the bot has executed a WorldEdit operation
**When** `//undo` is dispatched
**Then** the last operation is reversed

**Given** the bot has undone an operation
**When** `//redo` is dispatched
**Then** the undone operation is reapplied

### Story 3.9: Vanilla Fallback Construction

As a construction engine,
I want to use `/fill`, `/setblock`, `/clone` when WorldEdit is unavailable,
So that the bot works on vanilla servers.

**Acceptance Criteria:**

**Given** tier is Vanilla
**When** a region fill is requested
**Then** it's decomposed into `/fill` commands respecting the 32,768 block limit per command

**Given** a single block placement is requested
**When** tier is Vanilla
**Then** `/setblock x y z block_type` is dispatched

**Given** a clone operation is requested
**When** tier is Vanilla
**Then** `/clone` is used with appropriate source/destination coordinates

---

## Epic 4: RCON Channel

**Goal:** Add RCON as a secondary command dispatch channel for bulk operations that bypass chat rate limiting.

### Story 4.1: RCON Client Connection

As a system,
I want to connect to the Minecraft server's RCON port,
So that I can send commands directly to the server console.

**Acceptance Criteria:**

**Given** RCON is configured (`--rcon-port`, `--rcon-password`)
**When** the bot starts
**Then** it establishes an RCON connection and logs success

**Given** RCON is not configured
**When** the bot starts
**Then** RCON is marked as unavailable and the engine routes all commands through chat

**Given** RCON connection fails
**When** the bot starts
**Then** it logs a warning and falls back to chat-only mode (does not crash)

### Story 4.2: Command Routing (Chat vs RCON)

As a construction engine,
I want to route commands to chat or RCON based on operation type and availability,
So that bulk operations aren't throttled by chat rate limiting.

**Acceptance Criteria:**

**Given** a bulk operation (large `//set`, `//replace`, schematic paste) and RCON is available
**When** the command is dispatched
**Then** it goes through RCON

**Given** a player-session operation (wand selection, `//undo`) and RCON is available
**When** the command is dispatched
**Then** it goes through chat (because RCON runs as console, not as the bot player)

**Given** any operation and RCON is unavailable
**When** the command is dispatched
**Then** it goes through chat

---

## Epic 5: In-Game Chat Interface & LLM Integration

**Goal:** Implement the autonomous in-game agent — whisper listener, LLM integration, conversational building. This is the "magic" interface that differentiates v3.

### Story 5.1: Whisper Listener

As the bot,
I want to detect and parse whispered messages (`/msg`) from players,
So that players can talk to me in-game.

**Acceptance Criteria:**

**Given** a player sends `/msg Builder build me a house`
**When** the bot receives the message
**Then** it identifies the sender and message content, and routes to the LLM

**Given** a public chat message (not a whisper)
**When** the bot receives it
**Then** it does not treat it as a command (ignores non-whisper messages)

### Story 5.2: Chat Response System

As the bot,
I want to send chat responses back to players (whisper or public),
So that players see my replies in-game.

**Acceptance Criteria:**

**Given** the LLM generates a response for a player
**When** the response is ready
**Then** the bot sends it as a whisper to the requesting player

**Given** a response exceeds 256 characters
**When** formatting for chat
**Then** it's split across multiple messages with appropriate delays

### Story 5.3: LLM Provider Interface & Claude Implementation

As a developer,
I want a pluggable LLM provider interface with a Claude implementation,
So that the chat interface can use Claude API for natural language understanding.

**Acceptance Criteria:**

**Given** `CLAUDE_API_KEY` is set
**When** the ClaudeProvider is initialized
**Then** it can send messages to the Claude API with tool definitions and receive tool-call responses

**Given** the LLM provider interface
**When** a new provider is implemented (e.g., OpenAI)
**Then** it can be swapped in without changing chat or engine code

### Story 5.4: Tool-Calling Agent Loop

As a system,
I want the LLM to receive construction tools and execute multi-step builds,
So that a single "build me a cathedral" request triggers a sequence of WorldEdit commands.

**Acceptance Criteria:**

**Given** a player says "build me a stone wall 20 blocks long"
**When** the message is sent to the LLM with available tools
**Then** the LLM returns tool calls (e.g., set selection, `//set`), the system executes them, feeds results back, and the LLM continues until done

**Given** a tool call fails (e.g., WorldEdit error)
**When** the error is fed back to the LLM
**Then** the LLM adapts (retries with different parameters, reports the issue to the player, or tries an alternative approach)

### Story 5.5: Conversation Context & Natural Language Building

As a builder,
I want the bot to maintain conversation context and understand iterative requests,
So that I can say "make it taller" without repeating the full context.

**Acceptance Criteria:**

**Given** the bot built a wall and the player says "make it taller"
**When** the LLM receives the message with conversation history
**Then** it understands "it" refers to the wall and modifies accordingly

**Given** a conversation has been ongoing for 20+ messages
**When** a new message arrives
**Then** the full conversation context is available to the LLM (within token limits)

### Story 5.6: Bot Summoning & Teleportation

As a player,
I want to summon the builder to my location with a chat command,
So that it comes to where I'm building.

**Acceptance Criteria:**

**Given** a player whispers "come here" or "summon" (or similar intent)
**When** the LLM interprets the message
**Then** the bot teleports to the player's location (via `/tp` or movement)

---

## Epic 6: Distribution & Release

**Goal:** Package the binary for cross-platform distribution and ensure a frictionless install experience.

### Story 6.1: Goreleaser Configuration

As a developer,
I want goreleaser to build binaries for 6 targets on git tag,
So that users can download pre-built binaries for their platform.

**Acceptance Criteria:**

**Given** a git tag is pushed (e.g., `v3.0.0`)
**When** the GitHub Actions release workflow runs
**Then** goreleaser produces binaries for linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64, windows-arm64 with checksums

### Story 6.2: CI Pipeline

As a developer,
I want CI to lint, test, and build on every push and PR,
So that regressions are caught early.

**Acceptance Criteria:**

**Given** a push to main or a PR
**When** CI runs
**Then** it executes: `go vet`, `golangci-lint`, `go test ./...`, `go build`

### Story 6.3: Schematic Directory Loading

As a user,
I want the bot to load schematics from `~/.config/clankercraft/schematics/` on startup,
So that I can use saved building components.

**Acceptance Criteria:**

**Given** `.schem` files exist in the schematics directory
**When** the bot starts
**Then** it indexes available schematics and makes them available as a tool

**Given** the schematics directory doesn't exist
**When** the bot starts
**Then** it creates the directory and continues with no schematics loaded

---

## Cross-Epic NFR Mapping

| NFR | Addressed In |
|---|---|
| NFR1 (3s chat response) | Epic 5 — chat responder |
| NFR2 (100ms command dispatch) | Epic 3 — engine dispatch |
| NFR3 (1s scan-area) | Epic 2 — observation tools |
| NFR4 (200ms MCP round-trip) | Epic 2 — MCP transport |
| NFR5 (2s startup) | Epic 1 — binary scaffolding |
| NFR6 (<256MB memory) | Cross-cutting — all epics |
| NFR7 (1hr+ sessions) | Epic 1 — state machine, reconnect |
| NFR8 (10s reconnect) | Epic 1 — auto-reconnect |
| NFR9 (context preservation) | Epic 5 — conversation session |
| NFR10 (WE error handling) | Epic 3 — engine error wrapping |
| NFR11 (RCON degradation) | Epic 4 — fallback routing |
| NFR12 (LLM degradation) | Epic 5 — error handling in agent loop |
| NFR13 (MC protocol) | Epic 1 — go-mc connection |
| NFR14 (RCON protocol) | Epic 4 — RCON client |
| NFR15 (MCP protocol) | Epic 2 — MCP transport |
| NFR16 (LLM streaming) | Epic 5 — Claude provider |
| NFR17 (WE detection on connect) | Epic 3 — tier detection |
| NFR18 (RCON password security) | Epic 4 — RCON client logging |
| NFR19 (API key security) | Epic 1 — config, Epic 5 — LLM provider |
| NFR20 (token security) | Epic 1 — auth |
| NFR21 (command authorization) | Epic 5 — LLM agent loop (tools constrain actions) |

## Implementation Order

```
Epic 1 (Foundation) ──► Epic 2 (MCP + Observation) ──► Epic 6 (Distribution)
         │                                                      ▲
         ├──► Epic 3 (Construction Engine) ──► Epic 4 (RCON) ──┤
         │                                         │            │
         └──► Epic 5 (Chat + LLM) ◄───────────────┘────────────┘
```

**Critical path:** Epic 1 → Epic 3 → Epic 5 → Epic 6 (the demo video path)
**Parallel work:** Epic 2 (MCP) can proceed alongside Epic 3 after Epic 1 completes
