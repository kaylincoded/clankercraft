package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// validPatternRe matches valid WorldEdit block patterns (letters, digits, underscores, colons, commas, percentages, etc.).
var validPatternRe = regexp.MustCompile(`^[a-zA-Z0-9_:,%!^.\[\]]+$`)

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

// setSelectionInput is the input schema for the set-selection tool.
type setSelectionInput struct {
	X1 int `json:"x1" jsonschema:"first corner X coordinate"`
	Y1 int `json:"y1" jsonschema:"first corner Y coordinate"`
	Z1 int `json:"z1" jsonschema:"first corner Z coordinate"`
	X2 int `json:"x2" jsonschema:"second corner X coordinate"`
	Y2 int `json:"y2" jsonschema:"second corner Y coordinate"`
	Z2 int `json:"z2" jsonschema:"second corner Z coordinate"`
}

// setSelectionOutput is the output schema for the set-selection tool.
type setSelectionOutput struct {
	Message string `json:"message"`
}

// getSelectionInput is the input schema for the get-selection tool (no arguments).
type getSelectionInput struct{}

// getSelectionOutput is the output schema for the get-selection tool.
type getSelectionOutput struct {
	X1      int    `json:"x1"`
	Y1      int    `json:"y1"`
	Z1      int    `json:"z1"`
	X2      int    `json:"x2"`
	Y2      int    `json:"y2"`
	Z2      int    `json:"z2"`
	Message string `json:"message"`
}

// weSetInput is the input schema for the we-set tool.
type weSetInput struct {
	Pattern string `json:"pattern" jsonschema:"block pattern, e.g. stone_bricks or 50%stone,50%cobblestone"`
}

// weReplaceInput is the input schema for the we-replace tool.
type weReplaceInput struct {
	From string `json:"from" jsonschema:"source block pattern to replace"`
	To   string `json:"to" jsonschema:"target block pattern to replace with"`
}

// weWallsInput is the input schema for the we-walls tool.
type weWallsInput struct {
	Pattern string `json:"pattern" jsonschema:"block pattern for the walls"`
}

// weFacesInput is the input schema for the we-faces tool.
type weFacesInput struct {
	Pattern string `json:"pattern" jsonschema:"block pattern for all 6 faces"`
}

// weHollowInput is the input schema for the we-hollow tool.
type weHollowInput struct {
	Pattern string `json:"pattern,omitempty" jsonschema:"optional block pattern for the shell (default: existing blocks)"`
}

// weCommandOutput is the shared output schema for WorldEdit operation tools.
type weCommandOutput struct {
	Response string `json:"response"`
	Message  string `json:"message"`
}

// weSphereInput is the input schema for the we-sphere tool.
type weSphereInput struct {
	Pattern string `json:"pattern" jsonschema:"block pattern, e.g. stone"`
	Radius  int    `json:"radius" jsonschema:"sphere radius in blocks"`
	Hollow  bool   `json:"hollow,omitempty" jsonschema:"if true, creates a hollow sphere (//hsphere)"`
}

// weCylInput is the input schema for the we-cyl tool.
type weCylInput struct {
	Pattern string `json:"pattern" jsonschema:"block pattern, e.g. stone"`
	Radius  int    `json:"radius" jsonschema:"cylinder radius in blocks"`
	Height  int    `json:"height,omitempty" jsonschema:"cylinder height (default 1)"`
	Hollow  bool   `json:"hollow,omitempty" jsonschema:"if true, creates a hollow cylinder (//hcyl)"`
}

// wePyramidInput is the input schema for the we-pyramid tool.
type wePyramidInput struct {
	Pattern string `json:"pattern" jsonschema:"block pattern, e.g. stone"`
	Size    int    `json:"size" jsonschema:"pyramid size (base half-width)"`
	Hollow  bool   `json:"hollow,omitempty" jsonschema:"if true, creates a hollow pyramid (//hpyramid)"`
}

// weGenerateInput is the input schema for the we-generate tool.
type weGenerateInput struct {
	Expression string `json:"expression" jsonschema:"mathematical expression, e.g. (x*x + z*z < 100) * stone"`
}

