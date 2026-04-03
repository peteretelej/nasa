//go:build integration

package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildBinary compiles the MCP server binary to a temp directory.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "nasa-mcp")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/nasa-mcp/")
	cmd.Dir = filepath.Join("..", "..")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return binary
}

// sendMessage writes a JSON-RPC message to the server's stdin.
func sendMessage(w io.Writer, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// readResponse reads and parses a single JSON-RPC response line.
func readResponse(t *testing.T, scanner *bufio.Scanner) *JSONRPCResponse {
	t.Helper()
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			t.Fatalf("scanner error: %v", err)
		}
		t.Fatal("no more output from server")
	}
	var resp JSONRPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, scanner.Text())
	}
	return &resp
}

// loadAPIKey reads the NASA_API_KEY from env or from a .env file at the repo root.
func loadAPIKey(t *testing.T) string {
	t.Helper()
	if key := os.Getenv("NASA_API_KEY"); key != "" {
		return key
	}

	// Try reading ../../.env relative to cmd/nasa-mcp/.
	envPath := filepath.Join("..", "..", ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Skip("NASA_API_KEY not set and no .env file found; skipping integration test")
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Strip surrounding quotes if present.
		v = strings.Trim(v, `"'`)
		if k == "NASA_API_KEY" {
			os.Setenv("NASA_API_KEY", v)
			return v
		}
	}
	t.Skip("NASA_API_KEY not found in env or .env; skipping integration test")
	return ""
}

// startServer builds and starts the MCP binary, returning stdin/stdout handles
// and a cleanup function.
func startServer(t *testing.T, apiKey string) (io.WriteCloser, *bufio.Scanner, func()) {
	t.Helper()
	binary := buildBinary(t)

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(), "NASA_API_KEY="+apiKey)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}

	cleanup := func() {
		stdin.Close()
		// Give the process a moment to exit gracefully.
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			cmd.Process.Kill()
			<-done
		}
	}

	return stdin, scanner, cleanup
}

func TestIntegration_Initialize(t *testing.T) {
	apiKey := loadAPIKey(t)
	stdin, scanner, cleanup := startServer(t, apiKey)
	defer cleanup()

	err := sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "integration-test", "version": "1.0.0"},
		},
	})
	if err != nil {
		t.Fatalf("send initialize: %v", err)
	}

	resp := readResponse(t, scanner)
	if resp.Error != nil {
		t.Fatalf("initialize error: %+v", resp.Error)
	}
	if resp.ID != float64(1) {
		t.Errorf("expected ID 1, got %v", resp.ID)
	}
}

func TestIntegration_ToolsList(t *testing.T) {
	apiKey := loadAPIKey(t)
	stdin, scanner, cleanup := startServer(t, apiKey)
	defer cleanup()

	// Initialize first.
	sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	})
	readResponse(t, scanner)

	// Send initialized notification (no response expected).
	sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	// List tools.
	err := sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	if err != nil {
		t.Fatalf("send tools/list: %v", err)
	}

	resp := readResponse(t, scanner)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %+v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var m map[string]any
	json.Unmarshal(b, &m)
	toolsRaw, ok := m["tools"].([]any)
	if !ok {
		t.Fatal("tools missing or not array")
	}
	if len(toolsRaw) != 13 {
		t.Fatalf("expected 13 tools, got %d", len(toolsRaw))
	}
}

func TestIntegration_APOD(t *testing.T) {
	apiKey := loadAPIKey(t)
	stdin, scanner, cleanup := startServer(t, apiKey)
	defer cleanup()

	// Initialize.
	sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	})
	readResponse(t, scanner)

	// Call APOD (no args = today).
	err := sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "nasa_apod",
			"arguments": map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("send tools/call: %v", err)
	}

	resp := readResponse(t, scanner)
	if resp.Error != nil {
		t.Fatalf("tools/call error: %+v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var tr ToolResult
	json.Unmarshal(b, &tr)

	if tr.IsError {
		t.Fatalf("expected success, got error: %s", tr.Content[0].Text)
	}
	if len(tr.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	text := tr.Content[0].Text
	if !strings.Contains(text, "Title:") {
		t.Error("expected Title: in APOD output")
	}
	if !strings.Contains(text, "Date:") {
		t.Error("expected Date: in APOD output")
	}
	if !strings.Contains(text, "Explanation:") {
		t.Error("expected Explanation: in APOD output")
	}
}

func TestIntegration_NEOFeed(t *testing.T) {
	apiKey := loadAPIKey(t)
	stdin, scanner, cleanup := startServer(t, apiKey)
	defer cleanup()

	// Initialize.
	sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	})
	readResponse(t, scanner)

	// Rate limit: wait between live API calls.
	time.Sleep(2 * time.Second)

	// Call NEO feed for the last 7 days.
	now := time.Now()
	start := now.AddDate(0, 0, -7).Format("2006-01-02")

	err := sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "nasa_neo_feed",
			"arguments": map[string]any{
				"start_date": start,
			},
		},
	})
	if err != nil {
		t.Fatalf("send tools/call: %v", err)
	}

	resp := readResponse(t, scanner)
	if resp.Error != nil {
		t.Fatalf("tools/call error: %+v", resp.Error)
	}

	b, _ := json.Marshal(resp.Result)
	var tr ToolResult
	json.Unmarshal(b, &tr)

	if tr.IsError {
		t.Fatalf("expected success, got error: %s", tr.Content[0].Text)
	}

	text := tr.Content[0].Text
	if !strings.Contains(text, "Near-Earth Objects") {
		t.Error("expected 'Near-Earth Objects' in output")
	}
	if !strings.Contains(text, "Total:") {
		t.Error("expected Total: in output")
	}
}

func TestIntegration_UnknownTool(t *testing.T) {
	apiKey := loadAPIKey(t)
	stdin, scanner, cleanup := startServer(t, apiKey)
	defer cleanup()

	// Initialize.
	sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	})
	readResponse(t, scanner)

	// Call a non-existent tool.
	err := sendMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "nonexistent_tool",
			"arguments": map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("send tools/call: %v", err)
	}

	resp := readResponse(t, scanner)
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("expected error code -32602, got %d", resp.Error.Code)
	}
}
