package connection

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"net"
	"sync"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/bot/basic"
	"github.com/Tnze/go-mc/bot/msg"
	"github.com/Tnze/go-mc/bot/playerlist"
	mcworld "github.com/Tnze/go-mc/bot/world"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/data/packetid"
	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/level/block"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/offline"
	"github.com/kaylincoded/clankercraft/internal/config"
	"github.com/kaylincoded/clankercraft/internal/engine"
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

// Position holds the bot's tracked location and rotation.
type Position struct {
	X, Y, Z    float64
	Yaw, Pitch float32
}

// BlockInfo represents a block at a specific position, returned by ScanArea.
type BlockInfo struct {
	Block   string
	X, Y, Z int
}

const (
	// MaxScanVolume is the maximum number of blocks that can be scanned at once.
	MaxScanVolume = 10000
	// MaxFindSigns is the maximum number of signs returned by FindSigns.
	MaxFindSigns = 50
)

// SignText holds the text content of a sign's front and back faces.
type SignText struct {
	FrontLines [4]string
	BackLines  [4]string
}

// SignInfo represents a sign found in the world with its text and position.
type SignInfo struct {
	Sign    SignText
	Block   string
	X, Y, Z int
}

// chatListener is a one-shot system chat message listener.
type chatListener struct {
	ch chan string
}