// weSmoothInput is the input schema for the we-smooth tool.
type weSmoothInput struct {
	Iterations int `json:"iterations,omitempty" jsonschema:"number of smooth iterations (default 1)"`
}

// weNaturalizeInput is the input schema for the we-naturalize tool (no arguments).
type weNaturalizeInput struct{}

// weOverlayInput is the input schema for the we-overlay tool.
type weOverlayInput struct {
	Pattern string `json:"pattern" jsonschema:"block pattern to overlay, e.g. grass_block or 50%stone,50%cobblestone"`
}

// weCopyInput is the input schema for the we-copy tool (no arguments).
type weCopyInput struct{}

// wePasteInput is the input schema for the we-paste tool.
type wePasteInput struct {
	SkipAir bool `json:"skipAir,omitempty" jsonschema:"if true, skips air blocks when pasting (//paste -a)"`
}

// weRotateInput is the input schema for the we-rotate tool.
type weRotateInput struct {
	Degrees int `json:"degrees" jsonschema:"rotation in degrees: 90, 180, or 270"`
}

// weFlipInput is the input schema for the we-flip tool.
type weFlipInput struct {
	Direction string `json:"direction,omitempty" jsonschema:"flip direction: north, south, east, west, up, down (default: player facing)"`
}

// weUndoInput is the input schema for the we-undo tool (no arguments).
type weUndoInput struct{}

// weRedoInput is the input schema for the we-redo tool (no arguments).
type weRedoInput struct{}

// setblockInput is the input schema for the setblock tool.
type setblockInput struct {
	X     int    `json:"x" jsonschema:"X coordinate"`
	Y     int    `json:"y" jsonschema:"Y coordinate"`
	Z     int    `json:"z" jsonschema:"Z coordinate"`
	Block string `json:"block" jsonschema:"block type, e.g. minecraft:stone"`
}

// setblockOutput is the output schema for the setblock tool.
type setblockOutput struct {
	Response string `json:"response"`
	Message  string `json:"message"`
}

// fillInput is the input schema for the fill tool.
type fillInput struct {
	X1    int    `json:"x1" jsonschema:"first corner X coordinate"`
	Y1    int    `json:"y1" jsonschema:"first corner Y coordinate"`
	Z1    int    `json:"z1" jsonschema:"first corner Z coordinate"`
	X2    int    `json:"x2" jsonschema:"second corner X coordinate"`
	Y2    int    `json:"y2" jsonschema:"second corner Y coordinate"`
	Z2    int    `json:"z2" jsonschema:"second corner Z coordinate"`
	Block string `json:"block" jsonschema:"block type, e.g. minecraft:stone"`
}

// fillOutput is the output schema for the fill tool.
type fillOutput struct {
	Commands int    `json:"commands"`
	Message  string `json:"message"`
}

// cloneInput is the input schema for the clone tool.
type cloneInput struct {
	X1 int `json:"x1" jsonschema:"source first corner X"`
	Y1 int `json:"y1" jsonschema:"source first corner Y"`
	Z1 int `json:"z1" jsonschema:"source first corner Z"`
	X2 int `json:"x2" jsonschema:"source second corner X"`
	Y2 int `json:"y2" jsonschema:"source second corner Y"`
	Z2 int `json:"z2" jsonschema:"source second corner Z"`
	DX int `json:"dx" jsonschema:"destination X coordinate"`
	DY int `json:"dy" jsonschema:"destination Y coordinate"`
	DZ int `json:"dz" jsonschema:"destination Z coordinate"`
}

