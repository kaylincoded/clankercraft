<h1 align="center">
  <br>
  clankercraft
  <br>
</h1>

<p align="center">
  <b>Claude yearns for the mines.</b> An AI building partner for Minecraft — lives in your world, listens for whispers, builds with WorldEdit.
</p>

https://github.com/user-attachments/assets/f567d526-3b80-4e54-86e3-e24981f6c288

<p align="center">
  <img src="https://img.shields.io/badge/Minecraft-1.21-62b47a?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0id2hpdGUiIGQ9Ik0yIDJoMjB2MjBIMnoiLz48L3N2Zz4=" />
  <img src="https://img.shields.io/badge/Protocol-MCP-b4befe?style=flat-square" />
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/Claude_Code-Skills_Included-cba6f7?style=flat-square" />
</p>

---

## What It Does

A bot joins your Minecraft server as a player. You whisper to it in-game, it talks to Claude, and builds what you describe — using WorldEdit's full command vocabulary for selections, patterns, expressions, schematics, and terrain operations.

Also exposes 37 tools over [Model Context Protocol](https://modelcontextprotocol.io) so Claude Desktop, Claude Code, or any MCP client can drive it from outside the game.

One binary, zero runtime dependencies, sub-second startup.

## Two Ways to Use It

### In-Game Chat (whisper to the bot)
Set `ANTHROPIC_API_KEY` and the bot becomes an autonomous AI player. Whisper commands in natural language, it builds with WorldEdit and responds conversationally.

### MCP Server (Claude Desktop / Claude Code)
Connect any MCP client over stdio. Claude sees 37 tools and the building skills, and can move, build, scan, and interact with the world.

Both interfaces work simultaneously.

## What's Inside

```
├── main.go                    Entrypoint — wires everything together
├── internal/
│   ├── agent/                 LLM agent loop — whisper → Claude → tool calls → reply
│   ├── chat/                  Chat parsing and message handling
│   ├── config/                CLI flags, env vars, config file layering
│   ├── connection/            go-mc protocol connection with auto-reconnect
│   ├── engine/                WorldEdit capability detection and command routing
│   ├── llm/                   LLM provider interface (Claude implementation)
│   ├── log/                   Structured logging setup
│   ├── mcp/                   MCP stdio server — 37 tools over JSON-RPC
│   ├── rcon/                  RCON client for bulk WorldEdit operations
│   └── schematic/             Schematic directory indexer
├── .claude/skills/            Building knowledge for the LLM
│   ├── minecraft-player/      /fill over place-block, verify with scan-area, etc.
│   └── minecraft-urban-planner/  Material palettes, facade depth, floor spacing
└── tools/world-scanner/       Go tool — scan .minecraft saves offline (15M blocks/sec)
```

## Skills — What Makes This Different

Most MCP servers give the LLM tools and hope for the best. This one ships opinionated skills learned from hundreds of hours of in-game iteration:

### `minecraft-player`
The building fundamentals. Stuff the LLM gets wrong without guidance:
- **Use `/fill`, not `place-block`** — one command places an entire wall
- **Never send commands in parallel** — Minecraft chat rate-limits; only the first goes through
- **Verify with `scan-area`** — catch silently dropped commands before moving on
- **Real doors, not air gaps** — blockstate syntax for doors, pressure plates, redstone

### `minecraft-urban-planner`
Architectural patterns for builds that actually look good:
- **Material mixing** — never one material per wall; mix 3-5 from the same tonal family
- **Facade depth** — recessed windows, stair sills, cornice projections
- **Floor spacing** — 5 blocks per floor (1 plate + 4 air)
- **Three palettes** — gray (andesite/concrete), warm (sandstone/birch), light (diorite/quartz)

## Quick Start

### 1. Run a Minecraft Server

Any Java Edition server with [WorldEdit](https://enginehub.org/worldedit/) (Paper, Spigot, or Fabric). Vanilla servers work too — the bot falls back to `/fill`, `/setblock`, and `/clone`.

### 2. Install

**From release:**
```bash
# Download from GitHub Releases for your platform
curl -LO https://github.com/kaylincoded/clankercraft/releases/latest/download/clankercraft_linux_amd64.tar.gz
tar xzf clankercraft_linux_amd64.tar.gz
```

**From source:**
```bash
git clone https://github.com/kaylincoded/clankercraft.git
cd clankercraft
go build -o clankercraft .
```

### 3. Run

**In-game AI mode** (whisper to the bot):
```bash
export ANTHROPIC_API_KEY=sk-ant-...
./clankercraft --host localhost --port 25565 --username ClankerBot
```

**MCP-only mode** (no API key needed — the MCP client provides the LLM):
```bash
./clankercraft --host localhost --port 25565 --username ClankerBot
```

### 4. Configure Your MCP Client

**Claude Desktop** — edit `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "minecraft": {
      "command": "/path/to/clankercraft",
      "args": [
        "--host", "localhost",
        "--port", "25565",
        "--username", "ClankerBot"
      ]
    }
  }
}
```

**Claude Code** — add to `.claude/settings.json`:

```json
{
  "mcpServers": {
    "minecraft": {
      "command": "/path/to/clankercraft",
      "args": [
        "--host", "localhost",
        "--port", "25565",
        "--username", "ClankerBot"
      ]
    }
  }
}
```

### 5. Talk to It

In-game: whisper `/msg ClankerBot build a Japanese torii gate here`

Via MCP: "Build a small Japanese-style house near my position"

## Tools (37)

| Category | Tools |
|---|---|
| **Status** | `ping` `status` `detect-gamemode` `detect-worldedit` |
| **Position** | `get-position` `look-at` `teleport-to-player` |
| **Blocks** | `get-block-info` `find-block` `scan-area` `read-sign` `find-signs` |
| **Vanilla Build** | `setblock` `fill` `clone` |
| **WorldEdit Selection** | `set-selection` `get-selection` |
| **WorldEdit Region** | `we-set` `we-replace` `we-walls` `we-faces` `we-hollow` |
| **WorldEdit Generation** | `we-sphere` `we-cyl` `we-pyramid` `we-generate` |
| **WorldEdit Terrain** | `we-smooth` `we-naturalize` `we-overlay` |
| **WorldEdit Clipboard** | `we-copy` `we-paste` `we-rotate` `we-flip` |
| **WorldEdit History** | `we-undo` `we-redo` |
| **Schematics** | `list-schematics` `load-schematic` |

## Configuration

Config is resolved in priority order: CLI flags > environment variables > config file > defaults.

| Flag | Env Var | Default | Description |
|---|---|---|---|
| `--host` | `HOST` | `localhost` | Minecraft server address |
| `--port` | `PORT` | `25565` | Server port |
| `--username` | `USERNAME` | `LLMBot` | Bot's in-game name |
| `--offline` | `OFFLINE` | `false` | Use offline/cracked mode |
| `--rcon-port` | `RCON_PORT` | `25575` | RCON port |
| `--rcon-password` | `RCON_PASSWORD` | | RCON password (enables bulk WorldEdit) |
| `--log-level` | `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| | `ANTHROPIC_API_KEY` | | Enables in-game AI chat mode |
| | `LLM_MODEL` | | Override Claude model |

Config file: `~/.config/clankercraft/config.yaml`

Schematics: `~/.config/clankercraft/schematics/*.schem`

## World Scanner (Offline)

Analyze any `.minecraft` save without a running server. Written in Go, scans 15M blocks/sec.

```bash
cd tools/world-scanner
go build -o world-scanner .
./world-scanner "/path/to/world" --bounds minX minY minZ maxX maxY maxZ --json output.json
```

Use it to study reference builds before asking Claude to reproduce them.

## Development

```bash
git clone https://github.com/kaylincoded/clankercraft.git
cd clankercraft
go build -o clankercraft .
go test ./...
```

## License

[MIT](LICENSE)
