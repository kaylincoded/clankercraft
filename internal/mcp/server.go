package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP SDK server with clankercraft-specific configuration.
type Server struct {
	server *gomcp.Server
	logger *slog.Logger
	conn   BotState
}

// pingInput is the input schema for the ping tool (no arguments).
type pingInput struct{}

// pingOutput is the output schema for the ping tool.
type pingOutput struct {
	Status string `json:"status"`
}

// statusInput is the input schema for the status tool (no arguments).
type statusInput struct{}

// statusOutput is the output schema for the status tool.
type statusOutput struct {
	Connected bool   `json:"connected"`
	Status    string `json:"status"`
}

// getPositionInput is the input schema for the get-position tool (no arguments).
type getPositionInput struct{}

// getPositionOutput is the output schema for the get-position tool.
type getPositionOutput struct {
	X     int     `json:"x"`
	Y     int     `json:"y"`
	Z     int     `json:"z"`
	Yaw   float32 `json:"yaw"`
	Pitch float32 `json:"pitch"`
}

// lookAtInput is the input schema for the look-at tool.
type lookAtInput struct {
	X float64 `json:"x" jsonschema:"X coordinate to look at"`
	Y float64 `json:"y" jsonschema:"Y coordinate to look at"`
	Z float64 `json:"z" jsonschema:"Z coordinate to look at"`
}

// lookAtOutput is the output schema for the look-at tool.
type lookAtOutput struct {
	Message string `json:"message"`
}

// New creates a configured MCP server with registered tools.
func New(version string, logger *slog.Logger, conn BotState) *Server {
	s := gomcp.NewServer(
		&gomcp.Implementation{
			Name:    "clankercraft",
			Version: version,
		},
		&gomcp.ServerOptions{
			Logger: logger,
		},
	)

	srv := &Server{
		server: s,
		logger: logger,
		conn:   conn,
	}
	srv.registerTools()
	return srv
}

// registerTools registers all MCP tools on the server.
func (s *Server) registerTools() {
	// ping — no connection required (MCP transport health check)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "ping",
		Description: "Check if the bot is responsive",
	}, s.handlePing)

	// status — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "status",
		Description: "Get the bot's connection status to the Minecraft server",
	}, requireConnection(s.conn, s.handleStatus))

	// get-position — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "get-position",
		Description: "Get the bot's current position and facing direction in the Minecraft world",
	}, requireConnection(s.conn, s.handleGetPosition))

	// look-at — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "look-at",
		Description: "Make the bot face toward the specified coordinates",
	}, requireConnection(s.conn, s.handleLookAt))
}

// handlePing is a smoke-test tool that returns "pong".
func (s *Server) handlePing(_ context.Context, _ *gomcp.CallToolRequest, _ pingInput) (*gomcp.CallToolResult, pingOutput, error) {
	return nil, pingOutput{Status: "pong"}, nil
}

// handleStatus returns the bot's connection status. Only reachable when connected (middleware enforced).
func (s *Server) handleStatus(_ context.Context, _ *gomcp.CallToolRequest, _ statusInput) (*gomcp.CallToolResult, statusOutput, error) {
	return nil, statusOutput{Connected: true, Status: "connected"}, nil
}

// handleGetPosition returns the bot's current tracked position.
func (s *Server) handleGetPosition(_ context.Context, _ *gomcp.CallToolRequest, _ getPositionInput) (*gomcp.CallToolResult, getPositionOutput, error) {
	pos, ok := s.conn.GetPosition()
	if !ok {
		return toolError("position not yet known (waiting for server)"), getPositionOutput{}, nil
	}
	return nil, getPositionOutput{
		X:     int(math.Floor(pos.X)),
		Y:     int(math.Floor(pos.Y)),
		Z:     int(math.Floor(pos.Z)),
		Yaw:   pos.Yaw,
		Pitch: pos.Pitch,
	}, nil
}

// handleLookAt rotates the bot to face the target coordinates.
func (s *Server) handleLookAt(_ context.Context, _ *gomcp.CallToolRequest, input lookAtInput) (*gomcp.CallToolResult, lookAtOutput, error) {
	pos, ok := s.conn.GetPosition()
	if !ok {
		return toolError("position not yet known (waiting for server)"), lookAtOutput{}, nil
	}

	yaw, pitch := calcYawPitch(pos.X, pos.Y, pos.Z, input.X, input.Y, input.Z)
	if err := s.conn.SendRotation(yaw, pitch); err != nil {
		return toolError(fmt.Sprintf("failed to send rotation: %v", err)), lookAtOutput{}, nil
	}
	return nil, lookAtOutput{
		Message: fmt.Sprintf("Now looking at (%.0f, %.0f, %.0f)", input.X, input.Y, input.Z),
	}, nil
}

// calcYawPitch computes Minecraft yaw and pitch from a source to a target position.
// Yaw: 0 = +Z (south), 90 = -X (west), 180 = -Z (north), 270 = +X (east).
// Pitch: -90 = up, 0 = horizontal, 90 = down.
func calcYawPitch(fromX, fromY, fromZ, toX, toY, toZ float64) (yaw, pitch float32) {
	dx := toX - fromX
	dy := toY - fromY
	dz := toZ - fromZ
	dist := math.Sqrt(dx*dx + dz*dz)
	yaw = float32(-math.Atan2(dx, dz) * 180 / math.Pi)
	pitch = float32(-math.Atan2(dy, dist) * 180 / math.Pi)
	return yaw, pitch
}

// Run starts the MCP stdio transport. Blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting MCP server")
	return s.server.Run(ctx, &gomcp.StdioTransport{})
}