// cloneOutput is the output schema for the clone tool.
type cloneOutput struct {
	Response string `json:"response"`
	Message  string `json:"message"`
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

	// set-selection — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "set-selection",
		Description: "Set a WorldEdit selection by specifying two corner positions. On WorldEdit/FAWE servers, sends //pos1 and //pos2 commands.",
	}, requireConnection(s.conn, s.handleSetSelection))

	// get-selection — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "get-selection",
		Description: "Get the current WorldEdit selection coordinates, or report that no selection is set",
	}, requireConnection(s.conn, s.handleGetSelection))

	// detect-worldedit — requires connection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "detect-worldedit",
		Description: "Get the server's WorldEdit capability tier (fawe, worldedit, vanilla, or unknown if still detecting)",
	}, requireConnection(s.conn, s.handleDetectWorldedit))

	// we-set — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-set",
		Description: "Fill the current selection with a block pattern using WorldEdit //set",
	}, requireWorldEdit(s.conn, s.handleWESet))

	// we-replace — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-replace",
		Description: "Replace blocks in the current selection using WorldEdit //replace",
	}, requireWorldEdit(s.conn, s.handleWEReplace))

	// we-walls — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-walls",
		Description: "Set only the walls (not floor/ceiling) of the current selection using WorldEdit //walls",
	}, requireWorldEdit(s.conn, s.handleWEWalls))

	// we-faces — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-faces",
		Description: "Set all 6 faces of the current selection using WorldEdit //faces",
	}, requireWorldEdit(s.conn, s.handleWEFaces))

	// we-hollow — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-hollow",
		Description: "Hollow out the current selection using WorldEdit //hollow, optionally filling the shell with a pattern",
	}, requireWorldEdit(s.conn, s.handleWEHollow))

	// we-sphere — requires WorldEdit tier (position-based, no selection)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-sphere",
		Description: "Create a sphere at the bot's position using WorldEdit //sphere (set hollow=true for //hsphere)",
	}, requireWETier(s.conn, s.handleWESphere))

	// we-cyl — requires WorldEdit tier (position-based, no selection)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-cyl",
		Description: "Create a cylinder at the bot's position using WorldEdit //cyl (set hollow=true for //hcyl)",
	}, requireWETier(s.conn, s.handleWECyl))

	// we-pyramid — requires WorldEdit tier (position-based, no selection)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-pyramid",
		Description: "Create a pyramid at the bot's position using WorldEdit //pyramid (set hollow=true for //hpyramid)",
	}, requireWETier(s.conn, s.handleWEPyramid))

	// we-generate — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-generate",
		Description: "Generate blocks from a mathematical expression in the current selection using WorldEdit //generate",
	}, requireWorldEdit(s.conn, s.handleWEGenerate))

	// we-smooth — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-smooth",
		Description: "Smooth terrain in the current selection using WorldEdit //smooth (default 1 iteration)",
	}, requireWorldEdit(s.conn, s.handleWESmooth))

	// we-naturalize — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-naturalize",
		Description: "Naturalize terrain in the current selection using WorldEdit //naturalize (grass on top, dirt below, stone deep)",
	}, requireWorldEdit(s.conn, s.handleWENaturalize))

	// we-overlay — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-overlay",
		Description: "Overlay a block pattern on top of existing blocks in the current selection using WorldEdit //overlay",
	}, requireWorldEdit(s.conn, s.handleWEOverlay))

	// we-copy — requires WorldEdit + selection
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-copy",
		Description: "Copy the current selection to the clipboard using WorldEdit //copy",
	}, requireWorldEdit(s.conn, s.handleWECopy))

	// we-paste — requires WorldEdit tier (clipboard-based, no selection needed)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-paste",
		Description: "Paste the clipboard at the bot's position using WorldEdit //paste (set skipAir=true to skip air blocks)",
	}, requireWETier(s.conn, s.handleWEPaste))

	// we-rotate — requires WorldEdit tier (clipboard-based, no selection needed)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-rotate",
		Description: "Rotate the clipboard contents using WorldEdit //rotate (90, 180, or 270 degrees)",
	}, requireWETier(s.conn, s.handleWERotate))

	// we-flip — requires WorldEdit tier (clipboard-based, no selection needed)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-flip",
		Description: "Flip the clipboard contents using WorldEdit //flip (direction: north/south/east/west/up/down)",
	}, requireWETier(s.conn, s.handleWEFlip))

	// we-undo — requires WorldEdit tier (history-based, no selection needed)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-undo",
		Description: "Undo the last WorldEdit operation using //undo",
	}, requireWETier(s.conn, s.handleWEUndo))

	// we-redo — requires WorldEdit tier (history-based, no selection needed)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "we-redo",
		Description: "Redo the last undone WorldEdit operation using //redo",
	}, requireWETier(s.conn, s.handleWERedo))

	// setblock — requires connection (works on any tier)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "setblock",
		Description: "Place a single block at the specified coordinates using /setblock",
	}, requireConnection(s.conn, s.handleSetblock))

	// fill — requires connection (works on any tier, auto-decomposes large regions)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "fill",
		Description: "Fill a region with a block type using /fill (auto-decomposes regions larger than 32,768 blocks)",
	}, requireConnection(s.conn, s.handleFill))

	// clone — requires connection (works on any tier)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "clone",
		Description: "Clone a region to a destination using /clone (overlapping source and destination may produce undefined results)",
	}, requireConnection(s.conn, s.handleClone))
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

