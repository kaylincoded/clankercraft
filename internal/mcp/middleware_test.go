package mcp

import (
	"context"
	"fmt"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/kaylincoded/clankercraft/internal/connection"
)

// mockBotState satisfies BotState for testing.
type mockBotState struct {
	connected    bool
	pos          connection.Position
	hasPos       bool
	lastYaw      float32
	lastPitch    float32
	sendRotErr   error
	blockAtFn    func(x, y, z int) (string, error)
	findBlockFn  func(blockType string, maxDist int) (int, int, int, bool, error)
	scanAreaFn   func(x1, y1, z1, x2, y2, z2 int) ([]connection.BlockInfo, error)
}

func (m *mockBotState) IsConnected() bool { return m.connected }
func (m *mockBotState) GetPosition() (connection.Position, bool) { return m.pos, m.hasPos }
func (m *mockBotState) SendRotation(yaw, pitch float32) error {
	m.lastYaw = yaw
	m.lastPitch = pitch
	return m.sendRotErr
}
func (m *mockBotState) BlockAt(x, y, z int) (string, error) {
	if m.blockAtFn != nil {
		return m.blockAtFn(x, y, z)
	}
	return "", fmt.Errorf("not implemented")
}
func (m *mockBotState) FindBlock(blockType string, maxDist int) (int, int, int, bool, error) {
	if m.findBlockFn != nil {
		return m.findBlockFn(blockType, maxDist)
	}
	return 0, 0, 0, false, fmt.Errorf("not implemented")
}
func (m *mockBotState) ScanArea(x1, y1, z1, x2, y2, z2 int) ([]connection.BlockInfo, error) {
	if m.scanAreaFn != nil {
		return m.scanAreaFn(x1, y1, z1, x2, y2, z2)
	}
	return nil, fmt.Errorf("not implemented")
}

func TestRequireConnectionRejectsDisconnected(t *testing.T) {
	checker := &mockBotState{connected: false}
	handler := requireConnection(checker, func(_ context.Context, _ *gomcp.CallToolRequest, _ pingInput) (*gomcp.CallToolResult, pingOutput, error) {
		t.Fatal("handler should not be called when disconnected")
		return nil, pingOutput{}, nil
	})

	result, _, err := handler(context.Background(), nil, pingInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected error result, got nil")
	}
	if !result.IsError {
		t.Error("expected IsError=true")
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

func TestRequireConnectionAllowsConnected(t *testing.T) {
	checker := &mockBotState{connected: true}
	called := false
	handler := requireConnection(checker, func(_ context.Context, _ *gomcp.CallToolRequest, _ pingInput) (*gomcp.CallToolResult, pingOutput, error) {
		called = true
		return nil, pingOutput{Status: "ok"}, nil
	})

	result, out, err := handler(context.Background(), nil, pingInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if result != nil {
		t.Errorf("expected nil result (auto-populated), got %+v", result)
	}
	if out.Status != "ok" {
		t.Errorf("output status = %q, want %q", out.Status, "ok")
	}
}

func TestToolErrorFormat(t *testing.T) {
	result := toolError("something went wrong")
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(*gomcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want *TextContent", result.Content[0])
	}
	if text.Text != "something went wrong" {
		t.Errorf("text = %q, want %q", text.Text, "something went wrong")
	}
}
