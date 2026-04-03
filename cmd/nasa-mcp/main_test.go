package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/peteretelej/nasa"
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

// toolResultFromResp extracts a ToolResult from a JSONRPCResponse.
func toolResultFromResp(t *testing.T, resp *JSONRPCResponse) ToolResult {
	t.Helper()
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %+v", resp.Error)
	}
	b, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var tr ToolResult
	if err := json.Unmarshal(b, &tr); err != nil {
		t.Fatalf("unmarshal ToolResult: %v", err)
	}
	return tr
}

// toolText returns the text from a successful (non-error) tool call.
func toolText(t *testing.T, resp *JSONRPCResponse) string {
	t.Helper()
	tr := toolResultFromResp(t, resp)
	if tr.IsError {
		t.Fatalf("expected non-error result, got isError with: %s", tr.Content[0].Text)
	}
	if len(tr.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	return tr.Content[0].Text
}

// assertToolError asserts the response is a tool-level error containing substr.
func assertToolError(t *testing.T, resp *JSONRPCResponse, substr string) {
	t.Helper()
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("expected tool-level error, got RPC error: %+v", resp.Error)
	}
	tr := toolResultFromResp(t, resp)
	if !tr.IsError {
		t.Fatalf("expected isError true, got false; text: %s", tr.Content[0].Text)
	}
	if substr != "" && !strings.Contains(tr.Content[0].Text, substr) {
		t.Errorf("expected error containing %q, got %q", substr, tr.Content[0].Text)
	}
}

// --- Fixture-backed test client ---

// redirectTransport is an http.RoundTripper that rewrites all requests to
// point at a local httptest server, preserving the original path and query.
// This lets us mock all NASA services (api.nasa.gov, images-api.nasa.gov,
// ssd-api.jpl.nasa.gov, eonet.gsfc.nasa.gov) through a single test server.
type redirectTransport struct {
	target string // e.g. "http://127.0.0.1:12345"
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite scheme+host to point at test server, keep path+query.
	newURL := rt.target + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}

// fixtureHandler serves JSON fixture files from testdata/ based on request path.
func fixtureHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Route to fixture files based on API path.
	fixtures := map[string]string{
		// APOD: single vs range/count is determined by query params.
		"/planetary/apod":      "", // handled below
		"/neo/rest/v1/feed":    "../../testdata/neo_feed.json",
		"/DONKI/CME":           "../../testdata/donki_cme.json",
		"/DONKI/GST":           "../../testdata/donki_gst.json",
		"/DONKI/FLR":           "../../testdata/donki_flr.json",
		"/DONKI/SEP":           "../../testdata/donki_sep.json",
		"/DONKI/notifications": "../../testdata/donki_notifications.json",
		"/search":              "../../testdata/images_search.json",
		"/cad.api":             "../../testdata/ssd_cad.json",
		"/fireball.api":        "../../testdata/ssd_fireball.json",
		"/events":              "../../testdata/eonet_events.json",
	}

	// APOD: choose fixture based on query params.
	if path == "/planetary/apod" {
		q := r.URL.Query()
		if q.Get("start_date") != "" || q.Get("count") != "" {
			serveFixture(w, "../../testdata/apod_range.json")
		} else {
			serveFixture(w, "../../testdata/apod_single.json")
		}
		return
	}

	// NEO: /neo/rest/v1/neo/{id} for specific asteroid lookup.
	if strings.HasPrefix(path, "/neo/rest/v1/neo/") && path != "/neo/rest/v1/neo/browse" {
		serveFixture(w, "../../testdata/neo_single.json")
		return
	}

	if f, ok := fixtures[path]; ok && f != "" {
		serveFixture(w, f)
		return
	}

	// Also check path suffix for external services whose base URL adds
	// additional path segments (e.g. EONET /api/v3/events).
	for suffix, f := range fixtures {
		if f != "" && strings.HasSuffix(path, suffix) {
			serveFixture(w, f)
			return
		}
	}

	http.Error(w, "no fixture for path: "+path, http.StatusNotFound)
}

