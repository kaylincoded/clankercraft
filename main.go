package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/kaylincoded/clankercraft/internal/config"
	"github.com/kaylincoded/clankercraft/internal/connection"
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

	// Start MC connection and MCP server concurrently.
	// errgroup cancels gctx on first error, so a broken MCP transport
	// tears down the MC connection (and vice versa).
	conn := connection.New(cfg, logger)
	conn.SetRCON(rconClient)
	mcpServer := mcp.New(version, logger, conn)

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
