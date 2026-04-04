package connection

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/offline"
	"github.com/kaylincoded/clankercraft/internal/config"
	"github.com/kaylincoded/clankercraft/internal/engine"
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
	if conn.State() != StateDisconnected {
		t.Errorf("new connection state = %v, want StateDisconnected", conn.State())
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
	err := conn.setupAuth(client)
	if err != nil {
		t.Fatalf("setupAuth() returned error: %v", err)
	}

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

	// Inject mock auth function
	conn.authFn = func(cfg *config.Config, logger *slog.Logger) (*bot.Auth, error) {
		return &bot.Auth{
			Name: "MSAPlayer",
			UUID: "fake-uuid-1234",
			AsTk: "fake-access-token",
		}, nil
	}

	client := bot.NewClient()
	err := conn.setupAuth(client)
	if err != nil {
		t.Fatalf("setupAuth() returned error: %v", err)
	}

	if client.Auth.Name != "MSAPlayer" {
		t.Errorf("Auth.Name = %q, want %q", client.Auth.Name, "MSAPlayer")
	}
	if client.Auth.UUID != "fake-uuid-1234" {
		t.Errorf("Auth.UUID = %q, want %q", client.Auth.UUID, "fake-uuid-1234")
	}
	if client.Auth.AsTk != "fake-access-token" {
		t.Errorf("Auth.AsTk = %q, want %q", client.Auth.AsTk, "fake-access-token")
	}
}

func TestOnlineModeAuthFailure(t *testing.T) {
	cfg := &config.Config{
		Host:     "localhost",
		Port:     25565,
		Username: "OnlineBot",
		Offline:  false,
	}
	conn := New(cfg, testLogger())

	// Inject failing auth function
	conn.authFn = func(cfg *config.Config, logger *slog.Logger) (*bot.Auth, error) {
		return nil, fmt.Errorf("no MC ownership")
	}

	client := bot.NewClient()
	err := conn.setupAuth(client)
	if err == nil {
		t.Fatal("setupAuth() should return error when auth fails")
	}
	if client.Auth.AsTk != "" {
		t.Error("Auth.AsTk should be empty after failed auth")
	}
}

func TestCloseBeforeConnect(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// Close should be safe to call when not connected (no doneCh)
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

func TestCloseDrainsGameLoopGoroutine(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// Simulate a doneCh that closes quickly (goroutine finishes)
	doneCh := make(chan struct{})
	conn.mu.Lock()
	conn.doneCh = doneCh
	conn.mu.Unlock()

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(doneCh)
	}()

	start := time.Now()
	err := conn.Close()
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
	// Should have waited for doneCh, not timed out
	if elapsed >= ShutdownTimeout {
		t.Errorf("Close() took %v, should have drained quickly", elapsed)
	}
}

func TestCloseTimesOutOnStuckGoroutine(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())
	conn.shutdownTimeout = 50 * time.Millisecond

	// doneCh that never closes (simulates stuck goroutine)
	doneCh := make(chan struct{})
	conn.mu.Lock()
	conn.doneCh = doneCh
	conn.mu.Unlock()

	start := time.Now()
	conn.Close()
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Errorf("Close() returned in %v, should have waited at least 50ms", elapsed)
	}
	if elapsed > 1*time.Second {
		t.Errorf("Close() took %v, should have timed out near 50ms", elapsed)
	}
}

func TestShutdownTimeoutConstant(t *testing.T) {
	if ShutdownTimeout != 5*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 5s", ShutdownTimeout)
	}
}

