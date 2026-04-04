package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/kaylincoded/clankercraft/internal/connection"
)

// ConnChecker abstracts the connection state check for testability.
type ConnChecker interface {
	IsConnected() bool
}

// BotState extends ConnChecker with game state access for tools that need
// position data, rotation, or world queries.
type BotState interface {
	ConnChecker
	GetPosition() (connection.Position, bool)
	SendRotation(yaw, pitch float32) error
	BlockAt(x, y, z int) (string, error)
	FindBlock(blockType string, maxDist int) (bx, by, bz int, found bool, err error)
	ScanArea(x1, y1, z1, x2, y2, z2 int) ([]connection.BlockInfo, error)
	ReadSign(x, y, z int) (connection.SignText, string, error)
	FindSigns(maxDist int) ([]connection.SignInfo, error)
	GetGamemode() string
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
