package agent

import (
	"sync"
	"time"

	"github.com/kaylincoded/clankercraft/internal/llm"
)

// Conversation holds the message history for a single player.
type Conversation struct {
	Messages   []llm.Message
	LastActive time.Time
	mu         sync.Mutex
}

// ConversationStore manages per-player conversation histories.
type ConversationStore struct {
	mu    sync.RWMutex
	convs map[string]*Conversation
}

// NewConversationStore creates an empty store.
func NewConversationStore() *ConversationStore {
	return &ConversationStore{convs: make(map[string]*Conversation)}
}

// Get returns the conversation for the player, creating one if it doesn't exist.
func (s *ConversationStore) Get(player string) *Conversation {
	s.mu.RLock()
	conv, ok := s.convs[player]
	s.mu.RUnlock()
	if ok {
		return conv
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock.
	if conv, ok = s.convs[player]; ok {
		return conv
	}
	conv = &Conversation{LastActive: time.Now()}
	s.convs[player] = conv
	return conv
}

// Append adds messages to a player's conversation and updates LastActive.
func (s *ConversationStore) Append(player string, msgs ...llm.Message) {
	conv := s.Get(player)
	conv.mu.Lock()
	defer conv.mu.Unlock()
	conv.Messages = append(conv.Messages, msgs...)
	conv.LastActive = time.Now()
}

// Reset clears a player's conversation history.
func (s *ConversationStore) Reset(player string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.convs, player)
}

// Snapshot returns a copy of the player's messages (safe to use without locks).
func (s *ConversationStore) Snapshot(player string) []llm.Message {
	conv := s.Get(player)
	conv.mu.Lock()
	defer conv.mu.Unlock()
	out := make([]llm.Message, len(conv.Messages))
	copy(out, conv.Messages)
	return out
}

// Trim ensures a player's conversation doesn't exceed maxMessages.
// Preserves the first message (original context) and the most recent messages.
func (s *ConversationStore) Trim(player string, maxMessages int) {
	conv := s.Get(player)
	conv.mu.Lock()
	defer conv.mu.Unlock()
	if len(conv.Messages) <= maxMessages {
		return
	}
	// Keep first message + most recent (maxMessages-1).
	tail := conv.Messages[len(conv.Messages)-(maxMessages-1):]
	trimmed := make([]llm.Message, 0, maxMessages)
	trimmed = append(trimmed, conv.Messages[0])
	trimmed = append(trimmed, tail...)
	conv.Messages = trimmed
}

// CleanupIdle removes conversations that have been idle for longer than maxIdle.
func (s *ConversationStore) CleanupIdle(maxIdle time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	removed := 0
	for player, conv := range s.convs {
		conv.mu.Lock()
		idle := now.Sub(conv.LastActive) > maxIdle
		conv.mu.Unlock()
		if idle {
			delete(s.convs, player)
			removed++
		}
	}
	return removed
}

// Len returns the number of active conversations.
func (s *ConversationStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.convs)
}