func serveFixture(w http.ResponseWriter, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "fixture read error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(data); err != nil {
		http.Error(w, "fixture write error: "+err.Error(), http.StatusInternalServerError)
	}
}

// setupTestClient creates a nasa.Client backed by an httptest server that
// serves fixture data. All HTTP requests (regardless of target host) are
// redirected to the test server.
func setupTestClient(t *testing.T) *nasa.Client {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(fixtureHandler))
	t.Cleanup(ts.Close)

	return nasa.NewClient(
		nasa.WithBaseURL(ts.URL),
		nasa.WithAPIKey("TEST_KEY"),
		nasa.WithHTTPClient(&http.Client{
			Transport: &redirectTransport{target: ts.URL},
		}),
		nasa.WithRateLimit(1000), // no rate limit delay in tests
	)
}

// callTool is a helper to call a tool through handleMessage.
func callTool(client *nasa.Client, name string, args map[string]any) *JSONRPCResponse {
	req := makeRequest(1, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
	return handleMessage(req, client)
}

// --- Protocol tests (from Phase 1, preserved) ---

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
	badJSON := []byte(`{not valid json}`)
	var req JSONRPCRequest
	err := json.Unmarshal(badJSON, &req)
	if err == nil {
		t.Fatal("expected unmarshal error for malformed JSON")
	}
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      nil,
		Error:   &RPCError{Code: -32700, Message: "parse error: " + err.Error()},
	}
	if resp.Error.Code != -32700 {
		t.Errorf("expected error code -32700, got %d", resp.Error.Code)
	}
}

// --- Tool handler unit tests (Phase 3) ---

func TestHandlerAPOD(t *testing.T) {
	client := setupTestClient(t)

	t.Run("no_args_today", func(t *testing.T) {
		resp := callTool(client, "nasa_apod", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "Title:") {
			t.Error("expected Title: in output")
		}
		if !strings.Contains(text, "Date:") {
			t.Error("expected Date: in output")
		}
		if !strings.Contains(text, "Explanation:") {
			t.Error("expected Explanation: in output")
		}
	})

	t.Run("with_date", func(t *testing.T) {
		resp := callTool(client, "nasa_apod", map[string]any{"date": "2024-01-15"})
		text := toolText(t, resp)
		if !strings.Contains(text, "Title:") {
			t.Error("expected Title: in output")
		}
	})

	t.Run("with_count", func(t *testing.T) {
		resp := callTool(client, "nasa_apod", map[string]any{"count": float64(3)})
		text := toolText(t, resp)
		if !strings.Contains(text, "Title:") {
			t.Error("expected Title: in output")
		}
	})

	t.Run("invalid_date", func(t *testing.T) {
		resp := callTool(client, "nasa_apod", map[string]any{"date": "not-a-date"})
		assertToolError(t, resp, "invalid date")
	})
}

func TestHandlerAPODRange(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid", func(t *testing.T) {
		resp := callTool(client, "nasa_apod_range", map[string]any{
			"start_date": "2024-01-01",
			"end_date":   "2024-01-03",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "Date:") {
			t.Error("expected Date: in output")
		}
	})

	t.Run("missing_start_date", func(t *testing.T) {
		resp := callTool(client, "nasa_apod_range", map[string]any{
			"end_date": "2024-01-03",
		})
		assertToolError(t, resp, "start_date and end_date are required")
	})

	t.Run("missing_end_date", func(t *testing.T) {
		resp := callTool(client, "nasa_apod_range", map[string]any{
			"start_date": "2024-01-01",
		})
		assertToolError(t, resp, "start_date and end_date are required")
	})

	t.Run("invalid_date", func(t *testing.T) {
		resp := callTool(client, "nasa_apod_range", map[string]any{
			"start_date": "not-a-date",
			"end_date":   "2024-01-03",
		})
		assertToolError(t, resp, "invalid start_date")
	})
}

