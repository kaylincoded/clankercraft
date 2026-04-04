package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kaylincoded/clankercraft/internal/connection"
	"github.com/kaylincoded/clankercraft/internal/engine"
)

// mockBot implements mcp.BotState for testing.
type mockBot struct {
	connected     bool
	position      connection.Position
	positionKnown bool
	gamemode      string
	tier          engine.Tier
	selection     engine.Selection
	hasSelection  bool
	hasPos1       bool
	hasPos2       bool
	lastWECmd     string
	lastCmd       string
	weResp        string
	cmdResp       string
	weErr         error
	cmdErr        error
	blockAtResp   string
	blockAtErr    error
}

func (m *mockBot) IsConnected() bool                         { return m.connected }
func (m *mockBot) GetPosition() (connection.Position, bool)  { return m.position, m.positionKnown }
func (m *mockBot) SendRotation(yaw, pitch float32) error     { return nil }
func (m *mockBot) BlockAt(x, y, z int) (string, error)       { return m.blockAtResp, m.blockAtErr }
func (m *mockBot) FindBlock(blockType string, maxDist int) (int, int, int, bool, error) {
	return 0, 0, 0, false, nil
}
func (m *mockBot) ScanArea(x1, y1, z1, x2, y2, z2 int) ([]connection.BlockInfo, error) {
	return nil, nil
}
func (m *mockBot) ReadSign(x, y, z int) (connection.SignText, string, error) {
	return connection.SignText{}, "", nil
}
func (m *mockBot) FindSigns(maxDist int) ([]connection.SignInfo, error) { return nil, nil }
func (m *mockBot) GetGamemode() string                                 { return m.gamemode }
func (m *mockBot) GetTier() engine.Tier                                { return m.tier }
func (m *mockBot) SetSelection(x1, y1, z1, x2, y2, z2 int) error      { return nil }
func (m *mockBot) GetSelection() (engine.Selection, bool)              { return m.selection, m.hasSelection }
func (m *mockBot) HasPos1() bool                                       { return m.hasPos1 }
func (m *mockBot) HasPos2() bool                                       { return m.hasPos2 }
func (m *mockBot) RunWECommand(command string) (string, error) {
	m.lastWECmd = command
	return m.weResp, m.weErr
}
func (m *mockBot) RunCommand(command string) (string, error) { return m.cmdResp, m.cmdErr }
func (m *mockBot) RunBulkWECommand(command string) (string, error) {
	m.lastWECmd = command
	return m.weResp, m.weErr
}
func (m *mockBot) RunBulkCommand(command string) (string, error) {
	m.lastCmd = command
	return m.cmdResp, m.cmdErr
}
func (m *mockBot) OnWhisper(handler func(sender, message string)) {}
func (m *mockBot) SendWhisper(player, message string) error       { return nil }

func TestToolDefsCount(t *testing.T) {
	bot := &mockBot{}
	te := NewToolExecutor(bot)
	defs := te.ToolDefs()
	if len(defs) != 35 {
		t.Fatalf("got %d tool defs, want 35", len(defs))
	}
}

func TestToolDefsHaveNamesAndDescriptions(t *testing.T) {
	bot := &mockBot{}
	te := NewToolExecutor(bot)
	for _, def := range te.ToolDefs() {
		if def.Name == "" {
			t.Error("tool def has empty name")
		}
		if def.Description == "" {
			t.Errorf("tool %q has empty description", def.Name)
		}
	}
}

func TestToolDefsHaveValidSchemas(t *testing.T) {
	bot := &mockBot{}
	te := NewToolExecutor(bot)
	for _, def := range te.ToolDefs() {
		if len(def.InputSchema) > 0 {
			var m map[string]any
			if err := json.Unmarshal(def.InputSchema, &m); err != nil {
				t.Errorf("tool %q has invalid JSON schema: %v", def.Name, err)
			}
			if _, ok := m["properties"]; !ok {
				t.Errorf("tool %q schema missing 'properties'", def.Name)
			}
		}
	}
}

