package nasa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// setupEONETTestServer creates an httptest server that serves the given fixture
// file and returns a configured EONETService. The handler captures the last
// request for parameter verification.
func setupEONETTestServer(t *testing.T, fixturePath string) (*EONETService, **http.Request) {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixturePath, err)
	}
	var captured *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Errorf("writing EONET fixture: %v", err)
			return
		}
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("TEST_KEY"))
	svc := &EONETService{client: c, baseURL: ts.URL}
	return svc, &captured
}

// setupEONETTestServerNoAuth creates a test server with no auth key configured
// (DEMO_KEY), so addAuth should not inject api_key.
func setupEONETTestServerNoAuth(t *testing.T, fixturePath string) (*EONETService, **http.Request) {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixturePath, err)
	}
	var captured *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Errorf("writing EONET no-auth fixture: %v", err)
			return
		}
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("DEMO_KEY"))
	svc := &EONETService{client: c, baseURL: ts.URL}
	return svc, &captured
}

func TestEONETEvents(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_events.json")

	events, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	// First event: open wildfire.
	ev := events[0]
	if ev.ID != "EONET_19203" {
		t.Errorf("expected ID=EONET_19203, got %s", ev.ID)
	}
	if ev.Title != "rx-12_ 13_ 14 Prescribed Fire, Burnett, Wisconsin" {
		t.Errorf("unexpected title: %s", ev.Title)
	}
	if ev.Closed != nil {
		t.Errorf("expected Closed=nil for open event, got %v", ev.Closed)
	}
	if len(ev.Categories) != 1 || ev.Categories[0].ID != "wildfires" {
		t.Errorf("unexpected categories: %+v", ev.Categories)
	}
	if len(ev.Sources) != 1 || ev.Sources[0].ID != "IRWIN" {
		t.Errorf("unexpected sources: %+v", ev.Sources)
	}
	if len(ev.Geometry) != 1 {
		t.Fatalf("expected 1 geometry entry, got %d", len(ev.Geometry))
	}

	// Second event: also an open wildfire.
	ev2 := events[1]
	if ev2.ID != "EONET_19204" {
		t.Errorf("expected ID=EONET_19204, got %s", ev2.ID)
	}
	if ev2.Closed != nil {
		t.Errorf("expected Closed=nil for open event, got %v", ev2.Closed)
	}
}

