package chat

import (
	"fmt"
	"regexp"
	"sync"
	"time"
	"unicode/utf8"
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

// DefaultMessageDelay is the default delay between multi-part whisper messages.
const DefaultMessageDelay = 200 * time.Millisecond

// Sender sends chat messages (whispers) to players via the server.
type Sender struct {
	sendCommandFn func(string) error
	MessageDelay  time.Duration
}

// NewSender creates a Sender that dispatches commands via sendCommandFn.
// The function should behave like Connection.SendCommand — the command string
// is sent via ServerboundChatCommand with an implicit leading /.
func NewSender(sendCommandFn func(string) error) *Sender {
	return &Sender{
		sendCommandFn: sendCommandFn,
		MessageDelay:  DefaultMessageDelay,
	}
}

// SendWhisper sends a whisper (/msg) to the given player. Long messages are
// automatically split across multiple commands to stay within the 256-char
// Minecraft command limit, with MessageDelay between each send.
func (s *Sender) SendWhisper(player, message string) error {
	if player == "" {
		return fmt.Errorf("player name is required")
	}
	if message == "" {
		return fmt.Errorf("message is required")
	}

	prefix := "msg " + player + " "
	maxContent := 256 - len(prefix)
	if maxContent <= 0 {
		return fmt.Errorf("player name too long")
	}

	chunks := splitMessage(message, maxContent)
	for i, chunk := range chunks {
		if i > 0 && s.MessageDelay > 0 {
			time.Sleep(s.MessageDelay)
		}
		if err := s.sendCommandFn(prefix + chunk); err != nil {
			return fmt.Errorf("sending whisper chunk %d: %w", i, err)
		}
	}
	return nil
}

// splitMessage splits text into chunks whose byte length does not exceed
// maxLen, preferring to break at word (space) boundaries. Multi-byte UTF-8
// characters are never split mid-rune. If a single word's byte length
// exceeds maxLen, it is hard-split on rune boundaries.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}

		// Find the largest rune-safe byte offset <= maxLen.
		cut := runeByteLimit(text, maxLen)

		// Prefer splitting at the last space within that range.
		if idx := lastSpaceBefore(text, cut); idx > 0 {
			cut = idx
		}

		chunks = append(chunks, text[:cut])
		text = text[cut:]
		// Trim leading space from next chunk.
		if len(text) > 0 && text[0] == ' ' {
			text = text[1:]
		}
	}
	return chunks
}

// runeByteLimit returns the largest byte offset in s that is <= limit
// and falls on a rune boundary.
func runeByteLimit(s string, limit int) int {
	if limit >= len(s) {
		return len(s)
	}
	// Walk back from limit to find a rune boundary.
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	return limit
}

// lastSpaceBefore returns the byte index of the last space in s[:limit],
// or -1 if none.
func lastSpaceBefore(s string, limit int) int {
	for i := limit - 1; i >= 0; i-- {
		if s[i] == ' ' {
			return i
		}
	}
	return -1
}
