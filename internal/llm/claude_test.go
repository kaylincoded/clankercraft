package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func TestToAnthropicMessagesUserText(t *testing.T) {
	msgs := []Message{{Role: RoleUser, Content: "hello"}}
	out := toAnthropicMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("got %d messages, want 1", len(out))
	}
	if out[0].Role != anthropic.MessageParamRoleUser {
		t.Errorf("role = %q, want user", out[0].Role)
	}
	if len(out[0].Content) != 1 {
		t.Fatalf("got %d blocks, want 1", len(out[0].Content))
	}
	if out[0].Content[0].OfText == nil {
		t.Fatal("expected text block")
	}
	if out[0].Content[0].OfText.Text != "hello" {
		t.Errorf("text = %q, want %q", out[0].Content[0].OfText.Text, "hello")
	}
}

func TestToAnthropicMessagesAssistantWithToolCalls(t *testing.T) {
	msgs := []Message{{
		Role:    RoleAssistant,
		Content: "I'll help",
		ToolCalls: []ToolCall{{
			ID:    "tc_1",
			Name:  "set_block",
			Input: json.RawMessage(`{"x":1,"y":2,"z":3}`),
		}},
	}}
	out := toAnthropicMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("got %d messages, want 1", len(out))
	}
	if out[0].Role != anthropic.MessageParamRoleAssistant {
		t.Errorf("role = %q, want assistant", out[0].Role)
	}
	// Should have text block + tool_use block
	if len(out[0].Content) != 2 {
		t.Fatalf("got %d blocks, want 2", len(out[0].Content))
	}
	if out[0].Content[0].OfText == nil {
		t.Fatal("expected first block to be text")
	}
	if out[0].Content[1].OfToolUse == nil {
		t.Fatal("expected second block to be tool_use")
	}
	tu := out[0].Content[1].OfToolUse
	if tu.ID != "tc_1" {
		t.Errorf("tool use ID = %q, want %q", tu.ID, "tc_1")
	}
	if tu.Name != "set_block" {
		t.Errorf("tool use name = %q, want %q", tu.Name, "set_block")
	}
}

func TestToAnthropicMessagesToolResults(t *testing.T) {
	msgs := []Message{{
		Role: RoleUser,
		ToolResults: []ToolResult{
			{ToolCallID: "tc_1", Content: `{"ok":true}`, IsError: false},
			{ToolCallID: "tc_2", Content: "failed", IsError: true},
		},
	}}
	out := toAnthropicMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("got %d messages, want 1", len(out))
	}
	if out[0].Role != anthropic.MessageParamRoleUser {
		t.Errorf("role = %q, want user", out[0].Role)
	}
	if len(out[0].Content) != 2 {
		t.Fatalf("got %d blocks, want 2", len(out[0].Content))
	}
	// Both should be tool_result blocks
	for i, block := range out[0].Content {
		if block.OfToolResult == nil {
			t.Errorf("block[%d] expected tool_result", i)
		}
	}
}

func TestToAnthropicMessagesMultiTurn(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "build a wall"},
		{Role: RoleAssistant, Content: "", ToolCalls: []ToolCall{{ID: "tc_1", Name: "we_set", Input: json.RawMessage(`{"block":"stone"}`)}}},
		{Role: RoleUser, ToolResults: []ToolResult{{ToolCallID: "tc_1", Content: "done"}}},
		{Role: RoleAssistant, Content: "Wall built!"},
	}
	out := toAnthropicMessages(msgs)
	if len(out) != 4 {
		t.Fatalf("got %d messages, want 4", len(out))
	}
	// Verify roles alternate correctly
	roles := []anthropic.MessageParamRole{
		anthropic.MessageParamRoleUser,
		anthropic.MessageParamRoleAssistant,
		anthropic.MessageParamRoleUser,
		anthropic.MessageParamRoleAssistant,
	}
	for i, want := range roles {
		if out[i].Role != want {
			t.Errorf("msg[%d] role = %q, want %q", i, out[i].Role, want)
		}
	}
}

func TestToAnthropicTools(t *testing.T) {
	tools := []ToolDef{{
		Name:        "set_block",
		Description: "Places a block",
		InputSchema: json.RawMessage(`{"properties":{"x":{"type":"integer"},"y":{"type":"integer"}},"required":["x","y"]}`),
	}}
	out := toAnthropicTools(tools)
	if len(out) != 1 {
		t.Fatalf("got %d tools, want 1", len(out))
	}
	if out[0].OfTool == nil {
		t.Fatal("expected OfTool to be set")
	}
	if out[0].OfTool.Name != "set_block" {
		t.Errorf("name = %q, want %q", out[0].OfTool.Name, "set_block")
	}
	// Verify schema structure: Properties and Required extracted separately.
	schema := out[0].OfTool.InputSchema
	if schema.Properties == nil {
		t.Fatal("expected Properties to be set")
	}
	if len(schema.Required) != 2 {
		t.Errorf("required = %v, want [x y]", schema.Required)
	}
}

