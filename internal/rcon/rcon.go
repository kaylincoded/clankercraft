package rcon

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"

	mcnet "github.com/Tnze/go-mc/net"
	"github.com/kaylincoded/clankercraft/internal/config"
)

// DialFunc abstracts mcnet.DialRCON for testing.
type DialFunc func(addr, password string) (mcnet.RCONClientConn, error)

// Client manages an RCON connection to a Minecraft server.
type Client struct {
	cfg    *config.Config
	logger *slog.Logger

	mu        sync.Mutex
	conn      mcnet.RCONClientConn
	available bool
	dialFn    DialFunc
}

// New creates an RCON client. Call Connect to establish the connection.
func New(cfg *config.Config, logger *slog.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		dialFn: mcnet.DialRCON,
	}
}

// dialResult holds the result of a DialRCON call.
type dialResult struct {
	conn mcnet.RCONClientConn
	err  error
}

// Connect attempts to establish an RCON connection. If no password is configured,
// it returns immediately with RCON marked as unavailable. On connection failure,
// it logs a warning and returns nil (graceful degradation). The context controls
// the dial timeout — if cancelled, Connect returns without blocking.
func (c *Client) Connect(ctx context.Context) error {
	if c.cfg.RCONPassword == "" {
		c.logger.Info("RCON not configured (no password), using chat-only mode")
		return nil
	}

	addr := c.cfg.Host + ":" + strconv.Itoa(c.cfg.RCONPort)
	c.logger.Info("connecting to RCON", slog.String("addr", addr))

	// Run dial in a goroutine so context cancellation can interrupt it.
	ch := make(chan dialResult, 1)
	go func() {
		conn, err := c.dialFn(addr, c.cfg.RCONPassword)
		ch <- dialResult{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		c.logger.Warn("RCON connection cancelled, falling back to chat-only mode",
			slog.String("addr", addr),
			slog.String("error", ctx.Err().Error()),
		)
		return nil
	case result := <-ch:
		if result.err != nil {
			c.logger.Warn("RCON connection failed, falling back to chat-only mode",
				slog.String("addr", addr),
				slog.String("error", result.err.Error()),
			)
			return nil
		}

		c.mu.Lock()
		c.conn = result.conn
		c.available = true
		c.mu.Unlock()

		c.logger.Info("RCON connected", slog.String("addr", addr))
		return nil
	}
}

// Execute sends a command via RCON and returns the server response.
// Returns an error if RCON is not available.
func (c *Client) Execute(command string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.available || c.conn == nil {
		return "", fmt.Errorf("RCON is not available")
	}

	if err := c.conn.Cmd(command); err != nil {
		return "", fmt.Errorf("sending RCON command: %w", err)
	}

	resp, err := c.conn.Resp()
	if err != nil {
		return "", fmt.Errorf("reading RCON response: %w", err)
	}

	return resp, nil
}

// IsAvailable returns true if an RCON connection is established.
func (c *Client) IsAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.available
}

// Close closes the RCON connection. Safe to call on an unconnected client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	c.available = false
	err := c.conn.Close()
	c.conn = nil
	return err
}