func TestHandlerNEOFeed(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid", func(t *testing.T) {
		resp := callTool(client, "nasa_neo_feed", map[string]any{
			"start_date": "2024-01-01",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "Near-Earth Objects") {
			t.Error("expected 'Near-Earth Objects' in output")
		}
		if !strings.Contains(text, "Total:") {
			t.Error("expected Total: in output")
		}
	})

	t.Run("missing_start_date", func(t *testing.T) {
		resp := callTool(client, "nasa_neo_feed", map[string]any{})
		assertToolError(t, resp, "start_date is required")
	})

	t.Run("invalid_date", func(t *testing.T) {
		resp := callTool(client, "nasa_neo_feed", map[string]any{
			"start_date": "bad-date",
		})
		assertToolError(t, resp, "invalid start_date")
	})
}

func TestHandlerNEOGet(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid", func(t *testing.T) {
		resp := callTool(client, "nasa_neo_get", map[string]any{
			"asteroid_id": "3542519",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "Asteroid") {
			t.Error("expected 'Asteroid' in output")
		}
		if !strings.Contains(text, "ID:") {
			t.Error("expected ID: in output")
		}
	})

	t.Run("missing_asteroid_id", func(t *testing.T) {
		resp := callTool(client, "nasa_neo_get", map[string]any{})
		assertToolError(t, resp, "asteroid_id is required")
	})
}

func TestHandlerDONKICME(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid_defaults", func(t *testing.T) {
		resp := callTool(client, "nasa_donki_cme", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "Coronal Mass Ejections") {
			t.Error("expected 'Coronal Mass Ejections' in output")
		}
		if !strings.Contains(text, "Activity ID:") {
			t.Error("expected Activity ID: in output")
		}
	})

	t.Run("with_dates", func(t *testing.T) {
		resp := callTool(client, "nasa_donki_cme", map[string]any{
			"start_date": "2024-01-01",
			"end_date":   "2024-01-31",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "Coronal Mass Ejections") {
			t.Error("expected 'Coronal Mass Ejections' in output")
		}
	})
}

func TestHandlerDONKIGST(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid_empty", func(t *testing.T) {
		resp := callTool(client, "nasa_donki_gst", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "No geomagnetic storm events found") {
			t.Error("expected 'No geomagnetic storm events found' in output")
		}
	})
}

func TestHandlerDONKIFLR(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid", func(t *testing.T) {
		resp := callTool(client, "nasa_donki_flr", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "Solar Flares") {
			t.Error("expected 'Solar Flares' in output")
		}
		if !strings.Contains(text, "Flare ID:") {
			t.Error("expected Flare ID: in output")
		}
	})
}

func TestHandlerDONKISEP(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid", func(t *testing.T) {
		resp := callTool(client, "nasa_donki_sep", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "Solar Energetic Particles") {
			t.Error("expected 'Solar Energetic Particles' in output")
		}
	})
}

func TestHandlerDONKINotifications(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid_defaults", func(t *testing.T) {
		resp := callTool(client, "nasa_donki_notifications", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "DONKI Notifications") {
			t.Error("expected 'DONKI Notifications' in output")
		}
		if !strings.Contains(text, "Message Type:") {
			t.Error("expected Message Type: in output")
		}
	})

	t.Run("with_type_filter", func(t *testing.T) {
		resp := callTool(client, "nasa_donki_notifications", map[string]any{
			"type": "FLR",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "DONKI Notifications") {
			t.Error("expected 'DONKI Notifications' in output")
		}
	})
}

func TestHandlerImagesSearch(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid", func(t *testing.T) {
		resp := callTool(client, "nasa_images_search", map[string]any{
			"q": "apollo 11",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "Results for") {
			t.Error("expected 'Results for' in output")
		}
	})

	t.Run("missing_query", func(t *testing.T) {
		resp := callTool(client, "nasa_images_search", map[string]any{})
		assertToolError(t, resp, "q (query) is required")
	})
}

