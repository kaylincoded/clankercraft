package mcp

import (
	"context"
	"log/slog"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP SDK server with clankercraft-specific configuration.
type Server struct {
	server *gomcp.Server
	logger *slog.Logger
	conn   ConnChecker
}

// pingInput is the input schema for the ping tool (no arguments).
type pingInput struct{}

// pingOutput is the output schema for the ping tool.
type pingOutput struct {
	Status string `json:"status"`
}

// statusInput is the input schema for the status tool (no arguments).
type statusInput struct{}

// statusOutput is the output schema for the status tool.
type statusOutput struct {
	Connected bool   `json:"connected"`
	Status    string `json:"status"`
}

// New creates a configured MCP server with registered tools.
func New(version string, logger *slog.Logger, conn ConnChecker) *Server {
	s := gomcp.NewServer(
		&gomcp.Implementation{
			Name:    "clankercraft",
			Version: version,
		},
		&gomcp.ServerOptions{
			Logger: logger,
		},
	)

	srv := &Server{
		server: s,
		logger: logger,
		conn:   conn,
	}

	srv.registerTools()
	return srv
}

// registerTools registers all MCP tools on the server.
func (s *Server) registerTools() {
	// ping — no connection required (MCP transport health check)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "ping",
		Description: "Check if the bot is responsive",
	}, s.handlePing)

	// status — requires connection (demonstrates middleware pattern)
	gomcp.AddTool(s.server, &gomcp.Tool{
		Name:        "status",
		Description: "Get the bot's connection status to the Minecraft server",
	}, requireConnection(s.conn, s.handleStatus))
}

// handlePing is a smoke-test tool that returns "pong".
func (s *Server) handlePing(_ context.Context, _ *gomcp.CallToolRequest, _ pingInput) (*gomcp.CallToolResult, pingOutput, error) {
	return nil, pingOutput{Status: "pong"}, nil
}

// handleStatus returns the bot's connection status. Only reachable when connected (middleware enforced).
func (s *Server) handleStatus(_ context.Context, _ *gomcp.CallToolRequest, _ statusInput) (*gomcp.CallToolResult, statusOutput, error) {
	return nil, statusOutput{Connected: true, Status: "connected"}, nil
}

// Run starts the MCP stdio transport. Blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting MCP server")
	return s.server.Run(ctx, &gomcp.StdioTransport{})
}
