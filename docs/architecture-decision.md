---
stepsCompleted: ['step-01-init', 'step-02-context', 'step-03-template', 'step-04-core', 'step-05-patterns', 'step-06-structure', 'step-07-validation', 'step-08-complete']
inputDocuments:
  - docs/prd.md
  - docs/architecture.md
  - docs/api-contracts.md
  - docs/mineflayer-reference.md
workflowType: 'architecture'
project_name: 'Clankercraft v3'
user_name: 'Kaylin'
date: '2026-04-03'
---

# Architecture Decision Document — Clankercraft v3

## Project Context Analysis

### Requirements Overview

**46 Functional Requirements** across 8 capability areas:
- Connection & Authentication (FR1-FR5)
- In-Game Chat Interface (FR6-FR11)
- MCP Stdio Interface (FR12-FR15)
- WorldEdit Construction (FR16-FR25)
- Vanilla Fallback Construction (FR26-FR29)
- RCON Operations (FR30-FR32)
- World Observation (FR33-FR39)
- LLM Integration (FR40-FR43)
- Configuration & Distribution (FR44-FR46)

**21 Non-Functional Requirements** across 4 categories:
- Performance (NFR1-NFR6): sub-3s chat response, sub-100ms command dispatch, sub-2s startup, <256MB memory
- Reliability (NFR7-NFR12): 1hr+ sessions, auto-reconnect, graceful degradation
- Integration (NFR13-NFR17): MC protocol, RCON, MCP stdio, LLM streaming, WE capability detection
- Security (NFR18-NFR21): credential protection, token security, command authorization

### Scale & Complexity Assessment

- **Complexity:** Medium-High (brownfield rewrite with scope expansion)
- **Estimated MVP LOC:** ~2,500 Go lines
- **Concurrent users:** Single bot instance, multi-player interaction (one bot, many whisperers)
- **External integrations:** Minecraft server (protocol), RCON, Claude API, WorldEdit (via chat/RCON)
- **No database** — all state is in-memory or file-based (schematics, config)

### Cross-Cutting Concerns

| Concern | Approach |
|---|---|
| Connection lifecycle | State machine (disconnected → connecting → connected), auto-reconnect with backoff |
| Error handling | Graceful degradation at every tier — FAWE → WE → vanilla, RCON → chat-only, LLM failure → user notification |
| Logging | Structured JSON to stderr (preserves MCP stdout stream) |
| Configuration | Layered: CLI flags > env vars > config file > defaults |
| Testing | Unit tests with mocked MC protocol, integration tests against real server in CI |

## Technical Stack

### Language & Runtime

| Choice | Decision | Rationale |
|---|---|---|
| **Language** | Go 1.22+ | Single binary output, excellent concurrency (goroutines for dual interface), existing world scanner proof-of-concept |
| **Build** | `go build` + `goreleaser` | Cross-compile 6 targets, checksums, GitHub Releases |
| **MC Protocol** | `go-mc` | Proven in world scanner (15M blocks/sec chunk parsing), MSA auth support |
| **MCP SDK** | Custom stdio JSON-RPC | Go MCP SDK ecosystem is immature — stdio transport is simple enough to implement directly |
| **LLM Client** | Anthropic Go SDK | First-party Claude integration, tool use support |
| **RCON** | `go-mc/net/rcon` or equivalent | Same library family as protocol connection |
| **Config** | `viper` | Layered config (CLI + env + file + defaults), YAML config file support |
| **CLI** | `cobra` | Standard Go CLI framework, pairs with viper |

### No Starter Template

This is a Go binary, not a web app. No starter template — initialize with `go mod init github.com/kaylincoded/clankercraft` and build from scratch.

## Core Architectural Decisions

