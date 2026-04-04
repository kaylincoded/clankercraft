package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/kaylincoded/clankercraft/internal/connection"
	"github.com/kaylincoded/clankercraft/internal/engine"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// testSession creates a Server with the given BotState, connects an in-memory client,
// and returns the session. The server and client are torn down when the test ends.
func testSession(t *testing.T, state BotState) *gomcp.ClientSession {
	t.Helper()
	srv := New("test-version", testLogger(), state)

	clientTransport, serverTransport := gomcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go srv.server.Run(ctx, serverTransport)

	client := gomcp.NewClient(&gomcp.Implementation{
		Name:    "test-client",
		Version: "v1.0.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })

	return session
}

func TestNewCreatesServer(t *testing.T) {
	srv := New("test-version", testLogger(), &mockBotState{})
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.server == nil {
		t.Error("server field is nil")
	}
	if srv.logger == nil {
		t.Error("logger field is nil")
	}
}

func TestPingToolRegistered(t *testing.T) {
	session := testSession(t, &mockBotState{})

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "ping" {
			found = true
			if tool.Description == "" {
				t.Error("ping tool has empty description")
			}
		}
	}
	if !found {
		t.Error("ping tool not found in tool list")
	}
}

func TestStatusToolRegistered(t *testing.T) {
	session := testSession(t, &mockBotState{})

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "status" {
			found = true
			if tool.Description == "" {
				t.Error("status tool has empty description")
			}
		}
	}
	if !found {
		t.Error("status tool not found in tool list")
	}
}

func TestPingToolReturns(t *testing.T) {
	session := testSession(t, &mockBotState{})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "ping",
	})
	if err != nil {
		t.Fatalf("CallTool(ping): %v", err)
	}
	if result.IsError {
		t.Error("ping returned error")
	}
	if len(result.Content) == 0 {
		t.Fatal("ping returned no content")
	}
	text, ok := result.Content[0].(*gomcp.TextContent)
	if !ok {
		t.Fatalf("ping content type = %T, want *TextContent", result.Content[0])
	}
	if text.Text == "" {
		t.Error("ping returned empty text")
	}
}

func TestPingWorksWithoutConnection(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "ping",
	})
	if err != nil {
		t.Fatalf("CallTool(ping): %v", err)
	}
	if result.IsError {
		t.Error("ping should succeed even when disconnected")
	}
}

func TestStatusToolWhenConnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: true})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "status",
	})
	if err != nil {
		t.Fatalf("CallTool(status): %v", err)
	}
	if result.IsError {
		t.Error("status returned error when connected")
	}
}

func TestStatusToolWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "status",
	})
	if err != nil {
		t.Fatalf("CallTool(status): %v", err)
	}
	if !result.IsError {
		t.Error("status should return error when disconnected")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
	text, ok := result.Content[0].(*gomcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *TextContent", result.Content[0])
	}
	if text.Text != "bot is not connected to a Minecraft server" {
		t.Errorf("error text = %q, want connection error message", text.Text)
	}
}

// --- get-position tool tests ---

func TestGetPositionWhenConnectedWithPosition(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		hasPos:    true,
		pos:       connection.Position{X: 100.7, Y: 64.3, Z: -200.9, Yaw: 45.0, Pitch: -10.0},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get-position",
	})
	if err != nil {
		t.Fatalf("CallTool(get-position): %v", err)
	}
	if result.IsError {
		t.Errorf("get-position returned error: %v", result.Content)
	}
}

func TestGetPositionWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get-position",
	})
	if err != nil {
		t.Fatalf("CallTool(get-position): %v", err)
	}
	if !result.IsError {
		t.Error("get-position should return error when disconnected")
	}
}

func TestGetPositionBeforeServerSendsPosition(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		hasPos:    false, // no position received yet
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get-position",
	})
	if err != nil {
		t.Fatalf("CallTool(get-position): %v", err)
	}
	if !result.IsError {
		t.Error("get-position should return error when position not yet known")
	}
	text := result.Content[0].(*gomcp.TextContent)
	if text.Text != "position not yet known (waiting for server)" {
		t.Errorf("unexpected error text: %q", text.Text)
	}
}

// --- look-at tool tests ---