func TestEONETEventsEmpty(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_events_empty.json")

	events, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if events == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestEONETEventsOptions(t *testing.T) {
	svc, captured := setupEONETTestServer(t, "testdata/eonet_events_empty.json")

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	_, err := svc.Events(context.Background(),
		EONETWithStatus("open"),
		EONETWithLimit(10),
		EONETWithDays(30),
		EONETWithSource("InciWeb"),
		EONETWithStart(start),
		EONETWithEnd(end),
		EONETWithCategory("wildfires"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := *captured
	if req == nil {
		t.Fatal("no request captured")
	}

	q := req.URL.Query()
	tests := map[string]string{
		"status":   "open",
		"limit":    "10",
		"days":     "30",
		"source":   "InciWeb",
		"start":    "2026-01-01",
		"end":      "2026-03-01",
		"category": "wildfires",
		"api_key":  "TEST_KEY",
	}
	for key, want := range tests {
		got := q.Get(key)
		if got != want {
			t.Errorf("param %s: expected %q, got %q", key, want, got)
		}
	}
}

func TestEONETEvent(t *testing.T) {
	svc, captured := setupEONETTestServer(t, "testdata/eonet_event_single.json")

	event, err := svc.Event(context.Background(), "EONET_19203")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.ID != "EONET_19203" {
		t.Errorf("expected ID=EONET_19203, got %s", event.ID)
	}
	if event.Title != "rx-12_ 13_ 14 Prescribed Fire, Burnett, Wisconsin" {
		t.Errorf("unexpected title: %s", event.Title)
	}

	// Verify the path includes the event ID.
	req := *captured
	if req == nil {
		t.Fatal("no request captured")
	}
	if req.URL.Path != "/events/EONET_19203" {
		t.Errorf("expected path /events/EONET_19203, got %s", req.URL.Path)
	}
}

func TestEONETCategories(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_categories.json")

	cats, err := svc.Categories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cats) != 13 {
		t.Fatalf("expected 13 categories, got %d", len(cats))
	}

	if cats[0].ID != "drought" {
		t.Errorf("expected first category ID=drought, got %s", cats[0].ID)
	}
	if cats[0].Title != "Drought" {
		t.Errorf("unexpected title: %s", cats[0].Title)
	}
	if cats[0].Description != "Long lasting absence of precipitation affecting agriculture and livestock, and the overall availability of food and water." {
		t.Errorf("unexpected description: %s", cats[0].Description)
	}
	if cats[0].Layers != "https://eonet.gsfc.nasa.gov/api/v3/layers/drought" {
		t.Errorf("unexpected layers: %s", cats[0].Layers)
	}

	// Verify earthquakes category (third).
	if cats[2].ID != "earthquakes" {
		t.Errorf("expected third category ID=earthquakes, got %s", cats[2].ID)
	}
}

func TestEONETSources(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_sources.json")

	sources, err := svc.Sources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 33 {
		t.Fatalf("expected 33 sources, got %d", len(sources))
	}

	if sources[0].ID != "AVO" {
		t.Errorf("expected first source ID=AVO, got %s", sources[0].ID)
	}
	if sources[0].URL != "https://www.avo.alaska.edu/" {
		t.Errorf("expected first source URL=%q, got %q", "https://www.avo.alaska.edu/", sources[0].URL)
	}
	if sources[2].ID != "AU_BOM" {
		t.Errorf("expected third source ID=AU_BOM, got %s", sources[2].ID)
	}
}

func TestEONETGeometry(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_events.json")

	events, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First event: single Point geometry with magnitude.
	if len(events[0].Geometry) != 1 {
		t.Fatalf("expected 1 geometry entry for first event, got %d", len(events[0].Geometry))
	}
	geo := events[0].Geometry[0]
	if geo.Type != "Point" {
		t.Errorf("expected Type=Point, got %s", geo.Type)
	}
	pt, err := geo.PointCoordinates()
	if err != nil {
		t.Fatalf("PointCoordinates: %v", err)
	}
	if pt[0] != -92.6888328 {
		t.Errorf("expected lon=-92.6888328, got %f", pt[0])
	}
	if pt[1] != 45.7153853 {
		t.Errorf("expected lat=45.7153853, got %f", pt[1])
	}
	if geo.MagnitudeValue == nil {
		t.Fatal("expected MagnitudeValue to be non-nil for first event")
	}
	if *geo.MagnitudeValue != 655.7 {
		t.Errorf("expected MagnitudeValue=655.7, got %f", *geo.MagnitudeValue)
	}
	if geo.MagnitudeUnit != "acres" {
		t.Errorf("expected MagnitudeUnit=acres, got %s", geo.MagnitudeUnit)
	}

	// Second event: Point geometry with magnitude 1000 acres.
	geo2 := events[1].Geometry[0]
	if geo2.MagnitudeValue == nil {
		t.Fatal("expected MagnitudeValue to be non-nil")
	}
	if *geo2.MagnitudeValue != 1000 {
		t.Errorf("expected MagnitudeValue=1000, got %f", *geo2.MagnitudeValue)
	}
	if geo2.MagnitudeUnit != "acres" {
		t.Errorf("expected MagnitudeUnit=acres, got %s", geo2.MagnitudeUnit)
	}

	// Third event (severe storm): first geometry has null magnitude.
	geo3 := events[2].Geometry[0]
	if geo3.MagnitudeValue != nil {
		t.Errorf("expected MagnitudeValue=nil for storm first geometry, got %v", *geo3.MagnitudeValue)
	}
	// Third event has 6 geometry entries tracking the storm.
	if len(events[2].Geometry) != 6 {
		t.Errorf("expected 6 geometry entries for storm, got %d", len(events[2].Geometry))
	}
}

func TestEONETClosedField(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_events.json")

	events, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All five events in the real fixture are open (closed is null).
	for i, ev := range events {
		if ev.Closed != nil {
			t.Errorf("expected event[%d] Closed=nil, got %v", i, *ev.Closed)
		}
	}
}

func TestEONETAuthWithKey(t *testing.T) {
	svc, captured := setupEONETTestServer(t, "testdata/eonet_events_empty.json")

	_, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := *captured
	if req == nil {
		t.Fatal("no request captured")
	}

	apiKey := req.URL.Query().Get("api_key")
	if apiKey != "TEST_KEY" {
		t.Errorf("expected api_key=TEST_KEY, got %q", apiKey)
	}
}

func TestEONETAuthWithoutKey(t *testing.T) {
	svc, captured := setupEONETTestServerNoAuth(t, "testdata/eonet_events_empty.json")

	_, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := *captured
	if req == nil {
		t.Fatal("no request captured")
	}

	apiKey := req.URL.Query().Get("api_key")
	if apiKey != "" {
		t.Errorf("expected no api_key param for DEMO_KEY, got %q", apiKey)
	}
}

func TestEONETUsesRotatingClientKeys(t *testing.T) {
	var gotKeys []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKeys = append(gotKeys, r.URL.Query().Get("api_key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"title":"events","events":[]}`)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKeys([]string{"key1", "key2"}), WithRateLimit(1000))
	c.rateLimiter.keys.keys = []string{"key1", "key2"}
	c.rateLimiter.keys.current = 0
	svc := &EONETService{client: c, baseURL: ts.URL}

	for range 4 {
		if _, err := svc.Events(context.Background()); err != nil {
			t.Fatalf("Events() error: %v", err)
		}
	}

	want := []string{"key1", "key2", "key1", "key2"}
	if len(gotKeys) != len(want) {
		t.Fatalf("expected %d requests, got %d", len(want), len(gotKeys))
	}
	for i, wantKey := range want {
		if gotKeys[i] != wantKey {
			t.Fatalf("request %d used key %q, want %q", i+1, gotKeys[i], wantKey)
		}
	}
}

func TestEONETEventUsesRotatingClientKeys(t *testing.T) {
	var gotKeys []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKeys = append(gotKeys, r.URL.Query().Get("api_key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"EONET_1","title":"test"}`)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKeys([]string{"key1", "key2"}), WithRateLimit(1000))
	c.rateLimiter.keys.keys = []string{"key1", "key2"}
	c.rateLimiter.keys.current = 0
	svc := &EONETService{client: c, baseURL: ts.URL}

	for range 3 {
		if _, err := svc.Event(context.Background(), "EONET_1"); err != nil {
			t.Fatalf("Event() error: %v", err)
		}
	}

	want := []string{"key1", "key2", "key1"}
	for i, wantKey := range want {
		if gotKeys[i] != wantKey {
			t.Fatalf("request %d used key %q, want %q", i+1, gotKeys[i], wantKey)
		}
	}
}

func TestEONETEvent_EmptyID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("HTTP request should not be made for empty event ID")
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("TEST_KEY"))
	svc := &EONETService{client: c, baseURL: ts.URL}

	_, err := svc.Event(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty event ID")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error = %q, want to contain 'must not be empty'", err.Error())
	}
}

func TestEONETPointCoordinates_WrongLength(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		count int
	}{
		{"one element", `[1.0]`, 1},
		{"three elements", `[1.0, 2.0, 3.0]`, 3},
		{"empty array", `[]`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			geo := &EONETGeometry{
				Coordinates: json.RawMessage(tt.raw),
			}
			_, err := geo.PointCoordinates()
			if err == nil {
				t.Fatal("expected error for wrong tuple length")
			}
			want := fmt.Sprintf("exactly 2 elements, got %d", tt.count)
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), want)
			}
		})
	}
}