// DetectTimeout is how long to wait for a //version response before assuming vanilla.
const DetectTimeout = 3 * time.Second

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
	world   *mcworld.World

	authFn          AuthFunc
	connectAndRun   func(ctx context.Context) error   // injectable for testing RunWithReconnect
	backoffFn       func(attempt int) time.Duration    // injectable for testing backoff
	sendCommandFn   func(command string) error         // injectable for testing command dispatch
	shutdownTimeout time.Duration                      // injectable for testing Close drain

	mu             sync.Mutex
	state          ConnState
	pos            Position
	hasPos         bool             // true after first position update from server
	doneCh         chan struct{}     // closed when HandleGame goroutine exits
	tier           engine.Tier      // detected WorldEdit capability tier
	chatListeners  []chatListener   // one-shot system chat listeners
	selection      engine.Selection // current WorldEdit selection
	hasSelection   bool             // true after SetSelection is called
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
			go c.detectTier()
			return nil
		},
		Disconnect: func(reason chat.Message) error {
			c.setState(StateDisconnected)
			c.resetPosition()
			c.resetTier()
			c.resetSelection()
			c.logger.Warn("disconnected by server", slog.String("reason", reason.String()))
			return nil
		},
		Teleported: func(x, y, z float64, yaw, pitch float32, flags int32, teleportID int32) error {
			c.updatePosition(x, y, z, yaw, pitch, byte(flags))
			return c.player.AcceptTeleportation(pk.VarInt(teleportID))
		},
	})

	// Player list
	c.plist = playerlist.New(client)

	// World chunk storage — auto-loads chunks from server packets
	c.world = mcworld.NewWorld(client, c.player, mcworld.EventsListener{})

	// Chat message handling
	c.msgMgr = msg.New(client, c.player, c.plist, msg.EventsHandler{
		SystemChat: func(m chat.Message, overlay bool) error {
			text := m.ClearString()
			c.logger.Info("chat:system", slog.String("message", text), slog.Bool("overlay", overlay))
			c.dispatchChat(text)
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

// GetPosition returns the bot's tracked position and whether it has been set.
// Returns false if the server hasn't sent a position yet this session.
func (c *Connection) GetPosition() (Position, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pos, c.hasPos
}

// SendRotation sends a rotation packet to the server and updates tracked rotation.
func (c *Connection) SendRotation(yaw, pitch float32) error {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil || client.Conn == nil {
		return fmt.Errorf("not connected")
	}
	if err := client.Conn.WritePacket(pk.Marshal(
		packetid.ServerboundMovePlayerRot,
		pk.Float(yaw),
		pk.Float(pitch),
		pk.Boolean(true), // onGround
	)); err != nil {
		return fmt.Errorf("sending rotation: %w", err)
	}

	c.mu.Lock()
	c.pos.Yaw = yaw
	c.pos.Pitch = pitch
	c.mu.Unlock()
	return nil
}

// updatePosition applies a server position update, handling relative vs absolute flags.
// Flags bitfield: bit 0=X, 1=Y, 2=Z, 3=Yaw, 4=Pitch. If set, value is relative.
func (c *Connection) updatePosition(x, y, z float64, yaw, pitch float32, flags byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if flags&0x01 != 0 {
		c.pos.X += x
	} else {
		c.pos.X = x
	}
	if flags&0x02 != 0 {
		c.pos.Y += y
	} else {
		c.pos.Y = y
	}
	if flags&0x04 != 0 {
		c.pos.Z += z
	} else {
		c.pos.Z = z
	}
	if flags&0x08 != 0 {
		c.pos.Yaw += yaw
	} else {
		c.pos.Yaw = yaw
	}
	if flags&0x10 != 0 {
		c.pos.Pitch += pitch
	} else {
		c.pos.Pitch = pitch
	}
	c.hasPos = true

	c.logger.Debug("position updated",
		slog.Float64("x", c.pos.X),
		slog.Float64("y", c.pos.Y),
		slog.Float64("z", c.pos.Z),
		slog.Any("yaw", c.pos.Yaw),
		slog.Any("pitch", c.pos.Pitch),
	)
}

// resetPosition clears the tracked position (called on disconnect).
func (c *Connection) resetPosition() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pos = Position{}
	c.hasPos = false
}

// BlockAt returns the block name at the given world coordinates.
// Returns an error if the chunk is not loaded or coordinates are out of range.
func (c *Connection) BlockAt(x, y, z int) (string, error) {
	c.mu.Lock()
	w := c.world
	client := c.client
	player := c.player
	c.mu.Unlock()

	if w == nil || client == nil {
		return "", fmt.Errorf("not connected")
	}

	chunkPos := level.ChunkPos{int32(x >> 4), int32(z >> 4)}
	chunk, ok := w.Columns[chunkPos]
	if !ok {
		return "", fmt.Errorf("chunk at (%d, %d) not loaded", chunkPos[0], chunkPos[1])
	}

	dimType := client.Registries.DimensionType.GetByID(player.DimensionType)
	if dimType == nil {
		return "", fmt.Errorf("unknown dimension type %d", player.DimensionType)
	}
	minY := int(dimType.MinY)

	sectionIdx := (y - minY) >> 4
	if sectionIdx < 0 || sectionIdx >= len(chunk.Sections) {
		return "", fmt.Errorf("y=%d out of range (minY=%d, sections=%d)", y, minY, len(chunk.Sections))
	}

	section := &chunk.Sections[sectionIdx]
	blockIdx := ((y & 0xF) << 8) | ((z & 0xF) << 4) | (x & 0xF)
	stateID := section.GetBlock(blockIdx)

	if int(stateID) >= len(block.StateList) {
		return "", fmt.Errorf("unknown block state %d", stateID)
	}
	return block.StateList[stateID].ID(), nil
}

// FindBlock searches loaded chunks for the nearest block of the given type
// within maxDist blocks of the bot's position. Returns coordinates and whether
// a match was found.
func (c *Connection) FindBlock(blockType string, maxDist int) (bx, by, bz int, found bool, err error) {
	c.mu.Lock()
	w := c.world
	pos := c.pos
	hasPos := c.hasPos
	client := c.client
	player := c.player
	c.mu.Unlock()

	if w == nil || client == nil {
		return 0, 0, 0, false, fmt.Errorf("not connected")
	}
	if !hasPos {
		return 0, 0, 0, false, fmt.Errorf("position not yet known")
	}

	dimType := client.Registries.DimensionType.GetByID(player.DimensionType)
	if dimType == nil {
		return 0, 0, 0, false, fmt.Errorf("unknown dimension type")
	}
	minY := int(dimType.MinY)

	// Cap max distance to limit scan
	if maxDist > 64 {
		maxDist = 64
	}

	botX, botY, botZ := int(pos.X), int(pos.Y), int(pos.Z)
	chunkRadius := (maxDist >> 4) + 1
	botCX, botCZ := int32(botX>>4), int32(botZ>>4)

	bestDistSq := int64(maxDist+1) * int64(maxDist+1)
	found = false

	for cx := botCX - int32(chunkRadius); cx <= botCX+int32(chunkRadius); cx++ {
		for cz := botCZ - int32(chunkRadius); cz <= botCZ+int32(chunkRadius); cz++ {
			chunk, ok := w.Columns[level.ChunkPos{cx, cz}]
			if !ok {
				continue
			}

			for si, section := range chunk.Sections {
				if section.BlockCount == 0 {
					continue
				}
				sectionY := minY + si*16

				for i := 0; i < 16*16*16; i++ {
					stateID := section.GetBlock(i)
					if block.IsAir(stateID) {
						continue
					}
					if int(stateID) >= len(block.StateList) {
						continue
					}
					if block.StateList[stateID].ID() != blockType {
						continue
					}

					// Decode block position from index
					wy := sectionY + (i >> 8)
					wz := int(cz)*16 + ((i >> 4) & 0xF)
					wx := int(cx)*16 + (i & 0xF)

					dx := int64(wx - botX)
					dy := int64(wy - botY)
					dz := int64(wz - botZ)
					distSq := dx*dx + dy*dy + dz*dz

					if distSq < bestDistSq {
						bestDistSq = distSq
						bx, by, bz = wx, wy, wz
						found = true
					}
				}
			}
		}
	}

	return bx, by, bz, found, nil
}

// ScanArea scans a rectangular region and returns all non-air blocks.
// Returns an error if the region exceeds MaxScanVolume blocks.
// Blocks in unloaded chunks are silently skipped.
func (c *Connection) ScanArea(x1, y1, z1, x2, y2, z2 int) ([]BlockInfo, error) {
	// Normalize corners
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if z1 > z2 {
		z1, z2 = z2, z1
	}

	dx := x2 - x1 + 1
	dy := y2 - y1 + 1
	dz := z2 - z1 + 1
	volume := dx * dy * dz
	if volume > MaxScanVolume {
		return nil, fmt.Errorf("region too large: %d blocks (max %d)", volume, MaxScanVolume)
	}

	var blocks []BlockInfo
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
			for z := z1; z <= z2; z++ {
				name, err := c.BlockAt(x, y, z)
				if err != nil {
					continue // skip unloaded chunks
				}
				if name == "minecraft:air" || name == "minecraft:cave_air" || name == "minecraft:void_air" {
					continue
				}
				blocks = append(blocks, BlockInfo{Block: name, X: x, Y: y, Z: z})
			}
		}
	}

	return blocks, nil
}