func TestLookAtWhenConnected(t *testing.T) {
	mock := &mockBotState{
		connected: true,
		hasPos:    true,
		pos:       connection.Position{X: 0, Y: 65, Z: 0},
	}
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "look-at",
		Arguments: map[string]any{"x": 10.0, "y": 65.0, "z": 0.0},
	})
	if err != nil {
		t.Fatalf("CallTool(look-at): %v", err)
	}
	if result.IsError {
		t.Errorf("look-at returned error: %v", result.Content)
	}
	// Verify rotation was sent (looking east = +X from origin, yaw should be ~-90 or 270)
	if mock.lastYaw == 0 && mock.lastPitch == 0 {
		t.Error("SendRotation was not called")
	}
}

func TestLookAtWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "look-at",
		Arguments: map[string]any{"x": 10.0, "y": 65.0, "z": 0.0},
	})
	if err != nil {
		t.Fatalf("CallTool(look-at): %v", err)
	}
	if !result.IsError {
		t.Error("look-at should return error when disconnected")
	}
}

func TestLookAtBeforePositionKnown(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		hasPos:    false,
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "look-at",
		Arguments: map[string]any{"x": 10.0, "y": 65.0, "z": 0.0},
	})
	if err != nil {
		t.Fatalf("CallTool(look-at): %v", err)
	}
	if !result.IsError {
		t.Error("look-at should return error when position not yet known")
	}
}

// --- yaw/pitch calculation unit tests ---

func TestCalcYawPitchDirections(t *testing.T) {
	tests := []struct {
		name       string
		toX, toY, toZ float64
		wantYaw    float32
		wantPitch  float32
		tolerance  float32
	}{
		{
			name: "looking south (+Z)",
			toX: 0, toY: 65, toZ: 10,
			wantYaw: 0, wantPitch: 0, tolerance: 0.1,
		},
		{
			name: "looking north (-Z)",
			toX: 0, toY: 65, toZ: -10,
			wantYaw: -180, wantPitch: 0, tolerance: 0.1, // -180 and 180 are equivalent
		},
		{
			name: "looking east (+X)",
			toX: 10, toY: 65, toZ: 0,
			wantYaw: -90, wantPitch: 0, tolerance: 0.1,
		},
		{
			name: "looking west (-X)",
			toX: -10, toY: 65, toZ: 0,
			wantYaw: 90, wantPitch: 0, tolerance: 0.1,
		},
		{
			name: "looking straight up",
			toX: 0, toY: 75, toZ: 0,
			wantYaw: 0, wantPitch: -90, tolerance: 0.1,
		},
		{
			name: "looking straight down",
			toX: 0, toY: 55, toZ: 0,
			wantYaw: 0, wantPitch: 90, tolerance: 0.1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// From origin at y=65
			yaw, pitch := calcYawPitch(0, 65, 0, tc.toX, tc.toY, tc.toZ)
			if math.Abs(float64(yaw-tc.wantYaw)) > float64(tc.tolerance) {
				t.Errorf("yaw = %v, want %v (tolerance %v)", yaw, tc.wantYaw, tc.tolerance)
			}
			if math.Abs(float64(pitch-tc.wantPitch)) > float64(tc.tolerance) {
				t.Errorf("pitch = %v, want %v (tolerance %v)", pitch, tc.wantPitch, tc.tolerance)
			}
		})
	}
}

// --- get-block-info tool tests ---

func TestGetBlockInfoWhenConnected(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		blockAtFn: func(x, y, z int) (string, error) {
			return "minecraft:stone", nil
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get-block-info",
		Arguments: map[string]any{"x": 10, "y": 64, "z": -5},
	})
	if err != nil {
		t.Fatalf("CallTool(get-block-info): %v", err)
	}
	if result.IsError {
		t.Errorf("get-block-info returned error: %v", result.Content)
	}
}

func TestGetBlockInfoChunkNotLoaded(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		blockAtFn: func(x, y, z int) (string, error) {
			return "", fmt.Errorf("chunk at (0, 0) not loaded")
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get-block-info",
		Arguments: map[string]any{"x": 10, "y": 64, "z": -5},
	})
	if err != nil {
		t.Fatalf("CallTool(get-block-info): %v", err)
	}
	if !result.IsError {
		t.Error("get-block-info should return error for unloaded chunk")
	}
}

func TestGetBlockInfoWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get-block-info",
		Arguments: map[string]any{"x": 0, "y": 64, "z": 0},
	})
	if err != nil {
		t.Fatalf("CallTool(get-block-info): %v", err)
	}
	if !result.IsError {
		t.Error("get-block-info should return error when disconnected")
	}
}

// --- find-block tool tests ---

func TestFindBlockWhenFound(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		findBlockFn: func(blockType string, maxDist int) (int, int, int, bool, error) {
			if blockType == "minecraft:diamond_ore" {
				return 5, 12, -3, true, nil
			}
			return 0, 0, 0, false, nil
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "find-block",
		Arguments: map[string]any{"blockType": "minecraft:diamond_ore"},
	})
	if err != nil {
		t.Fatalf("CallTool(find-block): %v", err)
	}
	if result.IsError {
		t.Errorf("find-block returned error: %v", result.Content)
	}
}

func TestFindBlockNotFound(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		findBlockFn: func(blockType string, maxDist int) (int, int, int, bool, error) {
			return 0, 0, 0, false, nil
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "find-block",
		Arguments: map[string]any{"blockType": "minecraft:diamond_ore"},
	})
	if err != nil {
		t.Fatalf("CallTool(find-block): %v", err)
	}
	if result.IsError {
		t.Error("find-block should not return IsError for not-found (it's a valid response)")
	}
}

func TestFindBlockWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "find-block",
		Arguments: map[string]any{"blockType": "minecraft:stone"},
	})
	if err != nil {
		t.Fatalf("CallTool(find-block): %v", err)
	}
	if !result.IsError {
		t.Error("find-block should return error when disconnected")
	}
}

// --- scan-area tool tests ---

func TestScanAreaWhenConnected(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		scanAreaFn: func(x1, y1, z1, x2, y2, z2 int) ([]connection.BlockInfo, error) {
			return []connection.BlockInfo{
				{Block: "minecraft:stone", X: 0, Y: 64, Z: 0},
				{Block: "minecraft:dirt", X: 1, Y: 64, Z: 0},
			}, nil
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "scan-area",
		Arguments: map[string]any{"x1": 0, "y1": 64, "z1": 0, "x2": 5, "y2": 64, "z2": 5},
	})
	if err != nil {
		t.Fatalf("CallTool(scan-area): %v", err)
	}
	if result.IsError {
		t.Errorf("scan-area returned error: %v", result.Content)
	}
}

func TestScanAreaTooLarge(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		scanAreaFn: func(x1, y1, z1, x2, y2, z2 int) ([]connection.BlockInfo, error) {
			return nil, fmt.Errorf("region too large: 30000 blocks (max 10000)")
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "scan-area",
		Arguments: map[string]any{"x1": 0, "y1": 0, "z1": 0, "x2": 99, "y2": 29, "z2": 9},
	})
	if err != nil {
		t.Fatalf("CallTool(scan-area): %v", err)
	}
	if !result.IsError {
		t.Error("scan-area should return error for oversized region")
	}
}

func TestScanAreaWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "scan-area",
		Arguments: map[string]any{"x1": 0, "y1": 64, "z1": 0, "x2": 5, "y2": 64, "z2": 5},
	})
	if err != nil {
		t.Fatalf("CallTool(scan-area): %v", err)
	}
	if !result.IsError {
		t.Error("scan-area should return error when disconnected")
	}
}

// --- read-sign tool tests ---

func TestReadSignWhenConnected(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		readSignFn: func(x, y, z int) (connection.SignText, string, error) {
			return connection.SignText{
				FrontLines: [4]string{"Hello", "World", "", ""},
				BackLines:  [4]string{"Back", "", "", ""},
			}, "minecraft:oak_sign", nil
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "read-sign",
		Arguments: map[string]any{"x": 10, "y": 64, "z": -5},
	})
	if err != nil {
		t.Fatalf("CallTool(read-sign): %v", err)
	}
	if result.IsError {
		t.Errorf("read-sign returned error: %v", result.Content)
	}
}

