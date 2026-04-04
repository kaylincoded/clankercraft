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
	te := NewToolExecutor(bot)
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
	te := NewToolExecutor(bot)
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
	te := NewToolExecutor(bot)
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
	te := NewToolExecutor(bot)
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
	te := NewToolExecutor(bot)
	agent := NewAgent(provider, te, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := agent.HandleMessage(ctx, "Steve", "test", func(msg string) error { return nil })
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
