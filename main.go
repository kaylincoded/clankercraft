package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaylincoded/clankercraft/internal/config"
	"github.com/kaylincoded/clankercraft/internal/connection"
	cclog "github.com/kaylincoded/clankercraft/internal/log"
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

	// Graceful shutdown via context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Connect to Minecraft server with auto-reconnect
	conn := connection.New(cfg, logger)
	gameErr := conn.RunWithReconnect(ctx)

	logger.Info("shutting down")
	conn.Close()

	if gameErr != nil && gameErr != context.Canceled {
		return gameErr
	}
	return nil
}