func TestReadSignNoSignAtPosition(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		readSignFn: func(x, y, z int) (connection.SignText, string, error) {
			return connection.SignText{}, "", fmt.Errorf("no sign at (%d, %d, %d)", x, y, z)
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "read-sign",
		Arguments: map[string]any{"x": 10, "y": 64, "z": -5},
	})
	if err != nil {
		t.Fatalf("CallTool(read-sign): %v", err)
	}
	if !result.IsError {
		t.Error("read-sign should return error when no sign at position")
	}
}

func TestReadSignWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "read-sign",
		Arguments: map[string]any{"x": 0, "y": 64, "z": 0},
	})
	if err != nil {
		t.Fatalf("CallTool(read-sign): %v", err)
	}
	if !result.IsError {
		t.Error("read-sign should return error when disconnected")
	}
}

// --- find-signs tool tests ---

func TestFindSignsWhenFound(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		findSignsFn: func(maxDist int) ([]connection.SignInfo, error) {
			return []connection.SignInfo{
				{
					Sign:  connection.SignText{FrontLines: [4]string{"Shop", "Open", "", ""}},
					Block: "minecraft:oak_sign",
					X:     5, Y: 64, Z: 10,
				},
			}, nil
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "find-signs",
		Arguments: map[string]any{"maxDistance": 32},
	})
	if err != nil {
		t.Fatalf("CallTool(find-signs): %v", err)
	}
	if result.IsError {
		t.Errorf("find-signs returned error: %v", result.Content)
	}
}

func TestFindSignsNoneFound(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		findSignsFn: func(maxDist int) ([]connection.SignInfo, error) {
			return nil, nil
		},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "find-signs",
	})
	if err != nil {
		t.Fatalf("CallTool(find-signs): %v", err)
	}
	if result.IsError {
		t.Error("find-signs should not return error when none found")
	}
}

func TestFindSignsWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "find-signs",
	})
	if err != nil {
		t.Fatalf("CallTool(find-signs): %v", err)
	}
	if !result.IsError {
		t.Error("find-signs should return error when disconnected")
	}
}

// --- detect-gamemode tool tests ---

func TestDetectGamemodeWhenConnected(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		gamemode:  "creative",
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "detect-gamemode",
	})
	if err != nil {
		t.Fatalf("CallTool(detect-gamemode): %v", err)
	}
	if result.IsError {
		t.Errorf("detect-gamemode returned error: %v", result.Content)
	}
}

func TestDetectGamemodeWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "detect-gamemode",
	})
	if err != nil {
		t.Fatalf("CallTool(detect-gamemode): %v", err)
	}
	if !result.IsError {
		t.Error("detect-gamemode should return error when disconnected")
	}
}

// --- set-selection tool tests ---

func TestSetSelectionWhenConnected(t *testing.T) {
	mock := &mockBotState{connected: true}
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set-selection",
		Arguments: map[string]any{"x1": 0, "y1": 64, "z1": 0, "x2": 10, "y2": 70, "z2": 10},
	})
	if err != nil {
		t.Fatalf("CallTool(set-selection): %v", err)
	}
	if result.IsError {
		t.Errorf("set-selection returned error: %v", result.Content)
	}
	if !mock.hasPos1 || !mock.hasPos2 {
		t.Error("selection was not stored")
	}
	if mock.selection.X1 != 0 || mock.selection.Y2 != 70 {
		t.Errorf("selection = %+v, want pos1=(0,64,0) pos2=(10,70,10)", mock.selection)
	}
}

func TestSetSelectionWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set-selection",
		Arguments: map[string]any{"x1": 0, "y1": 64, "z1": 0, "x2": 10, "y2": 70, "z2": 10},
	})
	if err != nil {
		t.Fatalf("CallTool(set-selection): %v", err)
	}
	if !result.IsError {
		t.Error("set-selection should return error when disconnected")
	}
}

// --- get-selection tool tests ---

func TestGetSelectionWhenSet(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected:    true,
		hasPos1: true, hasPos2: true,
		selection:    engine.Selection{X1: 0, Y1: 64, Z1: 0, X2: 10, Y2: 70, Z2: 10},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get-selection",
	})
	if err != nil {
		t.Fatalf("CallTool(get-selection): %v", err)
	}
	if result.IsError {
		t.Errorf("get-selection returned error: %v", result.Content)
	}
}