// handleSetSelection sets the WorldEdit selection corners.
func (s *Server) handleSetSelection(_ context.Context, _ *gomcp.CallToolRequest, input setSelectionInput) (*gomcp.CallToolResult, setSelectionOutput, error) {
	if err := s.conn.SetSelection(input.X1, input.Y1, input.Z1, input.X2, input.Y2, input.Z2); err != nil {
		return toolError(fmt.Sprintf("failed to set selection: %v", err)), setSelectionOutput{}, nil
	}
	return nil, setSelectionOutput{
		Message: fmt.Sprintf("selection set: (%d, %d, %d) to (%d, %d, %d)", input.X1, input.Y1, input.Z1, input.X2, input.Y2, input.Z2),
	}, nil
}

// handleGetSelection returns the current selection or indicates partial/no state.
func (s *Server) handleGetSelection(_ context.Context, _ *gomcp.CallToolRequest, _ getSelectionInput) (*gomcp.CallToolResult, getSelectionOutput, error) {
	sel, ok := s.conn.GetSelection()
	if ok {
		return nil, getSelectionOutput{
			X1:      sel.X1,
			Y1:      sel.Y1,
			Z1:      sel.Z1,
			X2:      sel.X2,
			Y2:      sel.Y2,
			Z2:      sel.Z2,
			Message: fmt.Sprintf("selection: %s", sel.String()),
		}, nil
	}

	hasP1 := s.conn.HasPos1()
	hasP2 := s.conn.HasPos2()
	if hasP1 && !hasP2 {
		return nil, getSelectionOutput{
			X1:      sel.X1,
			Y1:      sel.Y1,
			Z1:      sel.Z1,
			Message: "partial selection: pos1 set, waiting for pos2",
		}, nil
	}
	if !hasP1 && hasP2 {
		return nil, getSelectionOutput{
			X2:      sel.X2,
			Y2:      sel.Y2,
			Z2:      sel.Z2,
			Message: "partial selection: pos2 set, waiting for pos1",
		}, nil
	}
	return toolError("no selection set"), getSelectionOutput{}, nil
}

// validatePattern checks that a WorldEdit pattern is safe (no command injection).
func validatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if !validPatternRe.MatchString(pattern) {
		return fmt.Errorf("invalid block pattern: %q", pattern)
	}
	return nil
}

// validateExpression checks that a //generate expression is safe (no command injection).
func validateExpression(expr string) error {
	if expr == "" {
		return fmt.Errorf("expression cannot be empty")
	}
	for _, c := range expr {
		if c == '\n' || c == '\r' || c == ';' || c == '/' {
			return fmt.Errorf("expression contains invalid characters")
		}
	}
	return nil
}

// handleWESet fills the selection with a block pattern.
func (s *Server) handleWESet(_ context.Context, _ *gomcp.CallToolRequest, input weSetInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.Pattern); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	resp, err := s.conn.RunBulkWECommand("set " + input.Pattern)
	if err != nil {
		return toolError(fmt.Sprintf("//set failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//set %s: %s", input.Pattern, resp)}, nil
}

// handleWEReplace replaces blocks in the selection.
func (s *Server) handleWEReplace(_ context.Context, _ *gomcp.CallToolRequest, input weReplaceInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.From); err != nil {
		return toolError(fmt.Sprintf("invalid 'from' pattern: %v", err)), weCommandOutput{}, nil
	}
	if err := validatePattern(input.To); err != nil {
		return toolError(fmt.Sprintf("invalid 'to' pattern: %v", err)), weCommandOutput{}, nil
	}
	resp, err := s.conn.RunBulkWECommand("replace " + input.From + " " + input.To)
	if err != nil {
		return toolError(fmt.Sprintf("//replace failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//replace %s %s: %s", input.From, input.To, resp)}, nil
}

