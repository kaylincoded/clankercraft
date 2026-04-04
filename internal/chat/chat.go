package chat

import (
	"regexp"
	"sync"
)

// Whisper represents an incoming whisper message from a player.
type Whisper struct {
	Sender  string
	Message string
}

// WhisperHandler is a callback invoked when a whisper is received.
type WhisperHandler func(Whisper)

// vanillaWhisperRe matches "PlayerName whispers to you: message content"
var vanillaWhisperRe = regexp.MustCompile(`^(\w+) whispers to you: (.+)$`)

// pluginWhisperRe matches "[PlayerName -> BotName] message content"
var pluginWhisperRe = regexp.MustCompile(`^\[(\w+) -> \w+\] (.+)$`)

// parseWhisper attempts to extract a Whisper from a system chat message.
// Returns nil if the message is not a whisper.
func parseWhisper(text string) *Whisper {
	if m := vanillaWhisperRe.FindStringSubmatch(text); m != nil {
		return &Whisper{Sender: m[1], Message: m[2]}
	}
	if m := pluginWhisperRe.FindStringSubmatch(text); m != nil {
		return &Whisper{Sender: m[1], Message: m[2]}
	}
	return nil
}

// Listener manages whisper event handlers and parses incoming chat messages.
type Listener struct {
	mu       sync.RWMutex
	handlers []WhisperHandler
}

// NewListener creates a Listener with no handlers registered.
func NewListener() *Listener {
	return &Listener{}
}

// OnWhisper registers a callback that fires when a whisper is received.
func (l *Listener) OnWhisper(handler WhisperHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, handler)
}

// HandleSystemChat parses a system chat message and dispatches whisper events.
// Non-whisper messages are silently ignored.
func (l *Listener) HandleSystemChat(text string) {
	w := parseWhisper(text)
	if w == nil {
		return
	}

	l.mu.RLock()
	handlers := make([]WhisperHandler, len(l.handlers))
	copy(handlers, l.handlers)
	l.mu.RUnlock()

	for _, h := range handlers {
		h(*w)
	}
}
