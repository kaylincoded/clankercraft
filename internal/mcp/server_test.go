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

func TestNewCreatesServer(t *testing.T) {
	srv := New("test-version", testLogger(), nil)
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
	srv := New("test-version", testLogger(), nil)

	// Use InMemoryTransport to test the server without stdio
	clientTransport, serverTransport := gomcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	go srv.server.Run(ctx, serverTransport)

	// Connect a client
	client := gomcp.NewClient(&gomcp.Implementation{
		Name:    "test-client",
		Version: "v1.0.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	// List tools
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("no tools registered")
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

func TestPingToolReturns(t *testing.T) {
	srv := New("test-version", testLogger(), nil)

	clientTransport, serverTransport := gomcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.server.Run(ctx, serverTransport)

	client := gomcp.NewClient(&gomcp.Implementation{
		Name:    "test-client",
		Version: "v1.0.0",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	// Call ping tool
	result, err := session.CallTool(ctx, &gomcp.CallToolParams{
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

func TestServerRunCancellation(t *testing.T) {
	srv := New("test-version", testLogger(), nil)

	_, serverTransport := gomcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- srv.server.Run(ctx, serverTransport)
	}()

	cancel()

	err := <-done
	// Server should return on context cancellation (nil or context error)
	if err != nil && err != context.Canceled {
		t.Errorf("Run() after cancel = %v, want nil or context.Canceled", err)
	}
}