// handleWEWalls sets the walls of the selection.
func (s *Server) handleWEWalls(_ context.Context, _ *gomcp.CallToolRequest, input weWallsInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.Pattern); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	resp, err := s.conn.RunBulkWECommand("walls " + input.Pattern)
	if err != nil {
		return toolError(fmt.Sprintf("//walls failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//walls %s: %s", input.Pattern, resp)}, nil
}

// handleWEFaces sets all 6 faces of the selection.
func (s *Server) handleWEFaces(_ context.Context, _ *gomcp.CallToolRequest, input weFacesInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.Pattern); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	resp, err := s.conn.RunBulkWECommand("faces " + input.Pattern)
	if err != nil {
		return toolError(fmt.Sprintf("//faces failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//faces %s: %s", input.Pattern, resp)}, nil
}

// handleWEHollow hollows out the selection.
func (s *Server) handleWEHollow(_ context.Context, _ *gomcp.CallToolRequest, input weHollowInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	cmd := "hollow"
	if input.Pattern != "" {
		if err := validatePattern(input.Pattern); err != nil {
			return toolError(err.Error()), weCommandOutput{}, nil
		}
		cmd += " " + input.Pattern
	}
	resp, err := s.conn.RunBulkWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//hollow failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//%s: %s", cmd, resp)}, nil
}

// handleWESphere creates a sphere at the bot's position.
func (s *Server) handleWESphere(_ context.Context, _ *gomcp.CallToolRequest, input weSphereInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.Pattern); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	if input.Radius <= 0 {
		return toolError("radius must be a positive integer"), weCommandOutput{}, nil
	}
	cmd := "sphere"
	if input.Hollow {
		cmd = "hsphere"
	}
	cmd += fmt.Sprintf(" %s %d", input.Pattern, input.Radius)
	resp, err := s.conn.RunBulkWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//%s failed: %v", cmd, err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//%s: %s", cmd, resp)}, nil
}

// handleWECyl creates a cylinder at the bot's position.
func (s *Server) handleWECyl(_ context.Context, _ *gomcp.CallToolRequest, input weCylInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.Pattern); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	if input.Radius <= 0 {
		return toolError("radius must be a positive integer"), weCommandOutput{}, nil
	}
	cmd := "cyl"
	if input.Hollow {
		cmd = "hcyl"
	}
	cmd += fmt.Sprintf(" %s %d", input.Pattern, input.Radius)
	if input.Height > 0 {
		cmd += fmt.Sprintf(" %d", input.Height)
	}
	resp, err := s.conn.RunBulkWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//%s failed: %v", cmd, err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//%s: %s", cmd, resp)}, nil
}

// handleWEPyramid creates a pyramid at the bot's position.
func (s *Server) handleWEPyramid(_ context.Context, _ *gomcp.CallToolRequest, input wePyramidInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.Pattern); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	if input.Size <= 0 {
		return toolError("size must be a positive integer"), weCommandOutput{}, nil
	}
	cmd := "pyramid"
	if input.Hollow {
		cmd = "hpyramid"
	}
	cmd += fmt.Sprintf(" %s %d", input.Pattern, input.Size)
	resp, err := s.conn.RunBulkWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//%s failed: %v", cmd, err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//%s: %s", cmd, resp)}, nil
}

// handleWEGenerate generates blocks from a mathematical expression in the selection.
func (s *Server) handleWEGenerate(_ context.Context, _ *gomcp.CallToolRequest, input weGenerateInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validateExpression(input.Expression); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	cmd := "generate " + input.Expression
	resp, err := s.conn.RunBulkWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//generate failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//generate: %s", resp)}, nil
}

// handleWESmooth smooths terrain in the selection.
func (s *Server) handleWESmooth(_ context.Context, _ *gomcp.CallToolRequest, input weSmoothInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	iterations := input.Iterations
	if iterations <= 0 {
		iterations = 1
	}
	cmd := fmt.Sprintf("smooth %d", iterations)
	resp, err := s.conn.RunBulkWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//smooth failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//smooth %d: %s", iterations, resp)}, nil
}