func TestConnStateString(t *testing.T) {
	tests := []struct {
		state ConnState
		want  string
	}{
		{StateDisconnected, "disconnected"},
		{StateConnecting, "connecting"},
		{StateConnected, "connected"},
		{ConnState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestStateTransitions(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// Initial state
	if conn.State() != StateDisconnected {
		t.Errorf("initial state = %v, want StateDisconnected", conn.State())
	}

	// Transition to connecting
	conn.setState(StateConnecting)
	if conn.State() != StateConnecting {
		t.Errorf("state = %v, want StateConnecting", conn.State())
	}
	if conn.IsConnected() {
		t.Error("IsConnected() should be false when connecting")
	}

	// Transition to connected
	conn.setState(StateConnected)
	if conn.State() != StateConnected {
		t.Errorf("state = %v, want StateConnected", conn.State())
	}
	if !conn.IsConnected() {
		t.Error("IsConnected() should be true when connected")
	}

	// Transition back to disconnected
	conn.setState(StateDisconnected)
	if conn.State() != StateDisconnected {
		t.Errorf("state = %v, want StateDisconnected", conn.State())
	}
	if conn.IsConnected() {
		t.Error("IsConnected() should be false when disconnected")
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

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // cap
		{6, 30 * time.Second}, // stays at cap
		{10, 30 * time.Second},
	}
	for _, tt := range tests {
		got := backoffDuration(tt.attempt)
		if got != tt.want {
			t.Errorf("backoffDuration(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestReconnectConstants(t *testing.T) {
	if MaxReconnectAttempts != 5 {
		t.Errorf("MaxReconnectAttempts = %d, want 5", MaxReconnectAttempts)
	}
	if MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", MaxBackoff)
	}
}

func TestRunWithReconnectContextCancellation(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565, Offline: true, Username: "Bot"}
	conn := New(cfg, testLogger())

	// Cancel context immediately — RunWithReconnect should return ctx.Err()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn.connectAndRun = func(ctx context.Context) error {
		return fmt.Errorf("connection refused")
	}

	err := conn.RunWithReconnect(ctx)
	if err != context.Canceled {
		t.Errorf("RunWithReconnect() with cancelled context = %v, want context.Canceled", err)
	}
}

func zeroBackoff(attempt int) time.Duration { return 0 }

func TestRunWithReconnectRetryExhaustion(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565, Offline: true, Username: "Bot"}
	conn := New(cfg, testLogger())
	conn.backoffFn = zeroBackoff

	var attempts int
	conn.connectAndRun = func(ctx context.Context) error {
		attempts++
		return fmt.Errorf("connection refused")
	}

	err := conn.RunWithReconnect(context.Background())
	if err == nil {
		t.Fatal("RunWithReconnect() should return error after max retries")
	}
	// 1 initial + 4 retries = 5 total attempts (MaxReconnectAttempts=5)
	if attempts != MaxReconnectAttempts {
		t.Errorf("attempts = %d, want %d", attempts, MaxReconnectAttempts)
	}
}

func TestRunWithReconnectResetsOnSuccess(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565, Offline: true, Username: "Bot"}
	conn := New(cfg, testLogger())
	conn.backoffFn = zeroBackoff

	var calls int
	conn.connectAndRun = func(ctx context.Context) error {
		calls++
		switch {
		case calls <= 3:
			// First 3 calls fail (building up retry count)
			return fmt.Errorf("connection refused")
		case calls == 4:
			// 4th call succeeds (returns nil = clean disconnect, triggers reconnect)
			return nil
		case calls <= 7:
			// Next 3 calls fail again
			return fmt.Errorf("connection refused")
		case calls == 8:
			// Success again, then we cancel
			return nil
		default:
			// Final failure to break the loop
			return context.Canceled
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Give it time to run through the attempts
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	conn.RunWithReconnect(ctx)
	// If retry counter wasn't reset after success on call 4,
	// calls 5-7 would exhaust retries (3 failures + prior 3 = 6 > 5).
	// The fact that we got past call 7 proves the counter reset.
	if calls < 7 {
		t.Errorf("calls = %d, want at least 7 (proving retry counter reset after success)", calls)
	}
}

// --- Position tracking tests ---

func TestGetPositionReturnsFalseBeforeUpdate(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	_, ok := conn.GetPosition()
	if ok {
		t.Error("GetPosition() should return false before any position update")
	}
}

func TestUpdatePositionAbsolute(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// flags=0 means all values are absolute
	conn.updatePosition(100.5, 64.0, -200.3, 45.0, -10.0, 0)

	pos, ok := conn.GetPosition()
	if !ok {
		t.Fatal("GetPosition() should return true after update")
	}
	if pos.X != 100.5 || pos.Y != 64.0 || pos.Z != -200.3 {
		t.Errorf("position = (%v, %v, %v), want (100.5, 64, -200.3)", pos.X, pos.Y, pos.Z)
	}
	if pos.Yaw != 45.0 || pos.Pitch != -10.0 {
		t.Errorf("rotation = (%v, %v), want (45, -10)", pos.Yaw, pos.Pitch)
	}
}

func TestUpdatePositionRelative(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// Set initial position
	conn.updatePosition(100.0, 64.0, 200.0, 0.0, 0.0, 0)

	// flags=0x1F means all values are relative
	conn.updatePosition(10.0, 5.0, -10.0, 45.0, -5.0, 0x1F)

	pos, _ := conn.GetPosition()
	if pos.X != 110.0 || pos.Y != 69.0 || pos.Z != 190.0 {
		t.Errorf("position = (%v, %v, %v), want (110, 69, 190)", pos.X, pos.Y, pos.Z)
	}
	if pos.Yaw != 45.0 || pos.Pitch != -5.0 {
		t.Errorf("rotation = (%v, %v), want (45, -5)", pos.Yaw, pos.Pitch)
	}
}

func TestUpdatePositionMixedFlags(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	// Set initial position
	conn.updatePosition(100.0, 64.0, 200.0, 90.0, 0.0, 0)

	// flags=0x01 means only X is relative, rest absolute
	conn.updatePosition(10.0, 70.0, 300.0, 180.0, -30.0, 0x01)

	pos, _ := conn.GetPosition()
	if pos.X != 110.0 {
		t.Errorf("X = %v, want 110 (relative)", pos.X)
	}
	if pos.Y != 70.0 {
		t.Errorf("Y = %v, want 70 (absolute)", pos.Y)
	}
	if pos.Z != 300.0 {
		t.Errorf("Z = %v, want 300 (absolute)", pos.Z)
	}
}

// --- parseSignMessages tests ---

func TestParseSignMessagesJSONTextComponent(t *testing.T) {
	messages := []string{
		`{"text":"Hello"}`,
		`{"text":"World"}`,
		`{"text":""}`,
		`{"text":""}`,
	}
	lines := parseSignMessages(messages)
	if lines[0] != "Hello" {
		t.Errorf("line 0 = %q, want %q", lines[0], "Hello")
	}
	if lines[1] != "World" {
		t.Errorf("line 1 = %q, want %q", lines[1], "World")
	}
	if lines[2] != "" {
		t.Errorf("line 2 = %q, want empty", lines[2])
	}
}

func TestParseSignMessagesPlainJSONString(t *testing.T) {
	// A plain JSON string literal (not a text component object)
	messages := []string{`"Hello plain"`, `""`, ``, ``}
	lines := parseSignMessages(messages)
	if lines[0] != "Hello plain" {
		t.Errorf("line 0 = %q, want %q", lines[0], "Hello plain")
	}
	if lines[1] != "" {
		t.Errorf("line 1 = %q, want empty", lines[1])
	}
}

func TestParseSignMessagesComplexTextComponent(t *testing.T) {
	messages := []string{
		`{"text":"","extra":[{"text":"Colored","color":"red"}]}`,
		`{"text":""}`,
		`{"text":""}`,
		`{"text":""}`,
	}
	lines := parseSignMessages(messages)
	if lines[0] != "Colored" {
		t.Errorf("line 0 = %q, want %q", lines[0], "Colored")
	}
}

func TestParseSignMessagesMalformedFallsToEmpty(t *testing.T) {
	messages := []string{`not valid json {`, `{"text":"OK"}`, ``, ``}
	lines := parseSignMessages(messages)
	if lines[0] != "" {
		t.Errorf("line 0 = %q, want empty for malformed JSON", lines[0])
	}
	if lines[1] != "OK" {
		t.Errorf("line 1 = %q, want %q", lines[1], "OK")
	}
}

func TestParseSignMessagesFewerThanFour(t *testing.T) {
	messages := []string{`{"text":"Only one"}`}
	lines := parseSignMessages(messages)
	if lines[0] != "Only one" {
		t.Errorf("line 0 = %q, want %q", lines[0], "Only one")
	}
	for i := 1; i < 4; i++ {
		if lines[i] != "" {
			t.Errorf("line %d = %q, want empty", i, lines[i])
		}
	}
}

// --- Chat listener and tier detection tests ---

func TestDispatchChatToListeners(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	ch := conn.listenChat()
	conn.dispatchChat("hello world")

	select {
	case msg := <-ch:
		if msg != "hello world" {
			t.Errorf("got %q, want %q", msg, "hello world")
		}
	default:
		t.Error("listener did not receive message")
	}

	conn.unlistenChat(ch)
}

func TestUnlistenRemovesListener(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	ch := conn.listenChat()
	conn.unlistenChat(ch)
	conn.dispatchChat("should not arrive")

	select {
	case msg := <-ch:
		t.Errorf("listener received %q after unlisten", msg)
	default:
		// expected
	}
}

func TestParseTierFromChat(t *testing.T) {
	tests := []struct {
		msg      string
		wantTier engine.Tier
		wantOK   bool
	}{
		{"WorldEdit version 7.4.0", engine.TierWorldEdit, true},
		{"WorldEdit version 7.4.0;4395bc1", engine.TierWorldEdit, true},
		{"FastAsyncWorldEdit version 2.12.3", engine.TierFAWE, true},
		{"FAWE version 2.12.3", engine.TierFAWE, true},
		{"Server running Paper 1.21.11", engine.TierUnknown, false},
		{"Unknown command", engine.TierUnknown, false},
		{"", engine.TierUnknown, false},
		{"worldedit version 7.4.0", engine.TierWorldEdit, true}, // case insensitive
		{"This server uses FastAsyncWorldEdit (FAWE)", engine.TierFAWE, true},
	}
	for _, tt := range tests {
		tier, ok := parseTierFromChat(tt.msg)
		if tier != tt.wantTier || ok != tt.wantOK {
			t.Errorf("parseTierFromChat(%q) = (%v, %v), want (%v, %v)", tt.msg, tier, ok, tt.wantTier, tt.wantOK)
		}
	}
}

func TestResetTierOnDisconnect(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	conn.mu.Lock()
	conn.tier = engine.TierWorldEdit
	conn.mu.Unlock()

	conn.resetTier()

	if tier := conn.GetTier(); tier != engine.TierUnknown {
		t.Errorf("tier after reset = %v, want TierUnknown", tier)
	}
}

func TestSetSelectionStoresCoordinates(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	// No selection initially
	_, ok := conn.GetSelection()
	if ok {
		t.Error("GetSelection() should return false before SetSelection")
	}

	conn.mu.Lock()
	conn.tier = engine.TierVanilla
	conn.mu.Unlock()

	err := conn.SetSelection(0, 64, 0, 10, 70, 10)
	if err != nil {
		t.Fatalf("SetSelection() returned error: %v", err)
	}

	sel, ok := conn.GetSelection()
	if !ok {
		t.Fatal("GetSelection() should return true after SetSelection")
	}
	if sel.X1 != 0 || sel.Y1 != 64 || sel.Z1 != 0 {
		t.Errorf("pos1 = (%d, %d, %d), want (0, 64, 0)", sel.X1, sel.Y1, sel.Z1)
	}
	if sel.X2 != 10 || sel.Y2 != 70 || sel.Z2 != 10 {
		t.Errorf("pos2 = (%d, %d, %d), want (10, 70, 10)", sel.X2, sel.Y2, sel.Z2)
	}
}

func TestSetSelectionSendsWECommandsWhenWorldEdit(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	var commands []string
	conn.sendCommandFn = func(cmd string) error {
		commands = append(commands, cmd)
		return nil
	}

	conn.mu.Lock()
	conn.tier = engine.TierWorldEdit
	conn.mu.Unlock()

	err := conn.SetSelection(0, 64, 0, 10, 70, 10)
	if err != nil {
		t.Fatalf("SetSelection() returned error: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d: %v", len(commands), commands)
	}
	if commands[0] != "/pos1 0,64,0" {
		t.Errorf("command[0] = %q, want %q", commands[0], "/pos1 0,64,0")
	}
	if commands[1] != "/pos2 10,70,10" {
		t.Errorf("command[1] = %q, want %q", commands[1], "/pos2 10,70,10")
	}
}

func TestSetSelectionSendsWECommandsWhenFAWE(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	var commands []string
	conn.sendCommandFn = func(cmd string) error {
		commands = append(commands, cmd)
		return nil
	}

	conn.mu.Lock()
	conn.tier = engine.TierFAWE
	conn.mu.Unlock()

	err := conn.SetSelection(-10, 50, -20, 5, 80, 15)
	if err != nil {
		t.Fatalf("SetSelection() returned error: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	if commands[0] != "/pos1 -10,50,-20" {
		t.Errorf("command[0] = %q, want %q", commands[0], "/pos1 -10,50,-20")
	}
	if commands[1] != "/pos2 5,80,15" {
		t.Errorf("command[1] = %q, want %q", commands[1], "/pos2 5,80,15")
	}
}

func TestSetSelectionSkipsCommandsWhenVanilla(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	var commands []string
	conn.sendCommandFn = func(cmd string) error {
		commands = append(commands, cmd)
		return nil
	}

	conn.mu.Lock()
	conn.tier = engine.TierVanilla
	conn.mu.Unlock()

	err := conn.SetSelection(0, 64, 0, 10, 70, 10)
	if err != nil {
		t.Fatalf("SetSelection() returned error: %v", err)
	}

	if len(commands) != 0 {
		t.Errorf("expected no commands for vanilla, got %d: %v", len(commands), commands)
	}

	// Selection should still be stored
	_, ok := conn.GetSelection()
	if !ok {
		t.Error("selection should be stored even in vanilla mode")
	}
}

func TestSetSelectionReturnsErrorOnCommandFailure(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	conn.sendCommandFn = func(cmd string) error {
		return fmt.Errorf("not connected")
	}

	conn.mu.Lock()
	conn.tier = engine.TierWorldEdit
	conn.mu.Unlock()

	err := conn.SetSelection(0, 64, 0, 10, 70, 10)
	if err == nil {
		t.Fatal("SetSelection() should return error when SendCommand fails")
	}

	// Selection should still be in memory despite command failure
	_, ok := conn.GetSelection()
	if !ok {
		t.Error("selection should be stored even when commands fail")
	}
}

func TestResetSelectionOnDisconnect(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())

	conn.mu.Lock()
	conn.selection = engine.Selection{X1: 1, Y1: 2, Z1: 3, X2: 4, Y2: 5, Z2: 6}
	conn.hasPos1 = true
	conn.hasPos2 = true
	conn.mu.Unlock()

	conn.resetSelection()

	_, ok := conn.GetSelection()
	if ok {
		t.Error("GetSelection() should return false after resetSelection")
	}
}

// --- Wand selection parsing tests ---

func TestParseWandPos1Valid(t *testing.T) {
	tests := []struct {
		msg          string
		wantX, wantY, wantZ int
	}{
		{"First position set to (100, 64, -200).", 100, 64, -200},
		{"First position set to (100, 64, -200) (7260).", 100, 64, -200},
		{"First position set to (0, 0, 0).", 0, 0, 0},
		{"First position set to (-50, -10, 300).", -50, -10, 300},
	}
	for _, tt := range tests {
		x, y, z, ok := parseWandPos1(tt.msg)
		if !ok {
			t.Errorf("parseWandPos1(%q) returned ok=false, want true", tt.msg)
			continue
		}
		if x != tt.wantX || y != tt.wantY || z != tt.wantZ {
			t.Errorf("parseWandPos1(%q) = (%d, %d, %d), want (%d, %d, %d)", tt.msg, x, y, z, tt.wantX, tt.wantY, tt.wantZ)
		}
	}
}

func TestParseWandPos1Invalid(t *testing.T) {
	invalids := []string{
		"Second position set to (100, 64, -200).",
		"WorldEdit version 7.4.0",
		"Unknown command",
		"",
	}
	for _, msg := range invalids {
		if _, _, _, ok := parseWandPos1(msg); ok {
			t.Errorf("parseWandPos1(%q) returned ok=true, want false", msg)
		}
	}
}

func TestParseWandPos2Valid(t *testing.T) {
	tests := []struct {
		msg          string
		wantX, wantY, wantZ int
	}{
		{"Second position set to (110, 70, -190).", 110, 70, -190},
		{"Second position set to (110, 70, -190) (7260).", 110, 70, -190},
		{"Second position set to (-1, -64, -1).", -1, -64, -1},
	}
	for _, tt := range tests {
		x, y, z, ok := parseWandPos2(tt.msg)
		if !ok {
			t.Errorf("parseWandPos2(%q) returned ok=false, want true", tt.msg)
			continue
		}
		if x != tt.wantX || y != tt.wantY || z != tt.wantZ {
			t.Errorf("parseWandPos2(%q) = (%d, %d, %d), want (%d, %d, %d)", tt.msg, x, y, z, tt.wantX, tt.wantY, tt.wantZ)
		}
	}
}

func TestParseWandPos2Invalid(t *testing.T) {
	invalids := []string{
		"First position set to (100, 64, -200).",
		"some random chat",
		"",
	}
	for _, msg := range invalids {
		if _, _, _, ok := parseWandPos2(msg); ok {
			t.Errorf("parseWandPos2(%q) returned ok=true, want false", msg)
		}
	}
}

func waitForCondition(t *testing.T, check func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error(msg)
}

func TestWandListenerUpdatesSelection(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())
	conn.wandDone = make(chan struct{})

	go conn.startWandListener()

	// Wait for listener to register
	waitForCondition(t, func() bool {
		conn.mu.Lock()
		defer conn.mu.Unlock()
		return len(conn.chatListeners) > 0
	}, 500*time.Millisecond, "wand listener should register a chat listener")

	// Send pos1
	conn.dispatchChat("First position set to (100, 64, -200).")
	waitForCondition(t, conn.HasPos1, 500*time.Millisecond, "HasPos1() should be true after wand pos1")

	if conn.HasPos2() {
		t.Error("HasPos2() should be false before wand pos2")
	}
	_, ok := conn.GetSelection()
	if ok {
		t.Error("GetSelection() should return false with only pos1")
	}

	// Send pos2
	conn.dispatchChat("Second position set to (110, 70, -190) (7260).")
	waitForCondition(t, conn.HasPos2, 500*time.Millisecond, "HasPos2() should be true after wand pos2")

	sel, ok := conn.GetSelection()
	if !ok {
		t.Fatal("GetSelection() should return true with both positions")
	}
	if sel.X1 != 100 || sel.Y1 != 64 || sel.Z1 != -200 {
		t.Errorf("pos1 = (%d, %d, %d), want (100, 64, -200)", sel.X1, sel.Y1, sel.Z1)
	}
	if sel.X2 != 110 || sel.Y2 != 70 || sel.Z2 != -190 {
		t.Errorf("pos2 = (%d, %d, %d), want (110, 70, -190)", sel.X2, sel.Y2, sel.Z2)
	}

	close(conn.wandDone)
}

func TestWandListenerStopsOnDone(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())
	conn.wandDone = make(chan struct{})

	done := make(chan struct{})
	go func() {
		conn.startWandListener()
		close(done)
	}()

	close(conn.wandDone)

	select {
	case <-done:
		// expected
	case <-time.After(1 * time.Second):
		t.Error("wand listener did not stop after wandDone closed")
	}
}

func TestDisconnectClearsWandStateAndStopsListener(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())
	conn.wandDone = make(chan struct{})

	listenerDone := make(chan struct{})
	go func() {
		conn.startWandListener()
		close(listenerDone)
	}()

	// Wait for listener to register
	waitForCondition(t, func() bool {
		conn.mu.Lock()
		defer conn.mu.Unlock()
		return len(conn.chatListeners) > 0
	}, 500*time.Millisecond, "wand listener should register")

	// Set wand positions
	conn.dispatchChat("First position set to (10, 20, 30).")
	conn.dispatchChat("Second position set to (40, 50, 60).")
	waitForCondition(t, func() bool {
		_, ok := conn.GetSelection()
		return ok
	}, 500*time.Millisecond, "both positions should be set")

	// Simulate disconnect: close wandDone, then resetSelection
	conn.mu.Lock()
	close(conn.wandDone)
	conn.wandDone = nil
	conn.mu.Unlock()
	conn.resetSelection()

	// Verify listener stopped
	select {
	case <-listenerDone:
	case <-time.After(1 * time.Second):
		t.Error("wand listener did not stop after disconnect")
	}

	// Verify state cleared
	if conn.HasPos1() {
		t.Error("HasPos1() should be false after disconnect")
	}
	if conn.HasPos2() {
		t.Error("HasPos2() should be false after disconnect")
	}
	_, ok := conn.GetSelection()
	if ok {
		t.Error("GetSelection() should return false after disconnect")
	}
}

func TestSetSelectionSetsBothPosFlags(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())
	conn.mu.Lock()
	conn.tier = engine.TierVanilla
	conn.mu.Unlock()

	conn.SetSelection(0, 64, 0, 10, 70, 10)

	if !conn.HasPos1() || !conn.HasPos2() {
		t.Error("SetSelection should set both hasPos1 and hasPos2")
	}
}

func TestGetGamemodeValues(t *testing.T) {
	conn := New(&config.Config{Host: "localhost", Port: 25565}, testLogger())
	// No player set — should return "unknown"
	if gm := conn.GetGamemode(); gm != "unknown" {
		t.Errorf("GetGamemode() without player = %q, want %q", gm, "unknown")
	}
}

func TestResetPositionOnDisconnect(t *testing.T) {
	cfg := &config.Config{Host: "localhost", Port: 25565}
	conn := New(cfg, testLogger())

	conn.updatePosition(100.0, 64.0, 200.0, 0.0, 0.0, 0)
	_, ok := conn.GetPosition()
	if !ok {
		t.Fatal("position should be set")
	}

	conn.resetPosition()

	_, ok = conn.GetPosition()
	if ok {
		t.Error("GetPosition() should return false after resetPosition")
	}
}
