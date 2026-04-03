package nasa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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
		w.Write(data)
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("TEST_KEY"))
	svc := &EONETService{client: c, baseURL: ts.URL, apiKey: "TEST_KEY"}
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
		w.Write(data)
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("DEMO_KEY"))
	svc := &EONETService{client: c, baseURL: ts.URL, apiKey: "DEMO_KEY"}
	return svc, &captured
}

func TestEONETEvents(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_events.json")

	events, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// First event: open (Closed == nil), wildfire.
	ev := events[0]
	if ev.ID != "EONET_6354" {
		t.Errorf("expected ID=EONET_6354, got %s", ev.ID)
	}
	if ev.Title != "Wildfire - Los Angeles, California" {
		t.Errorf("unexpected title: %s", ev.Title)
	}
	if ev.Closed != nil {
		t.Errorf("expected Closed=nil for open event, got %v", ev.Closed)
	}
	if len(ev.Categories) != 1 || ev.Categories[0].ID != "wildfires" {
		t.Errorf("unexpected categories: %+v", ev.Categories)
	}
	if len(ev.Sources) != 1 || ev.Sources[0].ID != "InciWeb" {
		t.Errorf("unexpected sources: %+v", ev.Sources)
	}
	if len(ev.Geometry) != 2 {
		t.Fatalf("expected 2 geometry entries, got %d", len(ev.Geometry))
	}

	// Second event: closed.
	ev2 := events[1]
	if ev2.ID != "EONET_6280" {
		t.Errorf("expected ID=EONET_6280, got %s", ev2.ID)
	}
	if ev2.Closed == nil {
		t.Fatal("expected Closed to be non-nil for closed event")
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

	event, err := svc.Event(context.Background(), "EONET_6354")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.ID != "EONET_6354" {
		t.Errorf("expected ID=EONET_6354, got %s", event.ID)
	}
	if event.Title != "Wildfire - Los Angeles, California" {
		t.Errorf("unexpected title: %s", event.Title)
	}

	// Verify the path includes the event ID.
	req := *captured
	if req == nil {
		t.Fatal("no request captured")
	}
	if req.URL.Path != "/events/EONET_6354" {
		t.Errorf("expected path /events/EONET_6354, got %s", req.URL.Path)
	}
}

func TestEONETCategories(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_categories.json")

	cats, err := svc.Categories(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cats) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(cats))
	}

	if cats[0].ID != "drought" {
		t.Errorf("expected first category ID=drought, got %s", cats[0].ID)
	}
	if cats[0].Title != "Drought" {
		t.Errorf("unexpected title: %s", cats[0].Title)
	}
	if cats[0].Description != "Long lasting absence of precipitation." {
		t.Errorf("unexpected description: %s", cats[0].Description)
	}
	if cats[0].Layers != "MODIS_Terra_NDVI_8Day" {
		t.Errorf("unexpected layers: %s", cats[0].Layers)
	}

	// Verify wildfires category.
	if cats[2].ID != "wildfires" {
		t.Errorf("expected third category ID=wildfires, got %s", cats[2].ID)
	}
}

func TestEONETSources(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_sources.json")

	sources, err := svc.Sources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}

	if sources[0].ID != "InciWeb" {
		t.Errorf("expected first source ID=InciWeb, got %s", sources[0].ID)
	}
	if sources[0].URL != "https://inciweb.nwcg.gov/" {
		t.Errorf("unexpected URL: %s", sources[0].URL)
	}
	if sources[2].ID != "USGS_CMT" {
		t.Errorf("expected third source ID=USGS_CMT, got %s", sources[2].ID)
	}
}

func TestEONETGeometry(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_events.json")

	events, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First event: geometry without magnitude.
	geo := events[0].Geometry[0]
	if geo.Type != "Point" {
		t.Errorf("expected Type=Point, got %s", geo.Type)
	}
	if len(geo.Coordinates) != 2 {
		t.Fatalf("expected 2 coordinates, got %d", len(geo.Coordinates))
	}
	if geo.Coordinates[0] != -118.52 {
		t.Errorf("expected lon=-118.52, got %f", geo.Coordinates[0])
	}
	if geo.Coordinates[1] != 34.05 {
		t.Errorf("expected lat=34.05, got %f", geo.Coordinates[1])
	}
	if geo.MagnitudeValue != nil {
		t.Errorf("expected MagnitudeValue=nil, got %v", *geo.MagnitudeValue)
	}

	// Second event: geometry with magnitude.
	geo2 := events[1].Geometry[0]
	if geo2.MagnitudeValue == nil {
		t.Fatal("expected MagnitudeValue to be non-nil")
	}
	if *geo2.MagnitudeValue != 65.0 {
		t.Errorf("expected MagnitudeValue=65.0, got %f", *geo2.MagnitudeValue)
	}
	if geo2.MagnitudeUnit != "kts" {
		t.Errorf("expected MagnitudeUnit=kts, got %s", geo2.MagnitudeUnit)
	}
}

func TestEONETClosedField(t *testing.T) {
	svc, _ := setupEONETTestServer(t, "testdata/eonet_events.json")

	events, err := svc.Events(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First event: open (Closed is null).
	if events[0].Closed != nil {
		t.Errorf("expected first event Closed=nil, got %v", *events[0].Closed)
	}

	// Second event: closed with timestamp.
	if events[1].Closed == nil {
		t.Fatal("expected second event Closed to be non-nil")
	}
	expected := time.Date(2025, 11, 15, 0, 0, 0, 0, time.UTC)
	if !events[1].Closed.Equal(expected) {
		t.Errorf("expected Closed=%v, got %v", expected, *events[1].Closed)
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
