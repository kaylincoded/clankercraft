# Story 2.1: MCP Stdio Transport

Status: review

## Story

As an MCP client (Claude Code/Desktop),
I want to communicate with clankercraft via JSON-RPC over stdin/stdout,
so that it works as a standard MCP tool server.

## Acceptance Criteria

1. **Given** the binary is started
   **When** a valid JSON-RPC `initialize` request arrives on stdin
   **Then** the server responds with its capabilities (server info, supported protocol version, tool list)

2. **Given** the MCP server is running
   **When** a `tools/list` request arrives
   **Then** all registered tools are returned with names, descriptions, and parameter schemas

3. **Given** a registered tool exists
   **When** a valid `tools/call` request arrives on stdin
   **Then** the server dispatches to the handler and writes the JSON-RPC response to stdout

4. **Given** stdout is used for MCP JSON-RPC
   **When** any non-MCP output would go to stdout (go-mc logging, etc.)
   **Then** all application logging goes to stderr only (already handled by slog setup)

5. **Given** the MCP server is running
   **When** SIGINT is received
   **Then** the MCP server shuts down cleanly alongside the MC connection

## Tasks / Subtasks

- [x] Task 1: Add official MCP Go SDK dependency (AC: #1)
  - [x] `go get github.com/modelcontextprotocol/go-sdk/mcp` (v1.4.1)
  - [x] Verify the dependency compiles and `go mod tidy` is clean
- [x] Task 2: Create `internal/mcp/server.go` — MCP server setup (AC: #1, #2)
  - [x] Create `internal/mcp/` package
  - [x] Define `Server` struct wrapping the SDK's MCP server
  - [x] `New(version, logger, conn)` — creates and configures the MCP server
  - [x] Register server info: name "clankercraft", version from build
  - [x] Register a `ping` tool as a smoke test (returns "pong" via typed output)
  - [x] `Run(ctx context.Context) error` — starts the stdio transport, blocks until ctx cancelled
- [x] Task 3: Stdio transport setup (AC: #3, #4)
  - [x] SDK's StdioTransport configured inline in server.go (no separate file needed)
  - [x] No application output leaks to stdout (slog writes to stderr, verified)
- [x] Task 4: Wire MCP server into main.go (AC: #5)
  - [x] Start MCP server in a goroutine alongside RunWithReconnect
  - [x] Both share the same context — SIGINT cancels both
  - [x] WaitGroup waits for both to finish before shutdown
- [x] Task 5: Write tests (all ACs)
  - [x] Test server creation and configuration
  - [x] Test ping tool registered and discoverable via tools/list
  - [x] Test ping tool returns expected response via CallTool
  - [x] Test server shuts down on context cancellation
  - [x] Existing tests still pass (43 total)

## Dev Notes

### MCP Go SDK (Official)

The architecture doc's assessment that "Go MCP SDK ecosystem is immature" is **outdated**. The official SDK is now production-ready:

- **Import:** `github.com/modelcontextprotocol/go-sdk` v1.4.1
- **Requires:** Go 1.25+ (check our go.mod — may need version bump)
- **Stdio transport:** Built-in `mcp.StdioTransport{}`
- **Tool registration:** Struct-based with `jsonschema` tags for automatic schema generation

```go
import "github.com/modelcontextprotocol/go-sdk/mcp"

// Create server
server := mcp.NewServer(&mcp.Implementation{
    Name:    "clankercraft",
    Version: "1.0.0",
}, nil)

// Register a tool with typed input
type PingInput struct{}
func pingHandler(ctx context.Context, req *mcp.CallToolRequest, input PingInput) (*mcp.CallToolResult, any, error) {
    return mcp.NewToolResultText("pong"), nil, nil
}
mcp.AddTool(server, &mcp.Tool{
    Name:        "ping",
    Description: "Check if the bot is responsive",
}, pingHandler)

// Run stdio transport
server.Run(ctx, &mcp.StdioTransport{})
```

**NOTE:** If the SDK requires Go 1.25+ and our go.mod has an older version, bump it. The SDK uses generics features from recent Go versions.

### Stdout Protection

All application logging already goes to stderr via slog (Story 1.1). The MCP SDK's StdioTransport reads stdin and writes to stdout. No filtering needed — just don't write to os.Stdout anywhere else.

go-mc-ms-auth uses `log.Print` which goes to stderr by default. Verified in Story 1.3.

### main.go Architecture After This Story

```go
func run(cmd *cobra.Command, args []string) error {
    // ... config, logger setup ...

    ctx, stop := signal.NotifyContext(...)
    defer stop()

    conn := connection.New(cfg, logger)
    mcpServer := mcp.New(logger, conn)

    // Run both concurrently
    g, gctx := errgroup.WithContext(ctx)
    g.Go(func() error { return conn.RunWithReconnect(gctx) })
    g.Go(func() error { return mcpServer.Run(gctx) })

    err := g.Wait()
    stop()
    logger.Info("shutting down, press Ctrl+C again to force quit")
    conn.Close()
    // ...
}
```

Consider using `golang.org/x/sync/errgroup` for concurrent goroutine management, or a simple goroutine+channel pattern if we want to avoid the dependency.

### Package Boundary

- `internal/mcp/` — MCP server, transport, tool registration
- `internal/mcp/` imports `internal/connection/` (to pass connection state to tools)
- `internal/connection/` does NOT import `internal/mcp/` (boundary rule from architecture)
- Tools that need connection data receive it via the Server struct, not by importing connection directly

### Previous Story Learnings (Story 1.5)

- Graceful shutdown: context cancellation propagates to all goroutines, Close() drains with timeout
- Force-quit on second signal via stop() before Close()
- Injectable fields (authFn, connectAndRun, backoffFn, shutdownTimeout) for testability
- 39 total tests across all packages, all passing

### What This Story Does NOT Do

- Register real observation tools (Stories 2.4-2.6)
- Tool registration framework with middleware (Story 2.2)
- Tool listing discovery (Story 2.3 — but the SDK handles `tools/list` automatically)
- Any construction/WorldEdit tools (Epic 3)
- In-game chat interface (Epic 5)

### Project Structure Notes

After this story:

```
internal/
├── config/
├── connection/
├── log/
└── mcp/
    ├── server.go        (new — MCP server setup, tool registration)
    ├── server_test.go   (new — server tests)
    └── transport.go     (new — stdio transport config)
main.go                  (modified — run MCP server alongside MC connection)
```

### References

- [Source: docs/architecture-decision.md#Dual Interface Architecture] — MCP stdio interface
- [Source: docs/architecture-decision.md#Technical Stack] — "Custom stdio JSON-RPC" (now replaced by official SDK)
- [Source: docs/architecture-decision.md#Project Structure] — internal/mcp/ package layout
- [Source: docs/epics.md#Story 2.1] — Original story definition
- [Source: docs/stories/1-5-graceful-shutdown.md] — Previous story learnings
- [External: github.com/modelcontextprotocol/go-sdk] — Official Go MCP SDK v1.4.1

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- Used official MCP Go SDK v1.4.1 (github.com/modelcontextprotocol/go-sdk/mcp) — architecture doc's "custom JSON-RPC" approach now outdated
- SDK provides struct-based tool schemas via jsonschema tags, automatic input validation, typed handlers
- Server.New() takes version string to pass build version to MCP implementation info
- ping tool uses ToolHandlerFor[pingInput, pingOutput] generic pattern — nil CallToolResult auto-populates from output struct
- StdioTransport configured inline — no separate transport.go needed (one-liner)
- main.go: sync.WaitGroup runs MCP server and MC connection concurrently, both share ctx
- Tests use NewInMemoryTransports() for fast in-process testing without stdio
- 43 total tests across all packages, all passing (4 new MCP tests)

### File List

- `internal/mcp/server.go` (new — MCP server, ping tool, stdio transport)
- `internal/mcp/server_test.go` (new — server creation, tool discovery, tool call, cancellation tests)
- `main.go` (modified — concurrent MCP + MC connection with WaitGroup)
- `go.mod` (modified — added MCP SDK and transitive deps)
- `go.sum` (modified)