### 1. Dual Interface Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Clankercraft v3                       │
│                                                         │
│  ┌──────────────┐          ┌──────────────────────┐     │
│  │ MCP Stdio    │          │ In-Game Chat          │     │
│  │ Interface    │          │ Interface             │     │
│  │              │          │                       │     │
│  │ JSON-RPC     │          │ Whisper Listener      │     │
│  │ stdin/stdout │          │ → LLM API (tools)     │     │
│  └──────┬───────┘          │ → Command Execution   │     │
│         │                  │ → Chat Response       │     │
│         │                  └──────────┬────────────┘     │
│         │                             │                  │
│         └──────────┬──────────────────┘                  │
│                    │                                     │
│         ┌──────────▼──────────────┐                      │
│         │ Construction Engine     │                      │
│         │                        │                      │
│         │ - WorldEdit Composer   │                      │
│         │ - Vanilla Fallback     │                      │
│         │ - Capability Tier Mgr  │                      │
│         │ - Command Dispatcher   │                      │
│         └──────────┬─────────────┘                      │
│                    │                                     │
│         ┌──────────▼──────────────┐                      │
│         │ Connection Layer        │                      │
│         │                        │                      │
│         │ - MC Protocol (go-mc)  │                      │
│         │ - RCON Client          │                      │
│         │ - Chat I/O             │                      │
│         │ - State Machine        │                      │
│         └─────────────────────────┘                      │
└─────────────────────────────────────────────────────────┘
```

**Decision:** Both interfaces share a single Construction Engine. The MCP interface receives tool calls and maps them to engine methods. The Chat interface receives natural language, calls the LLM API with the same tools, and executes the LLM's tool calls through the same engine.

**Why:** Prevents feature drift between interfaces. Any construction capability added to MCP is automatically available in chat mode and vice versa.

### 2. WorldEdit 3-Tier Capability Detection

**Decision:** Detect WorldEdit tier on connection by sending probe commands:
1. Send `//version` — if response mentions FAWE → Tier 1
2. If response mentions WorldEdit → Tier 2
3. No response or error → Tier 3 (vanilla only)

Cache tier for session lifetime. Re-detect on reconnect.

**Why:** The construction engine adapts its command vocabulary based on available capabilities. A `build_wall` operation uses `//set` with patterns on Tier 1/2, falls back to chunked `/fill` commands on Tier 3.

### 3. Dual Command Channels (Chat + RCON)

**Decision:** Two command dispatch paths:
- **Chat channel:** Bot sends commands as a player (e.g., `//set stone`). Has its own WorldEdit session, selection, history. Rate limited by server.
- **RCON channel:** Direct server console. No rate limits. Used for bulk operations. Commands run as console, not as a player — different permission context.

**Routing logic:**
- Operations requiring bot's WE session (selections, wand, undo) → Chat
- Bulk operations (large set/replace, schematic paste) → RCON (if available)
- RCON unavailable → fall back to chat for everything

**Why:** Chat rate limiting (typically 1 msg/tick = 20/sec) throttles large builds. RCON bypasses this. But RCON runs as console, not player — it can't use the bot's WE selection. So both channels are needed.

### 4. LLM Provider Interface

```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error)
    StreamChat(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamEvent, error)
}
```

**Decision:** Pluggable LLM backend. MVP: `ClaudeProvider` using Anthropic API. The MCP path doesn't use this interface — the MCP client handles its own LLM. Only the in-game chat interface uses `LLMProvider`.

**Why:** Community can contribute `OpenAIProvider`, `OllamaProvider` etc. without touching construction logic.

### 5. Connection State Machine

```
States: disconnected, connecting, connected
Transitions:
  disconnected → connecting (on connect() or auto-reconnect trigger)
  connecting → connected (on successful spawn)
  connecting → disconnected (on auth failure, timeout, max retries)
  connected → disconnected (on kick, error, server shutdown)
```

**Decision:** Exponential backoff reconnect: 1s, 2s, 4s, 8s, 16s, cap at 30s. Max 5 attempts. Preserve conversation context across reconnects (chat history stays in memory).

**Why:** Content creators need stable 1hr+ sessions. Server hiccups shouldn't lose the building conversation.

### 6. Configuration Layering

```
Priority (highest first):
1. CLI flags (--host, --port, --rcon-password, etc.)
2. Environment variables (CLANKERCRAFT_HOST, CLAUDE_API_KEY)
3. Config file (~/.config/clankercraft/config.yaml)
4. Defaults
```

**Decision:** Use `viper` for layered config. `cobra` for CLI parsing. Config file is optional — binary works with just CLI flags.

**Why:** Different deployment contexts need different config methods. Server admins use config files. Docker uses env vars. Quick testing uses CLI flags.

### 7. Authentication

**Decision:**
- **Minecraft:** MSA (Microsoft) auth via `go-mc`. Device code flow for first-time. Token caching in `~/.config/clankercraft/tokens/`. `--offline` flag for dev/cracked servers.
- **LLM API:** API key via `CLAUDE_API_KEY` env var or config file. Never in CLI args (visible in process list).
- **RCON:** Password via `--rcon-password` flag, `CLANKERCRAFT_RCON_PASSWORD` env var, or config file. Never logged.

### 8. No Database

**Decision:** All state is in-memory or filesystem:
- Conversation history: in-memory (per session)
- Schematics: filesystem (`~/.config/clankercraft/schematics/`)
- Config: filesystem (`~/.config/clankercraft/config.yaml`)
- Auth tokens: filesystem (`~/.config/clankercraft/tokens/`)

