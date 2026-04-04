package rcon

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"testing"

	mcnet "github.com/Tnze/go-mc/net"
	"github.com/kaylincoded/clankercraft/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewCreatesClient(t *testing.T) {
	cfg := &config.Config{RCONPort: 25575, RCONPassword: "test"}
	client := New(cfg, testLogger())
	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.IsAvailable() {
		t.Error("new client should not be available before Connect")
	}
}

func TestConnectNoPasswordSkips(t *testing.T) {
	cfg := &config.Config{RCONPort: 25575, RCONPassword: ""}
	client := New(cfg, testLogger())

	err := client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() with no password: %v", err)
	}
	if client.IsAvailable() {
		t.Error("client should not be available when no password configured")
	}
}

func TestConnectDialFailureGraceful(t *testing.T) {
	cfg := &config.Config{Host: "127.0.0.1", RCONPort: 1, RCONPassword: "secret"}
	client := New(cfg, testLogger())
	client.dialFn = func(addr, password string) (mcnet.RCONClientConn, error) {
		return nil, fmt.Errorf("connection refused")
	}

	err := client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() should not return error on dial failure: %v", err)
	}
	if client.IsAvailable() {
		t.Error("client should not be available after dial failure")
	}
}

func TestExecuteOnUnavailableClient(t *testing.T) {
	cfg := &config.Config{RCONPort: 25575, RCONPassword: ""}
	client := New(cfg, testLogger())

	_, err := client.Execute("say hello")
	if err == nil {
		t.Fatal("Execute() on unavailable client should return error")
	}
}

func TestConnectRespectsContextCancellation(t *testing.T) {
	cfg := &config.Config{Host: "127.0.0.1", RCONPort: 25575, RCONPassword: "secret"}
	client := New(cfg, testLogger())
	// Dial function that blocks until context is cancelled.
	client.dialFn = func(addr, password string) (mcnet.RCONClientConn, error) {
		select {} // block forever
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() should not return error on cancellation: %v", err)
	}
	if client.IsAvailable() {
		t.Error("client should not be available after cancelled connect")
	}
}

func TestCloseOnUnconnectedClient(t *testing.T) {
	cfg := &config.Config{RCONPort: 25575}
	client := New(cfg, testLogger())

	err := client.Close()
	if err != nil {
		t.Fatalf("Close() on unconnected client: %v", err)
	}
}

// mockRCONConn implements mcnet.RCONClientConn for unit tests.
type mockRCONConn struct {
	cmdFn   func(cmd string) error
	respFn  func() (string, error)
	closeFn func() error
	lastCmd string
}

func (m *mockRCONConn) Cmd(cmd string) error {
	m.lastCmd = cmd
	if m.cmdFn != nil {
		return m.cmdFn(cmd)
	}
	return nil
}

func (m *mockRCONConn) Resp() (string, error) {
	if m.respFn != nil {
		return m.respFn()
	}
	return "Done", nil
}

func (m *mockRCONConn) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

func TestConnectSuccess(t *testing.T) {
	mock := &mockRCONConn{}
	cfg := &config.Config{Host: "127.0.0.1", RCONPort: 25575, RCONPassword: "secret"}
	client := New(cfg, testLogger())
	client.dialFn = func(addr, password string) (mcnet.RCONClientConn, error) {
		if addr != "127.0.0.1:25575" {
			t.Errorf("dial addr = %q, want %q", addr, "127.0.0.1:25575")
		}
		if password != "secret" {
			t.Errorf("dial password = %q, want %q", password, "secret")
		}
		return mock, nil
	}

	err := client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect(): %v", err)
	}
	if !client.IsAvailable() {
		t.Error("client should be available after successful connect")
	}
}

func TestExecuteSendsCommand(t *testing.T) {
	mock := &mockRCONConn{
		respFn: func() (string, error) { return "42 blocks changed", nil },
	}
	cfg := &config.Config{Host: "127.0.0.1", RCONPort: 25575, RCONPassword: "secret"}
	client := New(cfg, testLogger())
	client.dialFn = func(addr, password string) (mcnet.RCONClientConn, error) {
		return mock, nil
	}
	client.Connect(context.Background())

	resp, err := client.Execute("fill 0 0 0 10 10 10 stone")
	if err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if mock.lastCmd != "fill 0 0 0 10 10 10 stone" {
		t.Errorf("command = %q, want %q", mock.lastCmd, "fill 0 0 0 10 10 10 stone")
	}
	if resp != "42 blocks changed" {
		t.Errorf("response = %q, want %q", resp, "42 blocks changed")
	}
}

