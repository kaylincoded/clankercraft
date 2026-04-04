package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/kaylincoded/clankercraft/internal/connection"
	"github.com/kaylincoded/clankercraft/internal/engine"
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
	GetTier() engine.Tier
	SetSelection(x1, y1, z1, x2, y2, z2 int) error
	GetSelection() (engine.Selection, bool)
	HasPos1() bool
	HasPos2() bool
	RunWECommand(command string) (string, error)
	RunCommand(command string) (string, error)
	RunBulkWECommand(command string) (string, error)
	RunBulkCommand(command string) (string, error)
}

// requireWETier wraps a handler with connection and WorldEdit tier checks (no selection required).
// Use for position-based commands like //sphere, //cyl, //pyramid.
func requireWETier[I, O any](bot BotState, handler gomcp.ToolHandlerFor[I, O]) gomcp.ToolHandlerFor[I, O] {
	return func(ctx context.Context, req *gomcp.CallToolRequest, input I) (*gomcp.CallToolResult, O, error) {
		var zero O
		if !bot.IsConnected() {
			return toolError("bot is not connected to a Minecraft server"), zero, nil
		}
		tier := bot.GetTier()
		if tier != engine.TierWorldEdit && tier != engine.TierFAWE {
			return toolError("WorldEdit is not available on this server (tier: " + tier.String() + ")"), zero, nil
		}
		return handler(ctx, req, input)
	}
}

// requireWorldEdit wraps a handler with connection, WorldEdit tier, and selection checks.
func requireWorldEdit[I, O any](bot BotState, handler gomcp.ToolHandlerFor[I, O]) gomcp.ToolHandlerFor[I, O] {
	return func(ctx context.Context, req *gomcp.CallToolRequest, input I) (*gomcp.CallToolResult, O, error) {
		var zero O
		if !bot.IsConnected() {
			return toolError("bot is not connected to a Minecraft server"), zero, nil
		}
		tier := bot.GetTier()
		if tier != engine.TierWorldEdit && tier != engine.TierFAWE {
			return toolError("WorldEdit is not available on this server (tier: " + tier.String() + ")"), zero, nil
		}
		if _, ok := bot.GetSelection(); !ok {
			return toolError("no selection set — use set-selection or wand to select a region first"), zero, nil
		}
		return handler(ctx, req, input)
	}
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
