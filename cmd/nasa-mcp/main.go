// nasa-mcp is a Model Context Protocol (MCP) server that exposes NASA APIs
// as tools for AI assistants. It communicates over stdio using JSON-RPC 2.0,
// reading one JSON message per line from stdin and writing responses to stdout.
// Diagnostic output goes to stderr.
//
// The server supports the MCP lifecycle (initialize, tools/list, tools/call)
// and registers 13 tools covering APOD, NeoWs, DONKI, NASA Image Library,
// SSD/CNEOS, and EONET APIs.
//
// Set the NASA_API_KEY environment variable to use a registered API key.
// Without it, the server falls back to DEMO_KEY with reduced rate limits.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/peteretelej/nasa"
)

// JSON-RPC 2.0 types for the MCP protocol.

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request or notification.
// The ID field is omitted for notifications (messages that expect no response).
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
// Exactly one of Result or Error is set per the JSON-RPC spec.
type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object returned for protocol-level
// failures such as parse errors (-32700), invalid requests (-32600), or
// unknown methods (-32601).
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool describes an MCP tool for the tools/list response. The InputSchema
// field is a JSON Schema object defining the tool's accepted parameters.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// Content represents a content block in a tool result. Currently only
// text content is used, but the MCP spec also supports image and resource types.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolResult is the result returned by a tool handler. When IsError is true,
// the content describes what went wrong (e.g. invalid arguments, API failure).
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// ToolDef pairs a tool definition with its handler function. The tools slice
// in tools.go is the single source of truth for tool registration; it is used
// by both tools/list (to report available tools) and tools/call (to dispatch).
type ToolDef struct {
	Tool    Tool
	Handler func(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult
}

// handleMessage is the core dispatch function for the MCP server. It routes
// JSON-RPC requests to the appropriate handler based on the method field:
// "initialize" for the MCP handshake, "tools/list" for tool discovery, and
// "tools/call" for tool execution. Returns nil for notifications (messages
// with no ID that expect no response).
func handleMessage(req JSONRPCRequest, client *nasa.Client) *JSONRPCResponse {
	// Per JSON-RPC 2.0, notifications have no ID and expect no response.
	if req.ID == nil {
		return nil
	}

	switch req.Method {
	case "initialize":
		return handleInitialize(req)
	case "tools/list":
		return handleToolsList(req)
	case "tools/call":
		return handleToolsCall(req, client)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: "method not found: " + req.Method},
		}
	}
}

// handleInitialize responds with the server's protocol version, capabilities,
// and server info per the MCP specification.
func handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "nasa-mcp",
				"version": "1.0.0",
			},
		},
	}
}

// handleToolsList returns the list of available tools and their input schemas
// in the format expected by the MCP tools/list method.
func handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	list := make([]Tool, len(tools))
	for i, td := range tools {
		list[i] = td.Tool
	}
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": list,
		},
	}
}

// handleToolsCall looks up the named tool in the tools registry and calls its
// handler with the provided arguments. Returns a JSON-RPC error if the tool
// name is not found or the params are malformed.
func handleToolsCall(req JSONRPCRequest, client *nasa.Client) *JSONRPCResponse {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "invalid params: " + err.Error()},
		}
	}

	for _, td := range tools {
		if td.Tool.Name == params.Name {
			result := td.Handler(context.Background(), params.Arguments, client)
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  result,
			}
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Error:   &RPCError{Code: -32602, Message: "unknown tool: " + params.Name},
	}
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	encoder := json.NewEncoder(os.Stdout)

	// Create NASA client. If NASA_API_KEY is empty, the SDK falls back to
	// DEMO_KEY via its own env-var resolution.
	var opts []nasa.Option
	if key := os.Getenv("NASA_API_KEY"); key != "" {
		opts = append(opts, nasa.WithAPIKey(key))
	}
	client := nasa.NewClient(opts...)

	for scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			resp := &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: -32700, Message: "parse error: " + err.Error()},
			}
			if err := encoder.Encode(resp); err != nil {
				log.Printf("encode error: %v", err)
			}
			continue
		}

		resp := handleMessage(req, client)
		if resp == nil {
			continue
		}
		if err := encoder.Encode(resp); err != nil {
			log.Printf("encode error: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "stdin read error: %v\n", err)
		os.Exit(1)
	}
}
