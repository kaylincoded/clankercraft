package connection

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/bot/basic"
	"github.com/Tnze/go-mc/bot/msg"
	"github.com/Tnze/go-mc/bot/playerlist"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/offline"
	"github.com/kaylincoded/clankercraft/internal/config"
)

// ConnState represents the connection state machine states.
type ConnState int

const (
	// StateDisconnected is the initial and post-disconnect state.
	StateDisconnected ConnState = iota
	// StateConnecting means a connection attempt is in progress.
	StateConnecting
	// StateConnected means the bot is logged in and handling game packets.
	StateConnected
)

// String returns the state name.
func (s ConnState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	default:
		return "unknown"
	}
}

const (
	// MaxReconnectAttempts is the maximum number of consecutive reconnect failures before giving up.
	MaxReconnectAttempts = 5
	// MaxBackoff is the maximum backoff duration between reconnect attempts.
	MaxBackoff = 30 * time.Second
	// ShutdownTimeout is the max time to wait for the game loop goroutine to drain during Close().
	ShutdownTimeout = 5 * time.Second
)

// AuthFunc is the function signature for MSA authentication.
type AuthFunc func(cfg *config.Config, logger *slog.Logger) (*bot.Auth, error)

// Connection manages the Minecraft server connection via go-mc.
type Connection struct {
	cfg    *config.Config
	logger *slog.Logger

	client  *bot.Client
	player  *basic.Player
	msgMgr  *msg.Manager
	plist   *playerlist.PlayerList

	authFn          AuthFunc
	connectAndRun   func(ctx context.Context) error   // injectable for testing RunWithReconnect
	backoffFn       func(attempt int) time.Duration    // injectable for testing backoff
	shutdownTimeout time.Duration                      // injectable for testing Close drain

	mu     sync.Mutex
	state  ConnState
	doneCh chan struct{} // closed when HandleGame goroutine exits
}

// New creates a new Connection configured from the given config.
func New(cfg *config.Config, logger *slog.Logger) *Connection {
	return &Connection{
		cfg:             cfg,
		logger:          logger,
		authFn:          Authenticate,
		shutdownTimeout: ShutdownTimeout,
	}
}

// Address returns the "host:port" string for the server.
func (c *Connection) Address() string {
	return net.JoinHostPort(c.cfg.Host, fmt.Sprintf("%d", c.cfg.Port))
}

// setupAuth configures authentication on the go-mc client.
func (c *Connection) setupAuth(client *bot.Client) error {
	if c.cfg.Offline {
		client.Auth.Name = c.cfg.Username
		id := offline.NameToUUID(c.cfg.Username)
		client.Auth.UUID = hex.EncodeToString(id[:])
		c.logger.Info("connecting in offline mode", slog.String("username", c.cfg.Username))
		return nil
	}

	auth, err := c.authFn(c.cfg, c.logger)
	if err != nil {
		return err
	}
	client.Auth = *auth
	return nil
}

// Connect joins the Minecraft server. Blocks until login+configuration is complete or fails.
func (c *Connection) Connect(ctx context.Context) error {
	c.setState(StateConnecting)

	client := bot.NewClient()
	if err := c.setupAuth(client); err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("authentication: %w", err)
	}
	c.client = client

	// basic.Player — handles keepalive, spawn, teleport acceptance
	c.player = basic.NewPlayer(client, basic.Settings{}, basic.EventsListener{
		GameStart: func() error {
			c.setState(StateConnected)
			c.logger.Info("spawned in world", slog.String("server", c.Address()))
			return nil
		},
		Disconnect: func(reason chat.Message) error {
			c.setState(StateDisconnected)
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
		c.setState(StateDisconnected)
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

	doneCh := make(chan struct{})
	c.mu.Lock()
	c.doneCh = doneCh
	c.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		defer close(doneCh)
		errCh <- c.client.HandleGame()
	}()

	select {
	case err := <-errCh:
		c.setState(StateDisconnected)
		if err != nil {
			return fmt.Errorf("game loop: %w", err)
		}
		return nil
	case <-ctx.Done():
		// Caller owns Close() — which will close TCP and drain the goroutine.
		return ctx.Err()
	}
}

// RunWithReconnect runs Connect+HandleGame in a loop with exponential backoff.
// On disconnect, it retries up to MaxReconnectAttempts times. On successful
// connection, the retry counter resets. Returns on context cancellation or
// after max retries are exhausted.
func (c *Connection) RunWithReconnect(ctx context.Context) error {
	runFn := c.defaultConnectAndRun
	if c.connectAndRun != nil {
		runFn = c.connectAndRun
	}

	var attempt int
	for {
		if err := runFn(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			attempt++
			if attempt >= MaxReconnectAttempts {
				return fmt.Errorf("max reconnect attempts (%d) exceeded: %w", MaxReconnectAttempts, err)
			}
			bf := backoffDuration
			if c.backoffFn != nil {
				bf = c.backoffFn
			}
			backoff := bf(attempt - 1)
			c.logger.Warn("connection failed, retrying",
				slog.Int("attempt", attempt),
				slog.Int("max_attempts", MaxReconnectAttempts),
				slog.Duration("backoff", backoff),
				slog.String("error", err.Error()),
			)
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Connected successfully, game loop ran and returned nil (clean disconnect)
		// Reset attempt counter and reconnect
		attempt = 0
		c.logger.Warn("connection lost, will reconnect")
	}
}

// defaultConnectAndRun is the production connect+run implementation.
func (c *Connection) defaultConnectAndRun(ctx context.Context) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	gameErr := c.HandleGame(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if gameErr != nil {
		return gameErr
	}
	return nil
}

// backoffDuration calculates exponential backoff: 1s, 2s, 4s, 8s, 16s, cap 30s.
func backoffDuration(attempt int) time.Duration {
	d := time.Second << uint(attempt)
	if d > MaxBackoff {
		d = MaxBackoff
	}
	return d
}

// Close disconnects from the server and waits for the HandleGame goroutine
// to drain (with timeout). Safe to call when not connected.
func (c *Connection) Close() error {
	c.mu.Lock()
	c.state = StateDisconnected
	client := c.client
	doneCh := c.doneCh
	c.mu.Unlock()

	if client != nil && client.Conn != nil {
		client.Conn.Close()
	}

	// Wait for HandleGame goroutine to finish
	if doneCh != nil {
		select {
		case <-doneCh:
		case <-time.After(c.shutdownTimeout):
			c.logger.Warn("shutdown timeout waiting for game loop to exit")
		}
	}

	return nil
}

// setState transitions the connection to a new state and logs the change.
func (c *Connection) setState(new ConnState) {
	c.mu.Lock()
	old := c.state
	c.state = new
	c.mu.Unlock()
	if old != new {
		c.logger.Info("connection state changed",
			slog.String("from", old.String()),
			slog.String("to", new.String()),
		)
	}
}

// State returns the current connection state.
func (c *Connection) State() ConnState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// IsConnected returns true if the connection state is StateConnected.
func (c *Connection) IsConnected() bool {
	return c.State() == StateConnected
}