func TestEONETPointCoordinates_Valid(t *testing.T) {
	geo := &EONETGeometry{
		Coordinates: json.RawMessage(`[-92.5, 45.7]`),
	}
	pt, err := geo.PointCoordinates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pt[0] != -92.5 || pt[1] != 45.7 {
		t.Errorf("got [%f, %f], want [-92.5, 45.7]", pt[0], pt[1])
	}
}

func TestEONETPolygonCoordinates_BadRingSize(t *testing.T) {
	// A ring with only 2 points (needs at least 3).
	raw := `[[[1.0, 2.0], [3.0, 4.0]]]`
	geo := &EONETGeometry{
		Coordinates: json.RawMessage(raw),
	}
	_, err := geo.PolygonCoordinates()
	if err == nil {
		t.Fatal("expected error for ring with < 3 points")
	}
	if !strings.Contains(err.Error(), "at least 3 points") {
		t.Errorf("error = %q, want to contain 'at least 3 points'", err.Error())
	}
}

func TestEONETPolygonCoordinates_BadPointSize(t *testing.T) {
	// A ring with a point that has 3 elements instead of 2.
	raw := `[[[1.0, 2.0], [3.0, 4.0], [5.0, 6.0, 7.0]]]`
	geo := &EONETGeometry{
		Coordinates: json.RawMessage(raw),
	}
	_, err := geo.PolygonCoordinates()
	if err == nil {
		t.Fatal("expected error for point with != 2 elements")
	}
	if !strings.Contains(err.Error(), "exactly 2 elements") {
		t.Errorf("error = %q, want to contain 'exactly 2 elements'", err.Error())
	}
}

func TestEONETPolygonCoordinates_Valid(t *testing.T) {
	raw := `[[[1.0, 2.0], [3.0, 4.0], [5.0, 6.0], [1.0, 2.0]]]`
	geo := &EONETGeometry{
		Coordinates: json.RawMessage(raw),
	}
	coords, err := geo.PolygonCoordinates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(coords) != 1 {
		t.Fatalf("expected 1 ring, got %d", len(coords))
	}
	if len(coords[0]) != 4 {
		t.Fatalf("expected 4 points in ring, got %d", len(coords[0]))
	}
}
