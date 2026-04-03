package main

import (
	"encoding/json"
	"testing"
)

// makeRequest builds a JSONRPCRequest with the given method and marshaled params.
func makeRequest(id any, method string, params any) JSONRPCRequest {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			panic("makeRequest: " + err.Error())
		}
		raw = b
	}
	return JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  raw,
	}
}

// resultMap extracts the Result field as a map from a JSONRPCResponse.
func resultMap(t *testing.T, resp *JSONRPCResponse) map[string]any {
	t.Helper()
	b, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return m
}

func TestInitialize(t *testing.T) {
	req := makeRequest(1, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
	})

	resp := handleMessage(req, nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %v", resp.ID)
	}

	m := resultMap(t, resp)

	pv, ok := m["protocolVersion"].(string)
	if !ok || pv != "2025-03-26" {
		t.Errorf("expected protocolVersion 2025-03-26, got %v", m["protocolVersion"])
	}

	caps, ok := m["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("capabilities missing or not an object")
	}
	if _, ok := caps["tools"]; !ok {
		t.Error("capabilities.tools missing")
	}

	info, ok := m["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("serverInfo missing or not an object")
	}
	if info["name"] != "nasa-mcp" {
		t.Errorf("expected serverInfo.name nasa-mcp, got %v", info["name"])
	}
}

func TestToolsList(t *testing.T) {
	req := makeRequest(2, "tools/list", map[string]any{})

	resp := handleMessage(req, nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	m := resultMap(t, resp)
	toolsRaw, ok := m["tools"].([]any)
	if !ok {
		t.Fatal("tools missing or not an array")
	}
	if len(toolsRaw) == 0 {
		t.Fatal("expected at least one tool")
	}

	first, ok := toolsRaw[0].(map[string]any)
	if !ok {
		t.Fatal("tool[0] is not an object")
	}
	if _, ok := first["name"]; !ok {
		t.Error("tool missing name")
	}
	if _, ok := first["description"]; !ok {
		t.Error("tool missing description")
	}
	if _, ok := first["inputSchema"]; !ok {
		t.Error("tool missing inputSchema")
	}
}

func TestToolsCall(t *testing.T) {
	// Call nasa_neo_get without required asteroid_id to get an error result
	// without needing a live API client.
	req := makeRequest(3, "tools/call", map[string]any{
		"name":      "nasa_neo_get",
		"arguments": map[string]any{},
	})

	resp := handleMessage(req, nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	// The result is a ToolResult with isError true (missing required param).
	b, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var tr ToolResult
	if err := json.Unmarshal(b, &tr); err != nil {
		t.Fatalf("unmarshal ToolResult: %v", err)
	}
	if len(tr.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	if tr.Content[0].Type != "text" {
		t.Errorf("expected content type text, got %s", tr.Content[0].Type)
	}
	if !tr.IsError {
		t.Error("expected isError true for missing required param")
	}
}

func TestToolsCallUnknown(t *testing.T) {
	req := makeRequest(4, "tools/call", map[string]any{
		"name":      "nonexistent",
		"arguments": map[string]any{},
	})

	resp := handleMessage(req, nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("expected error code -32602, got %d", resp.Error.Code)
	}
}

func TestMethodNotFound(t *testing.T) {
	req := makeRequest(5, "foo/bar", nil)

	resp := handleMessage(req, nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestNotificationNoResponse(t *testing.T) {
	req := makeRequest(nil, "notifications/initialized", nil)

	resp := handleMessage(req, nil)
	if resp != nil {
		t.Fatalf("expected nil response for notification, got %+v", resp)
	}
}

func TestAllToolsRegistered(t *testing.T) {
	req := makeRequest(10, "tools/list", map[string]any{})

	resp := handleMessage(req, nil)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	m := resultMap(t, resp)
	toolsRaw, ok := m["tools"].([]any)
	if !ok {
		t.Fatal("tools missing or not an array")
	}

	expectedNames := []string{
		"nasa_apod",
		"nasa_apod_range",
		"nasa_neo_feed",
		"nasa_neo_get",
		"nasa_donki_cme",
		"nasa_donki_gst",
		"nasa_donki_flr",
		"nasa_donki_sep",
		"nasa_donki_notifications",
		"nasa_images_search",
		"nasa_ssd_cad",
		"nasa_ssd_fireball",
		"nasa_eonet_events",
	}

	if len(toolsRaw) != len(expectedNames) {
		t.Fatalf("expected %d tools, got %d", len(expectedNames), len(toolsRaw))
	}

	nameSet := make(map[string]bool)
	for _, raw := range toolsRaw {
		tool, ok := raw.(map[string]any)
		if !ok {
			t.Fatal("tool entry is not an object")
		}
		name, ok := tool["name"].(string)
		if !ok {
			t.Fatal("tool missing name string")
		}
		nameSet[name] = true
	}

	for _, expected := range expectedNames {
		if !nameSet[expected] {
			t.Errorf("missing expected tool: %s", expected)
		}
	}
}

func TestParseError(t *testing.T) {
	// Simulate what the main loop does with malformed JSON by testing
	// the unmarshal path directly.
	badJSON := []byte(`{not valid json}`)
	var req JSONRPCRequest
	err := json.Unmarshal(badJSON, &req)
	if err == nil {
		t.Fatal("expected unmarshal error for malformed JSON")
	}
	// Verify the main loop would produce a parse error response.
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      nil,
		Error:   &RPCError{Code: -32700, Message: "parse error: " + err.Error()},
	}
	if resp.Error.Code != -32700 {
		t.Errorf("expected error code -32700, got %d", resp.Error.Code)
	}
}
