---
stepsCompleted: ['step-01-init', 'step-02-discovery', 'step-02b-vision', 'step-02c-executive-summary', 'step-03-success', 'step-04-journeys', 'step-05-domain-skipped', 'step-06-innovation', 'step-07-project-type', 'step-08-scoping', 'step-09-functional', 'step-10-nonfunctional', 'step-11-polish', 'step-12-complete']
completedAt: '2026-04-03'
inputDocuments:
  - docs/project-overview.md
  - docs/architecture.md
  - docs/api-contracts.md
  - docs/source-tree-analysis.md
  - docs/development-guide.md
  - docs/deployment-guide.md
  - docs/contribution-guide.md
  - docs/mineflayer-reference.md
  - docs/index.md
workflowType: 'prd'
documentCounts:
  briefs: 0
  research: 0
  projectDocs: 9
classification:
  projectType: creative_agent_platform
  domain: creative_tools_entertainment
  complexity: medium-high
  projectContext: brownfield_rewrite_with_scope_expansion
---

# Product Requirements Document - Clankercraft

**Author:** Kaylin
**Date:** 2026-04-03

## Executive Summary

Clankercraft is an AI building partner that lives inside your Minecraft world. You summon it, point at terrain with a WorldEdit wand, describe what you want in plain English through in-game chat, and it builds it — orchestrating WorldEdit's full command vocabulary (selections, patterns, expressions, schematics, brushes) behind a natural conversation.

The current system (v2.0.4) is a TypeScript MCP server connecting Claude to Minecraft via Mineflayer, exposing 25 tools over stdio JSON-RPC. It works, but the architecture fights its own product vision: the interaction happens in a terminal, not in the game. A stdout filter hacks around Mineflayer's logging to protect the JSON-RPC stream. 19 npm dependencies provide an API surface used at roughly 15%.

The v3 rewrite rebuilds clankercraft in Go as a dual-interface construction engine. The MCP stdio interface remains for Claude Code/Desktop integration. A new in-game chat interface lets the builder bot operate as an autonomous AI player — summoned to your location, listening for whispered commands, executing builds using WorldEdit, and responding conversationally. The Go binary connects via `go-mc` for protocol, RCON for bulk WorldEdit operations, and embeds the existing world scanner (15M blocks/sec) for terrain analysis. One binary, zero runtime dependencies, sub-second startup.

### What Makes This Special

No one has built an LLM-to-WorldEdit bridge. Minecraft builders type WorldEdit commands manually. AI coding assistants operate from terminals. Clankercraft v3 puts the AI *in the game* as a player you collaborate with — you select regions with a wand, the bot inherits your selection, you describe what you want, it builds. The wand is the pointing device. Chat is the voice. WorldEdit is the hands.

WorldEdit's expression engine (`//generate` with Perlin noise, trig, variables), pattern system (`#simplex` for organic textures, percentage-based material mixing), and schematic library (save/load/rotate/compose architectural components) give the AI a construction vocabulary that vanilla `/fill` cannot match. Combined with an LLM's ability to interpret "build a weathered Gothic cathedral" into specific expressions, patterns, and command sequences, this creates a creative tool category that doesn't exist yet.

The system supports two capability tiers: WorldEdit-enhanced (full construction engine on Paper/Spigot/Fabric servers) and vanilla fallback (`/fill`, `/setblock`, `/clone` on unmodded servers). It targets creative mode as the primary path, with survival mode support (resource gathering, crafting, inventory management) preserved from v2.

## Project Classification

| Attribute | Value |
|---|---|
| **Project Type** | Creative Agent Platform |
| **Domain** | Creative Tools / Entertainment |
| **Complexity** | Medium-High |
| **Project Context** | Brownfield rewrite with scope expansion |
| **Current Version** | 2.0.4 (TypeScript/Mineflayer/MCP) |
| **Target Version** | 3.0 (Go/go-mc/WorldEdit/dual-interface) |

## Success Criteria

### User Success
- Build beautiful structures from plain English without leaving the game
- Iterate on builds by pointing (wand) and talking (chat) — tight creative feedback loop
- AI interprets architectural *intent*, not just literal commands — "weathered Gothic" produces contextually appropriate materials and shapes
- Zero-config experience: summon the builder, start talking, watch it build