func TestParseResponseTextOnly(t *testing.T) {
	msg := &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			textContentBlock("Hello there!"),
		},
		StopReason: "end_turn",
	}
	r := parseResponse(msg)
	if r.Content != "Hello there!" {
		t.Errorf("content = %q, want %q", r.Content, "Hello there!")
	}
	if len(r.ToolCalls) != 0 {
		t.Errorf("tool calls = %d, want 0", len(r.ToolCalls))
	}
	if r.StopReason != StopReasonEndTurn {
		t.Errorf("stop reason = %q, want %q", r.StopReason, StopReasonEndTurn)
	}
}

func TestParseResponseToolCalls(t *testing.T) {
	msg := &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			toolUseContentBlock("tc_1", "set_block", `{"x":1}`),
		},
		StopReason: "tool_use",
	}
	r := parseResponse(msg)
	if len(r.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(r.ToolCalls))
	}
	tc := r.ToolCalls[0]
	if tc.ID != "tc_1" {
		t.Errorf("id = %q, want %q", tc.ID, "tc_1")
	}
	if tc.Name != "set_block" {
		t.Errorf("name = %q, want %q", tc.Name, "set_block")
	}
	if string(tc.Input) != `{"x":1}` {
		t.Errorf("input = %q, want %q", string(tc.Input), `{"x":1}`)
	}
	if r.StopReason != StopReasonToolUse {
		t.Errorf("stop reason = %q, want %q", r.StopReason, StopReasonToolUse)
	}
}

func TestParseResponseMixedContent(t *testing.T) {
	msg := &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			textContentBlock("Let me help. "),
			toolUseContentBlock("tc_1", "we_set", `{"block":"stone"}`),
			textContentBlock("Setting stone."),
		},
		StopReason: "tool_use",
	}
	r := parseResponse(msg)
	if r.Content != "Let me help. Setting stone." {
		t.Errorf("content = %q, want %q", r.Content, "Let me help. Setting stone.")
	}
	if len(r.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(r.ToolCalls))
	}
}

func TestNewClaudeProviderDefaults(t *testing.T) {
	p := NewClaudeProvider("test-key")
	if p.model != anthropic.ModelClaudeSonnet4_6 {
		t.Errorf("model = %q, want %q", p.model, anthropic.ModelClaudeSonnet4_6)
	}
	if p.maxTokens != 4096 {
		t.Errorf("maxTokens = %d, want 4096", p.maxTokens)
	}
	if p.systemPrompt != "" {
		t.Errorf("systemPrompt = %q, want empty", p.systemPrompt)
	}
}

func TestNewClaudeProviderOptions(t *testing.T) {
	p := NewClaudeProvider("test-key",
		WithModel("custom-model"),
		WithMaxTokens(8192),
		WithSystemPrompt("You are a builder bot"),
	)
	if p.model != "custom-model" {
		t.Errorf("model = %q, want %q", p.model, "custom-model")
	}
	if p.maxTokens != 8192 {
		t.Errorf("maxTokens = %d, want 8192", p.maxTokens)
	}
	if p.systemPrompt != "You are a builder bot" {
		t.Errorf("systemPrompt = %q, want %q", p.systemPrompt, "You are a builder bot")
	}
}

func TestClaudeProviderChatMockServer(t *testing.T) {
	// Mock Anthropic API server that returns a simple text response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)

		// Verify request structure
		if req["model"] == nil {
			t.Error("request missing model")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "msg_test",
			"type": "message",
			"role": "assistant",
			"content": []map[string]any{
				{"type": "text", "text": "I'll build that for you!"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 10, "output_tokens": 8},
		})
	}))
	defer server.Close()

	p := &ClaudeProvider{
		client:    anthropic.NewClient(option.WithAPIKey("test-key"), option.WithBaseURL(server.URL)),
		model:     "test-model",
		maxTokens: 1024,
	}

	resp, err := p.Chat(context.Background(), []Message{
		{Role: RoleUser, Content: "build me a house"},
	}, nil)
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if resp.Content != "I'll build that for you!" {
		t.Errorf("content = %q, want %q", resp.Content, "I'll build that for you!")
	}
	if resp.StopReason != StopReasonEndTurn {
		t.Errorf("stop reason = %q, want %q", resp.StopReason, StopReasonEndTurn)
	}
}

// --- helpers to construct ContentBlockUnion for tests ---

func textContentBlock(text string) anthropic.ContentBlockUnion {
	raw := json.RawMessage(`{"type":"text","text":` + mustJSON(text) + `}`)
	var block anthropic.ContentBlockUnion
	_ = json.Unmarshal(raw, &block)
	return block
}

func toolUseContentBlock(id, name, inputJSON string) anthropic.ContentBlockUnion {
	raw := json.RawMessage(`{"type":"tool_use","id":` + mustJSON(id) + `,"name":` + mustJSON(name) + `,"input":` + inputJSON + `}`)
	var block anthropic.ContentBlockUnion
	_ = json.Unmarshal(raw, &block)
	return block
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
