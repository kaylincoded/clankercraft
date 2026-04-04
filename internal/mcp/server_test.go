package mcp

import (
	"context"
	"log/slog"
	"os"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// testSession creates a Server with the given ConnChecker, connects an in-memory client,
// and returns the session. The server and client are torn down when the test ends.
func testSession(t *testing.T, checker ConnChecker) *gomcp.ClientSession {
	t.Helper()
	srv := New("test-version", testLogger(), checker)

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
	srv := New("test-version", testLogger(), &mockConnChecker{})
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
	session := testSession(t, &mockConnChecker{})

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
	session := testSession(t, &mockConnChecker{})

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
	session := testSession(t, &mockConnChecker{})

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
	// ping should work even when disconnected — it's a transport health check
	session := testSession(t, &mockConnChecker{connected: false})

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
	session := testSession(t, &mockConnChecker{connected: true})

	result, err := session.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "status",
	})
	if err != nil {
		t.Fatalf("CallTool(status): %v", err)
	}
	if result.IsError {
		t.Error("status returned error when connected")
	}
	if len(result.Content) == 0 {
		t.Fatal("status returned no content")
	}
}

func TestStatusToolWhenDisconnected(t *testing.T) {
	session := testSession(t, &mockConnChecker{connected: false})

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

func TestServerRunCancellation(t *testing.T) {
	srv := New("test-version", testLogger(), &mockConnChecker{})

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
