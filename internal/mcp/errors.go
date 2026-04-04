package mcp

import (
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolError returns an MCP error result with the given message.
// MCP clients display this text to the user/LLM.
func toolError(msg string) *gomcp.CallToolResult {
	return &gomcp.CallToolResult{
		IsError: true,
		Content: []gomcp.Content{&gomcp.TextContent{Text: msg}},
	}
}
