package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/kaylincoded/clankercraft/internal/llm"
)

// mockProvider implements llm.Provider for testing.
type mockProvider struct {
	responses []*llm.Response
	callIndex int
	lastMsgs  []llm.Message
}

func (m *mockProvider) Chat(_ context.Context, messages []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
	m.lastMsgs = messages
	if m.callIndex >= len(m.responses) {
		return nil, fmt.Errorf("no more mock responses")
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return resp, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandleMessageSingleTextResponse(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.Response{
			{Content: "I'll build that!", StopReason: llm.StopReasonEndTurn},
		},
	}
	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	agent := NewAgent(provider, te, testLogger())

	var reply string
	err := agent.HandleMessage(context.Background(), "Steve", "build a house", func(msg string) error {
		reply = msg
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "I'll build that!" {
		t.Errorf("reply = %q, want 'I'll build that!'", reply)
	}
}

func TestHandleMessageToolCallThenText(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.Response{
			{
				Content: "",
				ToolCalls: []llm.ToolCall{{
					ID:    "tc_1",
					Name:  "ping",
					Input: json.RawMessage(`{}`),
				}},
				StopReason: llm.StopReasonToolUse,
			},
			{Content: "Bot is responsive!", StopReason: llm.StopReasonEndTurn},
		},
	}
	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	agent := NewAgent(provider, te, testLogger())

	var reply string
	err := agent.HandleMessage(context.Background(), "Steve", "check the bot", func(msg string) error {
		reply = msg
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "Bot is responsive!" {
		t.Errorf("reply = %q, want 'Bot is responsive!'", reply)
	}
	// Verify provider was called twice (tool call + final).
	if provider.callIndex != 2 {
		t.Errorf("provider called %d times, want 2", provider.callIndex)
	}
}

func TestHandleMessageToolCallError(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID:    "tc_1",
					Name:  "get-position",
					Input: json.RawMessage(`{}`),
				}},
				StopReason: llm.StopReasonToolUse,
			},
			{Content: "Sorry, I can't get my position right now.", StopReason: llm.StopReasonEndTurn},
		},
	}
	// Bot is not connected — get-position will fail.
	bot := &mockBot{connected: false}
	te := NewToolExecutor(bot, nil)
	agent := NewAgent(provider, te, testLogger())

	var reply string
	err := agent.HandleMessage(context.Background(), "Steve", "where are you?", func(msg string) error {
		reply = msg
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "Sorry, I can't get my position right now." {
		t.Errorf("reply = %q", reply)
	}

	// Verify the tool result was sent back with IsError=true.
	if len(provider.lastMsgs) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(provider.lastMsgs))
	}
	toolResultMsg := provider.lastMsgs[2]
	if len(toolResultMsg.ToolResults) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(toolResultMsg.ToolResults))
	}
	if !toolResultMsg.ToolResults[0].IsError {
		t.Error("expected tool result to be an error")
	}
}

func TestHandleMessageMaxIterations(t *testing.T) {
	// Provider always returns tool calls — should hit max iterations.
	responses := make([]*llm.Response, 5)
	for i := range responses {
		responses[i] = &llm.Response{
			ToolCalls: []llm.ToolCall{{
				ID:    fmt.Sprintf("tc_%d", i),
				Name:  "ping",
				Input: json.RawMessage(`{}`),
			}},
			StopReason: llm.StopReasonToolUse,
		}
	}
	provider := &mockProvider{responses: responses}
	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	agent := NewAgent(provider, te, testLogger(), WithMaxIterations(3))

	var reply string
	err := agent.HandleMessage(context.Background(), "Steve", "loop forever", func(msg string) error {
		reply = msg
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "I've reached my step limit. Please try a simpler request or continue from where I left off." {
		t.Errorf("reply = %q", reply)
	}
	if provider.callIndex != 3 {
		t.Errorf("provider called %d times, want 3", provider.callIndex)
	}
}

func TestHandleMessageContextCancellation(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "tc_1", Name: "ping", Input: json.RawMessage(`{}`),
				}},
				StopReason: llm.StopReasonToolUse,
			},
		},
	}
	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	agent := NewAgent(provider, te, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := agent.HandleMessage(ctx, "Steve", "test", func(msg string) error { return nil })
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestHandleMessagePreservesHistory(t *testing.T) {
	callCount := 0
	provider := &trackingProvider{
		chatFn: func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: only 1 message (new user msg, no history).
				if len(msgs) != 1 {
					t.Errorf("call 1: got %d messages, want 1", len(msgs))
				}
				return &llm.Response{Content: "Built a wall!", StopReason: llm.StopReasonEndTurn}, nil
			}
			// Second call: should have 3 messages (user + assistant from call 1, + new user).
			if len(msgs) != 3 {
				t.Errorf("call 2: got %d messages, want 3", len(msgs))
			}
			if msgs[0].Content != "[Steve]: build a wall" {
				t.Errorf("call 2: msg[0] = %q, want '[Steve]: build a wall'", msgs[0].Content)
			}
			if msgs[1].Content != "Built a wall!" {
				t.Errorf("call 2: msg[1] = %q, want 'Built a wall!'", msgs[1].Content)
			}
			if msgs[2].Content != "[Steve]: make it taller" {
				t.Errorf("call 2: msg[2] = %q, want '[Steve]: make it taller'", msgs[2].Content)
			}
			return &llm.Response{Content: "Made it taller!", StopReason: llm.StopReasonEndTurn}, nil
		},
	}

	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	ag := NewAgent(provider, te, testLogger())

	var reply string
	noopReply := func(msg string) error { reply = msg; return nil }

	_ = ag.HandleMessage(context.Background(), "Steve", "build a wall", noopReply)
	if reply != "Built a wall!" {
		t.Fatalf("first reply = %q", reply)
	}

	_ = ag.HandleMessage(context.Background(), "Steve", "make it taller", noopReply)
	if reply != "Made it taller!" {
		t.Fatalf("second reply = %q", reply)
	}
}

