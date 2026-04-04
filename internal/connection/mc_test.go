package connection

import (
	"context"
	"encoding/hex"
	"log/slog"
	"os"
	"testing"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/offline"
	"github.com/kaylincoded/clankercraft/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewReturnsConfiguredConnection(t *testing.T) {
	cfg := &config.Config{
		Host:     "mc.example.com",
		Port:     25565,
		Username: "TestBot",
		Offline:  true,
	}
	conn := New(cfg, testLogger())

	if conn.cfg != cfg {
		t.Error("Connection.cfg not set correctly")
	}
	if conn.connected {
		t.Error("new connection should not be connected")
	}
}

func TestAddressFormatting(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"localhost", 25565, "localhost:25565"},
		{"mc.example.com", 25566, "mc.example.com:25566"},
		{"192.168.1.1", 12345, "192.168.1.1:12345"},
	}

	for _, tt := range tests {
		cfg := &config.Config{Host: tt.host, Port: tt.port}
		conn := New(cfg, testLogger())
		got := conn.Address()
		if got != tt.want {
			t.Errorf("Address() = %q, want %q", got, tt.want)
		}
	}
}

func TestOfflineModeAuthSetup(t *testing.T) {
	cfg := &config.Config{
		Host:     "localhost",
		Port:     25565,
		Username: "OfflineBot",
		Offline:  true,
	}
	conn := New(cfg, testLogger())

	client := bot.NewClient()
	conn.setupAuth(client)

	if client.Auth.Name != "OfflineBot" {
		t.Errorf("Auth.Name = %q, want %q", client.Auth.Name, "OfflineBot")
	}

	id := offline.NameToUUID("OfflineBot")
	expectedUUID := hex.EncodeToString(id[:])
	if client.Auth.UUID != expectedUUID {
		t.Errorf("Auth.UUID = %q, want %q", client.Auth.UUID, expectedUUID)
	}

	if client.Auth.AsTk != "" {
		t.Errorf("Auth.AsTk = %q, want empty (offline mode)", client.Auth.AsTk)
	}
}

func TestOnlineModeAuthSetup(t *testing.T) {
	cfg := &config.Config{
		Host:     "localhost",
		Port:     25565,
		Username: "OnlineBot",
		Offline:  false,
	}
	conn := New(cfg, testLogger())

	client := bot.NewClient()
	conn.setupAuth(client)

	if client.Auth.Name != "OnlineBot" {
		t.Errorf("Auth.Name = %q, want %q", client.Auth.Name, "OnlineBot")
	}

	// Online mode should not set offline UUID
	if client.Auth.UUID != "" {
		t.Errorf("Auth.UUID = %q, want empty (no offline UUID for online mode)", client.Auth.UUID)
	}
}

func TestCloseBeforeConnect(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// Close should be safe to call when not connected
	err := conn.Close()
	if err != nil {
		t.Errorf("Close() before Connect should not error, got: %v", err)
	}
}

func TestCloseMultipleTimes(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// Multiple Close calls should be safe
	for i := 0; i < 3; i++ {
		err := conn.Close()
		if err != nil {
			t.Errorf("Close() call %d should not error, got: %v", i+1, err)
		}
	}
}

func TestIsConnectedDefaultFalse(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	if conn.IsConnected() {
		t.Error("IsConnected() should be false before Connect")
	}
}

func TestHandleGameWithoutConnect(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	err := conn.HandleGame(context.Background())
	if err == nil {
		t.Error("HandleGame() without Connect should error")
	}
}
