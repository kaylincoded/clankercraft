# Story 6.3: Schematic Directory Loading

Status: done

## Story

As a user,
I want the bot to load schematics from `~/.config/clankercraft/schematics/` on startup,
so that I can use saved building components.

## Acceptance Criteria

1. **Given** `.schem` files exist in the schematics directory
   **When** the bot starts
   **Then** it indexes available schematics and makes them available as a tool

2. **Given** the schematics directory doesn't exist
   **When** the bot starts
   **Then** it creates the directory and continues with no schematics loaded

## Tasks / Subtasks

- [x] Task 1: Create schematic library package (AC: #1, #2)
  - [x] Create `internal/schematic/library.go`
  - [x] Implement `Library` struct with `schematics map[string]SchematicInfo` (name → metadata)
  - [x] `SchematicInfo` struct: `Name string`, `Path string`, `SizeBytes int64`
  - [x] `NewLibrary(dir string, logger *slog.Logger) *Library` — constructor
  - [x] `Load() error` — ensures directory exists (creates with `os.MkdirAll` 0755 if missing), scans for `*.schem` files, populates index
  - [x] `List() []SchematicInfo` — returns indexed schematics sorted by name
  - [x] `Has(name string) bool` — check if schematic exists
  - [x] `Path(name string) (string, error)` — return full path for a schematic name
  - [x] Schematic names are filename without `.schem` extension (e.g., `torii_gate.schem` → `torii_gate`)
- [x] Task 2: Add schematic tools to agent (AC: #1)
  - [x] Add `list-schematics` tool def to `tooldefs.go` (no params, returns list of available schematics)
  - [x] Add `load-schematic` tool def to `tooldefs.go` (param: `name` string — the schematic to load and paste)
  - [x] Add `list-schematics` case in `tools.go` Execute() — calls `Library.List()`, returns JSON array
  - [x] Add `load-schematic` case in `tools.go` Execute() — validates name exists via `Library.Has()`, then runs `//schem load <name>` + `//paste` via WorldEdit commands
  - [x] Update `ToolExecutor` to accept `*schematic.Library` (add field + constructor param)
  - [x] Update tool count comment in `tooldefs.go` (35 → 37)
- [x] Task 3: Add schematic tools to MCP server (AC: #1)
  - [x] Add `list-schematics` handler to `server.go` — returns indexed schematics
  - [x] Add `load-schematic` handler to `server.go` — validates name, runs `//schem load` + `//paste`
  - [x] Register both tools in `registerTools()` with `gomcp.AddTool()`
  - [x] `load-schematic` requires WorldEdit tier (use `requireWorldEdit()` middleware)
  - [x] Pass `*schematic.Library` to MCP `Server` struct
- [x] Task 4: Wire schematic library into main.go (AC: #1, #2)
  - [x] Create `schematic.NewLibrary(schematicsDir, logger)` in `main.go`
  - [x] Call `library.Load()` — log warning and continue if error (non-fatal)
  - [x] Log number of schematics indexed at startup
  - [x] Pass library to `agent.NewToolExecutor(conn, library)`
  - [x] Pass library to `mcp.New(version, logger, conn, library)`
  - [x] Schematics directory: `~/.config/clankercraft/schematics/`
- [x] Task 5: Write tests (AC: #1, #2)
  - [x] `internal/schematic/library_test.go`:
    - [x] Test Load with existing `.schem` files (use `os.MkdirTemp` + create dummy files)
    - [x] Test Load creates directory when missing
    - [x] Test List returns sorted schematics
    - [x] Test Has returns true/false correctly
    - [x] Test Path returns correct path / error for missing
    - [x] Test non-`.schem` files are ignored
  - [x] `internal/agent/tools_test.go`:
    - [x] Test `list-schematics` returns empty array when no schematics
    - [x] Test `list-schematics` returns schematics when loaded
    - [x] Test `load-schematic` with invalid name returns error
    - [x] Update TestToolDefsCount from 35 to 37
- [x] Task 6: Validate locally (AC: #1, #2)
  - [x] All tests pass (`go test ./...`)
  - [x] Lint passes (`golangci-lint run ./...`)
  - [x] Build succeeds (`go build -o /dev/null .`)

## Dev Notes

### Scope: Index-Only, No Parsing

This story indexes `.schem` files by filename — it does NOT parse the NBT/schematic format. The bot uses WorldEdit's `//schem load` command to actually load schematics server-side. The library is a directory index, not a file parser.

### WorldEdit Schematic Commands

WorldEdit's schematic commands expect files in the server's `worldedit/schematics/` directory, NOT the bot's local directory. For `//schem load <name>` to work, schematics must be on the server filesystem.

**Important design decision:** The `load-schematic` tool should:
1. Verify the schematic name exists in the bot's local index (prevents blind command injection)
2. Run `//schem load <name>` via WorldEdit (assumes server has a matching file)
3. Run `//paste` to place the loaded schematic

The local index serves as a validation layer and discovery tool — the LLM can ask "what schematics are available?" and get a list, then request one by name. The actual loading is done by WorldEdit on the server side.

If the server doesn't have the schematic, WorldEdit will return an error which propagates back to the LLM.

### Schematic Name Validation

Schematic names derive from filenames. Validate names before interpolation into `//schem load <name>`:
- Only allow `[a-zA-Z0-9_-]` characters (no spaces, no path separators, no dots)
- This prevents path traversal attacks (`../../etc/passwd`) in the WorldEdit command

Use a validation function similar to `isValidPlayerName` in `tools.go:674`.

### Directory Pattern

Follow the established pattern from `internal/connection/auth.go:59-76`:

```go
dir := filepath.Join(home, ".config", "clankercraft", "schematics")
if err := os.MkdirAll(dir, 0755); err != nil {
    return fmt.Errorf("creating schematics directory: %w", err)
}
```

Use `0755` (not `0700`) — schematics are not sensitive data (unlike auth tokens).

### ToolExecutor Changes

`ToolExecutor` currently takes only `BotState`:

```go
type ToolExecutor struct {
    bot mcp.BotState
}
func NewToolExecutor(bot mcp.BotState) *ToolExecutor {
```

Add `library *schematic.Library` field. The library can be `nil` (no schematics loaded) — tool handlers should check for nil and return a helpful error.

### MCP Server Changes

`mcp.New()` currently takes `(version string, logger *slog.Logger, bot BotState)`. Add `library *schematic.Library` parameter. Same nil-safety pattern.

### Command Routing

`//schem load` is a player-session command (uses bot's clipboard), so it goes through **chat** not RCON. `//paste` also goes through chat. Use `RunWECommand()` (not `RunBulkWECommand()`).

### What NOT to Build

- **No schematic parsing** — don't parse NBT or `.schem` binary format
- **No schematic saving** — `//schem save` is a Growth feature, not MVP
- **No schematic uploading/syncing** — no copying files to server
- **No subdirectory recursion** — only scan top-level `schematics/` directory
- **No file watching** — index once at startup, no hot-reload

### Previous Story Intelligence

- Story 6-2: golangci-lint v2 config requires `version: "2"`, `gosimple` merged into `staticcheck`, exclusions under `linters.exclusions`
- Story 6-2: All errcheck issues in production code now fixed — maintain this standard
- Story 5-6: `isValidPlayerName` pattern for input validation before command interpolation (`tools.go:674`)
- Story 5-6: Tool count is now 35 (will become 37)
- Story 5-4: Agent tool executor pattern — switch/case in `Execute()`, `jsonResult()` helper
- `ensureCacheDir()` in `auth.go:59-76` — established pattern for `~/.config/clankercraft/` subdirectory creation
- `weTierCmd()` in `tools.go:620-647` — helper for routing WE commands through correct channel
- Tool registration in both `tooldefs.go` (LLM agent) and `server.go` (MCP) — both must be updated for new tools

### References

- [Source: docs/epics.md#Story 6.3] — Original story definition
- [Source: docs/prd.md#FR24] — WorldEdit schematic management (`//schem save/load/list`)
- [Source: docs/prd.md#FR46] — Load user schematics from configurable directory on startup
- [Source: docs/architecture-decision.md#Data Storage] — Schematics stored at `~/.config/clankercraft/schematics/`
- [Source: internal/connection/auth.go#L59-76] — Directory creation pattern (`ensureCacheDir`)
- [Source: internal/agent/tools.go#L674-686] — Input validation pattern (`isValidPlayerName`)
- [Source: internal/agent/tools.go#L620-647] — WE command routing (`weTierCmd`)
- [Source: internal/agent/tooldefs.go] — Tool definition pattern (35 current tools)
- [Source: internal/mcp/server.go#L370-574] — MCP tool registration pattern
- [Source: main.go#L86-104] — Component wiring flow

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- `internal/schematic/library.go` — New package: indexes `.schem` files by filename, creates directory if missing, provides List/Has/Path methods
- 2 new tools added (`list-schematics`, `load-schematic`) — tool count 35 → 37
- `load-schematic` validates name with `isValidSchematicName` (alphanumeric + underscore + hyphen only) to prevent command injection
- `load-schematic` runs `//schem load <name>` then `//paste` via `RunWECommand()` (chat, not RCON — clipboard is player-session)
- Both agent (`tools.go`/`tooldefs.go`) and MCP (`server.go`) tool systems updated
- `ToolExecutor` and `mcp.Server` now accept optional `*schematic.Library` (nil-safe)
- Library wired in `main.go` — loads from `~/.config/clankercraft/schematics/`, non-fatal on error
- 6 library tests + 4 agent tool tests + TestIsValidSchematicName
- All tests pass, golangci-lint 0 issues, build succeeds

### File List
- `internal/schematic/library.go` — NEW: Schematic directory indexer
- `internal/schematic/library_test.go` — NEW: 6 library tests
- `internal/agent/tools.go` — MODIFIED: Added `list-schematics`/`load-schematic` cases, `isValidSchematicName`, library field on ToolExecutor
- `internal/agent/tooldefs.go` — MODIFIED: Added 2 tool defs (37 total)
- `internal/agent/tools_test.go` — MODIFIED: Added 4 schematic tool tests, updated tool count, fixed NewToolExecutor calls
- `internal/agent/agent_test.go` — MODIFIED: Fixed NewToolExecutor calls (added nil library param)
- `internal/mcp/server.go` — MODIFIED: Added list-schematics/load-schematic handlers, library field on Server
- `internal/mcp/server_test.go` — MODIFIED: Fixed New() calls (added nil library param)
- `main.go` — MODIFIED: Added schematic library initialization and wiring
- `docs/stories/6-3-schematic-directory-loading.md` — MODIFIED: Task completion and Dev Agent Record
- `docs/stories/sprint-status.yaml` — MODIFIED: Story 6-3 status