// signNBT represents the NBT structure of a sign block entity (1.20+).
type signNBT struct {
	FrontText signSideNBT `nbt:"front_text"`
	BackText  signSideNBT `nbt:"back_text"`
}

type signSideNBT struct {
	Messages []string `nbt:"messages"`
}

// signEntityType and hangingSignEntityType are the block entity type IDs for signs.
var (
	signEntityType        = block.EntityTypes["minecraft:sign"]
	hangingSignEntityType = block.EntityTypes["minecraft:hanging_sign"]
)

// parseSignMessages extracts plain text from sign message JSON text components.
// Handles standard JSON text components ({"text":"..."}) and plain JSON strings ("...").
// Falls back to empty string on unparseable data rather than exposing raw JSON.
func parseSignMessages(messages []string) [4]string {
	var lines [4]string
	for i := 0; i < 4 && i < len(messages); i++ {
		if messages[i] == "" {
			continue
		}
		var msg chat.Message
		if err := json.Unmarshal([]byte(messages[i]), &msg); err != nil {
			// Try as a plain JSON string literal (e.g., "Hello")
			var plain string
			if err2 := json.Unmarshal([]byte(messages[i]), &plain); err2 == nil {
				lines[i] = plain
			}
			// Otherwise leave as empty — don't expose raw JSON to callers
			continue
		}
		lines[i] = msg.ClearString()
	}
	return lines
}

// ReadSign reads the text of a sign block entity at the given coordinates.
// Returns the sign text and block type name.
// Returns an error if no sign exists at those coordinates or the chunk is not loaded.
func (c *Connection) ReadSign(x, y, z int) (SignText, string, error) {
	c.mu.Lock()
	w := c.world
	c.mu.Unlock()

	if w == nil {
		return SignText{}, "", fmt.Errorf("not connected")
	}

	chunkPos := level.ChunkPos{int32(x >> 4), int32(z >> 4)}
	chunk, ok := w.Columns[chunkPos]
	if !ok {
		return SignText{}, "", fmt.Errorf("chunk at (%d, %d) not loaded", chunkPos[0], chunkPos[1])
	}

	localX := x & 0xF
	localZ := z & 0xF

	for _, be := range chunk.BlockEntity {
		if be.Type != signEntityType && be.Type != hangingSignEntityType {
			continue
		}
		beX, beZ := be.UnpackXZ()
		if beX != localX || beZ != localZ || int(be.Y) != y {
			continue
		}

		var data signNBT
		if err := be.Data.Unmarshal(&data); err != nil {
			return SignText{}, "", fmt.Errorf("failed to read sign NBT: %w", err)
		}

		blockName, _ := c.BlockAt(x, y, z)

		return SignText{
			FrontLines: parseSignMessages(data.FrontText.Messages),
			BackLines:  parseSignMessages(data.BackText.Messages),
		}, blockName, nil
	}

	return SignText{}, "", fmt.Errorf("no sign at (%d, %d, %d)", x, y, z)
}

