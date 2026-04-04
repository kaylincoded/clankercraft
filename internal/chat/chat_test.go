package chat

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

func TestParseWhisperVanilla(t *testing.T) {
	w := parseWhisper("Steve whispers to you: build me a house")
	if w == nil {
		t.Fatal("expected whisper, got nil")
	}
	if w.Sender != "Steve" {
		t.Errorf("sender = %q, want %q", w.Sender, "Steve")
	}
	if w.Message != "build me a house" {
		t.Errorf("message = %q, want %q", w.Message, "build me a house")
	}
}

func TestParseWhisperPlugin(t *testing.T) {
	w := parseWhisper("[Steve -> LLMBot] build me a house")
	if w == nil {
		t.Fatal("expected whisper, got nil")
	}
	if w.Sender != "Steve" {
		t.Errorf("sender = %q, want %q", w.Sender, "Steve")
	}
	if w.Message != "build me a house" {
		t.Errorf("message = %q, want %q", w.Message, "build me a house")
	}
}

func TestParseWhisperPublicChat(t *testing.T) {
	if w := parseWhisper("<Steve> hello everyone"); w != nil {
		t.Errorf("public chat should not parse as whisper, got %+v", w)
	}
}

func TestParseWhisperSystemMessage(t *testing.T) {
	cases := []string{
		"42 block(s) have been changed.",
		"First position set to (0, 64, 0).",
		"Steve joined the game",
		"",
	}
	for _, msg := range cases {
		if w := parseWhisper(msg); w != nil {
			t.Errorf("system message %q should not parse as whisper, got %+v", msg, w)
		}
	}
}

func TestListenerDispatchesWhisper(t *testing.T) {
	l := NewListener()

	var got Whisper
	l.OnWhisper(func(w Whisper) {
		got = w
	})

	l.HandleSystemChat("Steve whispers to you: hello")

	if got.Sender != "Steve" {
		t.Errorf("sender = %q, want %q", got.Sender, "Steve")
	}
	if got.Message != "hello" {
		t.Errorf("message = %q, want %q", got.Message, "hello")
	}
}

func TestListenerIgnoresNonWhisper(t *testing.T) {
	l := NewListener()

	called := false
	l.OnWhisper(func(w Whisper) {
		called = true
	})

	l.HandleSystemChat("42 block(s) have been changed.")

	if called {
		t.Error("handler should not be called for non-whisper message")
	}
}