// handleWENaturalize naturalizes terrain in the selection.
func (s *Server) handleWENaturalize(_ context.Context, _ *gomcp.CallToolRequest, _ weNaturalizeInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	resp, err := s.conn.RunBulkWECommand("naturalize")
	if err != nil {
		return toolError(fmt.Sprintf("//naturalize failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//naturalize: %s", resp)}, nil
}

// handleWEOverlay overlays a pattern on top of existing blocks in the selection.
func (s *Server) handleWEOverlay(_ context.Context, _ *gomcp.CallToolRequest, input weOverlayInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if err := validatePattern(input.Pattern); err != nil {
		return toolError(err.Error()), weCommandOutput{}, nil
	}
	cmd := "overlay " + input.Pattern
	resp, err := s.conn.RunBulkWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//overlay failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//overlay %s: %s", input.Pattern, resp)}, nil
}

// validFlipDirections is the set of valid directions for //flip.
var validFlipDirections = map[string]bool{
	"north": true, "south": true, "east": true, "west": true, "up": true, "down": true,
}

// handleWECopy copies the selection to the clipboard.
func (s *Server) handleWECopy(_ context.Context, _ *gomcp.CallToolRequest, _ weCopyInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	resp, err := s.conn.RunWECommand("copy")
	if err != nil {
		return toolError(fmt.Sprintf("//copy failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//copy: %s", resp)}, nil
}

// handleWEPaste pastes the clipboard at the bot's position.
func (s *Server) handleWEPaste(_ context.Context, _ *gomcp.CallToolRequest, input wePasteInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	cmd := "paste"
	if input.SkipAir {
		cmd = "paste -a"
	}
	resp, err := s.conn.RunWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//%s failed: %v", cmd, err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//%s: %s", cmd, resp)}, nil
}

// handleWERotate rotates the clipboard contents.
func (s *Server) handleWERotate(_ context.Context, _ *gomcp.CallToolRequest, input weRotateInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	if input.Degrees != 90 && input.Degrees != 180 && input.Degrees != 270 {
		return toolError(fmt.Sprintf("invalid rotation: %d degrees (must be 90, 180, or 270)", input.Degrees)), weCommandOutput{}, nil
	}
	cmd := fmt.Sprintf("rotate %d", input.Degrees)
	resp, err := s.conn.RunWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//rotate failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//rotate %d: %s", input.Degrees, resp)}, nil
}

// handleWEFlip flips the clipboard contents.
func (s *Server) handleWEFlip(_ context.Context, _ *gomcp.CallToolRequest, input weFlipInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	cmd := "flip"
	if input.Direction != "" {
		if !validFlipDirections[input.Direction] {
			return toolError(fmt.Sprintf("invalid flip direction: %q (must be north, south, east, west, up, or down)", input.Direction)), weCommandOutput{}, nil
		}
		cmd += " " + input.Direction
	}
	resp, err := s.conn.RunWECommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("//%s failed: %v", cmd, err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//%s: %s", cmd, resp)}, nil
}

// handleWEUndo undoes the last WorldEdit operation.
func (s *Server) handleWEUndo(_ context.Context, _ *gomcp.CallToolRequest, _ weUndoInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	resp, err := s.conn.RunWECommand("undo")
	if err != nil {
		return toolError(fmt.Sprintf("//undo failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//undo: %s", resp)}, nil
}

// handleWERedo redoes the last undone WorldEdit operation.
func (s *Server) handleWERedo(_ context.Context, _ *gomcp.CallToolRequest, _ weRedoInput) (*gomcp.CallToolResult, weCommandOutput, error) {
	resp, err := s.conn.RunWECommand("redo")
	if err != nil {
		return toolError(fmt.Sprintf("//redo failed: %v", err)), weCommandOutput{}, nil
	}
	return nil, weCommandOutput{Response: resp, Message: fmt.Sprintf("//redo: %s", resp)}, nil
}

// maxFillVolume is the Minecraft vanilla /fill block limit per command.
const maxFillVolume = 32768

// decomposeFill splits a region into sub-regions that each fit within maxFillVolume.
// Returns a slice of [6]int{x1,y1,z1,x2,y2,z2} regions.
func decomposeFill(x1, y1, z1, x2, y2, z2 int) [][6]int {
	// Normalize coordinates so min <= max
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if z1 > z2 {
		z1, z2 = z2, z1
	}

	dx := x2 - x1 + 1
	dy := y2 - y1 + 1
	dz := z2 - z1 + 1
	volume := dx * dy * dz

	if volume <= maxFillVolume {
		return [][6]int{{x1, y1, z1, x2, y2, z2}}
	}

	// Split along the longest axis
	if dx >= dy && dx >= dz {
		mid := x1 + dx/2 - 1
		left := decomposeFill(x1, y1, z1, mid, y2, z2)
		right := decomposeFill(mid+1, y1, z1, x2, y2, z2)
		return append(left, right...)
	} else if dy >= dz {
		mid := y1 + dy/2 - 1
		left := decomposeFill(x1, y1, z1, x2, mid, z2)
		right := decomposeFill(x1, mid+1, z1, x2, y2, z2)
		return append(left, right...)
	} else {
		mid := z1 + dz/2 - 1
		left := decomposeFill(x1, y1, z1, x2, y2, mid)
		right := decomposeFill(x1, y1, mid+1, x2, y2, z2)
		return append(left, right...)
	}
}

// handleSetblock places a single block.
func (s *Server) handleSetblock(_ context.Context, _ *gomcp.CallToolRequest, input setblockInput) (*gomcp.CallToolResult, setblockOutput, error) {
	if err := validatePattern(input.Block); err != nil {
		return toolError(err.Error()), setblockOutput{}, nil
	}
	cmd := fmt.Sprintf("setblock %d %d %d %s", input.X, input.Y, input.Z, input.Block)
	resp, err := s.conn.RunBulkCommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("/setblock failed: %v", err)), setblockOutput{}, nil
	}
	return nil, setblockOutput{
		Response: resp,
		Message:  fmt.Sprintf("/setblock %d %d %d %s: %s", input.X, input.Y, input.Z, input.Block, resp),
	}, nil
}

// handleFill fills a region, auto-decomposing large regions.
func (s *Server) handleFill(_ context.Context, _ *gomcp.CallToolRequest, input fillInput) (*gomcp.CallToolResult, fillOutput, error) {
	if err := validatePattern(input.Block); err != nil {
		return toolError(err.Error()), fillOutput{}, nil
	}

	regions := decomposeFill(input.X1, input.Y1, input.Z1, input.X2, input.Y2, input.Z2)
	var lastResp string
	for _, r := range regions {
		cmd := fmt.Sprintf("fill %d %d %d %d %d %d %s", r[0], r[1], r[2], r[3], r[4], r[5], input.Block)
		resp, err := s.conn.RunBulkCommand(cmd)
		if err != nil {
			return toolError(fmt.Sprintf("/fill failed: %v", err)), fillOutput{}, nil
		}
		lastResp = resp
	}

	msg := fmt.Sprintf("/fill: %s", lastResp)
	if len(regions) > 1 {
		msg = fmt.Sprintf("/fill: decomposed into %d commands, last response: %s", len(regions), lastResp)
	}
	return nil, fillOutput{
		Commands: len(regions),
		Message:  msg,
	}, nil
}

// handleClone clones a region to a destination.
func (s *Server) handleClone(_ context.Context, _ *gomcp.CallToolRequest, input cloneInput) (*gomcp.CallToolResult, cloneOutput, error) {
	cmd := fmt.Sprintf("clone %d %d %d %d %d %d %d %d %d",
		input.X1, input.Y1, input.Z1, input.X2, input.Y2, input.Z2,
		input.DX, input.DY, input.DZ)
	resp, err := s.conn.RunBulkCommand(cmd)
	if err != nil {
		return toolError(fmt.Sprintf("/clone failed: %v", err)), cloneOutput{}, nil
	}
	return nil, cloneOutput{
		Response: resp,
		Message:  fmt.Sprintf("/clone: %s", resp),
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
