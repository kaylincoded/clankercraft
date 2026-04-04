# Story 1.1: Project Initialization & CLI Scaffolding

Status: done

## Story

As a developer,
I want a Go module with cobra/viper CLI scaffolding and a working binary that parses flags and loads layered config,
so that the v3 project has a runnable foundation for all subsequent epics to build on.

## Acceptance Criteria

1. **Given** the binary is invoked with `--host`, `--port`, `--username` flags
   **When** it starts
   **Then** it parses all flags, merges with env vars and config file (`~/.config/clankercraft/config.yaml`), and logs the resolved configuration to stderr as structured JSON

2. **Given** no flags or config are provided
   **When** the binary starts
   **Then** it uses defaults: host=localhost, port=25565, username=LLMBot

3. **Given** `config.yaml` sets `host: server1` and CLI flag sets `--host server2`
   **When** config is resolved
   **Then** CLI flag wins: host = server2

4. **Given** `CLANKERCRAFT_HOST=envhost` is set and config.yaml has `host: filehost`
   **When** config is resolved
   **Then** env var wins over config file (priority: CLI > env > file > defaults)

5. **Given** the binary starts
   **When** initialization completes
   **Then** it logs a startup message to stderr with resolved config (host, port, username) and exits cleanly (placeholder — no MC connection yet)

6. **Given** `--version` flag is passed
   **When** the binary starts
   **Then** it prints the version string and exits

7. **Given** SIGINT or SIGTERM is received
   **When** the binary is running
   **Then** it shuts down gracefully via context cancellation

## Tasks / Subtasks