func TestListenerMultipleHandlers(t *testing.T) {
	l := NewListener()

	var mu sync.Mutex
	count := 0
	for i := 0; i < 3; i++ {
		l.OnWhisper(func(w Whisper) {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	l.HandleSystemChat("Alex whispers to you: hi")

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("handler call count = %d, want 3", count)
	}
}

func TestListenerNoHandlersNoPanic(t *testing.T) {
	l := NewListener()
	// Should not panic with no handlers registered.
	l.HandleSystemChat("Steve whispers to you: test")
}

func TestParseWhisperVanillaWithSpacesInMessage(t *testing.T) {
	w := parseWhisper("Player123 whispers to you: make it bigger and add windows")
	if w == nil {
		t.Fatal("expected whisper")
	}
	if w.Sender != "Player123" {
		t.Errorf("sender = %q, want %q", w.Sender, "Player123")
	}
	if w.Message != "make it bigger and add windows" {
		t.Errorf("message = %q, want %q", w.Message, "make it bigger and add windows")
	}
}

func TestParseWhisperPluginWithUnderscore(t *testing.T) {
	w := parseWhisper("[Cool_Player -> Builder_Bot] hello there")
	if w == nil {
		t.Fatal("expected whisper")
	}
	if w.Sender != "Cool_Player" {
		t.Errorf("sender = %q, want %q", w.Sender, "Cool_Player")
	}
	if w.Message != "hello there" {
		t.Errorf("message = %q, want %q", w.Message, "hello there")
	}
}

// --- splitMessage tests ---

func TestSplitMessageUnderLimit(t *testing.T) {
	chunks := splitMessage("short message", 100)
	if len(chunks) != 1 || chunks[0] != "short message" {
		t.Errorf("got %v, want [short message]", chunks)
	}
}

func TestSplitMessageAtWordBoundary(t *testing.T) {
	// maxLen=13: "hello world" (11) fits in first chunk via word boundary,
	// "goodbye now" (11) fits in second
	chunks := splitMessage("hello world goodbye now", 13)
	want := []string{"hello world", "goodbye now"}
	if len(chunks) != len(want) {
		t.Fatalf("got %d chunks %v, want %v", len(chunks), chunks, want)
	}
	for i := range want {
		if chunks[i] != want[i] {
			t.Errorf("chunk[%d] = %q, want %q", i, chunks[i], want[i])
		}
	}
}

func TestSplitMessageHardSplit(t *testing.T) {
	long := strings.Repeat("x", 30)
	chunks := splitMessage(long, 10)
	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3", len(chunks))
	}
	for i, c := range chunks {
		if len(c) > 10 {
			t.Errorf("chunk[%d] len=%d exceeds maxLen=10", i, len(c))
		}
	}
	if strings.Join(chunks, "") != long {
		t.Errorf("reassembled = %q, want %q", strings.Join(chunks, ""), long)
	}
}

func TestSplitMessageAccountsForPlayerNameLength(t *testing.T) {
	// Simulate: prefix = "msg LongPlayerName " = 19 chars, so maxContent = 256-19 = 237
	player := "LongPlayerName"
	prefix := "msg " + player + " "
	maxContent := 256 - len(prefix)

	msg := strings.Repeat("a ", maxContent) // longer than maxContent
	chunks := splitMessage(msg, maxContent)
	for i, c := range chunks {
		if len(c) > maxContent {
			t.Errorf("chunk[%d] len=%d exceeds maxContent=%d", i, len(c), maxContent)
		}
	}
}

func TestSplitMessageMultiByteRune(t *testing.T) {
	// Each '世' is 3 bytes. With maxLen=10, we can fit 3 runes (9 bytes) but not 4 (12).
	text := "世世世世世世" // 6 runes, 18 bytes
	chunks := splitMessage(text, 10)
	for i, c := range chunks {
		if len(c) > 10 {
			t.Errorf("chunk[%d] byte len=%d exceeds 10", i, len(c))
		}
		// Verify each chunk is valid UTF-8
		if !utf8.ValidString(c) {
			t.Errorf("chunk[%d] = %q is not valid UTF-8", i, c)
		}
	}
	if strings.Join(chunks, "") != text {
		t.Errorf("reassembled = %q, want %q", strings.Join(chunks, ""), text)
	}
}

// --- SendWhisper tests ---

func TestSendWhisperBasic(t *testing.T) {
	var sent []string
	s := NewSender(func(cmd string) error {
		sent = append(sent, cmd)
		return nil
	})
	s.MessageDelay = 0

	err := s.SendWhisper("Steve", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if len(sent) != 1 {
		t.Fatalf("sent %d commands, want 1", len(sent))
	}
	if sent[0] != "msg Steve hello" {
		t.Errorf("command = %q, want %q", sent[0], "msg Steve hello")
	}
}

func TestSendWhisperSplitsLongMessage(t *testing.T) {
	var sent []string
	s := NewSender(func(cmd string) error {
		sent = append(sent, cmd)
		return nil
	})
	s.MessageDelay = 0

	// "msg Steve " = 10 chars, so maxContent = 246
	longMsg := strings.Repeat("word ", 60) // 300 chars
	longMsg = strings.TrimSpace(longMsg)

	err := s.SendWhisper("Steve", longMsg)
	if err != nil {
		t.Fatal(err)
	}
	if len(sent) < 2 {
		t.Fatalf("expected multiple sends for long message, got %d", len(sent))
	}
	for i, cmd := range sent {
		if len(cmd) > 256 {
			t.Errorf("command[%d] len=%d exceeds 256", i, len(cmd))
		}
		if !strings.HasPrefix(cmd, "msg Steve ") {
			t.Errorf("command[%d] = %q, missing prefix", i, cmd)
		}
	}
}

func TestSendWhisperEmptyPlayer(t *testing.T) {
	s := NewSender(func(cmd string) error { return nil })
	if err := s.SendWhisper("", "hello"); err == nil {
		t.Error("expected error for empty player")
	}
}

func TestSendWhisperEmptyMessage(t *testing.T) {
	s := NewSender(func(cmd string) error { return nil })
	if err := s.SendWhisper("Steve", ""); err == nil {
		t.Error("expected error for empty message")
	}
}

func TestSendWhisperRateLimiting(t *testing.T) {
	var sent []string
	s := NewSender(func(cmd string) error {
		sent = append(sent, cmd)
		return nil
	})
	s.MessageDelay = 50 * time.Millisecond

	// Build a message that will split into 2+ chunks
	longMsg := strings.Repeat("word ", 60)
	longMsg = strings.TrimSpace(longMsg)

	start := time.Now()
	err := s.SendWhisper("Steve", longMsg)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(sent) < 2 {
		t.Fatalf("expected multiple sends, got %d", len(sent))
	}
	// Expect at least (len(sent)-1) * 50ms delay
	minDelay := time.Duration(len(sent)-1) * 50 * time.Millisecond
	if elapsed < minDelay/2 { // generous tolerance
		t.Errorf("elapsed %v too short for %d sends with 50ms delay (expected >= %v)", elapsed, len(sent), minDelay/2)
	}
}

func TestSendWhisperCommandError(t *testing.T) {
	s := NewSender(func(cmd string) error {
		return fmt.Errorf("connection lost")
	})
	s.MessageDelay = 0

	err := s.SendWhisper("Steve", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "connection lost") {
		t.Errorf("error = %q, want to contain 'connection lost'", err.Error())
	}
}
