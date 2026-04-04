package connection

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/bot/basic"
	"github.com/Tnze/go-mc/bot/msg"
	"github.com/Tnze/go-mc/bot/playerlist"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/offline"
	"github.com/kaylincoded/clankercraft/internal/config"
)

// Connection manages the Minecraft server connection via go-mc.
type Connection struct {
	cfg    *config.Config
	logger *slog.Logger

	client  *bot.Client
	player  *basic.Player
	msgMgr  *msg.Manager
	plist   *playerlist.PlayerList

	mu        sync.Mutex
	connected bool
}

// New creates a new Connection configured from the given config.
func New(cfg *config.Config, logger *slog.Logger) *Connection {
	return &Connection{
		cfg:    cfg,
		logger: logger,
	}
}

// Address returns the "host:port" string for the server.
func (c *Connection) Address() string {
	return net.JoinHostPort(c.cfg.Host, fmt.Sprintf("%d", c.cfg.Port))
}

// setupAuth configures authentication on the go-mc client.
func (c *Connection) setupAuth(client *bot.Client) {
	client.Auth.Name = c.cfg.Username
	if c.cfg.Offline {
		id := offline.NameToUUID(c.cfg.Username)
		client.Auth.UUID = hex.EncodeToString(id[:])
		c.logger.Info("connecting in offline mode", slog.String("username", c.cfg.Username))
	} else {
		// MSA auth token would be set here (Story 1.3)
		c.logger.Info("connecting with auth", slog.String("username", c.cfg.Username))
	}
}

// Connect joins the Minecraft server. Blocks until login+configuration is complete or fails.
func (c *Connection) Connect(ctx context.Context) error {
	client := bot.NewClient()
	c.setupAuth(client)
	c.client = client

	// basic.Player — handles keepalive, spawn, teleport acceptance
	c.player = basic.NewPlayer(client, basic.Settings{}, basic.EventsListener{
		GameStart: func() error {
			c.mu.Lock()
			c.connected = true
			c.mu.Unlock()
			c.logger.Info("spawned in world", slog.String("server", c.Address()))
			return nil
		},
		Disconnect: func(reason chat.Message) error {
			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()
			c.logger.Warn("disconnected by server", slog.String("reason", reason.String()))
			return nil
		},
	})

	// Player list
	c.plist = playerlist.New(client)

	// Chat message handling
	c.msgMgr = msg.New(client, c.player, c.plist, msg.EventsHandler{
		SystemChat: func(m chat.Message, overlay bool) error {
			c.logger.Info("chat:system", slog.String("message", m.String()), slog.Bool("overlay", overlay))
			return nil
		},
		PlayerChatMessage: func(m chat.Message, validated bool) error {
			c.logger.Info("chat:player", slog.String("message", m.String()), slog.Bool("validated", validated))
			return nil
		},
		DisguisedChat: func(m chat.Message) error {
			c.logger.Info("chat:disguised", slog.String("message", m.String()))
			return nil
		},
	})

	// Connect to server
	addr := c.Address()
	c.logger.Info("joining server", slog.String("address", addr))

	if err := client.JoinServer(addr); err != nil {
		return fmt.Errorf("joining server %s: %w", addr, err)
	}

	c.logger.Info("login complete", slog.String("address", addr))
	return nil
}

// HandleGame runs the blocking packet read loop. Returns when the connection
// drops, an error occurs, or the context is cancelled.
func (c *Connection) HandleGame(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.client.HandleGame()
	}()

	select {
	case err := <-errCh:
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		if err != nil {
			return fmt.Errorf("game loop: %w", err)
		}
		return nil
	case <-ctx.Done():
		// Caller owns Close() — which will unblock the client.HandleGame goroutine.
		// errCh is buffered so the goroutine won't leak.
		return ctx.Err()
	}
}

// Close disconnects from the server. Safe to call when not connected.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	if c.client != nil && c.client.Conn != nil {
		c.client.Conn.Close()
	}
	return nil
}

// IsConnected returns the current connection state.
func (c *Connection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
