package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConnChecker abstracts the connection state check for testability.
type ConnChecker interface {
	IsConnected() bool
}

// requireConnection wraps a typed tool handler with a connection check.
// If the bot is not connected, it returns an MCP error without calling the handler.
func requireConnection[I, O any](checker ConnChecker, handler gomcp.ToolHandlerFor[I, O]) gomcp.ToolHandlerFor[I, O] {
	return func(ctx context.Context, req *gomcp.CallToolRequest, input I) (*gomcp.CallToolResult, O, error) {
		if !checker.IsConnected() {
			var zero O
			return toolError("bot is not connected to a Minecraft server"), zero, nil
		}
		return handler(ctx, req, input)
	}
}