### Business Success
- **3-month:** Working demo video that showcases in-game chat building with WorldEdit — the "holy shit" moment for the AI + Minecraft community
- **6-month:** GitHub stars > 500, active users on multiple servers, content creators adopting it
- **12-month:** Community-contributed building styles and schematics, "clankercraft" recognized as the AI Minecraft building tool
- Distribution: open-source, single binary, frictionless install

### Technical Success
- Single Go binary, zero runtime dependencies, sub-second startup
- Dual interface: MCP stdio (Claude Code/Desktop) + in-game chat (autonomous agent)
- WorldEdit integration: selection sharing, region operations, expressions, schematics, patterns
- Vanilla fallback: `/fill`/`/setblock`/`/clone` when WorldEdit unavailable
- Stable 1-hour+ building sessions with no crashes or disconnects
- < 5 second response time for chat-based build requests
- Cross-platform: Linux, macOS, Windows binaries from single `go build`

### Measurable Outcomes
- Spike gate: Go binary connects + places blocks via WorldEdit in 1 day
- Demo gate: Full in-game chat building session recorded as demo video
- Parity gate: All 25 v2 MCP tools functional in Go
- Distribution gate: Binary installable by a new user in < 2 minutes

## Product Scope

See [Project Scoping & Phased Development](#project-scoping--phased-development) for the detailed MVP feature set, phase roadmap, and risk analysis.

## User Journeys

### Journey 1: Kaylin the Builder — "Make This Beautiful"

**Opening Scene:** Kaylin is flying over her Minecraft world in creative mode, surveying a river valley she's been wanting to develop. She's got a vision — a medieval village with a central cathedral, terraced gardens along the riverbank, and a stone bridge. Manually placing thousands of blocks would take days. Opening a terminal to type JSON-RPC commands would kill the immersion.

**Rising Action:** She lands on the hilltop overlooking the valley.

```
Kaylin:   /msg Builder come here
Builder:  *teleports to Kaylin's position*
Builder:  Hey! Nice spot. What are we building?
Kaylin:   I want a Gothic cathedral right here on the hilltop.
          Big one — stone bricks, flying buttresses, rose window.
Builder:  Got it. How about 40 wide, 60 tall, 80 long? I'll use
          stone bricks with mossy variation for a weathered look.
Kaylin:   Yeah, go for it.
```

The builder sets WorldEdit selections, fires `//generate` expressions for the arched nave, `//set` with `#simplex` patterns for the stone walls, `//smooth` for organic contours. Blocks appear in waves — the walls rise, buttresses extend, the roof arches close.

```
Builder:  Nave and walls are up. Take a look.
Kaylin:   *flies around the structure*
Kaylin:   Love it. But the entrance feels too flat.
Kaylin:   *selects the entrance facade with wand*
Kaylin:   /msg Builder make this more grand — taller arch,
          add some depth and detail
Builder:  Got your selection (40,64,100) to (52,90,105).
          Adding a recessed portal with pointed arch and
          column details.
          *modifies the selection*
Builder:  How's that?
Kaylin:   Perfect. Save this as a schematic — I want to
          reuse this entrance style.
Builder:  Saved as "gothic_entrance_grand". Ready for the
          bridge next?
```

**Climax:** An hour in, the cathedral is complete. Kaylin stands inside, looking up at the vaulted ceiling with light streaming through stained glass. She didn't place a single block manually. She described, pointed, iterated — and the builder made it real.

**Resolution:** Kaylin flies up for an aerial view. Cathedral on the hill, village taking shape below, the bridge spanning the river. She screenshots it, posts it. The build that would have taken a week took an evening. She saves the cathedral as a schematic for her next world.

---

### Journey 2: Marcus the MCP User — "Claude, Build Me a Base"

**Opening Scene:** Marcus is a developer who uses Claude Code daily. He just discovered clankercraft on GitHub and wants to try it with his local Minecraft server. He's comfortable in a terminal but doesn't want to learn WorldEdit commands.

**Rising Action:** He installs clankercraft in 90 seconds:

```bash
# Download the binary
curl -L https://github.com/kaylincoded/clankercraft/releases/latest/download/clankercraft-linux-amd64 -o clankercraft
chmod +x clankercraft

# Add to Claude Code MCP config
claude mcp add clankercraft ./clankercraft --host localhost --port 25565 --username BuilderBot
```

In Claude Code, he starts a conversation:

```
Marcus:   I just spawned in a plains biome. Build me a modern house
          with a pool and a garden.
Claude:   I can see you're at (128, 64, -200) in a flat plains area.
          I'll build a 20x15 modern house with clean lines, large
          windows, a rooftop terrace, lap pool to the east, and a
          landscaped garden to the south. Starting now.
```

Claude orchestrates through MCP — the same WorldEdit command composer, same construction engine, just driven by MCP tool calls instead of in-game chat.

**Climax:** Marcus joins his Minecraft server and sees a modern house sitting in the plains. Clean quartz walls, dark oak accents, blue pool. He didn't learn a single Minecraft command. He just talked to Claude like he always does.

**Resolution:** Marcus shares the screenshot in his dev Discord. Three people ask "how." He links the GitHub repo. They each install the binary that evening.

---

### Journey 3: Alex the Server Admin — "Setting Up the Playground"

**Opening Scene:** Alex runs a small creative server for her friend group (8 players). They've seen clankercraft demos and want a builder bot on their server. Alex is comfortable with server management but not Go or MCP.

**Rising Action:** Alex checks the prerequisites:

```
- Paper 1.21.11 server (already running)
- WorldEdit installed (already have it)
- FAWE? — installs it (drop-in replacement, 2 minutes)
- RCON enabled in server.properties (sets password, port 25575)
```

She downloads the clankercraft binary, creates a systemd service or just runs it:

```bash
./clankercraft --host localhost --port 25565 --rcon-port 25575 \
  --rcon-password secret --username Builder
```

The bot logs in. It appears in the player list. Alex sets permissions — the Builder account gets WorldEdit access, operator for commands.

**Complication:** One player, Jake, whispers to the builder to "delete everything in a 500 block radius." The builder needs guardrails.

```
Builder:  That would affect 250,000,000 blocks across areas
          other players have built. I can only modify areas
          within your claimed region or selections you make
          with the wand. Want to select a specific area instead?
```

**Climax:** All 8 players are building simultaneously. Three are talking to the builder in chat, each getting personalized responses. The builder manages its WorldEdit session per-request, undoing when asked, saving schematics players want to keep.

**Resolution:** Alex checks server performance — FAWE's async operations mean the builder's WorldEdit commands don't lag the server. The bot has been running for 6 hours without a crash. She adds it to the server's auto-start script.

---

### Journey 4: Priya the Content Creator — "The Viral Build"

**Opening Scene:** Priya runs a Minecraft YouTube channel (50K subscribers). She's planning a "I let AI build my entire city" video. She needs the building process to be visually dramatic, not instant.

**Rising Action:** Priya starts recording, flies to an empty mesa biome.

```
Priya:    /msg Builder I want you to build me an entire desert
          city. Sandstone, terracotta domes, narrow market streets,
          a central palace. Build it piece by piece so my viewers
          can watch.
Builder:  Love it. I'll build in sections — palace first, then
          market district, residential quarter, city walls.
          I'll place blocks in visible waves so the camera
          catches the construction. Ready when you are.
```

The builder constructs in "animation mode" — FAWE's streaming placement means blocks appear progressively, not all at once. Priya circles with the camera, narrating as walls rise and domes form.

**Climax:** The palace dome completes — a massive terracotta structure generated with `//generate` using a sphere expression with noise-varied radius. The market streets wind organically between buildings, placed with `//curve` and `//line`. Priya's chat is going wild.

**Complication:** Mid-build, the bot disconnects (server hiccup). It auto-reconnects in 3 seconds, picks up the conversation context, and continues from where it left off. Priya barely has to edit the footage.

**Resolution:** The video gets 200K views. Comments are full of "how is this real" and "link to the bot?" The GitHub repo gets 300 stars overnight.

---

### Journey 5: Dev the Contributor — "Adding a Japanese Style"

**Opening Scene:** Dev is a Go developer and Minecraft builder who loves Japanese architecture. He's been using clankercraft and wants to contribute a "Japanese temple" building skill.

**Rising Action:** He forks the repo, reads the existing skills:

```
.claude/skills/
  minecraft-player/SKILL.md        - building patterns
  minecraft-urban-planner/SKILL.md - material palettes, spacing rules
```

He creates a new skill: `minecraft-japanese-architect/SKILL.md` containing:
- Material palette: dark oak, spruce, white concrete, acacia
- Roof rules: curved eaves using stair blocks, 3-layer depth
- Spacing: tatami-proportioned rooms (2:1 ratio)
- Garden patterns: gravel paths, water features, bamboo placement
- Temple typologies: pagoda, torii gate, zen garden, tea house

He also creates schematics for reusable components — a torii gate, a pagoda roof cap, a zen rock garden.

**Climax:** He submits a PR. Kaylin reviews it, tests it in-game:

```
Kaylin:   /msg Builder build me a Japanese temple complex
          using the japanese-architect style
Builder:  *loads the japanese-architect skill*
          Building a 3-tier pagoda with curved eaves, stone
          garden courtyard, torii gate entrance. Using dark
          oak and spruce palette with white concrete accents.
```

It works. The skill guides the AI's material choices and proportions. The schematics provide reusable structural elements.

**Resolution:** The PR merges. Other users start requesting "Japanese style" builds. Dev's contribution becomes one of the most-used community skills. He submits more — Chinese courtyard, Art Deco, Brutalist.

---

### Journey Requirements Summary

| Journey | Key Capabilities Revealed |
|---|---|
| **Kaylin (Builder)** | In-game chat interface, wand selection sharing, WorldEdit command composition, schematic save/load, iterative refinement, spatial awareness |
| **Marcus (MCP User)** | MCP stdio transport, single binary install, Claude Code/Desktop integration, same construction engine via different interface |
| **Alex (Admin)** | Binary distribution, RCON configuration, permission/guardrail system, multi-player session management, auto-reconnect, server performance (FAWE async) |
| **Priya (Creator)** | Build animation mode, session stability (1hr+), auto-reconnect with context preservation, visually dramatic construction |
| **Dev (Contributor)** | Skill authoring (markdown), schematic contribution, PR workflow, extensible architecture |

**Cross-cutting requirements surfaced:**
- Guardrails / build area limits (Alex's journey)
- Conversation context preservation across reconnects (Priya's journey)
- Skill loading system that supports community contributions (Dev's journey)
- Two interfaces (chat + MCP) sharing one construction engine (Kaylin + Marcus)

## Innovation & Novel Patterns

### Detected Innovation Areas

**1. LLM-to-WorldEdit Bridge (New Category)**
No existing tool connects an LLM directly to WorldEdit's command vocabulary. WorldEdit has existed for 10+ years. LLM-powered coding assistants have existed for 2+ years. The combination — an AI that interprets "build a weathered Gothic cathedral" and translates it into `//generate` expressions, `#simplex` patterns, and schematic compositions — creates a new product category.

**2. In-Game AI Player (Interface Innovation)**
Current AI-Minecraft integrations operate from terminals or external tools. Clankercraft v3 puts the AI *inside the game* as a visible player with a conversational chat interface. The interaction model — summon, point with wand, describe in chat, iterate — has no precedent in the Minecraft ecosystem or the broader AI agent space.

**3. Wand-as-Cursor Interaction Model (UX Innovation)**
Using WorldEdit's existing wand tool as a shared pointing device between human and AI. The player selects a region with left/right click, the builder bot reads those coordinates and operates on the player's selection. This repurposes a familiar Minecraft tool as an AI interaction primitive.

**4. Construction Compiler Architecture (Technical Innovation)**
The Go binary functions as a "construction compiler" — it takes high-level architectural intent from an LLM and compiles it into optimized WorldEdit command sequences across two channels (chat for player-context operations, RCON for bulk operations). This dual-channel command dispatch with capability-tier detection (FAWE vs WorldEdit vs vanilla) is a novel architecture.

### Market Context & Competitive Landscape

- **Minecraft bot frameworks** (Mineflayer, go-mc): Protocol libraries, not AI agents. No LLM integration, no WorldEdit awareness.
- **AI coding assistants** (Claude Code, Cursor, Copilot): Operate in terminals/IDEs. Don't connect to game worlds.
- **WorldEdit automation** (command blocks, CraftScript): Programmatic but not conversational. Require Minecraft/Java expertise.
- **Minecraft AI research** (MineRL, Voyager, STEVE-1): Academic projects focused on survival/navigation. Not creative building tools. Not shipped products.
- **No direct competitor** combines LLM + WorldEdit + in-game presence + conversational building.

### Validation Approach

1. **Spike gate (day 1):** Go binary connects to MC server, sends `//set` via chat. Validates go-mc + WorldEdit integration is feasible.
2. **Chat interface proof (week 1):** Whisper → Claude API → WorldEdit command → chat response round-trip working. Validates the autonomous agent architecture.
3. **Demo video (week 2-3):** Full building session recorded. Validates the product vision is demonstrable and compelling.
4. **Community reaction (month 1):** Ship binary, post demo. GitHub stars and Discord engagement validate market interest.

### Risk Mitigation

See [Risk Mitigation Strategy](#risk-mitigation-strategy) for the consolidated risk analysis covering technical, market, and resource risks.

## Creative Agent Platform — Technical Requirements

### Project-Type Overview

Clankercraft v3 is a creative agent platform that operates as both an MCP tool server (programmatic interface) and an autonomous in-game AI player (conversational interface). It bridges LLM intelligence to Minecraft's construction capabilities via WorldEdit, delivered as a single Go binary.

### Technical Architecture Considerations

**Runtime & Distribution:**
- Single statically-linked Go binary, cross-compiled for 6 targets (linux/macOS/windows x amd64/arm64)
- GitHub Releases via `goreleaser` with checksums. Package managers (brew, AUR, scoop) are community contributions.
- Zero runtime dependencies — no Node.js, no JVM, no Python

**Protocol & Version Support:**
- Target: Minecraft Java Edition 1.21.x (pin to latest stable, currently 1.21.11)
- Single version at a time — update when new versions release, don't maintain multi-version adapters
- Protocol layer: `go-mc` library for Minecraft protocol, MSA authentication, chunk parsing

**Authentication:**
- Microsoft (MSA) authentication as default via `go-mc`
- `--offline` flag for offline/cracked servers and development
- Token caching for session persistence

**Configuration (layered, highest priority first):**
1. CLI flags (`--host`, `--port`, `--rcon-password`, etc.)
2. Environment variables (`CLANKERCRAFT_HOST`, `CLAUDE_API_KEY`)
3. Config file (`~/.config/clankercraft/config.yaml`)
4. Sensible defaults

**LLM Integration:**
- `LLMProvider` interface for pluggable LLM backends
- MVP: `ClaudeProvider` (Anthropic API) for in-game chat autonomous mode
- MCP path is provider-agnostic — the MCP client handles its own LLM
- Community can contribute `OpenAIProvider`, `OllamaProvider` etc. via PR

**Dual Interface Architecture:**
- **MCP Stdio:** JSON-RPC over stdin/stdout for Claude Code/Desktop integration. Tool registration pattern with validation middleware.
- **In-Game Chat:** Whisper listener → LLM API call with tools → WorldEdit command execution → chat response. Autonomous agent mode.
- Both interfaces share one construction engine (WorldEdit command composer)

**WorldEdit Integration (3-tier capability detection):**
- **Tier 1 — FAWE:** Full pattern system (`#simplex`, `#color`, gradients), async operations, extended brushes, `//generate` with noise functions
- **Tier 2 — WorldEdit:** Core commands (`//set`, `//replace`, `//generate`, `//smooth`, `//schem`), standard patterns, expression engine
- **Tier 3 — Vanilla:** `/fill`, `/setblock`, `/clone` only. No undo, no expressions, 32K block limit per command.
- Detect tier on connect. Degrade gracefully. Same MCP tools regardless of tier.

**Command Channels:**
- **Chat:** Bot sends WorldEdit commands as a player. Has its own WE session, selection, history. Used for player-context operations, wand interaction.
- **RCON:** Direct server console. No chat rate limiting. Used for bulk operations, large `//set`/`//replace`, schematic pastes.

**Extension Points:**
- **Skills (markdown):** Building knowledge injected into LLM context. Primary community extension mechanism. No Go knowledge required.
- **Schematics (`.schem` files):** `~/.config/clankercraft/schematics/` directory scanned on startup. File-based extensibility.
- **Go source (PR):** New tools are Go files in `tools/`, registered in `main.go`. Clean enough for community PRs, but not a plugin system.
- **No Go plugin system** — `plugin` package is fragile and unmaintainable for solo projects.

### Implementation Considerations

**Connection Management:**
- State machine: `disconnected` → `connecting` → `connected`
- Auto-reconnect with exponential backoff
- Conversation context preservation across reconnects (for content creator sessions)
- Graceful shutdown on SIGINT/SIGTERM

**Guardrails:**
- Build area limits configurable per-server (`max_radius`, `max_blocks_per_operation`)
- Destructive operation confirmation (large `//replace`, `//set` affecting > N blocks)
- No access to other players' WorldEdit sessions without explicit `/sharesel`

**Observability:**
- Structured logging to stderr (JSON format, configurable level)
- Operation metrics: commands sent, blocks modified, LLM API latency
- Health check endpoint (optional, for monitoring in server deployments)

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Experience MVP — "The Demo Video"

The fastest path to validated learning is a compelling demo that proves the core interaction model: summon an AI builder in-game, point with a wand, describe what you want, watch it build. The demo video is both the validation artifact and the distribution mechanism.

**Resource Requirements:** Solo developer (Kaylin), Go expertise, access to a Paper server with WorldEdit/FAWE.

**MVP Philosophy:** If it's not needed for the demo video, it's not MVP. The demo requires: connect → chat → build → iterate. Everything else is post-MVP.

### MVP Feature Set (Phase 1) — "The Demo Video"

**Core User Journeys Supported:**
- Journey 1 (Kaylin the Builder) — partial: in-game chat building with WorldEdit, wand selection, iterative refinement
- Journey 2 (Marcus the MCP User) — full: MCP stdio tools, same construction engine

**Must-Have Capabilities:**

| Capability | Justification | Without It... |
|---|---|---|
| go-mc connection + MSA auth | Can't join a server | Product doesn't exist |
| In-game chat listener (whisper) | Core interaction model | No in-game experience — it's just v2 again |
| Claude API integration | Bot needs to think | Chat responses are hardcoded, not AI |
| WorldEdit command composer | Core construction engine | Falls back to `/fill` only — unimpressive demo |
| Wand selection sharing | Point-and-describe workflow | User must type coordinates — kills the magic |
| `scan-area` / `get-position` / `get-block-info` | Bot needs spatial awareness | Builds blindly, can't verify or iterate |
| MCP stdio transport | Claude Code/Desktop compatibility | Lose existing user base, break v2 workflows |
| RCON channel | Bulk WorldEdit operations | Chat rate limiting throttles large builds |
| Vanilla fallback (`/fill`, `/setblock`) | Servers without WorldEdit | Narrows audience unnecessarily at launch |
| Auto-reconnect | Session stability | Demo crashes mid-build, unusable |

**Explicitly NOT MVP:**
- Schematic save/load (Growth)
- FAWE pattern detection — use basic WE patterns first (Growth)
- `//generate` expression composition — use `//set`/`//replace` first (Growth)
- Crafting/furnace/entity/flight tools — creative mode doesn't need them (Growth)
- Build animation mode (Growth)
- Survival mode support (Growth)
- Multi-player builder management (Vision)
- Companion Paper plugin (Vision)
- Guardrails/permission system — single user MVP (Growth)

### Post-MVP Features

**Phase 2 — Growth (months 2-3):**
- Schematic library: `//schem save/load/list`, `~/.config/clankercraft/schematics/`
- FAWE detection + advanced patterns (`#simplex`, `#color`, gradients)
- `//generate` expression composition for domes, arches, organic shapes
- Embedded world scanner for terrain analysis
- Full 25-tool MCP parity (port remaining v2 tools)
- Builder personality tuning (system prompt customization)
- Guardrails: `max_radius`, `max_blocks_per_operation`, build area limits
- Multi-player session management (Alex's journey)

**Phase 3 — Expansion (months 4-6):**
- Build animation mode for content creators (Priya's journey)
- Survival mode support (resource gathering, crafting, pathfinding)
- Community skill/schematic contribution workflow (Dev's journey)
- Companion Paper plugin (`/sharesel`, structured WE responses)
- Cross-platform binary distribution (goreleaser + homebrew tap)

**Phase 4 — Vision (6+ months):**
- Multi-builder support (per-player AI agents)
- Architectural style transfer
- Community schematic marketplace
- Terrain generation from reference images (FAWE CFI)
- Local LLM support (Ollama provider)

### Risk Mitigation Strategy

**Technical Risks:**

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| go-mc can't handle MSA auth for 1.21.x | Low | Critical | Spike gate validates day 1. Fallback: `--offline` for MVP, fix auth post-launch |
| WorldEdit command composition produces wrong results | Medium | High | Test against real server in CI. WE's `//undo` provides safety net. |
| Chat rate limiting throttles build speed | Medium | Medium | RCON channel bypasses chat limits for bulk operations |
| Claude API latency > 5s breaks conversational flow | Low | Medium | Async pattern: acknowledge immediately, build while composing next message |
| go-mc chunk parsing incomplete for 1.21.x | Low | High | World scanner already parses chunks in Go — proven path |
| In-game chat interface too limiting (256 char limit) | Medium | Low | Builder uses multiple messages, summarizes intent, confirms before executing |
| Wand selection sharing requires server plugin | Low | Medium | MVP: bot listens for WorldEdit's coordinate feedback in chat. Growth: companion Paper plugin |

**Market Risks:**

| Risk | Mitigation |
|---|---|
| Nobody wants an AI Minecraft builder | Demo video validates interest before heavy investment. If < 100 stars in first week, reassess. |
| Minecraft updates break protocol | Pin to 1.21.x, update reactively. go-mc community tracks versions. |
| WorldEdit dependency narrows audience | Vanilla fallback ensures basic functionality everywhere. WE is standard on creative servers. |

**Resource Risks:**

| Risk | Mitigation |
|---|---|
| Solo developer burnout during rewrite | MVP scoped to ~2,500 lines Go. Milestone gates allow stopping at any point with a working product. |
| Dual codebase maintenance (TS + Go) | Don't maintain both. Ship Go MVP, deprecate TypeScript version immediately. |
| Community doesn't contribute | Skills and schematics are markdown/files — lowest possible contribution barrier. But product works without community. |

## Functional Requirements

### Connection & Authentication

- **FR1:** Bot can connect to a Minecraft Java Edition 1.21.x server via the Minecraft protocol
- **FR2:** Bot can authenticate via Microsoft (MSA) account with device code flow and token caching
- **FR3:** Bot can connect in offline mode when configured with `--offline` flag
- **FR4:** Bot can auto-reconnect after disconnection with exponential backoff
- **FR5:** Bot can gracefully shut down on SIGINT/SIGTERM, cleaning up server-side state

### In-Game Chat Interface

- **FR6:** Bot can detect and respond to whispered messages (`/msg`) from players
- **FR7:** Bot can send chat messages visible to nearby players or whispered to a specific player
- **FR8:** Bot can interpret natural language build requests and translate them into construction actions
- **FR9:** Bot can maintain conversational context across multiple message exchanges within a session
- **FR10:** Bot can preserve conversation context across reconnections within the same session
- **FR11:** Bot can teleport to the requesting player's location when summoned

### MCP Stdio Interface

- **FR12:** System can expose construction and observation tools via MCP JSON-RPC over stdin/stdout
- **FR13:** System can validate tool arguments before execution
- **FR14:** System can return structured error responses for failed tool calls
- **FR15:** System can check connection health before every tool execution, triggering reconnect if needed

### WorldEdit Construction

- **FR16:** Bot can detect WorldEdit capability tier on connect (FAWE / WorldEdit / vanilla)
- **FR17:** Bot can set WorldEdit selections (`//pos1`, `//pos2`) programmatically
- **FR18:** Bot can read a player's WorldEdit selection coordinates (wand selection sharing)
- **FR19:** Bot can execute WorldEdit region operations (`//set`, `//replace`, `//walls`, `//faces`, `//hollow`)
- **FR20:** Bot can execute WorldEdit generation commands (`//generate` with expressions, `//sphere`, `//cyl`, `//pyramid`)
- **FR21:** Bot can execute WorldEdit terrain operations (`//smooth`, `//naturalize`, `//overlay`)
- **FR22:** Bot can use WorldEdit pattern syntax for material mixing (e.g., `50%stone,30%cobblestone,20%mossy_stone_bricks`)
- **FR23:** Bot can execute WorldEdit clipboard operations (`//copy`, `//paste`, `//rotate`, `//flip`)
- **FR24:** Bot can manage WorldEdit schematics (`//schem save`, `//schem load`, `//schem list`)
- **FR25:** Bot can undo/redo WorldEdit operations (`//undo`, `//redo`)

### Vanilla Fallback Construction

- **FR26:** Bot can place blocks via `/setblock` command
- **FR27:** Bot can fill regions via `/fill` command (respecting 32K block limit)
- **FR28:** Bot can clone regions via `/clone` command
- **FR29:** Bot can teleport via `/tp` command

### RCON Operations

- **FR30:** System can connect to Minecraft server via RCON protocol
- **FR31:** System can dispatch WorldEdit commands via RCON for bulk operations (bypassing chat rate limits)
- **FR32:** System can detect RCON availability and fall back to chat-only when unavailable

### World Observation

- **FR33:** Bot can report its current position in the world
- **FR34:** Bot can scan a rectangular area and return block types and positions
- **FR35:** Bot can query block information at a specific coordinate
- **FR36:** Bot can find the nearest block of a specific type within a search radius
- **FR37:** Bot can read sign text at a specific position
- **FR38:** Bot can find all signs within a search radius and read their text
- **FR39:** Bot can detect the current game mode

### LLM Integration

- **FR40:** System can send player messages to an LLM API with construction tools as available functions
- **FR41:** System can execute tool calls returned by the LLM and feed results back into the conversation
- **FR42:** System can be configured with different LLM providers via a pluggable interface
- **FR43:** System can include skill files (markdown) as context when calling the LLM

### Configuration & Distribution

- **FR44:** System can be configured via CLI flags, environment variables, and config file (layered priority)
- **FR45:** System can be distributed as a single binary for Linux, macOS, and Windows (amd64 and arm64)
- **FR46:** System can load user schematics from a configurable directory on startup

## Non-Functional Requirements

### Performance

- **NFR1:** In-game chat responses (bot acknowledges a whispered request) must complete within 3 seconds — the threshold where conversational flow feels natural
- **NFR2:** WorldEdit command dispatch (from tool call to command sent) must complete within 100ms — the construction engine should never be the bottleneck
- **NFR3:** `scan-area` observation tool must return results for a 50x50x50 region within 1 second
- **NFR4:** MCP tool call round-trip (stdin request → stdout response) must complete within 200ms excluding external calls (Minecraft server, LLM API)
- **NFR5:** Binary startup to "ready for connections" must complete within 2 seconds
- **NFR6:** Memory usage must stay below 256MB during normal operation (building sessions of 1hr+)

### Reliability

- **NFR7:** System must maintain stable sessions of 1+ hours without crashes or memory leaks
- **NFR8:** Auto-reconnect must succeed within 10 seconds of detecting disconnection, with up to 5 retry attempts
- **NFR9:** Conversation context must survive reconnections — the bot resumes the building conversation after reconnect without the player repeating themselves
- **NFR10:** WorldEdit command failures must not crash the system — errors are caught, logged, and reported to the player via chat or MCP response
- **NFR11:** RCON connection failure must degrade gracefully to chat-only mode, not block operation
- **NFR12:** LLM API failures (rate limit, timeout, error) must degrade gracefully — bot reports "I'm having trouble thinking right now, try again in a moment" rather than crashing

### Integration

- **NFR13:** Minecraft protocol integration must support Java Edition 1.21.x packet format via go-mc
- **NFR14:** RCON integration must conform to Minecraft's RCON protocol specification (port 25575 default, max 4096 byte payload)
- **NFR15:** MCP integration must conform to the Model Context Protocol specification for stdio transport (JSON-RPC 2.0)
- **NFR16:** LLM provider interface must support streaming responses for reduced perceived latency in chat mode
- **NFR17:** WorldEdit capability detection must complete during initial connection, before any construction tools are available

### Security

- **NFR18:** RCON password must never be logged or included in error messages
- **NFR19:** Claude API key must be stored via environment variable or config file with appropriate file permissions — never in CLI arguments (visible in process list)
- **NFR20:** MSA auth tokens must be cached securely in the user's config directory with restricted file permissions
- **NFR21:** In-game chat interface must not execute arbitrary server commands from untrusted players — only recognized build-related actions through the LLM