**Why:** A Minecraft building bot doesn't need persistence across sessions. The Minecraft world *is* the persistence layer.

## Implementation Patterns & Consistency Rules

### Naming Conventions

| Category | Convention | Example |
|---|---|---|
| Go packages | lowercase, single word | `engine`, `mcp`, `chat`, `rcon` |
| Go files | lowercase with hyphens | `world-edit.go`, `capability-tier.go` |
| Go types | PascalCase | `ConstructionEngine`, `LLMProvider` |
| Go functions | PascalCase (exported), camelCase (unexported) | `PlaceBlock()`, `detectTier()` |
| MCP tool names | kebab-case | `place-block`, `scan-area`, `get-position` |
| Config keys | snake_case in YAML, SCREAMING_SNAKE for env | `rcon_password`, `CLANKERCRAFT_RCON_PASSWORD` |
| CLI flags | kebab-case | `--rcon-password`, `--max-radius` |
| Log fields | snake_case | `block_count`, `command_latency_ms` |

### Error Handling Pattern

```go
// Construction engine errors are always wrapped with context
// and returned to the interface layer (MCP or chat) for formatting.
// Never panic. Never os.Exit outside of main.

func (e *Engine) SetBlocks(ctx context.Context, sel Selection, pattern string) error {
    if err := e.ensureConnected(ctx); err != nil {
        return fmt.Errorf("set blocks: %w", err)
    }
    cmd := e.composer.Set(sel, pattern)
    if err := e.dispatch(ctx, cmd); err != nil {
        return fmt.Errorf("set blocks at %v: %w", sel, err)
    }
    return nil
}
```

### Command Dispatch Pattern

```go
// dispatch routes commands to the appropriate channel based on
// operation type and RCON availability.
func (e *Engine) dispatch(ctx context.Context, cmd Command) error {
    if cmd.RequiresPlayerSession || !e.rcon.Available() {
        return e.chat.Send(ctx, cmd.String())
    }
    return e.rcon.Send(ctx, cmd.String())
}
```

### MCP Tool Registration Pattern

```go
// Tools are registered declaratively. The MCP server handles
// JSON-RPC framing. Each tool gets connection-check middleware.
func registerBlockTools(srv *MCPServer, engine *Engine) {
    srv.RegisterTool("place-block", placeBlockSchema, func(ctx context.Context, args map[string]any) (string, error) {
        // args already validated against schema
        x, y, z := args["x"].(float64), args["y"].(float64), args["z"].(float64)
        return engine.PlaceBlock(ctx, x, y, z)
    })
}
```

### Logging Pattern

```go
// All logging to stderr. JSON structured format.
// Levels: debug, info, warn, error
// MCP stdout stream must never be contaminated.
log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
log.Info("worldedit tier detected", "tier", tier, "version", version)
```

### Testing Pattern

- **Unit tests:** Mock the MC connection and RCON. Test construction engine logic, command composition, capability detection, config parsing.
- **Integration tests (CI):** Spin up a Paper server with WorldEdit in Docker. Run actual commands. Validate block placement.
- **Test file naming:** `*_test.go` in same package (white-box testing for internal logic).

## Project Structure & Boundaries

```
clankercraft/
├── main.go                    # Entry point: CLI parsing, startup sequence
├── go.mod
├── go.sum
├── goreleaser.yaml            # Cross-platform release config
├── .github/
│   └── workflows/
│       └── build.yml          # CI: lint, test, build
│       └── release.yml        # CD: goreleaser on tag
│
├── internal/                  # All application code (unexported)
│   ├── config/
│   │   └── config.go          # Viper config, CLI flags, layered resolution
│   │
│   ├── connection/
│   │   ├── mc.go              # go-mc Minecraft connection, state machine
│   │   ├── rcon.go            # RCON client, availability detection
│   │   └── auth.go            # MSA auth, token caching, offline mode
│   │
│   ├── engine/
│   │   ├── engine.go          # Construction engine — shared by both interfaces
│   │   ├── composer.go        # WorldEdit command composition
│   │   ├── tier.go            # Capability tier detection (FAWE/WE/vanilla)
│   │   ├── dispatch.go        # Command routing (chat vs RCON)
│   │   └── vanilla.go         # Vanilla fallback (/fill, /setblock, /clone)
│   │
│   ├── mcp/
│   │   ├── server.go          # MCP stdio JSON-RPC server
│   │   ├── transport.go       # Stdin/stdout transport with filtering
│   │   └── tools.go           # MCP tool registration (maps to engine)
│   │
│   ├── chat/
│   │   ├── listener.go        # Whisper listener, message routing
│   │   ├── responder.go       # Chat response formatting (multi-message)
│   │   └── session.go         # Per-player conversation context
│   │
│   ├── llm/
│   │   ├── provider.go        # LLMProvider interface
│   │   ├── claude.go          # ClaudeProvider (Anthropic API)
│   │   └── tools.go           # Tool definitions for LLM function calling
│   │
│   ├── observation/
│   │   ├── scanner.go         # Area scanning, block queries
│   │   ├── position.go        # Position tracking
│   │   └── signs.go           # Sign reading
│   │
│   └── log/
│       └── log.go             # Structured slog setup, stderr-only
│
├── skills/                    # Building knowledge (markdown, community-contributed)
│   ├── minecraft-player/
│   │   └── SKILL.md
│   └── minecraft-urban-planner/
│       └── SKILL.md
│
├── tools/                     # Auxiliary tools
│   └── world-scanner/         # Go world scanner (existing, 15M blocks/sec)
│
├── test/
│   └── integration/           # Integration tests (require Docker + Paper server)
│       └── worldedit_test.go
│
└── docs/                      # Documentation
    ├── prd.md
    ├── architecture-decision.md
    ├── epics.md
    └── ...
```

