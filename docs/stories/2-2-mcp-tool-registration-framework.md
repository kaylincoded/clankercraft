# Story 2.2: MCP Tool Registration Framework

Status: done

## Story

As a developer,
I want a tool registration pattern with connection-check middleware and structured error responses,
so that all tools share consistent error handling and connection awareness.

## Acceptance Criteria

1. **Given** a tool is registered with a JSON schema
   **When** a tool call arrives with invalid arguments
   **Then** a structured error response is returned without crashing

2. **Given** a tool is called while the bot is disconnected
   **When** the connection check runs
   **Then** an error response is returned indicating the bot is not connected

3. **Given** a tool is called while the bot is connected
   **When** the connection check passes
   **Then** the tool handler executes normally

4. **Given** any tool handler returns an error
   **When** the error propagates through the framework
   **Then** it is wrapped in a consistent MCP error response with `isError: true`

5. **Given** a tool is registered via the framework
   **When** `tools/list` is requested
   **Then** the tool appears with its name, description, and parameter schema (SDK handles this)

## Tasks / Subtasks

- [x] Task 1: Create `internal/mcp/middleware.go` — connection-check middleware (AC: #2, #3)
  - [x] Define `ConnChecker` interface: `IsConnected() bool` (Connection already implements this)
  - [x] Create `requireConnection` wrapper that returns a typed MCP error if `!IsConnected()`
  - [x] Wrapper is generic — works with any `ToolHandlerFor` signature
- [x] Task 2: Create `internal/mcp/errors.go` — structured error responses (AC: #1, #4)
  - [x] Define `toolError(msg string) *mcp.CallToolResult` helper — returns `{isError: true, content: [TextContent]}`
  - [x] Standardize error text format for MCP clients to parse
- [x] Task 3: Refactor `server.go` — use framework for tool registration (AC: #5)
  - [x] Update `registerTools()` to use middleware wrappers where appropriate
  - [x] `ping` tool does NOT require connection (it's a smoke test) — skip middleware for it
  - [x] Added `status` tool using `requireConnection` middleware as the pattern for future tools
- [x] Task 4: Write tests (all ACs)
  - [x] Test connection-check middleware rejects when disconnected
  - [x] Test connection-check middleware allows when connected
  - [x] Test error helper produces correct MCP error response format
  - [x] Test ping tool still works without connection middleware
  - [x] Test status tool returns error when disconnected (integration via InMemoryTransports)
  - [x] Test status tool succeeds when connected (integration via InMemoryTransports)
  - [x] Existing tests still pass (50 total — 7 new)

## Dev Notes

### SDK Argument Validation (AC #1)

The MCP Go SDK already handles argument validation automatically via typed input structs and `jsonschema` tags. When a `tools/call` arrives with invalid arguments, the SDK returns a JSON-RPC error before the handler is invoked. **We don't need to build argument validation** — AC #1 is satisfied by the SDK.

Verify this in tests by calling a tool with malformed arguments and confirming the SDK returns an appropriate error.

### Connection-Check Middleware Pattern

The key decision is how to wrap tool handlers. The SDK uses this signature for typed handlers:

```go
func handler(ctx context.Context, req *mcp.CallToolRequest, input T) (*mcp.CallToolResult, O, error)
```

A `requireConnection` wrapper should:
1. Check `s.conn.IsConnected()` before calling the inner handler
2. Return `toolError("bot is not connected to a Minecraft server")` if disconnected
3. Pass through to the inner handler if connected

```go
// ConnChecker abstracts the connection state check for testability.
type ConnChecker interface {
    IsConnected() bool
}

// requireConnection wraps a typed tool handler with a connection check.
func requireConnection[I, O any](checker ConnChecker, handler mcp.ToolHandlerFor[I, O]) mcp.ToolHandlerFor[I, O] {
    return func(ctx context.Context, req *mcp.CallToolRequest, input I) (*mcp.CallToolResult, O, error) {
        if !checker.IsConnected() {
            var zero O
            return toolError("bot is not connected to a Minecraft server"), zero, nil
        }
        return handler(ctx, req, input)
    }
}
```

**Note:** `ToolHandlerFor[I, O]` is a type alias in the SDK:
```go
type ToolHandlerFor[I, O any] func(context.Context, *CallToolRequest, I) (*CallToolResult, O, error)
```

Verify this type exists in the SDK before using it. If it doesn't, use the raw function signature directly.

### Error Response Pattern

MCP error responses use `isError: true` with text content:

```go
func toolError(msg string) *mcp.CallToolResult {
    return &mcp.CallToolResult{
        IsError: true,
        Content: []mcp.Content{&mcp.TextContent{Text: msg}},
    }
}
```

This is the standard MCP pattern — clients (Claude Code, Claude Desktop) display the text to the user/LLM.

### What Doesn't Need Middleware

- `ping` — intentionally works without connection (it's a health check for the MCP transport itself)
- Future tools that are purely informational about the MCP server (if any)

### Testing Strategy

Use a mock `ConnChecker` in tests:

```go
type mockConnChecker struct{ connected bool }
func (m *mockConnChecker) IsConnected() bool { return m.connected }
```

Test the middleware independently from the MCP server — no need for in-memory transports for middleware unit tests. Integration tests (tool call through transport → middleware → handler) use the existing InMemoryTransports pattern from Story 2-1.

### Previous Story Learnings (Story 2.1)

- SDK uses `gomcp.AddTool(server, &gomcp.Tool{...}, handler)` for typed registration
- `NewInMemoryTransports()` for fast in-process testing
- Returning `nil` CallToolResult with a typed output auto-populates content from output struct
- Returning a non-nil CallToolResult takes precedence over the output struct

### Package Boundary (Improved)

- `internal/mcp/` no longer imports `internal/connection/` — fully decoupled via ConnChecker interface
- `internal/connection/` does NOT import `internal/mcp/` (unchanged)
- `*connection.Connection` satisfies `mcp.ConnChecker` implicitly (structural typing)

### What This Story Does NOT Do

- Register real observation tools (Stories 2.4-2.6)
- Implement reconnection-on-tool-call (epic AC says "attempts reconnection" but that adds complexity — for now, return error; reconnection happens via RunWithReconnect)
- Rate limiting or tool authorization
- Any construction/WorldEdit tools (Epic 3)

### Project Structure After This Story

```
internal/mcp/
├── server.go           (modified — uses middleware for registration pattern)
├── server_test.go      (modified — additional middleware integration tests)
├── middleware.go        (new — requireConnection wrapper)
├── middleware_test.go   (new — middleware unit tests)
└── errors.go           (new — toolError helper)
```

### References

- [Source: docs/epics.md#Story 2.2] — Original story definition
- [Source: docs/stories/2-1-mcp-stdio-transport.md] — Previous story learnings
- [Source: internal/connection/mc.go#IsConnected] — Connection state check method
- [External: github.com/modelcontextprotocol/go-sdk] — Official Go MCP SDK v1.4.1

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- `ConnChecker` interface decouples mcp package from concrete connection type — `*connection.Connection` satisfies it implicitly
- `requireConnection[I, O any]` uses SDK's `ToolHandlerFor[I, O]` generic type alias for type-safe wrapping
- `toolError()` returns non-nil `*CallToolResult` with `IsError: true` — takes precedence over typed output (SDK behavior)
- `status` tool added as a real middleware-wrapped tool (not just a test fixture) — useful for MCP clients to check bot state
- `Server.conn` field changed from `*connection.Connection` to `ConnChecker` interface — no import of connection package needed
- `ping` explicitly skips middleware — it's a transport health check, not a game state query
- 50 total tests across all packages (7 new: 3 middleware unit + 4 server integration)

### File List

- `internal/mcp/middleware.go` (new — ConnChecker interface, requireConnection generic wrapper)
- `internal/mcp/middleware_test.go` (new — 3 unit tests for middleware and error helper)
- `internal/mcp/errors.go` (new — toolError helper)
- `internal/mcp/server.go` (modified — ConnChecker interface, status tool with middleware)
- `internal/mcp/server_test.go` (modified — testSession helper, 4 new integration tests)
