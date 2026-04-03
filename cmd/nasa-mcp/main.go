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
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool describes an MCP tool for the tools/list response.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// Content represents a content block in a tool result.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ToolResult is the result returned by a tool handler.
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// ToolDef pairs a tool definition with its handler function.
type ToolDef struct {
	Tool    Tool
	Handler func(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult
}

// handleMessage dispatches a JSON-RPC request to the appropriate handler.
// Returns nil for notifications (no response expected).
func handleMessage(req JSONRPCRequest, client *nasa.Client) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return handleInitialize(req)
	case "notifications/initialized":
		return nil
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

// handleToolsList returns the list of available tools.
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

// handleToolsCall looks up the named tool and calls its handler.
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