func TestHandleMessageIsolatesPlayers(t *testing.T) {
	provider := &trackingProvider{
		chatFn: func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
			return &llm.Response{Content: "ok", StopReason: llm.StopReasonEndTurn}, nil
		},
	}

	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	ag := NewAgent(provider, te, testLogger())
	noopReply := func(msg string) error { return nil }

	// Steve sends a message.
	_ = ag.HandleMessage(context.Background(), "Steve", "build a house", noopReply)
	// Alex sends a message — should not see Steve's history.
	var alexMsgCount int
	provider.chatFn = func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
		alexMsgCount = len(msgs)
		return &llm.Response{Content: "ok", StopReason: llm.StopReasonEndTurn}, nil
	}
	_ = ag.HandleMessage(context.Background(), "Alex", "build a farm", noopReply)

	if alexMsgCount != 1 {
		t.Errorf("Alex should see 1 message (own), got %d", alexMsgCount)
	}
}

func TestHandleMessageResetCommand(t *testing.T) {
	provider := &trackingProvider{
		chatFn: func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
			return &llm.Response{Content: "ok", StopReason: llm.StopReasonEndTurn}, nil
		},
	}

	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	ag := NewAgent(provider, te, testLogger())

	var reply string
	replyFn := func(msg string) error { reply = msg; return nil }

	// Build up history.
	_ = ag.HandleMessage(context.Background(), "Steve", "build a wall", replyFn)

	// Reset.
	_ = ag.HandleMessage(context.Background(), "Steve", "reset", replyFn)
	if reply != "Conversation cleared. What would you like to build?" {
		t.Errorf("reset reply = %q", reply)
	}

	// Next message should have no history.
	var msgCount int
	provider.chatFn = func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
		msgCount = len(msgs)
		return &llm.Response{Content: "fresh", StopReason: llm.StopReasonEndTurn}, nil
	}
	_ = ag.HandleMessage(context.Background(), "Steve", "build a tower", replyFn)
	if msgCount != 1 {
		t.Errorf("after reset, should have 1 message, got %d", msgCount)
	}
}

func TestHandleMessageResetCommandVariants(t *testing.T) {
	provider := &trackingProvider{
		chatFn: func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
			return &llm.Response{Content: "ok", StopReason: llm.StopReasonEndTurn}, nil
		},
	}

	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	ag := NewAgent(provider, te, testLogger())

	variants := []string{"reset", "clear", "new conversation", "forget", "Reset", "CLEAR", "  forget  "}
	for _, v := range variants {
		// Build up history first.
		_ = ag.HandleMessage(context.Background(), "Steve", "build something", func(string) error { return nil })

		var reply string
		_ = ag.HandleMessage(context.Background(), "Steve", v, func(msg string) error { reply = msg; return nil })
		if reply != "Conversation cleared. What would you like to build?" {
			t.Errorf("variant %q: reply = %q, want reset confirmation", v, reply)
		}

		// Verify history is actually cleared.
		snap := ag.conversations.Snapshot("Steve")
		if len(snap) != 0 {
			t.Errorf("variant %q: expected 0 messages after reset, got %d", v, len(snap))
		}
	}
}

func TestHandleMessagePlayerNamePrefix(t *testing.T) {
	var capturedMsgs []llm.Message
	provider := &trackingProvider{
		chatFn: func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
			capturedMsgs = msgs
			return &llm.Response{Content: "ok", StopReason: llm.StopReasonEndTurn}, nil
		},
	}
	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	ag := NewAgent(provider, te, testLogger())

	_ = ag.HandleMessage(context.Background(), "PixiisRobot", "come here", func(string) error { return nil })

	if len(capturedMsgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(capturedMsgs))
	}
	if capturedMsgs[0].Content != "[PixiisRobot]: come here" {
		t.Errorf("message = %q, want '[PixiisRobot]: come here'", capturedMsgs[0].Content)
	}
}

func TestHandleMessageTrimming(t *testing.T) {
	callCount := 0
	provider := &trackingProvider{
		chatFn: func(_ context.Context, msgs []llm.Message, _ []llm.ToolDef) (*llm.Response, error) {
			callCount++
			return &llm.Response{Content: fmt.Sprintf("reply-%d", callCount), StopReason: llm.StopReasonEndTurn}, nil
		},
	}

	bot := &mockBot{connected: true}
	te := NewToolExecutor(bot, nil)
	ag := NewAgent(provider, te, testLogger(), WithMaxConversationMessages(4))
	noopReply := func(msg string) error { return nil }

	// Send 5 messages — each produces user+assistant = 2 stored messages = 10 total, trimmed to 4.
	for i := 0; i < 5; i++ {
		_ = ag.HandleMessage(context.Background(), "Steve", fmt.Sprintf("msg-%d", i), noopReply)
	}

	snap := ag.conversations.Snapshot("Steve")
	if len(snap) != 4 {
		t.Fatalf("after trimming, got %d messages, want 4", len(snap))
	}
	// First message preserved.
	if snap[0].Content != "[Steve]: msg-0" {
		t.Errorf("first = %q, want original context", snap[0].Content)
	}
}

// trackingProvider is a mock that allows per-call function customization.
type trackingProvider struct {
	chatFn func(context.Context, []llm.Message, []llm.ToolDef) (*llm.Response, error)
}

func (p *trackingProvider) Chat(ctx context.Context, msgs []llm.Message, tools []llm.ToolDef) (*llm.Response, error) {
	return p.chatFn(ctx, msgs, tools)
}
