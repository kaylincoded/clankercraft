package agent

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kaylincoded/clankercraft/internal/llm"
)

func TestConversationStoreGetNew(t *testing.T) {
	s := NewConversationStore()
	conv := s.Get("Steve")
	if conv == nil {
		t.Fatal("expected non-nil conversation")
	}
	if len(conv.Messages) != 0 {
		t.Errorf("new conversation should have 0 messages, got %d", len(conv.Messages))
	}
}

func TestConversationStoreGetExisting(t *testing.T) {
	s := NewConversationStore()
	conv1 := s.Get("Steve")
	conv2 := s.Get("Steve")
	if conv1 != conv2 {
		t.Error("expected same conversation object for same player")
	}
}

func TestConversationStoreAppend(t *testing.T) {
	s := NewConversationStore()
	s.Append("Steve", llm.Message{Role: llm.RoleUser, Content: "hello"})
	s.Append("Steve", llm.Message{Role: llm.RoleAssistant, Content: "hi"})
	s.Append("Alex", llm.Message{Role: llm.RoleUser, Content: "hey"})

	steveSnap := s.Snapshot("Steve")
	if len(steveSnap) != 2 {
		t.Fatalf("Steve should have 2 messages, got %d", len(steveSnap))
	}
	if steveSnap[0].Content != "hello" {
		t.Errorf("first msg = %q, want 'hello'", steveSnap[0].Content)
	}

	alexSnap := s.Snapshot("Alex")
	if len(alexSnap) != 1 {
		t.Fatalf("Alex should have 1 message, got %d", len(alexSnap))
	}
}

func TestConversationStoreReset(t *testing.T) {
	s := NewConversationStore()
	s.Append("Steve", llm.Message{Role: llm.RoleUser, Content: "hello"})
	s.Reset("Steve")

	snap := s.Snapshot("Steve")
	if len(snap) != 0 {
		t.Errorf("after reset, should have 0 messages, got %d", len(snap))
	}
}

func TestConversationStoreConcurrentAccess(t *testing.T) {
	s := NewConversationStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			player := "Steve"
			if n%2 == 0 {
				player = "Alex"
			}
			s.Append(player, llm.Message{Role: llm.RoleUser, Content: "msg"})
			_ = s.Snapshot(player)
		}(i)
	}
	wg.Wait()

	steveCount := len(s.Snapshot("Steve"))
	alexCount := len(s.Snapshot("Alex"))
	if steveCount+alexCount != 50 {
		t.Errorf("total messages = %d, want 50", steveCount+alexCount)
	}
}

func TestConversationStoreTrim(t *testing.T) {
	s := NewConversationStore()
	// Add 10 messages.
	for i := 0; i < 10; i++ {
		s.Append("Steve", llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf("msg-%d", i)})
	}

	s.Trim("Steve", 5)
	snap := s.Snapshot("Steve")
	if len(snap) != 5 {
		t.Fatalf("after trim, got %d messages, want 5", len(snap))
	}
	// First message preserved.
	if snap[0].Content != "msg-0" {
		t.Errorf("first msg = %q, want 'msg-0'", snap[0].Content)
	}
	// Last message is the most recent.
	if snap[4].Content != "msg-9" {
		t.Errorf("last msg = %q, want 'msg-9'", snap[4].Content)
	}
}

func TestConversationStoreTrimPreservesFirstMessage(t *testing.T) {
	s := NewConversationStore()
	s.Append("Steve",
		llm.Message{Role: llm.RoleUser, Content: "build a castle"},
		llm.Message{Role: llm.RoleAssistant, Content: "ok building"},
		llm.Message{Role: llm.RoleUser, Content: "make it bigger"},
		llm.Message{Role: llm.RoleAssistant, Content: "ok bigger"},
		llm.Message{Role: llm.RoleUser, Content: "add towers"},
	)

	s.Trim("Steve", 3)
	snap := s.Snapshot("Steve")
	if len(snap) != 3 {
		t.Fatalf("got %d messages, want 3", len(snap))
	}
	if snap[0].Content != "build a castle" {
		t.Errorf("first = %q, want original context preserved", snap[0].Content)
	}
	if snap[2].Content != "add towers" {
		t.Errorf("last = %q, want most recent", snap[2].Content)
	}
}

func TestConversationStoreCleanupIdle(t *testing.T) {
	s := NewConversationStore()
	s.Append("Steve", llm.Message{Role: llm.RoleUser, Content: "hello"})

	// Manually set LastActive to the past.
	conv := s.Get("Steve")
	conv.mu.Lock()
	conv.LastActive = time.Now().Add(-1 * time.Hour)
	conv.mu.Unlock()

	removed := s.CleanupIdle(30 * time.Minute)
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if s.Len() != 0 {
		t.Errorf("store should be empty after cleanup, got %d", s.Len())
	}
}

func TestConversationStoreCleanupKeepsActive(t *testing.T) {
	s := NewConversationStore()
	s.Append("Steve", llm.Message{Role: llm.RoleUser, Content: "hello"})

	removed := s.CleanupIdle(30 * time.Minute)
	if removed != 0 {
		t.Errorf("removed = %d, want 0 (conversation is active)", removed)
	}
	if s.Len() != 1 {
		t.Errorf("store should have 1 conversation, got %d", s.Len())
	}
}