func TestExecutePing(t *testing.T) {
	bot := &mockBot{}
	te := NewToolExecutor(bot)
	result, err := te.Execute(context.Background(), "ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]string
	json.Unmarshal([]byte(result), &m)
	if m["status"] != "pong" {
		t.Errorf("status = %q, want pong", m["status"])
	}
}

func TestExecuteUnknownTool(t *testing.T) {
	bot := &mockBot{}
	te := NewToolExecutor(bot)
	_, err := te.Execute(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if err.Error() != "unknown tool: nonexistent" {
		t.Errorf("error = %q, want 'unknown tool: nonexistent'", err.Error())
	}
}

func TestExecuteGetPositionConnected(t *testing.T) {
	bot := &mockBot{
		connected:     true,
		positionKnown: true,
		position:      connection.Position{X: 10.5, Y: 64.0, Z: -30.7},
	}
	te := NewToolExecutor(bot)
	result, err := te.Execute(context.Background(), "get-position", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var m map[string]any
	json.Unmarshal([]byte(result), &m)
	if m["x"] != float64(10) {
		t.Errorf("x = %v, want 10", m["x"])
	}
	if m["y"] != float64(64) {
		t.Errorf("y = %v, want 64", m["y"])
	}
}

func TestExecuteGetPositionDisconnected(t *testing.T) {
	bot := &mockBot{connected: false}
	te := NewToolExecutor(bot)
	_, err := te.Execute(context.Background(), "get-position", nil)
	if err == nil {
		t.Fatal("expected error when disconnected")
	}
}

func TestExecuteWESetWithSelection(t *testing.T) {
	bot := &mockBot{
		connected:    true,
		tier:         engine.TierWorldEdit,
		hasSelection: true,
		selection:    engine.Selection{X1: 0, Y1: 0, Z1: 0, X2: 10, Y2: 10, Z2: 10},
		weResp:       "1000 blocks changed",
	}
	te := NewToolExecutor(bot)
	result, err := te.Execute(context.Background(), "we-set", json.RawMessage(`{"pattern":"stone"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bot.lastWECmd != "set stone" {
		t.Errorf("WE cmd = %q, want 'set stone'", bot.lastWECmd)
	}
	var m map[string]string
	json.Unmarshal([]byte(result), &m)
	if m["response"] != "1000 blocks changed" {
		t.Errorf("response = %q", m["response"])
	}
}

func TestExecuteWESetNoSelection(t *testing.T) {
	bot := &mockBot{
		connected:    true,
		tier:         engine.TierWorldEdit,
		hasSelection: false,
	}
	te := NewToolExecutor(bot)
	_, err := te.Execute(context.Background(), "we-set", json.RawMessage(`{"pattern":"stone"}`))
	if err == nil {
		t.Fatal("expected error without selection")
	}
}

func TestExecuteWESetNoWorldEdit(t *testing.T) {
	bot := &mockBot{
		connected:    true,
		tier:         engine.TierVanilla,
		hasSelection: true,
	}
	te := NewToolExecutor(bot)
	_, err := te.Execute(context.Background(), "we-set", json.RawMessage(`{"pattern":"stone"}`))
	if err == nil {
		t.Fatal("expected error without WorldEdit")
	}
}

func TestExecuteTeleportToPlayerConnected(t *testing.T) {
	bot := &mockBot{
		connected: true,
		cmdResp:   "Teleported Bot to Steve",
	}
	te := NewToolExecutor(bot)
	result, err := te.Execute(context.Background(), "teleport-to-player", json.RawMessage(`{"player":"Steve"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bot.lastCmd != "tp @s Steve" {
		t.Errorf("cmd = %q, want 'tp @s Steve'", bot.lastCmd)
	}
	var m map[string]string
	json.Unmarshal([]byte(result), &m)
	if m["message"] != "Teleported to Steve" {
		t.Errorf("message = %q", m["message"])
	}
}

func TestExecuteTeleportToPlayerDisconnected(t *testing.T) {
	bot := &mockBot{connected: false}
	te := NewToolExecutor(bot)
	_, err := te.Execute(context.Background(), "teleport-to-player", json.RawMessage(`{"player":"Steve"}`))
	if err == nil {
		t.Fatal("expected error when disconnected")
	}
}

func TestExecuteTeleportToPlayerInvalidName(t *testing.T) {
	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot)

	cases := []struct {
		name  string
		input string
	}{
		{"spaces", `{"player":"Steve Jobs"}`},
		{"special chars", `{"player":"Steve;drop"}`},
		{"too short", `{"player":"AB"}`},
		{"too long", `{"player":"ABCDEFGHIJKLMNOPQR"}`},
		{"slash injection", `{"player":"../etc/passwd"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := te.Execute(context.Background(), "teleport-to-player", json.RawMessage(tc.input))
			if err == nil {
				t.Error("expected error for invalid player name")
			}
		})
	}
}

func TestIsValidPlayerName(t *testing.T) {
	valid := []string{"Steve", "Alex", "abc", "A_B_C_D_E_F_G_H", "Player_123"}
	for _, name := range valid {
		if !isValidPlayerName(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	invalid := []string{"", "AB", "ABCDEFGHIJKLMNOPQR", "has space", "semi;colon", "slash/path", "dot.name", "a-b"}
	for _, name := range invalid {
		if isValidPlayerName(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

func TestExecuteSetblock(t *testing.T) {
	bot := &mockBot{
		connected: true,
		cmdResp:   "Changed the block",
	}
	te := NewToolExecutor(bot)
	result, err := te.Execute(context.Background(), "setblock", json.RawMessage(`{"x":10,"y":64,"z":-30,"block":"stone"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bot.lastCmd != "setblock 10 64 -30 stone" {
		t.Errorf("cmd = %q", bot.lastCmd)
	}
	var m map[string]string
	json.Unmarshal([]byte(result), &m)
	if m["response"] != "Changed the block" {
		t.Errorf("response = %q", m["response"])
	}
}