// signWithDist is used internally for distance-sorted sign results.
type signWithDist struct {
	info   SignInfo
	distSq int64
}

// FindSigns searches loaded chunks within maxDist blocks of the bot for sign block entities.
// Returns up to MaxFindSigns results, sorted nearest first.
func (c *Connection) FindSigns(maxDist int) ([]SignInfo, error) {
	c.mu.Lock()
	w := c.world
	pos := c.pos
	hasPos := c.hasPos
	c.mu.Unlock()

	if w == nil {
		return nil, fmt.Errorf("not connected")
	}
	if !hasPos {
		return nil, fmt.Errorf("position not yet known")
	}

	if maxDist <= 0 {
		maxDist = 16
	}
	if maxDist > 64 {
		maxDist = 64
	}

	botX, botY, botZ := int(pos.X), int(pos.Y), int(pos.Z)
	chunkRadius := (maxDist >> 4) + 1
	botCX, botCZ := int32(botX>>4), int32(botZ>>4)
	maxDistSq := int64(maxDist) * int64(maxDist)

	var found []signWithDist
	for cx := botCX - int32(chunkRadius); cx <= botCX+int32(chunkRadius); cx++ {
		for cz := botCZ - int32(chunkRadius); cz <= botCZ+int32(chunkRadius); cz++ {
			chunk, ok := w.Columns[level.ChunkPos{cx, cz}]
			if !ok {
				continue
			}

			for _, be := range chunk.BlockEntity {
				if be.Type != signEntityType && be.Type != hangingSignEntityType {
					continue
				}

				beX, beZ := be.UnpackXZ()
				wx := int(cx)*16 + beX
				wy := int(be.Y)
				wz := int(cz)*16 + beZ

				dx := int64(wx - botX)
				dy := int64(wy - botY)
				dz := int64(wz - botZ)
				distSq := dx*dx + dy*dy + dz*dz
				if distSq > maxDistSq {
					continue
				}

				var data signNBT
				if err := be.Data.Unmarshal(&data); err != nil {
					continue // skip unreadable signs
				}

				blockName, _ := c.BlockAt(wx, wy, wz)

				found = append(found, signWithDist{
					info: SignInfo{
						Sign: SignText{
							FrontLines: parseSignMessages(data.FrontText.Messages),
							BackLines:  parseSignMessages(data.BackText.Messages),
						},
						Block: blockName,
						X:     wx,
						Y:     wy,
						Z:     wz,
					},
					distSq: distSq,
				})
			}
		}
	}

	sort.Slice(found, func(i, j int) bool {
		return found[i].distSq < found[j].distSq
	})

	limit := len(found)
	if limit > MaxFindSigns {
		limit = MaxFindSigns
	}

	signs := make([]SignInfo, limit)
	for i := 0; i < limit; i++ {
		signs[i] = found[i].info
	}

	return signs, nil
}

// gamemodeNames maps gamemode byte values to human-readable strings.
var gamemodeNames = [4]string{"survival", "creative", "adventure", "spectator"}

// GetGamemode returns the bot's current game mode as a string.
func (c *Connection) GetGamemode() string {
	c.mu.Lock()
	player := c.player
	c.mu.Unlock()

	if player == nil {
		return "unknown"
	}

	gm := player.Gamemode
	if int(gm) < len(gamemodeNames) {
		return gamemodeNames[gm]
	}
	return fmt.Sprintf("unknown(%d)", gm)
}

// dispatchChat sends a system chat message to all registered listeners.
func (c *Connection) dispatchChat(text string) {
	c.mu.Lock()
	listeners := make([]chatListener, len(c.chatListeners))
	copy(listeners, c.chatListeners)
	c.mu.Unlock()

	for _, l := range listeners {
		select {
		case l.ch <- text:
		default:
		}
	}
}

