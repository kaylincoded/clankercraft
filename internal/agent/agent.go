package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kaylincoded/clankercraft/internal/llm"
)

const defaultMaxIterations = 25
const defaultMaxConversationMessages = 50
const cleanupInterval = 5 * time.Minute
const idleTimeout = 30 * time.Minute

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

// WithMaxIterations overrides the default max iterations per message.
func WithMaxIterations(n int) AgentOption {
	return func(a *Agent) { a.maxIterations = n }
}

// WithMaxConversationMessages overrides the max messages kept per conversation.
func WithMaxConversationMessages(n int) AgentOption {
	return func(a *Agent) { a.maxConversationMessages = n }
}

// Agent orchestrates whisper → LLM conversation → tool execution → whisper reply.
type Agent struct {
	provider                llm.Provider
	executor                *ToolExecutor
	logger                  *slog.Logger
	conversations           *ConversationStore
	systemPrompt            string
	maxIterations           int
	maxConversationMessages int
}

// NewAgent creates an Agent with a per-player conversation store.
func NewAgent(provider llm.Provider, executor *ToolExecutor, logger *slog.Logger, opts ...AgentOption) *Agent {
	a := &Agent{
		provider:                provider,
		executor:                executor,
		logger:                  logger,
		conversations:           NewConversationStore(),
		systemPrompt:            DefaultSystemPrompt,
		maxIterations:           defaultMaxIterations,
		maxConversationMessages: defaultMaxConversationMessages,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Start launches background goroutines (idle conversation cleanup).
// Must be called after NewAgent. Respects context cancellation.
func (a *Agent) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if removed := a.conversations.CleanupIdle(idleTimeout); removed > 0 {
					a.logger.Debug("cleaned up idle conversations", slog.Int("removed", removed))
				}
			}
		}
	}()
}

// isResetCommand checks if the message is a conversation reset command.
func isResetCommand(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	return lower == "reset" || lower == "clear" || lower == "new conversation" || lower == "forget"
}

// HandleMessage runs the agent loop for a single player message.
// Conversation history is maintained across calls for the same player.
func (a *Agent) HandleMessage(ctx context.Context, player, message string, sendReply func(string) error) error {
	// Check for reset command before anything else.
	if isResetCommand(message) {
		a.conversations.Reset(player)
		a.logger.Info("conversation reset", slog.String("player", player))
		return sendReply("Conversation cleared. What would you like to build?")
	}

	a.logger.Info("agent handling message",
		slog.String("player", player),
		slog.String("message", message),
	)

	// Load existing conversation history + new user message.
	history := a.conversations.Snapshot(player)
	messages := make([]llm.Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: message})

	historyLen := len(history)
	tools := a.executor.ToolDefs()

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
			// Persist new messages (everything after history).
			a.conversations.Append(player, messages[historyLen:]...)
			a.conversations.Trim(player, a.maxConversationMessages)

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

	// Persist even on max iterations — the conversation happened.
	a.conversations.Append(player, messages[historyLen:]...)
	a.conversations.Trim(player, a.maxConversationMessages)

	return sendReply("I've reached my step limit. Please try a simpler request or continue from where I left off.")
}
