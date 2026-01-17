package main

import (
	"encoding/json"

	"github.com/longregen/alicia/cmd/mcp-web/tools"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  any             `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// InitializeResult is the response to the initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ServerCapabilities describes the server's capabilities
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability describes tool-related capabilities
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ServerInfo describes the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsListResult is the response to tools/list
type ToolsListResult struct {
	Tools []tools.ToolDefinition `json:"tools"`
}

// ToolCallParams are the parameters for tools/call
type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolCallResult is the response to tools/call
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in tool results
type ContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}