// listenChat registers a listener that receives system chat messages.
// All system chat messages are broadcast to all listeners; callers should
// filter for relevant content. Call unlistenChat when done to avoid leaking.
func (c *Connection) listenChat() chan string {
	ch := make(chan string, 32)
	c.mu.Lock()
	c.chatListeners = append(c.chatListeners, chatListener{ch: ch})
	c.mu.Unlock()
	return ch
}

// unlistenChat removes a previously registered chat listener.
func (c *Connection) unlistenChat(ch chan string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, l := range c.chatListeners {
		if l.ch == ch {
			c.chatListeners = append(c.chatListeners[:i], c.chatListeners[i+1:]...)
			return
		}
	}
}

// SendCommand sends a chat command to the server. The command string is sent
// via the ServerboundChatCommand packet, which the server interprets with an
// implicit leading /. So SendCommand("version") sends /version, and
// SendCommand("/version") sends //version (used for WorldEdit commands).
func (c *Connection) SendCommand(command string) error {
	c.mu.Lock()
	fn := c.sendCommandFn
	mgr := c.msgMgr
	c.mu.Unlock()

	if fn != nil {
		return fn(command)
	}
	if mgr == nil {
		return fmt.Errorf("not connected")
	}
	return mgr.SendCommand(command)
}

// GetTier returns the detected WorldEdit capability tier.
func (c *Connection) GetTier() engine.Tier {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tier
}

// resetTier clears the detected tier (called on disconnect).
func (c *Connection) resetTier() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tier = engine.TierUnknown
}

// SetSelection sets the WorldEdit selection to the given coordinates.
// If the server has WorldEdit or FAWE, it also sends //pos1 and //pos2 commands.
func (c *Connection) SetSelection(x1, y1, z1, x2, y2, z2 int) error {
	c.mu.Lock()
	c.selection = engine.Selection{X1: x1, Y1: y1, Z1: z1, X2: x2, Y2: y2, Z2: z2}
	c.hasSelection = true
	tier := c.tier
	c.mu.Unlock()

	if tier == engine.TierWorldEdit || tier == engine.TierFAWE {
		if err := c.SendCommand(fmt.Sprintf("/pos1 %d,%d,%d", x1, y1, z1)); err != nil {
			return fmt.Errorf("sending //pos1: %w", err)
		}
		if err := c.SendCommand(fmt.Sprintf("/pos2 %d,%d,%d", x2, y2, z2)); err != nil {
			return fmt.Errorf("sending //pos2: %w", err)
		}
	}

	return nil
}

// GetSelection returns the current selection and whether one has been set.
func (c *Connection) GetSelection() (engine.Selection, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.selection, c.hasSelection
}

// resetSelection clears the selection (called on disconnect).
func (c *Connection) resetSelection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.selection = engine.Selection{}
	c.hasSelection = false
}

// parseTierFromChat checks if a chat message indicates a WorldEdit tier.
// Returns the detected tier and true if a match was found, or TierUnknown and false otherwise.
func parseTierFromChat(msg string) (engine.Tier, bool) {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "fastasyncworldedit") || strings.Contains(lower, "fawe") {
		return engine.TierFAWE, true
	}
	if strings.Contains(lower, "worldedit") {
		return engine.TierWorldEdit, true
	}
	return engine.TierUnknown, false
}

// detectTier sends //version and parses the response to determine the WorldEdit tier.
func (c *Connection) detectTier() {
	ch := c.listenChat()
	defer c.unlistenChat(ch)

	if err := c.SendCommand("/version"); err != nil {
		c.logger.Warn("failed to send //version for tier detection", slog.String("error", err.Error()))
		c.mu.Lock()
		c.tier = engine.TierVanilla
		c.mu.Unlock()
		return
	}

	timeout := time.After(DetectTimeout)

	for {
		select {
		case msg := <-ch:
			if tier, ok := parseTierFromChat(msg); ok {
				c.mu.Lock()
				c.tier = tier
				c.mu.Unlock()
				c.logger.Info("worldedit tier detected", slog.String("tier", tier.String()))
				return
			}
			continue
		case <-timeout:
			c.mu.Lock()
			c.tier = engine.TierVanilla
			c.mu.Unlock()
			c.logger.Info("worldedit tier detected", slog.String("tier", engine.TierVanilla.String()))
			return
		}
	}
}
