package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/kaylincoded/clankercraft/internal/engine"
	"github.com/kaylincoded/clankercraft/internal/llm"
	"github.com/kaylincoded/clankercraft/internal/mcp"
)

// ToolExecutor bridges LLM tool calls to BotState methods.
type ToolExecutor struct {
	bot  mcp.BotState
	defs []llm.ToolDef
}

// NewToolExecutor creates a ToolExecutor backed by the given BotState.
func NewToolExecutor(bot mcp.BotState) *ToolExecutor {
	te := &ToolExecutor{bot: bot}
	te.defs = te.buildToolDefs()
	return te
}

// ToolDefs returns all available tools as LLM tool definitions.
func (te *ToolExecutor) ToolDefs() []llm.ToolDef {
	return te.defs
}

// Execute dispatches a tool call by name. Returns JSON result string on success.
// On failure, returns an error whose message should be sent as ToolResult.IsError=true.
func (te *ToolExecutor) Execute(_ context.Context, name string, input json.RawMessage) (string, error) {
	switch name {
	case "ping":
		return jsonResult(map[string]string{"status": "pong"})

	case "status":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		return jsonResult(map[string]any{"connected": true, "status": "connected"})

	case "get-position":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		pos, ok := te.bot.GetPosition()
		if !ok {
			return "", fmt.Errorf("position not yet known (waiting for server)")
		}
		return jsonResult(map[string]any{
			"x": int(math.Floor(pos.X)), "y": int(math.Floor(pos.Y)), "z": int(math.Floor(pos.Z)),
			"yaw": pos.Yaw, "pitch": pos.Pitch,
		})

	case "look-at":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		pos, ok := te.bot.GetPosition()
		if !ok {
			return "", fmt.Errorf("position not yet known (waiting for server)")
		}
		yaw, pitch := calcYawPitch(pos.X, pos.Y, pos.Z, in.X, in.Y, in.Z)
		if err := te.bot.SendRotation(yaw, pitch); err != nil {
			return "", fmt.Errorf("failed to send rotation: %w", err)
		}
		return jsonResult(map[string]string{"message": fmt.Sprintf("Now looking at (%.0f, %.0f, %.0f)", in.X, in.Y, in.Z)})

	case "get-block-info":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X int `json:"x"`
			Y int `json:"y"`
			Z int `json:"z"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		block, err := te.bot.BlockAt(in.X, in.Y, in.Z)
		if err != nil {
			return "", fmt.Errorf("cannot read block at (%d, %d, %d): %w", in.X, in.Y, in.Z, err)
		}
		return jsonResult(map[string]any{"block": block, "x": in.X, "y": in.Y, "z": in.Z})

	case "find-block":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			BlockType   string `json:"blockType"`
			MaxDistance int    `json:"maxDistance"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		maxDist := in.MaxDistance
		if maxDist <= 0 {
			maxDist = 16
		}
		bx, by, bz, found, err := te.bot.FindBlock(in.BlockType, maxDist)
		if err != nil {
			return "", fmt.Errorf("block search failed: %w", err)
		}
		if !found {
			return jsonResult(map[string]string{"message": fmt.Sprintf("no %s found within %d blocks", in.BlockType, maxDist)})
		}
		return jsonResult(map[string]any{"block": in.BlockType, "x": bx, "y": by, "z": bz,
			"message": fmt.Sprintf("found %s at (%d, %d, %d)", in.BlockType, bx, by, bz)})

	case "scan-area":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X1 int `json:"x1"`
			Y1 int `json:"y1"`
			Z1 int `json:"z1"`
			X2 int `json:"x2"`
			Y2 int `json:"y2"`
			Z2 int `json:"z2"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		blocks, err := te.bot.ScanArea(in.X1, in.Y1, in.Z1, in.X2, in.Y2, in.Z2)
		if err != nil {
			return "", fmt.Errorf("%w", err)
		}
		type blockInfo struct {
			Block string `json:"block"`
			X     int    `json:"x"`
			Y     int    `json:"y"`
			Z     int    `json:"z"`
		}
		result := make([]blockInfo, len(blocks))
		for i, b := range blocks {
			result[i] = blockInfo{Block: b.Block, X: b.X, Y: b.Y, Z: b.Z}
		}
		return jsonResult(map[string]any{"blocks": result, "count": len(result),
			"message": fmt.Sprintf("scanned region, found %d non-air blocks", len(result))})

	case "read-sign":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X int `json:"x"`
			Y int `json:"y"`
			Z int `json:"z"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		sign, blockName, err := te.bot.ReadSign(in.X, in.Y, in.Z)
		if err != nil {
			return "", fmt.Errorf("%w", err)
		}
		return jsonResult(map[string]any{"front": sign.FrontLines, "back": sign.BackLines, "block": blockName,
			"message": fmt.Sprintf("read sign at (%d, %d, %d)", in.X, in.Y, in.Z)})

	case "find-signs":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			MaxDistance int `json:"maxDistance"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		signs, err := te.bot.FindSigns(in.MaxDistance)
		if err != nil {
			return "", fmt.Errorf("%w", err)
		}
		type signInfo struct {
			Front [4]string `json:"front"`
			Back  [4]string `json:"back"`
			Block string    `json:"block"`
			X     int       `json:"x"`
			Y     int       `json:"y"`
			Z     int       `json:"z"`
		}
		result := make([]signInfo, len(signs))
		for i, si := range signs {
			result[i] = signInfo{Front: si.Sign.FrontLines, Back: si.Sign.BackLines, Block: si.Block, X: si.X, Y: si.Y, Z: si.Z}
		}
		return jsonResult(map[string]any{"signs": result, "count": len(result),
			"message": fmt.Sprintf("found %d signs", len(result))})

	case "detect-gamemode":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		return jsonResult(map[string]string{"gamemode": te.bot.GetGamemode()})

	case "detect-worldedit":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		tier := te.bot.GetTier()
		return jsonResult(map[string]string{"tier": tier.String(), "message": fmt.Sprintf("WorldEdit tier: %s", tier)})

	case "set-selection":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X1 int `json:"x1"`
			Y1 int `json:"y1"`
			Z1 int `json:"z1"`
			X2 int `json:"x2"`
			Y2 int `json:"y2"`
			Z2 int `json:"z2"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		if err := te.bot.SetSelection(in.X1, in.Y1, in.Z1, in.X2, in.Y2, in.Z2); err != nil {
			return "", fmt.Errorf("failed to set selection: %w", err)
		}
		return jsonResult(map[string]string{
			"message": fmt.Sprintf("selection set: (%d, %d, %d) to (%d, %d, %d)", in.X1, in.Y1, in.Z1, in.X2, in.Y2, in.Z2),
		})

	case "get-selection":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		sel, ok := te.bot.GetSelection()
		if ok {
			return jsonResult(map[string]any{
				"x1": sel.X1, "y1": sel.Y1, "z1": sel.Z1,
				"x2": sel.X2, "y2": sel.Y2, "z2": sel.Z2,
				"message": fmt.Sprintf("selection: %s", sel.String()),
			})
		}
		if te.bot.HasPos1() && !te.bot.HasPos2() {
			return jsonResult(map[string]any{"x1": sel.X1, "y1": sel.Y1, "z1": sel.Z1,
				"message": "partial selection: pos1 set, waiting for pos2"})
		}
		if !te.bot.HasPos1() && te.bot.HasPos2() {
			return jsonResult(map[string]any{"x2": sel.X2, "y2": sel.Y2, "z2": sel.Z2,
				"message": "partial selection: pos2 set, waiting for pos1"})
		}
		return "", fmt.Errorf("no selection set")

	// WorldEdit selection-required tools
	case "we-set":
		return te.weSelectionCmd(input, func(in wePatternInput) (string, error) {
			return "set " + in.Pattern, nil
		})
	case "we-replace":
		return te.weSelectionCmd(input, func(_ wePatternInput) (string, error) {
			var in struct {
				From string `json:"from"`
				To   string `json:"to"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if err := validatePattern(in.From); err != nil {
				return "", fmt.Errorf("invalid 'from' pattern: %w", err)
			}
			if err := validatePattern(in.To); err != nil {
				return "", fmt.Errorf("invalid 'to' pattern: %w", err)
			}
			return "replace " + in.From + " " + in.To, nil
		})
	case "we-walls":
		return te.weSelectionCmd(input, func(in wePatternInput) (string, error) {
			return "walls " + in.Pattern, nil
		})
	case "we-faces":
		return te.weSelectionCmd(input, func(in wePatternInput) (string, error) {
			return "faces " + in.Pattern, nil
		})
	case "we-hollow":
		return te.weSelectionCmd(input, func(_ wePatternInput) (string, error) {
			var in struct {
				Pattern string `json:"pattern"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			cmd := "hollow"
			if in.Pattern != "" {
				if err := validatePattern(in.Pattern); err != nil {
					return "", err
				}
				cmd += " " + in.Pattern
			}
			return cmd, nil
		})
	case "we-generate":
		return te.weSelectionCmd(input, func(_ wePatternInput) (string, error) {
			var in struct {
				Expression string `json:"expression"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if err := validateExpression(in.Expression); err != nil {
				return "", err
			}
			return "generate " + in.Expression, nil
		})
	case "we-smooth":
		return te.weSelectionCmd(input, func(_ wePatternInput) (string, error) {
			var in struct {
				Iterations int `json:"iterations"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			iterations := in.Iterations
			if iterations <= 0 {
				iterations = 1
			}
			return fmt.Sprintf("smooth %d", iterations), nil
		})
	case "we-naturalize":
		return te.weSelectionCmd(input, func(_ wePatternInput) (string, error) {
			return "naturalize", nil
		})
	case "we-overlay":
		return te.weSelectionCmd(input, func(in wePatternInput) (string, error) {
			return "overlay " + in.Pattern, nil
		})
	case "we-copy":
		return te.weSelectionCmd(input, func(_ wePatternInput) (string, error) {
			return "copy", nil
		})

	// WorldEdit tier-only tools (no selection required)
	case "we-sphere":
		return te.weTierCmd(input, func() (string, error) {
			var in struct {
				Pattern string `json:"pattern"`
				Radius  int    `json:"radius"`
				Hollow  bool   `json:"hollow"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if err := validatePattern(in.Pattern); err != nil {
				return "", err
			}
			if in.Radius <= 0 {
				return "", fmt.Errorf("radius must be a positive integer")
			}
			cmd := "sphere"
			if in.Hollow {
				cmd = "hsphere"
			}
			return fmt.Sprintf("%s %s %d", cmd, in.Pattern, in.Radius), nil
		})
	case "we-cyl":
		return te.weTierCmd(input, func() (string, error) {
			var in struct {
				Pattern string `json:"pattern"`
				Radius  int    `json:"radius"`
				Height  int    `json:"height"`
				Hollow  bool   `json:"hollow"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if err := validatePattern(in.Pattern); err != nil {
				return "", err
			}
			if in.Radius <= 0 {
				return "", fmt.Errorf("radius must be a positive integer")
			}
			cmd := "cyl"
			if in.Hollow {
				cmd = "hcyl"
			}
			cmd = fmt.Sprintf("%s %s %d", cmd, in.Pattern, in.Radius)
			if in.Height > 0 {
				cmd += fmt.Sprintf(" %d", in.Height)
			}
			return cmd, nil
		})
	case "we-pyramid":
		return te.weTierCmd(input, func() (string, error) {
			var in struct {
				Pattern string `json:"pattern"`
				Size    int    `json:"size"`
				Hollow  bool   `json:"hollow"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if err := validatePattern(in.Pattern); err != nil {
				return "", err
			}
			if in.Size <= 0 {
				return "", fmt.Errorf("size must be a positive integer")
			}
			cmd := "pyramid"
			if in.Hollow {
				cmd = "hpyramid"
			}
			return fmt.Sprintf("%s %s %d", cmd, in.Pattern, in.Size), nil
		})
	case "we-paste":
		return te.weTierCmd(input, func() (string, error) {
			var in struct {
				SkipAir bool `json:"skipAir"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if in.SkipAir {
				return "paste -a", nil
			}
			return "paste", nil
		})
	case "we-rotate":
		return te.weTierCmd(input, func() (string, error) {
			var in struct {
				Degrees int `json:"degrees"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			if in.Degrees != 90 && in.Degrees != 180 && in.Degrees != 270 {
				return "", fmt.Errorf("invalid rotation: %d degrees (must be 90, 180, or 270)", in.Degrees)
			}
			return fmt.Sprintf("rotate %d", in.Degrees), nil
		})
	case "we-flip":
		return te.weTierCmd(input, func() (string, error) {
			var in struct {
				Direction string `json:"direction"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			cmd := "flip"
			if in.Direction != "" {
				valid := map[string]bool{"north": true, "south": true, "east": true, "west": true, "up": true, "down": true}
				if !valid[in.Direction] {
					return "", fmt.Errorf("invalid flip direction: %q", in.Direction)
				}
				cmd += " " + in.Direction
			}
			return cmd, nil
		})
	case "we-undo":
		return te.weTierCmd(input, func() (string, error) { return "undo", nil })
	case "we-redo":
		return te.weTierCmd(input, func() (string, error) { return "redo", nil })

	// Teleportation
	case "teleport-to-player":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			Player string `json:"player"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		if !isValidPlayerName(in.Player) {
			return "", fmt.Errorf("invalid player name: %q", in.Player)
		}
		cmd := fmt.Sprintf("tp @s %s", in.Player)
		resp, err := te.bot.RunBulkCommand(cmd)
		if err != nil {
			return "", fmt.Errorf("/tp failed: %w", err)
		}
		return jsonResult(map[string]string{"response": resp, "message": fmt.Sprintf("Teleported to %s", in.Player)})

	// Vanilla commands
	case "setblock":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X     int    `json:"x"`
			Y     int    `json:"y"`
			Z     int    `json:"z"`
			Block string `json:"block"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		if err := validatePattern(in.Block); err != nil {
			return "", err
		}
		cmd := fmt.Sprintf("setblock %d %d %d %s", in.X, in.Y, in.Z, in.Block)
		resp, err := te.bot.RunBulkCommand(cmd)
		if err != nil {
			return "", fmt.Errorf("/setblock failed: %w", err)
		}
		return jsonResult(map[string]string{"response": resp, "message": fmt.Sprintf("/setblock %d %d %d %s: %s", in.X, in.Y, in.Z, in.Block, resp)})

	case "fill":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X1    int    `json:"x1"`
			Y1    int    `json:"y1"`
			Z1    int    `json:"z1"`
			X2    int    `json:"x2"`
			Y2    int    `json:"y2"`
			Z2    int    `json:"z2"`
			Block string `json:"block"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		if err := validatePattern(in.Block); err != nil {
			return "", err
		}
		regions := decomposeFill(in.X1, in.Y1, in.Z1, in.X2, in.Y2, in.Z2)
		for _, r := range regions {
			cmd := fmt.Sprintf("fill %d %d %d %d %d %d %s", r[0], r[1], r[2], r[3], r[4], r[5], in.Block)
			if _, err := te.bot.RunBulkCommand(cmd); err != nil {
				return "", fmt.Errorf("/fill failed: %w", err)
			}
		}
		return jsonResult(map[string]any{"commands": len(regions),
			"message": fmt.Sprintf("fill complete (%d commands)", len(regions))})

	case "clone":
		if !te.bot.IsConnected() {
			return "", fmt.Errorf("bot is not connected to a Minecraft server")
		}
		var in struct {
			X1 int `json:"x1"`
			Y1 int `json:"y1"`
			Z1 int `json:"z1"`
			X2 int `json:"x2"`
			Y2 int `json:"y2"`
			Z2 int `json:"z2"`
			DX int `json:"dx"`
			DY int `json:"dy"`
			DZ int `json:"dz"`
		}
		if err := json.Unmarshal(input, &in); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}
		cmd := fmt.Sprintf("clone %d %d %d %d %d %d %d %d %d", in.X1, in.Y1, in.Z1, in.X2, in.Y2, in.Z2, in.DX, in.DY, in.DZ)
		resp, err := te.bot.RunBulkCommand(cmd)
		if err != nil {
			return "", fmt.Errorf("/clone failed: %w", err)
		}
		return jsonResult(map[string]string{"response": resp, "message": fmt.Sprintf("/clone: %s", resp)})

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// wePatternInput is used by WE tools that take a single pattern parameter.
type wePatternInput struct {
	Pattern string `json:"pattern"`
}

// weSelectionCmd checks connection, tier, and selection, then runs a WE command.
func (te *ToolExecutor) weSelectionCmd(input json.RawMessage, buildCmd func(wePatternInput) (string, error)) (string, error) {
	if !te.bot.IsConnected() {
		return "", fmt.Errorf("bot is not connected to a Minecraft server")
	}
	tier := te.bot.GetTier()
	if tier != engine.TierWorldEdit && tier != engine.TierFAWE {
		return "", fmt.Errorf("WorldEdit is not available on this server (tier: %s)", tier)
	}
	if _, ok := te.bot.GetSelection(); !ok {
		return "", fmt.Errorf("no selection set — use set-selection or wand to select a region first")
	}

	var patIn wePatternInput
	if len(input) > 0 {
		_ = json.Unmarshal(input, &patIn)
	}
	// For tools that require a pattern, validate it.
	if patIn.Pattern != "" {
		if err := validatePattern(patIn.Pattern); err != nil {
			return "", err
		}
	}

	cmd, err := buildCmd(patIn)
	if err != nil {
		return "", err
	}

	// Use RunBulkWECommand for most WE operations, RunWECommand for copy.
	var resp string
	if cmd == "copy" {
		resp, err = te.bot.RunWECommand(cmd)
	} else {
		resp, err = te.bot.RunBulkWECommand(cmd)
	}
	if err != nil {
		return "", fmt.Errorf("//%s failed: %w", cmd, err)
	}
	return jsonResult(map[string]string{"response": resp, "message": fmt.Sprintf("//%s: %s", cmd, resp)})
}

// weTierCmd checks connection and tier (no selection), then runs a WE command.
func (te *ToolExecutor) weTierCmd(input json.RawMessage, buildCmd func() (string, error)) (string, error) {
	if !te.bot.IsConnected() {
		return "", fmt.Errorf("bot is not connected to a Minecraft server")
	}
	tier := te.bot.GetTier()
	if tier != engine.TierWorldEdit && tier != engine.TierFAWE {
		return "", fmt.Errorf("WorldEdit is not available on this server (tier: %s)", tier)
	}

	_ = input // consumed by buildCmd closure
	cmd, err := buildCmd()
	if err != nil {
		return "", err
	}

	// paste, rotate, flip, undo, redo use RunWECommand; sphere/cyl/pyramid use RunBulkWECommand.
	var resp string
	switch {
	case cmd == "undo" || cmd == "redo" || startsWith(cmd, "paste") || startsWith(cmd, "rotate") || startsWith(cmd, "flip"):
		resp, err = te.bot.RunWECommand(cmd)
	default:
		resp, err = te.bot.RunBulkWECommand(cmd)
	}
	if err != nil {
		return "", fmt.Errorf("//%s failed: %w", cmd, err)
	}
	return jsonResult(map[string]string{"response": resp, "message": fmt.Sprintf("//%s: %s", cmd, resp)})
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func jsonResult(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(b), nil
}

// calcYawPitch computes Minecraft yaw and pitch from a source to a target position.
func calcYawPitch(fromX, fromY, fromZ, toX, toY, toZ float64) (yaw, pitch float32) {
	dx := toX - fromX
	dy := toY - fromY
	dz := toZ - fromZ
	dist := math.Sqrt(dx*dx + dz*dz)
	yaw = float32(-math.Atan2(dx, dz) * 180 / math.Pi)
	pitch = float32(-math.Atan2(dy, dist) * 180 / math.Pi)
	return yaw, pitch
}

// isValidPlayerName checks that a Minecraft username is safe for command interpolation.
// Valid names: 3-16 characters, alphanumeric + underscore only.
func isValidPlayerName(name string) bool {
	if len(name) < 3 || len(name) > 16 {
		return false
	}
	for _, c := range name {
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		if !isAlpha && !isDigit && c != '_' {
			return false
		}
	}
	return true
}

// validatePattern checks that a WorldEdit pattern is safe.
var validPatternRe = mcp.ValidPatternRe

func validatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if !validPatternRe.MatchString(pattern) {
		return fmt.Errorf("invalid block pattern: %q", pattern)
	}
	return nil
}

// validateExpression checks that a //generate expression is safe.
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

// decomposeFill splits a region into sub-regions that each fit within 32768 blocks.
func decomposeFill(x1, y1, z1, x2, y2, z2 int) [][6]int {
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
	if dx*dy*dz <= 32768 {
		return [][6]int{{x1, y1, z1, x2, y2, z2}}
	}
	if dx >= dy && dx >= dz {
		mid := x1 + dx/2 - 1
		return append(decomposeFill(x1, y1, z1, mid, y2, z2), decomposeFill(mid+1, y1, z1, x2, y2, z2)...)
	} else if dy >= dz {
		mid := y1 + dy/2 - 1
		return append(decomposeFill(x1, y1, z1, x2, mid, z2), decomposeFill(x1, mid+1, z1, x2, y2, z2)...)
	}
	mid := z1 + dz/2 - 1
	return append(decomposeFill(x1, y1, z1, x2, y2, mid), decomposeFill(x1, y1, mid+1, x2, y2, z2)...)
}
