package agent

import (
	"encoding/json"

	"github.com/kaylincoded/clankercraft/internal/llm"
)

// buildToolDefs returns the static list of all 34 tools as LLM ToolDef.
func (te *ToolExecutor) buildToolDefs() []llm.ToolDef {
	return []llm.ToolDef{
		td("ping", "Check if the bot is responsive", nil),
		td("status", "Get the bot's connection status to the Minecraft server", nil),
		td("get-position", "Get the bot's current position and facing direction in the Minecraft world", nil),
		td("look-at", "Make the bot face toward the specified coordinates", schema(
			prop("x", "number", "X coordinate to look at"),
			prop("y", "number", "Y coordinate to look at"),
			prop("z", "number", "Z coordinate to look at"),
		).require("x", "y", "z")),
		td("get-block-info", "Get the block type at specific coordinates in the Minecraft world", schema(
			prop("x", "integer", "X coordinate"),
			prop("y", "integer", "Y coordinate"),
			prop("z", "integer", "Z coordinate"),
		).require("x", "y", "z")),
		td("find-block", "Find the nearest block of a given type within a search distance", schema(
			prop("blockType", "string", "block type to search for, e.g. minecraft:stone"),
			prop("maxDistance", "integer", "max search distance in blocks (default 16, max 64)"),
		).require("blockType")),
		td("scan-area", "Scan a rectangular region and return all non-air blocks with their types and positions (max 10,000 blocks)", schema(
			prop("x1", "integer", "first corner X coordinate"),
			prop("y1", "integer", "first corner Y coordinate"),
			prop("z1", "integer", "first corner Z coordinate"),
			prop("x2", "integer", "second corner X coordinate"),
			prop("y2", "integer", "second corner Y coordinate"),
			prop("z2", "integer", "second corner Z coordinate"),
		).require("x1", "y1", "z1", "x2", "y2", "z2")),
		td("read-sign", "Read the text on a sign at the specified coordinates (returns front and back text)", schema(
			prop("x", "integer", "X coordinate of the sign"),
			prop("y", "integer", "Y coordinate of the sign"),
			prop("z", "integer", "Z coordinate of the sign"),
		).require("x", "y", "z")),
		td("find-signs", "Find all signs within a distance of the bot and return their text and positions (max 50 signs)", schema(
			prop("maxDistance", "integer", "max search distance in blocks (default 16, max 64)"),
		)),
		td("detect-gamemode", "Get the bot's current game mode (survival, creative, adventure, spectator)", nil),
		td("detect-worldedit", "Get the server's WorldEdit capability tier (fawe, worldedit, vanilla, or unknown if still detecting)", nil),
		td("set-selection", "Set a WorldEdit selection by specifying two corner positions", schema(
			prop("x1", "integer", "first corner X coordinate"),
			prop("y1", "integer", "first corner Y coordinate"),
			prop("z1", "integer", "first corner Z coordinate"),
			prop("x2", "integer", "second corner X coordinate"),
			prop("y2", "integer", "second corner Y coordinate"),
			prop("z2", "integer", "second corner Z coordinate"),
		).require("x1", "y1", "z1", "x2", "y2", "z2")),
		td("get-selection", "Get the current WorldEdit selection coordinates, or report that no selection is set", nil),
		// WorldEdit selection-required
		td("we-set", "Fill the current selection with a block pattern using WorldEdit //set", schema(
			prop("pattern", "string", "block pattern, e.g. stone_bricks or 50%stone,50%cobblestone"),
		).require("pattern")),
		td("we-replace", "Replace blocks in the current selection using WorldEdit //replace", schema(
			prop("from", "string", "source block pattern to replace"),
			prop("to", "string", "target block pattern to replace with"),
		).require("from", "to")),
		td("we-walls", "Set only the walls (not floor/ceiling) of the current selection using WorldEdit //walls", schema(
			prop("pattern", "string", "block pattern for the walls"),
		).require("pattern")),
		td("we-faces", "Set all 6 faces of the current selection using WorldEdit //faces", schema(
			prop("pattern", "string", "block pattern for all 6 faces"),
		).require("pattern")),
		td("we-hollow", "Hollow out the current selection using WorldEdit //hollow, optionally filling the shell with a pattern", schema(
			prop("pattern", "string", "optional block pattern for the shell (default: existing blocks)"),
		)),
		td("we-sphere", "Create a sphere at the bot's position using WorldEdit //sphere (set hollow=true for //hsphere)", schema(
			prop("pattern", "string", "block pattern, e.g. stone"),
			prop("radius", "integer", "sphere radius in blocks"),
			prop("hollow", "boolean", "if true, creates a hollow sphere"),
		).require("pattern", "radius")),
		td("we-cyl", "Create a cylinder at the bot's position using WorldEdit //cyl (set hollow=true for //hcyl)", schema(
			prop("pattern", "string", "block pattern, e.g. stone"),
			prop("radius", "integer", "cylinder radius in blocks"),
			prop("height", "integer", "cylinder height (default 1)"),
			prop("hollow", "boolean", "if true, creates a hollow cylinder"),
		).require("pattern", "radius")),
		td("we-pyramid", "Create a pyramid at the bot's position using WorldEdit //pyramid (set hollow=true for //hpyramid)", schema(
			prop("pattern", "string", "block pattern, e.g. stone"),
			prop("size", "integer", "pyramid size (base half-width)"),
			prop("hollow", "boolean", "if true, creates a hollow pyramid"),
		).require("pattern", "size")),
		td("we-generate", "Generate blocks from a mathematical expression in the current selection using WorldEdit //generate", schema(
			prop("expression", "string", "mathematical expression, e.g. (x*x + z*z < 100) * stone"),
		).require("expression")),
		td("we-smooth", "Smooth terrain in the current selection using WorldEdit //smooth (default 1 iteration)", schema(
			prop("iterations", "integer", "number of smooth iterations (default 1)"),
		)),
		td("we-naturalize", "Naturalize terrain in the current selection using WorldEdit //naturalize (grass on top, dirt below, stone deep)", nil),
		td("we-overlay", "Overlay a block pattern on top of existing blocks in the current selection using WorldEdit //overlay", schema(
			prop("pattern", "string", "block pattern to overlay, e.g. grass_block"),
		).require("pattern")),
		td("we-copy", "Copy the current selection to the clipboard using WorldEdit //copy", nil),
		td("we-paste", "Paste the clipboard at the bot's position using WorldEdit //paste (set skipAir=true to skip air blocks)", schema(
			prop("skipAir", "boolean", "if true, skips air blocks when pasting"),
		)),
		td("we-rotate", "Rotate the clipboard contents using WorldEdit //rotate (90, 180, or 270 degrees)", schema(
			prop("degrees", "integer", "rotation in degrees: 90, 180, or 270"),
		).require("degrees")),
		td("we-flip", "Flip the clipboard contents using WorldEdit //flip (direction: north/south/east/west/up/down)", schema(
			prop("direction", "string", "flip direction: north, south, east, west, up, down (default: player facing)"),
		)),
		td("we-undo", "Undo the last WorldEdit operation using //undo", nil),
		td("we-redo", "Redo the last undone WorldEdit operation using //redo", nil),
		// Vanilla commands
		td("setblock", "Place a single block at the specified coordinates using /setblock", schema(
			prop("x", "integer", "X coordinate"),
			prop("y", "integer", "Y coordinate"),
			prop("z", "integer", "Z coordinate"),
			prop("block", "string", "block type, e.g. minecraft:stone"),
		).require("x", "y", "z", "block")),
		td("fill", "Fill a region with a block type using /fill (auto-decomposes regions larger than 32,768 blocks)", schema(
			prop("x1", "integer", "first corner X coordinate"),
			prop("y1", "integer", "first corner Y coordinate"),
			prop("z1", "integer", "first corner Z coordinate"),
			prop("x2", "integer", "second corner X coordinate"),
			prop("y2", "integer", "second corner Y coordinate"),
			prop("z2", "integer", "second corner Z coordinate"),
			prop("block", "string", "block type, e.g. minecraft:stone"),
		).require("x1", "y1", "z1", "x2", "y2", "z2", "block")),
		td("clone", "Clone a region to a destination using /clone", schema(
			prop("x1", "integer", "source first corner X"),
			prop("y1", "integer", "source first corner Y"),
			prop("z1", "integer", "source first corner Z"),
			prop("x2", "integer", "source second corner X"),
			prop("y2", "integer", "source second corner Y"),
			prop("z2", "integer", "source second corner Z"),
			prop("dx", "integer", "destination X coordinate"),
			prop("dy", "integer", "destination Y coordinate"),
			prop("dz", "integer", "destination Z coordinate"),
		).require("x1", "y1", "z1", "x2", "y2", "z2", "dx", "dy", "dz")),
	}
}

// td builds an llm.ToolDef.
func td(name, description string, s *schemaBuilder) llm.ToolDef {
	def := llm.ToolDef{Name: name, Description: description}
	if s != nil {
		def.InputSchema = s.build()
	}
	return def
}

// schemaBuilder helps construct JSON Schema objects for tool definitions.
type schemaBuilder struct {
	props    map[string]any
	required []string
}

func schema(props ...propDef) *schemaBuilder {
	s := &schemaBuilder{props: make(map[string]any)}
	for _, p := range props {
		s.props[p.name] = map[string]string{"type": p.typ, "description": p.desc}
	}
	return s
}

func (s *schemaBuilder) require(names ...string) *schemaBuilder {
	s.required = names
	return s
}

func (s *schemaBuilder) build() json.RawMessage {
	m := map[string]any{"properties": s.props}
	if len(s.required) > 0 {
		m["required"] = s.required
	}
	b, _ := json.Marshal(m)
	return b
}

type propDef struct {
	name string
	typ  string
	desc string
}

func prop(name, typ, desc string) propDef {
	return propDef{name: name, typ: typ, desc: desc}
}
