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

// getBlockInfoInput is the input schema for the get-block-info tool.
type getBlockInfoInput struct {
	X int `json:"x" jsonschema:"X coordinate"`
	Y int `json:"y" jsonschema:"Y coordinate"`
	Z int `json:"z" jsonschema:"Z coordinate"`
}

// getBlockInfoOutput is the output schema for the get-block-info tool.
type getBlockInfoOutput struct {
	Block string `json:"block"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Z     int    `json:"z"`
}

// findBlockInput is the input schema for the find-block tool.
type findBlockInput struct {
	BlockType   string `json:"blockType" jsonschema:"block type to search for, e.g. minecraft:stone"`
	MaxDistance int    `json:"maxDistance,omitempty" jsonschema:"max search distance in blocks (default 16, max 64)"`
}

// findBlockOutput is the output schema for the find-block tool.
type findBlockOutput struct {
	Block   string `json:"block"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Z       int    `json:"z"`
	Message string `json:"message"`
}

// readSignInput is the input schema for the read-sign tool.
type readSignInput struct {
	X int `json:"x" jsonschema:"X coordinate of the sign"`
	Y int `json:"y" jsonschema:"Y coordinate of the sign"`
	Z int `json:"z" jsonschema:"Z coordinate of the sign"`
}

// readSignOutput is the output schema for the read-sign tool.
type readSignOutput struct {
	Front   [4]string `json:"front"`
	Back    [4]string `json:"back"`
	Block   string    `json:"block,omitempty"`
	Message string    `json:"message"`
}

// findSignsInput is the input schema for the find-signs tool.
type findSignsInput struct {
	MaxDistance int `json:"maxDistance,omitempty" jsonschema:"max search distance in blocks (default 16, max 64)"`
}

// findSignsSign is a single sign in the find-signs output.
type findSignsSign struct {
	Front [4]string `json:"front"`
	Back  [4]string `json:"back"`
	Block string    `json:"block"`
	X     int       `json:"x"`
	Y     int       `json:"y"`
	Z     int       `json:"z"`
}

// findSignsOutput is the output schema for the find-signs tool.
type findSignsOutput struct {
	Signs   []findSignsSign `json:"signs"`
	Count   int             `json:"count"`
	Message string          `json:"message"`
}

// detectGamemodeInput is the input schema for the detect-gamemode tool (no arguments).
type detectGamemodeInput struct{}

// detectGamemodeOutput is the output schema for the detect-gamemode tool.
type detectGamemodeOutput struct {
	Gamemode string `json:"gamemode"`
}

// detectWorldeditInput is the input schema for the detect-worldedit tool (no arguments).
type detectWorldeditInput struct{}

// detectWorldeditOutput is the output schema for the detect-worldedit tool.
type detectWorldeditOutput struct {
	Tier    string `json:"tier"`
	Message string `json:"message"`
}

// scanAreaInput is the input schema for the scan-area tool.
type scanAreaInput struct {
	X1 int `json:"x1" jsonschema:"first corner X coordinate"`
	Y1 int `json:"y1" jsonschema:"first corner Y coordinate"`
	Z1 int `json:"z1" jsonschema:"first corner Z coordinate"`
	X2 int `json:"x2" jsonschema:"second corner X coordinate"`
	Y2 int `json:"y2" jsonschema:"second corner Y coordinate"`
	Z2 int `json:"z2" jsonschema:"second corner Z coordinate"`
}

