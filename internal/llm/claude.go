package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ClaudeOption configures a ClaudeProvider.
type ClaudeOption func(*ClaudeProvider)

// WithModel overrides the default model.
func WithModel(model string) ClaudeOption {
	return func(p *ClaudeProvider) { p.model = model }
}

// WithMaxTokens overrides the default max tokens.
func WithMaxTokens(n int64) ClaudeOption {
	return func(p *ClaudeProvider) { p.maxTokens = n }
}

// WithSystemPrompt sets the system prompt sent with every request.
func WithSystemPrompt(prompt string) ClaudeOption {
	return func(p *ClaudeProvider) { p.systemPrompt = prompt }
}

// ClaudeProvider implements Provider using the Anthropic Claude API.
type ClaudeProvider struct {
	client       anthropic.Client
	model        string
	maxTokens    int64
	systemPrompt string
}

// NewClaudeProvider creates a Provider backed by Claude.
func NewClaudeProvider(apiKey string, opts ...ClaudeOption) *ClaudeProvider {
	p := &ClaudeProvider{
		client:    anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:     anthropic.ModelClaudeSonnet4_6,
		maxTokens: 4096,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// Chat sends a conversation to Claude and returns the response.
func (p *ClaudeProvider) Chat(ctx context.Context, messages []Message, tools []ToolDef) (*Response, error) {
	params := anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: p.maxTokens,
		Messages:  toAnthropicMessages(messages),
	}

	if len(tools) > 0 {
		params.Tools = toAnthropicTools(tools)
	}

	if p.systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: p.systemPrompt},
		}
	}

	msg, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("claude API: %w", err)
	}

	return parseResponse(msg), nil
}

// toAnthropicMessages converts our Message slice to Anthropic SDK format.
func toAnthropicMessages(msgs []Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch {
		case len(m.ToolResults) > 0:
			// User message carrying tool results.
			blocks := make([]anthropic.ContentBlockParamUnion, len(m.ToolResults))
			for i, tr := range m.ToolResults {
				blocks[i] = anthropic.NewToolResultBlock(tr.ToolCallID, tr.Content, tr.IsError)
			}
			out = append(out, anthropic.NewUserMessage(blocks...))

		case m.Role == RoleAssistant && len(m.ToolCalls) > 0:
			// Assistant message with tool calls (may also have text).
			var blocks []anthropic.ContentBlockParamUnion
			if m.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(m.Content))
			}
			for _, tc := range m.ToolCalls {
				var input any
				if len(tc.Input) > 0 {
					_ = json.Unmarshal(tc.Input, &input)
				}
				blocks = append(blocks, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: input,
					},
				})
			}
			out = append(out, anthropic.MessageParam{
				Role:    anthropic.MessageParamRoleAssistant,
				Content: blocks,
			})

		default:
			// Plain text message (user or assistant).
			role := anthropic.MessageParamRoleUser
			if m.Role == RoleAssistant {
				role = anthropic.MessageParamRoleAssistant
			}
			out = append(out, anthropic.MessageParam{
				Role:    role,
				Content: []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(m.Content)},
			})
		}
	}
	return out
}

// toAnthropicTools converts our ToolDef slice to Anthropic SDK format.
func toAnthropicTools(tools []ToolDef) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, len(tools))
	for i, td := range tools {
		schema := anthropic.ToolInputSchemaParam{}
		if len(td.InputSchema) > 0 {
			// InputSchema is a JSON Schema object like:
			//   {"properties":{...},"required":["x","y"]}
			// We extract Properties and Required into their respective fields.
			var raw struct {
				Properties any      `json:"properties"`
				Required   []string `json:"required"`
			}
			if err := json.Unmarshal(td.InputSchema, &raw); err == nil {
				schema.Properties = raw.Properties
				schema.Required = raw.Required
			}
		}
		out[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        td.Name,
				Description: anthropic.String(td.Description),
				InputSchema: schema,
			},
		}
	}
	return out
}

// parseResponse extracts text and tool calls from an Anthropic API response.
func parseResponse(msg *anthropic.Message) *Response {
	r := &Response{
		StopReason: string(msg.StopReason),
	}

	var textParts []string
	for _, block := range msg.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			textParts = append(textParts, v.Text)
		case anthropic.ToolUseBlock:
			r.ToolCalls = append(r.ToolCalls, ToolCall{
				ID:    v.ID,
				Name:  v.Name,
				Input: v.Input,
			})
		}
	}
	r.Content = strings.Join(textParts, "")
	return r
}