### Architectural Boundaries

| Boundary | Rule |
|---|---|
| `internal/engine` | Never imports `mcp` or `chat` — it doesn't know which interface is calling it |
| `internal/mcp` | Only imports `engine` — translates JSON-RPC to engine calls |
| `internal/chat` | Imports `engine` and `llm` — orchestrates whisper → LLM → engine → response |
| `internal/connection` | Provides MC protocol and RCON to engine. No knowledge of MCP or chat semantics. |
| `internal/llm` | Pure LLM client. No MC protocol knowledge. |
| `internal/observation` | Uses `connection` to read world state. No construction capability. |
| `main.go` | Wires everything together. Only file that imports multiple internal packages. |

### Data Flow

**MCP Path:**
```
Claude Code → stdin (JSON-RPC) → mcp/server → mcp/tools → engine → connection (chat/RCON) → MC Server
MC Server → connection → engine result → mcp/tools → stdout (JSON-RPC) → Claude Code
```

**Chat Path:**
```
Player whisper → connection/mc → chat/listener → llm/claude (with tools) → engine → connection → MC Server
MC Server → connection → engine result → llm/claude (next turn) → chat/responder → connection/mc → Player chat
```

## Architecture Validation Results

### Coherence Assessment

| Check | Status | Notes |
|---|---|---|
| Technology compatibility | ✅ | go-mc, viper, cobra, slog all standard Go ecosystem |
| Pattern consistency | ✅ | All patterns use Go conventions, error wrapping, interface-based dispatch |
| Structure alignment | ✅ | Clean separation: engine knows nothing about interfaces |
| Dual interface sharing | ✅ | Both MCP and chat use same engine — no feature drift |
| Tier degradation | ✅ | Engine handles tier at dispatch level, transparent to callers |

### Requirements Coverage

| Requirement Area | Coverage | Notes |
|---|---|---|
| Connection & Auth (FR1-5) | ✅ | `internal/connection/` — mc.go, auth.go, state machine |
| In-Game Chat (FR6-11) | ✅ | `internal/chat/` — listener, responder, session |
| MCP Stdio (FR12-15) | ✅ | `internal/mcp/` — server, transport, tools |
| WorldEdit (FR16-25) | ✅ | `internal/engine/` — composer, tier, dispatch |
| Vanilla Fallback (FR26-29) | ✅ | `internal/engine/vanilla.go` |
| RCON (FR30-32) | ✅ | `internal/connection/rcon.go` + engine dispatch |
| Observation (FR33-39) | ✅ | `internal/observation/` |
| LLM Integration (FR40-43) | ✅ | `internal/llm/` — provider interface, claude impl |
| Config & Distribution (FR44-46) | ✅ | `internal/config/` + goreleaser |

### Implementation Readiness

- **Spike gate:** `main.go` + `internal/connection/mc.go` + `internal/engine/engine.go` — connect and send a WE command. Day 1 validation.
- **All decisions made.** No deferred architectural choices blocking MVP.
- **No external dependencies requiring setup** beyond a Minecraft server with WorldEdit.

### Confidence Level: **High**

The PRD's technical requirements are unusually detailed for a PRD — they already contain architectural decisions. This document formalizes and validates those decisions. The main risk is in the `go-mc` integration (MSA auth, packet handling for 1.21.x), which the spike gate addresses on day 1.