- [x] Task 1: Initialize Go module and project structure (AC: #5)
  - [x] `go mod init github.com/kaylincoded/clankercraft`
  - [x] Create directory layout: `internal/config/`, `internal/log/`, `cmd/` (or flat main.go)
  - [x] Add `.gitignore` for Go (binaries, vendor/, etc.)
- [x] Task 2: Implement cobra root command with all CLI flags (AC: #1, #2, #6)
  - [x] Root command with `--host`, `--port`, `--username`, `--log-level`, `--version`
  - [x] RCON flags: `--rcon-port`, `--rcon-password` (used by later epics, define now)
  - [x] `--offline` flag (boolean, used by Story 1.3)
  - [x] `--config` flag for custom config file path
- [x] Task 3: Implement viper layered config (AC: #1, #2, #3, #4)
  - [x] Bind cobra flags to viper
  - [x] Set env prefix `CLANKERCRAFT_` with automatic env binding
  - [x] Set config file path: `~/.config/clankercraft/config.yaml`
  - [x] Set all defaults (host=localhost, port=25565, username=LLMBot, log-level=info)
  - [x] Create `internal/config/config.go` with a `Config` struct and `Load() (*Config, error)`
- [x] Task 4: Implement structured logging (AC: #5)
  - [x] Create `internal/log/log.go` using `log/slog` with `slog.NewJSONHandler(os.Stderr, opts)`
  - [x] Configurable level from `--log-level` flag (debug, info, warn, error)
  - [x] Log resolved config on startup (mask sensitive fields like rcon-password)
- [x] Task 5: Implement graceful shutdown (AC: #7)
  - [x] `signal.NotifyContext` for SIGINT/SIGTERM
  - [x] Pass context through to future components (connection, MCP server)
  - [x] Log shutdown initiation
- [x] Task 6: Write tests (all ACs)
  - [x] Config layering priority test (CLI > env > file > defaults)
  - [x] Default values test
  - [x] Version flag test
  - [x] Log level configuration test

## Dev Notes

### Architecture Compliance

- **Entry point:** `main.go` at project root. Wires cobra command, loads config, initializes logger, waits for signal.
- **Internal packages:** All application code under `internal/` (unexported). `internal/config/` for config, `internal/log/` for logging.
- **No `cmd/` package pattern** — single binary, keep it simple. Root command in `main.go`, config logic in `internal/config/`.
- **Config struct is the single source of truth** — all components receive config values, never call viper directly.

### Technical Stack (pinned versions)

| Dependency | Version | Import Path |
|---|---|---|
| Go | 1.22+ (use latest stable, currently 1.26.x) | — |
| cobra | v1.10.2 | `github.com/spf13/cobra` |
| viper | v1.21.0 | `github.com/spf13/viper` |
| slog | stdlib | `log/slog` (since Go 1.21) |
| go-mc | v1.20.2+ | `github.com/Tnze/go-mc` (added in Story 1.2, not this story) |

**Do NOT add go-mc in this story** — that's Story 1.2. This story only needs cobra, viper, and stdlib.

### Config File Format

```yaml
# ~/.config/clankercraft/config.yaml
host: localhost
port: 25565
username: LLMBot
log_level: info
offline: false
rcon_port: 25575
rcon_password: ""
```

### Environment Variable Mapping

| Config Key | Env Var | CLI Flag |
|---|---|---|
| host | CLANKERCRAFT_HOST | --host |
| port | CLANKERCRAFT_PORT | --port |
| username | CLANKERCRAFT_USERNAME | --username |
| log_level | CLANKERCRAFT_LOG_LEVEL | --log-level |
| offline | CLANKERCRAFT_OFFLINE | --offline |
| rcon_port | CLANKERCRAFT_RCON_PORT | --rcon-port |
| rcon_password | CLANKERCRAFT_RCON_PASSWORD | --rcon-password |

### Viper-Cobra Integration Pattern

```go
// In main.go or internal/config/config.go:
// 1. Define cobra flags
// 2. Bind flags to viper keys: viper.BindPFlag("host", cmd.Flags().Lookup("host"))
// 3. Set env prefix: viper.SetEnvPrefix("CLANKERCRAFT")
// 4. Enable automatic env: viper.AutomaticEnv()
// 5. Replace hyphens with underscores for env: viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
// 6. Set config file: viper.SetConfigFile(path) or viper.AddConfigPath + SetConfigName
// 7. viper.ReadInConfig() — ignore "not found" error (config file is optional)
// 8. Unmarshal into Config struct: viper.Unmarshal(&config)
```

### Logging Pattern

```go
// internal/log/log.go
func Setup(level string) *slog.Logger {
    var lvl slog.Level
    switch strings.ToLower(level) {
        case "debug": lvl = slog.LevelDebug
        case "warn":  lvl = slog.LevelWarn
        case "error": lvl = slog.LevelError
        default:      lvl = slog.LevelInfo
    }
    return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
```

All logging MUST go to stderr. stdout is reserved for the MCP JSON-RPC stream (Story 2.1).

### Security Notes

- RCON password MUST be masked in startup log output (show `"***"` not the value)
- API keys (added in later stories) MUST NEVER appear in logs
- Config file permissions: warn if `config.yaml` is world-readable and contains rcon_password

### Existing Codebase Context

The world scanner at `tools/world-scanner/` uses go-mc v1.20.2 with Go 1.26.1. It's a standalone module (`module world-scanner`), not part of the main project. The v3 project should be a new Go module at the repo root: `module github.com/kaylincoded/clankercraft`. The world scanner will remain separate for now (embedded in a later epic).

### Critical go-mc Intelligence (for later stories, NOT this one)

- **go-mc latest is v1.20.2** — pinned to MC 1.20.2 protocol (version 767). No 1.21.x support tagged yet. PRs #294-297 are attempting 1.21.11 support but are incomplete.
- **MSA auth is NOT in go-mc** — only old Yggdrasil auth + a slot to pass a pre-obtained access token. Story 1.3 will need a separate Microsoft Device Code OAuth2 flow (e.g., `golang.org/x/oauth2` against Microsoft endpoints).
- **Spike gate risk:** The day-1 spike (Story 1.2) should validate whether go-mc's 1.20.2 protocol connects to a 1.21.x server, or whether we need the in-progress 1.21.11 fork.
- **This story does NOT depend on go-mc.** Only cobra (v1.10.2), viper (v1.21.0), and stdlib.

The v2 TypeScript bootstrap sequence (for reference, not to replicate):
1. setupStdioFiltering() → 2. parseConfig(yargs) → 3. MessageStore → 4. BotConnection → 5. connect() → 6. McpServer → 7. ToolFactory → 8. registerTools → 9. server.connect(StdioTransport)

This story covers the Go equivalent of steps 1-2 only (config + stdio protection setup).

### Testing Standards

- Test framework: Go stdlib `testing` package
- Test files: `*_test.go` in same package (white-box)
- Run: `go test ./...`
- Config tests should use `t.Setenv()` for env var testing (auto-cleaned up)
- Use `t.TempDir()` for config file tests

### Project Structure Notes

After this story, the project should look like:

```
clankercraft/
├── main.go                      # cobra root command, startup, signal handling
├── go.mod                       # module github.com/kaylincoded/clankercraft
├── go.sum
├── internal/
│   ├─�� config/
│   │   ├── config.go            # Config struct, Load(), viper integration
│   │   └── config_test.go       # Layering priority tests, defaults tests
│   └── log/
│       ��── log.go               # slog setup, JSON handler to stderr
│       └── log_test.go          # Level configuration tests
├── tools/
│   └── world-scanner/           # Existing, untouched
├── docs/                        # Existing, untouched
├── src/                         # Existing v2 TypeScript, untouched
└── ...                          # Existing v2 files, untouched
```

**Do NOT delete or modify any existing v2 files.** The Go and TypeScript codebases coexist until v3 ships.

### References

- [Source: docs/architecture-decision.md#Technical Stack] — Go 1.22+, cobra, viper, slog
- [Source: docs/architecture-decision.md#Configuration Layering] — Priority order, viper pattern
- [Source: docs/architecture-decision.md#Logging Pattern] — slog JSON to stderr
- [Source: docs/architecture-decision.md#Project Structure & Boundaries] — Directory layout
- [Source: docs/prd.md#Configuration] — CLI > env > file > defaults
- [Source: docs/prd.md#FR44] — Layered config requirement
- [Source: docs/prd.md#FR45] — Single binary distribution
- [Source: docs/prd.md#NFR5] — Sub-2s startup
- [Source: docs/prd.md#NFR18-19] — Credential security requirements
- [Source: docs/epics.md#Story 1.1] — Original story definition

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- Initialized Go module `github.com/kaylincoded/clankercraft` with cobra v1.10.2 + viper v1.21.0
- Config layering verified: CLI > env > config file > defaults (all 4 layers tested)
- RCON password masking implemented and tested
- Structured JSON logging to stderr via slog (stdout reserved for MCP)
- Graceful shutdown via `signal.NotifyContext` with context propagation
- Existing v2 TypeScript files untouched — coexistence preserved
- Version flag injected via `var version` (goreleaser will set via ldflags)
- 16 tests total, all passing

### Code Review Fixes Applied

- **M1 — Malformed default config silently ignored**: `config.go` now returns an error for any non-NotFound config read failure, regardless of whether the path was explicit or default. Users with a typo in `~/.config/clankercraft/config.yaml` will now see the parse error.
- **M2 — Version flag test missing**: Added `TestVersionFlag` to `config_test.go` — executes root command with `--version`, verifies output contains the version string.

### File List

- `main.go` (new)
- `go.mod` (new)
- `go.sum` (new)
- `internal/config/config.go` (new)
- `internal/config/config_test.go` (new)
- `internal/log/log.go` (new)
- `internal/log/log_test.go` (new)
- `.gitignore` (modified)