func TestExecuteCmdError(t *testing.T) {
	mock := &mockRCONConn{
		cmdFn: func(cmd string) error { return fmt.Errorf("write error") },
	}
	cfg := &config.Config{Host: "127.0.0.1", RCONPort: 25575, RCONPassword: "secret"}
	client := New(cfg, testLogger())
	client.dialFn = func(addr, password string) (mcnet.RCONClientConn, error) {
		return mock, nil
	}
	client.Connect(context.Background())

	_, err := client.Execute("say hello")
	if err == nil {
		t.Fatal("Execute() should return error on Cmd failure")
	}
}

func TestExecuteRespError(t *testing.T) {
	mock := &mockRCONConn{
		respFn: func() (string, error) { return "", fmt.Errorf("read error") },
	}
	cfg := &config.Config{Host: "127.0.0.1", RCONPort: 25575, RCONPassword: "secret"}
	client := New(cfg, testLogger())
	client.dialFn = func(addr, password string) (mcnet.RCONClientConn, error) {
		return mock, nil
	}
	client.Connect(context.Background())

	_, err := client.Execute("say hello")
	if err == nil {
		t.Fatal("Execute() should return error on Resp failure")
	}
}

func TestCloseAfterConnect(t *testing.T) {
	closed := false
	mock := &mockRCONConn{
		closeFn: func() error { closed = true; return nil },
	}
	cfg := &config.Config{Host: "127.0.0.1", RCONPort: 25575, RCONPassword: "secret"}
	client := New(cfg, testLogger())
	client.dialFn = func(addr, password string) (mcnet.RCONClientConn, error) {
		return mock, nil
	}
	client.Connect(context.Background())

	err := client.Close()
	if err != nil {
		t.Fatalf("Close(): %v", err)
	}
	if !closed {
		t.Error("underlying connection was not closed")
	}
	if client.IsAvailable() {
		t.Error("client should not be available after Close")
	}
}

// Integration test using go-mc's built-in RCON server.
func TestIntegrationRCON(t *testing.T) {
	// Start a test RCON server on a random port.
	listener, err := mcnet.ListenRCON("127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenRCON: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	password := "testpass123"

	// Server goroutine: accept one client, handle login and one command.
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- fmt.Errorf("accept: %w", err)
			return
		}
		defer conn.Close()

		if err := conn.AcceptLogin(password); err != nil {
			serverDone <- fmt.Errorf("accept login: %w", err)
			return
		}

		cmd, err := conn.AcceptCmd()
		if err != nil {
			serverDone <- fmt.Errorf("accept cmd: %w", err)
			return
		}

		if cmd != "say hello world" {
			serverDone <- fmt.Errorf("unexpected command: %q", cmd)
			return
		}

		if err := conn.RespCmd("Message sent"); err != nil {
			serverDone <- fmt.Errorf("resp cmd: %w", err)
			return
		}

		serverDone <- nil
	}()

	// Client side: use real DialRCON against the test server.
	// Parse host and port from listener address.
	host, port := parseAddr(t, addr)
	cfg := &config.Config{Host: host, RCONPort: port, RCONPassword: password}
	client := New(cfg, testLogger())

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	if !client.IsAvailable() {
		t.Fatal("client should be available after connect")
	}

	resp, err := client.Execute("say hello world")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp != "Message sent" {
		t.Errorf("response = %q, want %q", resp, "Message sent")
	}

	// Wait for server goroutine to finish.
	if err := <-serverDone; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

func TestIntegrationRCONWrongPassword(t *testing.T) {
	listener, err := mcnet.ListenRCON("127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenRCON: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Server goroutine: reject wrong password.
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.AcceptLogin("correct-password")
	}()

	host, port := parseAddr(t, addr)
	cfg := &config.Config{Host: host, RCONPort: port, RCONPassword: "wrong-password"}
	client := New(cfg, testLogger())

	err = client.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect should not return error (graceful): %v", err)
	}
	if client.IsAvailable() {
		t.Error("client should not be available with wrong password")
	}
}

func parseAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("parse addr %q: %v", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port %q: %v", portStr, err)
	}
	return host, port
}
