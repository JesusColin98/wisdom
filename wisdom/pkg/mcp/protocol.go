package mcp

import "encoding/json"

// JSON-RPC Constants
const (
	Version = "2.0"
)

// Request represents a standard MCP/JSON-RPC request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a standard MCP/JSON-RPC response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error object.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ToolListResponse is the result for tools/list.
type ToolListResponse struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest defines params for tools/call.
type CallToolRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallToolResponse is the result for tools/call.
type CallToolResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents a piece of content in a tool response.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// InitializeRequest represents the first message from client.
type InitializeRequest struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo `json:"clientInfo"`
}

// ClientInfo metadata about the client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResponse result for initialize.
type InitializeResponse struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

// ServerInfo metadata about the server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
