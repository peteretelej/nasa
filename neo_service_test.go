package nasa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func newTestNEOServer(t *testing.T) *httptest.Server {
	t.Helper()
	fixture, err := os.ReadFile("testdata/neo_feed.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/neo/rest/v1/feed") {
			http.NotFound(w, r)
			return
		}
		// Verify date params are present
		q := r.URL.Query()
		if q.Get("start_date") == "" {
			http.Error(w, "missing start_date", http.StatusBadRequest)
			return
		}
		if q.Get("end_date") == "" {
			http.Error(w, "missing end_date", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
}

func TestNEOServiceFeed(t *testing.T) {
	ts := newTestNEOServer(t)
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	nl, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		t.Fatalf("NEO.Feed returned error: %v", err)
	}

	if nl.ElementCount != 1 {
		t.Errorf("got element count %d, want 1", nl.ElementCount)
	}
	if nl.Start != "2024-01-15" {
		t.Errorf("got start %q, want %q", nl.Start, "2024-01-15")
	}
	if nl.End != "2024-01-15" {
		t.Errorf("got end %q, want %q", nl.End, "2024-01-15")
	}
	if len(nl.NearEarthObjects) == 0 {
		t.Fatal("expected at least one date in NearEarthObjects")
	}
	asteroids, ok := nl.NearEarthObjects["2024-01-15"]
	if !ok {
		t.Fatal("expected key 2024-01-15 in NearEarthObjects")
	}
	if len(asteroids) != 1 {
		t.Errorf("expected 1 asteroid, got %d", len(asteroids))
	}
	if asteroids[0].Name != "(2010 PK9)" {
		t.Errorf("got asteroid name %q, want %q", asteroids[0].Name, "(2010 PK9)")
	}
}

func TestNEOServiceFeedDateFormat(t *testing.T) {
	var receivedStartDate, receivedEndDate string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		receivedStartDate = q.Get("start_date")
		receivedEndDate = q.Get("end_date")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"element_count":0,"near_earth_objects":{}}`))
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	start := time.Date(2024, 3, 5, 10, 30, 0, 0, time.UTC)
	end := time.Date(2024, 3, 7, 15, 0, 0, 0, time.UTC)

	_, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		t.Fatalf("NEO.Feed returned error: %v", err)
	}

	if receivedStartDate != "2024-03-05" {
		t.Errorf("start_date param = %q, want %q", receivedStartDate, "2024-03-05")
	}
	if receivedEndDate != "2024-03-07" {
		t.Errorf("end_date param = %q, want %q", receivedEndDate, "2024-03-07")
	}
}
