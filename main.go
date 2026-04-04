package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/kaylincoded/clankercraft/internal/agent"
	"github.com/kaylincoded/clankercraft/internal/config"
	"github.com/kaylincoded/clankercraft/internal/connection"
	"github.com/kaylincoded/clankercraft/internal/llm"
	cclog "github.com/kaylincoded/clankercraft/internal/log"
	"github.com/kaylincoded/clankercraft/internal/mcp"
	"github.com/kaylincoded/clankercraft/internal/rcon"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "clankercraft",
		Short:   "AI building partner for Minecraft via WorldEdit",
		Version: version,
		RunE:    run,
	}

	config.BindFlags(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cmd)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := cclog.Setup(cfg.LogLevel)

	logger.Info("clankercraft starting",
		slog.String("version", version),
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("username", cfg.Username),
		slog.String("log_level", cfg.LogLevel),
		slog.Bool("offline", cfg.Offline),
		slog.Int("rcon_port", cfg.RCONPort),
		slog.String("rcon_password", cfg.MaskedRCONPassword()),
	)

	// Graceful shutdown via context — first signal cancels context,
	// stop() restores default handler so second signal force-kills.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect RCON (optional — logs warning and continues on failure).
	rconClient := rcon.New(cfg, logger)
	rconClient.Connect(ctx)

	// Initialize LLM provider (optional — nil means no LLM features).
	var llmProvider llm.Provider
	if cfg.AnthropicAPIKey != "" {
		opts := []llm.ClaudeOption{
			llm.WithSystemPrompt(agent.DefaultSystemPrompt),
		}
		if cfg.LLMModel != "" {
			opts = append(opts, llm.WithModel(cfg.LLMModel))
		}
		llmProvider = llm.NewClaudeProvider(cfg.AnthropicAPIKey, opts...)
		logger.Info("LLM provider initialized", slog.String("model", "claude"))
	} else {
		logger.Warn("ANTHROPIC_API_KEY not set — LLM features disabled")
	}

	// Start MC connection and MCP server concurrently.
	// errgroup cancels gctx on first error, so a broken MCP transport
	// tears down the MC connection (and vice versa).
	conn := connection.New(cfg, logger)
	conn.SetRCON(rconClient)
	mcpServer := mcp.New(version, logger, conn)

	// Wire agent loop: whisper → LLM → tool execution → whisper reply.
	if llmProvider != nil {
		toolExec := agent.NewToolExecutor(conn)
		agentLoop := agent.NewAgent(llmProvider, toolExec, logger)
		conn.OnWhisper(func(sender, msg string) {
			go func() {
				replyFn := func(reply string) error { return conn.SendWhisper(sender, reply) }
				if err := agentLoop.HandleMessage(ctx, sender, msg, replyFn); err != nil {
					logger.Error("agent error", slog.String("player", sender), slog.Any("error", err))
				}
			}()
		})
		logger.Info("agent loop wired — whisper to interact")
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return conn.RunWithReconnect(gctx) })
	g.Go(func() error { return mcpServer.Run(gctx) })

	err = g.Wait()

	// Restore default signal handling so second Ctrl+C force-quits
	stop()
	logger.Info("shutting down, press Ctrl+C again to force quit")
	rconClient.Close()
	conn.Close()

	if err != nil && err != context.Canceled {
		return err
	}
	return nil
}
