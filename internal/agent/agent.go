package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kaylincoded/clankercraft/internal/llm"
)

const defaultMaxIterations = 25

// DefaultSystemPrompt is the system prompt sent with every LLM request.
const DefaultSystemPrompt = `You are Clankercraft, a Minecraft building assistant bot. You help players build structures using WorldEdit commands and vanilla Minecraft commands.

When a player asks you to build something:
1. Think about what tools you need and plan your approach
2. Use set-selection to define regions, then WorldEdit commands like we-set, we-walls, we-hollow etc.
3. For simple builds, use setblock or fill commands
4. Check your position with get-position to orient yourself
5. Use detect-worldedit to check what capabilities are available

You have access to WorldEdit commands (sphere, cylinder, pyramid, copy/paste, etc.), vanilla commands (setblock, fill, clone), and world query tools (scan-area, find-block, read-sign).

Be concise in your responses. Report what you built and any issues encountered.`

// AgentOption configures an Agent.
type AgentOption func(*Agent)

// WithSystemPrompt overrides the default system prompt.
func WithSystemPrompt(prompt string) AgentOption {
	return func(a *Agent) { a.systemPrompt = prompt }
}

// WithMaxIterations overrides the default max iterations.
func WithMaxIterations(n int) AgentOption {
	return func(a *Agent) { a.maxIterations = n }
}

// Agent orchestrates whisper → LLM conversation → tool execution → whisper reply.
type Agent struct {
	provider      llm.Provider
	executor      *ToolExecutor
	logger        *slog.Logger
	systemPrompt  string
	maxIterations int
}

// NewAgent creates an Agent.
func NewAgent(provider llm.Provider, executor *ToolExecutor, logger *slog.Logger, opts ...AgentOption) *Agent {
	a := &Agent{
		provider:      provider,
		executor:      executor,
		logger:        logger,
		systemPrompt:  DefaultSystemPrompt,
		maxIterations: defaultMaxIterations,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// HandleMessage runs the agent loop for a single player message.
// It calls sendReply with the final text response.
func (a *Agent) HandleMessage(ctx context.Context, player, message string, sendReply func(string) error) error {
	messages := []llm.Message{{Role: llm.RoleUser, Content: message}}
	tools := a.executor.ToolDefs()

	// If provider has a system prompt option, set it.
	// The system prompt is configured on the provider at creation time (via llm.WithSystemPrompt).

	a.logger.Info("agent handling message",
		slog.String("player", player),
		slog.String("message", message),
	)

	for i := 0; i < a.maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := a.provider.Chat(ctx, messages, tools)
		if err != nil {
			return fmt.Errorf("llm chat: %w", err)
		}

		// Append assistant response to conversation history.
		messages = append(messages, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		if resp.StopReason == llm.StopReasonEndTurn {
			if resp.Content != "" {
				return sendReply(resp.Content)
			}
			return nil
		}

		// Execute tool calls.
		var results []llm.ToolResult
		for _, tc := range resp.ToolCalls {
			a.logger.Debug("executing tool",
				slog.String("tool", tc.Name),
				slog.String("id", tc.ID),
			)

			result, execErr := a.executor.Execute(ctx, tc.Name, tc.Input)
			if execErr != nil {
				a.logger.Warn("tool execution failed",
					slog.String("tool", tc.Name),
					slog.Any("error", execErr),
				)
				errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
				results = append(results, llm.ToolResult{
					ToolCallID: tc.ID,
					Content:    string(errJSON),
					IsError:    true,
				})
			} else {
				results = append(results, llm.ToolResult{
					ToolCallID: tc.ID,
					Content:    result,
					IsError:    false,
				})
			}
		}

		// Append tool results as a user message.
		messages = append(messages, llm.Message{
			Role:        llm.RoleUser,
			ToolResults: results,
		})
	}

	return sendReply("I've reached my step limit. Please try a simpler request or continue from where I left off.")
}