func TestGetSelectionPartialPos1Only(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		hasPos1:   true,
		hasPos2:   false,
		selection: engine.Selection{X1: 100, Y1: 64, Z1: -200},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get-selection",
	})
	if err != nil {
		t.Fatalf("CallTool(get-selection): %v", err)
	}
	if result.IsError {
		t.Error("get-selection should not return error for partial selection (pos1 only)")
	}
}

func TestGetSelectionWhenNotSet(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected:    true,
		hasPos1: false, hasPos2: false,
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get-selection",
	})
	if err != nil {
		t.Fatalf("CallTool(get-selection): %v", err)
	}
	if !result.IsError {
		t.Error("get-selection should return error when no selection set")
	}
}

func TestGetSelectionWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get-selection",
	})
	if err != nil {
		t.Fatalf("CallTool(get-selection): %v", err)
	}
	if !result.IsError {
		t.Error("get-selection should return error when disconnected")
	}
}

// --- detect-worldedit tool tests ---

func TestDetectWorldeditReturnsWorldEdit(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		tier:      engine.TierWorldEdit,
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "detect-worldedit",
	})
	if err != nil {
		t.Fatalf("CallTool(detect-worldedit): %v", err)
	}
	if result.IsError {
		t.Errorf("detect-worldedit returned error: %v", result.Content)
	}
}

func TestDetectWorldeditReturnsFAWE(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		tier:      engine.TierFAWE,
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "detect-worldedit",
	})
	if err != nil {
		t.Fatalf("CallTool(detect-worldedit): %v", err)
	}
	if result.IsError {
		t.Errorf("detect-worldedit returned error: %v", result.Content)
	}
}

func TestDetectWorldeditWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "detect-worldedit",
	})
	if err != nil {
		t.Fatalf("CallTool(detect-worldedit): %v", err)
	}
	if !result.IsError {
		t.Error("detect-worldedit should return error when disconnected")
	}
}

// --- WorldEdit region operation tool tests ---

func weConnectedMock() *mockBotState {
	return &mockBotState{
		connected: true,
		tier:      engine.TierWorldEdit,
		hasPos1:   true,
		hasPos2:   true,
		selection: engine.Selection{X1: 0, Y1: 64, Z1: 0, X2: 10, Y2: 70, Z2: 10},
		runWECommandFn: func(command string) (string, error) {
			return "42 block(s) have been changed.", nil
		},
	}
}

func weCommandCaptureMock() (*mockBotState, *string) {
	var capturedCmd string
	mock := &mockBotState{
		connected: true,
		tier:      engine.TierWorldEdit,
		hasPos1:   true,
		hasPos2:   true,
		selection: engine.Selection{X1: 0, Y1: 64, Z1: 0, X2: 10, Y2: 70, Z2: 10},
		runWECommandFn: func(command string) (string, error) {
			capturedCmd = command
			return "42 block(s) have been changed.", nil
		},
	}
	return mock, &capturedCmd
}

func TestWESetSendsCorrectCommand(t *testing.T) {
	mock, cmd := weCommandCaptureMock()
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-set",
		Arguments: map[string]any{"pattern": "stone_bricks"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-set): %v", err)
	}
	if result.IsError {
		t.Errorf("we-set returned error: %v", result.Content)
	}
	if *cmd != "set stone_bricks" {
		t.Errorf("command = %q, want %q", *cmd, "set stone_bricks")
	}
}

func TestWEReplaceSendsCorrectCommand(t *testing.T) {
	mock, cmd := weCommandCaptureMock()
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-replace",
		Arguments: map[string]any{"from": "stone", "to": "cobblestone"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-replace): %v", err)
	}
	if result.IsError {
		t.Errorf("we-replace returned error: %v", result.Content)
	}
	if *cmd != "replace stone cobblestone" {
		t.Errorf("command = %q, want %q", *cmd, "replace stone cobblestone")
	}
}

func TestWEWallsSendsCorrectCommand(t *testing.T) {
	mock, cmd := weCommandCaptureMock()
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-walls",
		Arguments: map[string]any{"pattern": "glass"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-walls): %v", err)
	}
	if result.IsError {
		t.Errorf("we-walls returned error: %v", result.Content)
	}
	if *cmd != "walls glass" {
		t.Errorf("command = %q, want %q", *cmd, "walls glass")
	}
}