// scanAreaBlock is a single block in a scan result.
type scanAreaBlock struct {
	Block string `json:"block"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Z     int    `json:"z"`
}

// scanAreaOutput is the output schema for the scan-area tool.
type scanAreaOutput struct {
	Blocks  []scanAreaBlock `json:"blocks"`
	Count   int             `json:"count"`
	Message string          `json:"message,omitempty"`
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

	// get-block-info — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "get-block-info",
		Description: "Get the block type at specific coordinates in the Minecraft world",
	}, requireConnection(s.conn, s.handleGetBlockInfo))

	// find-block — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "find-block",
		Description: "Find the nearest block of a given type within a search distance",
	}, requireConnection(s.conn, s.handleFindBlock))

	// scan-area — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "scan-area",
		Description: "Scan a rectangular region and return all non-air blocks with their types and positions (max 10,000 blocks)",
	}, requireConnection(s.conn, s.handleScanArea))

	// read-sign — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "read-sign",
		Description: "Read the text on a sign at the specified coordinates (returns front and back text)",
	}, requireConnection(s.conn, s.handleReadSign))

	// find-signs — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "find-signs",
		Description: "Find all signs within a distance of the bot and return their text and positions (max 50 signs)",
	}, requireConnection(s.conn, s.handleFindSigns))

	// detect-gamemode — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "detect-gamemode",
		Description: "Get the bot's current game mode (survival, creative, adventure, spectator)",
	}, requireConnection(s.conn, s.handleDetectGamemode))

	// detect-worldedit — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "detect-worldedit",
		Description: "Get the server's WorldEdit capability tier (fawe, worldedit, vanilla, or unknown if still detecting)",
	}, requireConnection(s.conn, s.handleDetectWorldedit))
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

// handleGetBlockInfo returns the block type at the given coordinates.
func (s *Server) handleGetBlockInfo(_ context.Context, _ *gomcp.CallToolRequest, input getBlockInfoInput) (*gomcp.CallToolResult, getBlockInfoOutput, error) {
	name, err := s.conn.BlockAt(input.X, input.Y, input.Z)
	if err != nil {
		return toolError(fmt.Sprintf("cannot read block at (%d, %d, %d): %v", input.X, input.Y, input.Z, err)), getBlockInfoOutput{}, nil
	}
	return nil, getBlockInfoOutput{
		Block: name,
		X:     input.X,
		Y:     input.Y,
		Z:     input.Z,
	}, nil
}

// handleFindBlock searches for the nearest block of a given type.
func (s *Server) handleFindBlock(_ context.Context, _ *gomcp.CallToolRequest, input findBlockInput) (*gomcp.CallToolResult, findBlockOutput, error) {
	maxDist := input.MaxDistance
	if maxDist <= 0 {
		maxDist = 16
	}

	bx, by, bz, found, err := s.conn.FindBlock(input.BlockType, maxDist)
	if err != nil {
		return toolError(fmt.Sprintf("block search failed: %v", err)), findBlockOutput{}, nil
	}
	if !found {
		return nil, findBlockOutput{
			Message: fmt.Sprintf("no %s found within %d blocks", input.BlockType, maxDist),
		}, nil
	}
	return nil, findBlockOutput{
		Block:   input.BlockType,
		X:       bx,
		Y:       by,
		Z:       bz,
		Message: fmt.Sprintf("found %s at (%d, %d, %d)", input.BlockType, bx, by, bz),
	}, nil
}

// handleScanArea scans a rectangular region and returns non-air blocks.
func (s *Server) handleScanArea(_ context.Context, _ *gomcp.CallToolRequest, input scanAreaInput) (*gomcp.CallToolResult, scanAreaOutput, error) {
	blocks, err := s.conn.ScanArea(input.X1, input.Y1, input.Z1, input.X2, input.Y2, input.Z2)
	if err != nil {
		return toolError(err.Error()), scanAreaOutput{}, nil
	}

	result := make([]scanAreaBlock, len(blocks))
	for i, b := range blocks {
		result[i] = scanAreaBlock{Block: b.Block, X: b.X, Y: b.Y, Z: b.Z}
	}

	return nil, scanAreaOutput{
		Blocks:  result,
		Count:   len(result),
		Message: fmt.Sprintf("scanned region, found %d non-air blocks", len(result)),
	}, nil
}

// handleReadSign reads the text of a sign at the given coordinates.
func (s *Server) handleReadSign(_ context.Context, _ *gomcp.CallToolRequest, input readSignInput) (*gomcp.CallToolResult, readSignOutput, error) {
	sign, blockName, err := s.conn.ReadSign(input.X, input.Y, input.Z)
	if err != nil {
		return toolError(err.Error()), readSignOutput{}, nil
	}

	return nil, readSignOutput{
		Front:   sign.FrontLines,
		Back:    sign.BackLines,
		Block:   blockName,
		Message: fmt.Sprintf("read sign at (%d, %d, %d)", input.X, input.Y, input.Z),
	}, nil
}

// handleFindSigns searches for signs near the bot.
func (s *Server) handleFindSigns(_ context.Context, _ *gomcp.CallToolRequest, input findSignsInput) (*gomcp.CallToolResult, findSignsOutput, error) {
	signs, err := s.conn.FindSigns(input.MaxDistance)
	if err != nil {
		return toolError(err.Error()), findSignsOutput{}, nil
	}

	result := make([]findSignsSign, len(signs))
	for i, si := range signs {
		result[i] = findSignsSign{
			Front: si.Sign.FrontLines,
			Back:  si.Sign.BackLines,
			Block: si.Block,
			X:     si.X,
			Y:     si.Y,
			Z:     si.Z,
		}
	}

	return nil, findSignsOutput{
		Signs:   result,
		Count:   len(result),
		Message: fmt.Sprintf("found %d signs", len(result)),
	}, nil
}

// handleDetectGamemode returns the bot's current game mode.
func (s *Server) handleDetectGamemode(_ context.Context, _ *gomcp.CallToolRequest, _ detectGamemodeInput) (*gomcp.CallToolResult, detectGamemodeOutput, error) {
	return nil, detectGamemodeOutput{
		Gamemode: s.conn.GetGamemode(),
	}, nil
}

// handleDetectWorldedit returns the server's WorldEdit capability tier.
func (s *Server) handleDetectWorldedit(_ context.Context, _ *gomcp.CallToolRequest, _ detectWorldeditInput) (*gomcp.CallToolResult, detectWorldeditOutput, error) {
	tier := s.conn.GetTier()
	return nil, detectWorldeditOutput{
		Tier:    tier.String(),
		Message: fmt.Sprintf("worldedit tier: %s", tier.String()),
	}, nil
}

// Run starts the MCP stdio transport. Blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting MCP server")
	return s.server.Run(ctx, &gomcp.StdioTransport{})
}
