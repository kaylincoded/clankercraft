package chat

import (
	"sync"
	"testing"
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
