package mcp

import (
	"context"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type mockConnChecker struct{ connected bool }

func (m *mockConnChecker) IsConnected() bool { return m.connected }

func TestRequireConnectionRejectsDisconnected(t *testing.T) {
	checker := &mockConnChecker{connected: false}
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
	checker := &mockConnChecker{connected: true}
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