func TestWEFacesSendsCorrectCommand(t *testing.T) {
	mock, cmd := weCommandCaptureMock()
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-faces",
		Arguments: map[string]any{"pattern": "stone"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-faces): %v", err)
	}
	if result.IsError {
		t.Errorf("we-faces returned error: %v", result.Content)
	}
	if *cmd != "faces stone" {
		t.Errorf("command = %q, want %q", *cmd, "faces stone")
	}
}

func TestWEHollowNoPattern(t *testing.T) {
	mock, cmd := weCommandCaptureMock()
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "we-hollow",
	})
	if err != nil {
		t.Fatalf("CallTool(we-hollow): %v", err)
	}
	if result.IsError {
		t.Errorf("we-hollow returned error: %v", result.Content)
	}
	if *cmd != "hollow" {
		t.Errorf("command = %q, want %q", *cmd, "hollow")
	}
}

func TestWEHollowWithPattern(t *testing.T) {
	mock, cmd := weCommandCaptureMock()
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-hollow",
		Arguments: map[string]any{"pattern": "glass"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-hollow): %v", err)
	}
	if result.IsError {
		t.Errorf("we-hollow returned error: %v", result.Content)
	}
	if *cmd != "hollow glass" {
		t.Errorf("command = %q, want %q", *cmd, "hollow glass")
	}
}

func TestWEToolReturnsErrorFromRunWECommand(t *testing.T) {
	mock := weConnectedMock()
	mock.runWECommandFn = func(command string) (string, error) {
		return "", fmt.Errorf("no response from server within 5s")
	}
	session := testSession(t, mock)

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-set",
		Arguments: map[string]any{"pattern": "stone"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-set): %v", err)
	}
	if !result.IsError {
		t.Error("we-set should return error when RunWECommand fails")
	}
}

func TestWEToolRejectsInvalidPattern(t *testing.T) {
	session := testSession(t, weConnectedMock())

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-set",
		Arguments: map[string]any{"pattern": "stone\n/op hacker"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-set): %v", err)
	}
	if !result.IsError {
		t.Error("we-set should reject patterns with newlines")
	}
}

func TestWEToolRejectsVanillaTier(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		tier:      engine.TierVanilla,
		hasPos1:   true,
		hasPos2:   true,
		selection: engine.Selection{X1: 0, Y1: 64, Z1: 0, X2: 10, Y2: 70, Z2: 10},
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-set",
		Arguments: map[string]any{"pattern": "stone"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-set): %v", err)
	}
	if !result.IsError {
		t.Error("we-set should return error when tier is vanilla")
	}
	text := result.Content[0].(*gomcp.TextContent)
	if text.Text != "WorldEdit is not available on this server (tier: vanilla)" {
		t.Errorf("error text = %q", text.Text)
	}
}

func TestWEToolRejectsNoSelection(t *testing.T) {
	session := testSession(t, &mockBotState{
		connected: true,
		tier:      engine.TierWorldEdit,
		hasPos1:   false,
		hasPos2:   false,
	})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-set",
		Arguments: map[string]any{"pattern": "stone"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-set): %v", err)
	}
	if !result.IsError {
		t.Error("we-set should return error when no selection set")
	}
	text := result.Content[0].(*gomcp.TextContent)
	if text.Text != "no selection set — use set-selection or wand to select a region first" {
		t.Errorf("error text = %q", text.Text)
	}
}

func TestWEToolRejectsDisconnected(t *testing.T) {
	session := testSession(t, &mockBotState{connected: false})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "we-set",
		Arguments: map[string]any{"pattern": "stone"},
	})
	if err != nil {
		t.Fatalf("CallTool(we-set): %v", err)
	}
	if !result.IsError {
		t.Error("we-set should return error when disconnected")
	}
}

func TestServerRunCancellation(t *testing.T) {
	srv := New("test-version", testLogger(), &mockBotState{})

	_, serverTransport := gomcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- srv.server.Run(ctx, serverTransport)
	}()

	cancel()

	err := <-done
	if err != nil && err != context.Canceled {
		t.Errorf("Run() after cancel = %v, want nil or context.Canceled", err)
	}
}
