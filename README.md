<h1 align="center">
  <br>
  ⛏️ clankercraft
  <br>
</h1>

<p align="center">
  <b>Claude yearns for the mines.</b> an MCP server that connects LLMs to a live Minecraft world via <a href="https://github.com/PrismarineJS/mineflayer">Mineflayer</a>.
</p>

https://github.com/kaylincoded/clankercraft/releases/download/v2.0.4/clankertown-480p.webm

<p align="center">
  <img src="https://img.shields.io/badge/Minecraft-1.21.11-62b47a?style=flat-square&logo=data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAyNCAyNCI+PHBhdGggZmlsbD0id2hpdGUiIGQ9Ik0yIDJoMjB2MjBIMnoiLz48L3N2Zz4=" />
  <img src="https://img.shields.io/badge/Protocol-MCP-b4befe?style=flat-square" />
  <img src="https://img.shields.io/badge/Runtime-Node_%3E%3D20-339933?style=flat-square&logo=node.js&logoColor=white" />
  <img src="https://img.shields.io/badge/Claude_Code-Skills_Included-cba6f7?style=flat-square" />
</p>

---

## What It Does

A bot joins your Minecraft server and exposes 25 tools over [Model Context Protocol](https://modelcontextprotocol.io). Claude (or any MCP client) can then move, build, mine, craft, chat, and scan the world — all through natural language.

Ships with **Claude Code skills** — battle-tested building patterns so the LLM knows _how_ to build, not just _what tools exist_.

## What's Inside

```
├── src/
│   ├── main.ts              MCP server entrypoint
│   ├── bot-connection.ts    Mineflayer bot lifecycle
│   ├── config.ts            CLI arg parsing (--host, --port, --username)
│   └── tools/
│       ├── block-tools.ts       place, dig, scan, find blocks & signs
│       ├── chat-tools.ts        send & read chat messages
│       ├── crafting-tools.ts    recipes, crafting, ingredient checks
│       ├── entity-tools.ts      find entities
│       ├── flight-tools.ts      creative flight
│       ├── furnace-tools.ts     smelting
│       ├── gamestate-tools.ts   gamemode detection
│       ├── inventory-tools.ts   list, find, equip items
│       └── position-tools.ts    move, look, jump, teleport
├── .claude/skills/
│   ├── minecraft-player/        how to build efficiently with /fill
│   └── minecraft-urban-planner/ architectural patterns & material palettes
├── tools/world-scanner/         Go tool — scan .minecraft saves offline
└── tests/
```

## Skills — What Makes This Different

Most MCP servers give the LLM tools and hope for the best. This one ships opinionated skills learned from hundreds of hours of in-game iteration:

### `minecraft-player`
The building fundamentals. Stuff the LLM gets wrong without guidance:
- **Use `/fill`, not `place-block`** — one command places an entire wall; `place-block` needs chunk loading and has 1-block reach
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

Any Java Edition 1.21.11 server. For singleplayer, open to LAN (`ESC → Open to LAN`).

### 2. Configure Your MCP Client

**Claude Desktop** — edit `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "minecraft": {
      "command": "npx",
      "args": [
        "-y",
        "github:kaylincoded/clankercraft",
        "--host", "localhost",
        "--port", "25565",
        "--username", "ClaudeBot"
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
      "command": "npx",
      "args": [
        "-y",
        "github:kaylincoded/clankercraft",
        "--host", "localhost",
        "--port", "25565",
        "--username", "ClaudeBot"
      ]
    }
  }
}
```

### 3. Talk to It

> "Build a small Japanese-style house near my position"

The bot joins, Claude reads the skills, and building starts.

## Tools (25)

| Category | Tools |
|---|---|
| **Blocks** | `place-block` `dig-block` `get-block-info` `find-block` `scan-area` `read-sign` `find-signs` |
| **Movement** | `get-position` `move-to-position` `look-at` `jump` `move-in-direction` `fly-to` |
| **Inventory** | `list-inventory` `find-item` `equip-item` |
| **Crafting** | `list-recipes` `get-recipe` `can-craft` `craft-item` |
| **Chat** | `send-chat` `read-chat` |
| **World** | `detect-gamemode` `find-entity` `smelt-item` |

## World Scanner (Offline)

Analyze any `.minecraft` save without a running server. Written in Go, scans 15M blocks/sec.

```bash
cd tools/world-scanner
go build -o world-scanner .
./world-scanner "/path/to/world" --bounds minX minY minZ maxX maxY maxZ --json output.json
```

Use it to study reference builds before asking Claude to reproduce them.

## Config

```bash
cp .env.example .env
```

| Variable | Default | Description |
|---|---|---|
| `MC_HOST` | `localhost` | Minecraft server address |
| `MC_PORT` | `25565` | Server port |
| `MC_USERNAME` | `LLMBot` | Bot's in-game name |

These map to CLI args (`--host`, `--port`, `--username`) when running directly.

## Development

```bash
git clone git@github.com:kaylincoded/clankercraft.git
cd minecraft-mcp-server
npm install
npm run dev -- --host localhost --port 25565
```

## Contributing

PRs and issues welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