func TestHandlerSSDCAD(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid_defaults", func(t *testing.T) {
		resp := callTool(client, "nasa_ssd_cad", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "Close Approach Data") {
			t.Error("expected 'Close Approach Data' in output")
		}
	})

	t.Run("with_filters", func(t *testing.T) {
		resp := callTool(client, "nasa_ssd_cad", map[string]any{
			"date_min": "2024-01-01",
			"date_max": "2024-12-31",
			"dist_max": "0.05",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "Close Approach Data") {
			t.Error("expected 'Close Approach Data' in output")
		}
	})

	t.Run("invalid_date", func(t *testing.T) {
		resp := callTool(client, "nasa_ssd_cad", map[string]any{
			"date_min": "not-a-date",
		})
		assertToolError(t, resp, "invalid date_min")
	})
}

func TestHandlerSSDFireball(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid_defaults", func(t *testing.T) {
		resp := callTool(client, "nasa_ssd_fireball", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "Fireball Events") {
			t.Error("expected 'Fireball Events' in output")
		}
	})

	t.Run("with_dates", func(t *testing.T) {
		resp := callTool(client, "nasa_ssd_fireball", map[string]any{
			"date_min": "2024-01-01",
			"date_max": "2024-12-31",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "Fireball Events") {
			t.Error("expected 'Fireball Events' in output")
		}
	})

	t.Run("invalid_date", func(t *testing.T) {
		resp := callTool(client, "nasa_ssd_fireball", map[string]any{
			"date_min": "bad",
		})
		assertToolError(t, resp, "invalid date_min")
	})
}

func TestHandlerEONETEvents(t *testing.T) {
	client := setupTestClient(t)

	t.Run("valid_defaults", func(t *testing.T) {
		resp := callTool(client, "nasa_eonet_events", map[string]any{})
		text := toolText(t, resp)
		if !strings.Contains(text, "EONET Events") {
			t.Error("expected 'EONET Events' in output")
		}
	})

	t.Run("with_status_filter", func(t *testing.T) {
		resp := callTool(client, "nasa_eonet_events", map[string]any{
			"status": "open",
		})
		text := toolText(t, resp)
		if !strings.Contains(text, "EONET Events") {
			t.Error("expected 'EONET Events' in output")
		}
	})
}

// --- Full lifecycle test ---

func TestFullLifecycle(t *testing.T) {
	client := setupTestClient(t)

	// Step 1: Initialize.
	initReq := makeRequest(1, "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
	})
	initResp := handleMessage(initReq, client)
	if initResp == nil {
		t.Fatal("initialize: expected response")
	}
	if initResp.Error != nil {
		t.Fatalf("initialize: unexpected error: %+v", initResp.Error)
	}

	// Step 2: Notification (no response expected).
	notifReq := makeRequest(nil, "notifications/initialized", nil)
	notifResp := handleMessage(notifReq, client)
	if notifResp != nil {
		t.Fatalf("notifications/initialized: expected nil, got %+v", notifResp)
	}

	// Step 3: List tools, verify count.
	listReq := makeRequest(2, "tools/list", map[string]any{})
	listResp := handleMessage(listReq, client)
	if listResp == nil {
		t.Fatal("tools/list: expected response")
	}
	m := resultMap(t, listResp)
	toolsRaw, ok := m["tools"].([]any)
	if !ok {
		t.Fatal("tools/list: tools missing or not an array")
	}
	if len(toolsRaw) != 13 {
		t.Fatalf("tools/list: expected 13 tools, got %d", len(toolsRaw))
	}

	// Step 4: Call nasa_apod, verify text output.
	apodResp := callTool(client, "nasa_apod", map[string]any{})
	text := toolText(t, apodResp)
	if !strings.Contains(text, "Title:") {
		t.Error("nasa_apod: expected Title: in output")
	}
	if !strings.Contains(text, "Date:") {
		t.Error("nasa_apod: expected Date: in output")
	}
	if !strings.Contains(text, "Explanation:") {
		t.Error("nasa_apod: expected Explanation: in output")
	}
}
