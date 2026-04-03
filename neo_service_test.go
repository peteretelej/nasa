package nasa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNEOService_Feed(t *testing.T) {
	fixture, err := os.ReadFile("testdata/neo_feed.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/neo/rest/v1/feed") {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("start_date") == "" || q.Get("end_date") == "" {
			http.Error(w, "missing date params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(fixture); err != nil {
			t.Fatalf("writing neo service fixture: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	feed, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		t.Fatalf("NEO.Feed returned error: %v", err)
	}

	if feed.ElementCount != 1 {
		t.Errorf("ElementCount = %d, want 1", feed.ElementCount)
	}

	asteroids, ok := feed.NearEarthObjects["2024-01-15"]
	if !ok {
		t.Fatal("expected key 2024-01-15 in NearEarthObjects")
	}
	if len(asteroids) != 1 {
		t.Fatalf("expected 1 asteroid, got %d", len(asteroids))
	}

	a := asteroids[0]
	if a.ID != "3542519" {
		t.Errorf("ID = %q, want %q", a.ID, "3542519")
	}
	if a.Name != "(2010 PK9)" {
		t.Errorf("Name = %q, want %q", a.Name, "(2010 PK9)")
	}

	// Verify string-number conversion via StringFloat64.
	ca := a.CloseApproachData[0]
	if got := float64(ca.RelativeVelocity.KilometersPerSecond); got != 15.23 {
		t.Errorf("KilometersPerSecond = %f, want 15.23", got)
	}
	if got := float64(ca.MissDistance.Astronomical); got != 0.2 {
		t.Errorf("Astronomical = %f, want 0.2", got)
	}

	// Verify close_approach_date_full was parsed.
	wantDateFull := time.Date(2024, 1, 15, 8, 30, 0, 0, time.UTC)
	if !ca.CloseApproachDateFull.Equal(wantDateFull) {
		t.Errorf("CloseApproachDateFull = %v, want %v", ca.CloseApproachDateFull, wantDateFull)
	}

	// Verify links were fixed to HTTPS.
	if !strings.HasPrefix(feed.Links.Self, "https://api.nasa.gov") {
		t.Errorf("Links.Self not fixed to HTTPS: %s", feed.Links.Self)
	}
	if feed.Links.Next != "" && !strings.HasPrefix(feed.Links.Next, "https://api.nasa.gov") {
		t.Errorf("Links.Next not fixed to HTTPS: %s", feed.Links.Next)
	}
}

func TestNEOService_Feed_DateValidation(t *testing.T) {
	// Server that should never be called.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not have been made for invalid date range")
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC) // 8 days

	_, err := c.NEO.Feed(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error for date range > 7 days")
	}
	if !strings.Contains(err.Error(), "exceeds 7 days") {
		t.Errorf("error = %q, want to contain 'exceeds 7 days'", err.Error())
	}
}

func TestNEOService_Feed_InvalidRange(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not have been made for start > end")
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC) // before start

	_, err := c.NEO.Feed(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error for start > end")
	}
	if !strings.Contains(err.Error(), "after end date") {
		t.Errorf("error = %q, want to contain 'after end date'", err.Error())
	}
}

func TestNEOService_Get(t *testing.T) {
	fixture, err := os.ReadFile("testdata/neo_single.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/neo/rest/v1/neo/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(fixture); err != nil {
			t.Fatalf("writing neo get fixture: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	neo, err := c.NEO.Get(context.Background(), "2021277")
	if err != nil {
		t.Fatalf("NEO.Get returned error: %v", err)
	}

	if neo.ID != "2021277" {
		t.Errorf("ID = %q, want %q", neo.ID, "2021277")
	}
	if neo.Name != "21277 (1996 TO5)" {
		t.Errorf("Name = %q, want %q", neo.Name, "21277 (1996 TO5)")
	}

	// Verify orbital data is present.
	if neo.OrbitalData == nil {
		t.Fatal("expected OrbitalData to be present")
	}
	if neo.OrbitalData.OrbitID != "164" {
		t.Errorf("OrbitID = %q, want %q", neo.OrbitalData.OrbitID, "164")
	}
	if neo.OrbitalData.DataArcInDays != 10047 {
		t.Errorf("DataArcInDays = %d, want 10047", neo.OrbitalData.DataArcInDays)
	}

	// Verify orbit class.
	if neo.OrbitalData.OrbitClass == nil {
		t.Fatal("expected OrbitClass to be present")
	}
	if neo.OrbitalData.OrbitClass.Type != "AMO" {
		t.Errorf("OrbitClass.Type = %q, want %q", neo.OrbitalData.OrbitClass.Type, "AMO")
	}

	// Verify close approach date full parsing.
	if len(neo.CloseApproachData) == 0 {
		t.Fatal("expected close approach data")
	}
	wantDateFull := time.Date(2026, 4, 1, 17, 14, 0, 0, time.UTC)
	if !neo.CloseApproachData[0].CloseApproachDateFull.Equal(wantDateFull) {
		t.Errorf("CloseApproachDateFull = %v, want %v",
			neo.CloseApproachData[0].CloseApproachDateFull, wantDateFull)
	}
}

func TestNEOService_Get_EmptyID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not have been made for empty ID")
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	_, err := c.NEO.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty asteroid ID")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error = %q, want to contain 'must not be empty'", err.Error())
	}
}

func TestNEOService_Browse(t *testing.T) {
	fixture, err := os.ReadFile("testdata/neo_browse.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/neo/rest/v1/neo/browse") {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("page") == "" || q.Get("size") == "" {
			http.Error(w, "missing params", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(fixture); err != nil {
			t.Fatalf("writing neo browse fixture: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	browse, err := c.NEO.Browse(context.Background(), 0, 2)
	if err != nil {
		t.Fatalf("NEO.Browse returned error: %v", err)
	}

	if browse.Page.TotalElements != 35000 {
		t.Errorf("TotalElements = %d, want 35000", browse.Page.TotalElements)
	}
	if browse.Page.Size != 2 {
		t.Errorf("Page.Size = %d, want 2", browse.Page.Size)
	}
	if len(browse.NearEarthObjects) != 2 {
		t.Fatalf("expected 2 NEOs, got %d", len(browse.NearEarthObjects))
	}
	if browse.NearEarthObjects[0].Name != "433 Eros (A898 PA)" {
		t.Errorf("first NEO name = %q, want %q", browse.NearEarthObjects[0].Name, "433 Eros (A898 PA)")
	}

	// Verify links were fixed to HTTPS.
	if !strings.HasPrefix(browse.Links.Self, "https://api.nasa.gov") {
		t.Errorf("Links.Self not fixed to HTTPS: %s", browse.Links.Self)
	}
}

func TestStringFloat64_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"string value", `"34.02"`, 34.02, false},
		{"string zero", `"0"`, 0, false},
		{"empty string", `""`, 0, false},
		{"json number", `34.02`, 34.02, false},
		{"json integer", `42`, 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sf StringFloat64
			err := json.Unmarshal([]byte(tt.input), &sf)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && float64(sf) != tt.want {
				t.Errorf("got %f, want %f", float64(sf), tt.want)
			}
		})
	}
}

func TestMillisecondEpoch_Unmarshal(t *testing.T) {
	// 1712016000000 ms = 2024-04-02 00:00:00 UTC
	input := `1712016000000`
	var me MillisecondEpoch
	if err := json.Unmarshal([]byte(input), &me); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	want := time.Date(2024, 4, 2, 0, 0, 0, 0, time.UTC)
	got := me.Time()
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCloseApproach_DateFullParse(t *testing.T) {
	input := `{
		"close_approach_date": "2026-04-01",
		"close_approach_date_full": "2026-Apr-01 17:14",
		"epoch_date_close_approach": 1775070840000,
		"relative_velocity": {"kilometers_per_second": "12.0", "kilometers_per_hour": "43200.0", "miles_per_hour": "26843.0"},
		"miss_distance": {"astronomical": "0.1", "lunar": "38.9", "kilometers": "14960000", "miles": "9296000"},
		"orbiting_body": "Earth"
	}`

	var ca CloseApproach
	if err := json.Unmarshal([]byte(input), &ca); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	want := time.Date(2026, 4, 1, 17, 14, 0, 0, time.UTC)
	if !ca.CloseApproachDateFull.Equal(want) {
		t.Errorf("CloseApproachDateFull = %v, want %v", ca.CloseApproachDateFull, want)
	}
}

func TestCloseApproach_DateFullEmpty(t *testing.T) {
	input := `{
		"close_approach_date": "2026-04-01",
		"epoch_date_close_approach": 1775070840000,
		"relative_velocity": {"kilometers_per_second": "12.0", "kilometers_per_hour": "43200.0", "miles_per_hour": "26843.0"},
		"miss_distance": {"astronomical": "0.1", "lunar": "38.9", "kilometers": "14960000", "miles": "9296000"},
		"orbiting_body": "Earth"
	}`

	var ca CloseApproach
	if err := json.Unmarshal([]byte(input), &ca); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !ca.CloseApproachDateFull.IsZero() {
		t.Errorf("expected zero time for missing close_approach_date_full, got %v", ca.CloseApproachDateFull)
	}
}

func TestNEOService_Feed_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`)); err != nil {
			t.Fatalf("writing neo api error response: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	_, err := c.NEO.Feed(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	var rlErr *RateLimitError
	if !isRateLimitError(err, &rlErr) {
		t.Errorf("expected *RateLimitError, got %T: %v", err, err)
	}
}

// isRateLimitError checks if err is a *RateLimitError using errors.As.
func isRateLimitError(err error, target **RateLimitError) bool {
	return errorAs(err, target)
}

// errorAs is a thin wrapper to avoid importing errors in tests.
func errorAs(err error, target interface{}) bool {
	switch t := target.(type) {
	case **RateLimitError:
		for {
			if rl, ok := err.(*RateLimitError); ok {
				*t = rl
				return true
			}
			u, ok := err.(interface{ Unwrap() error })
			if !ok {
				return false
			}
			err = u.Unwrap()
		}
	}
	return false
}

func TestFixHTTPLinks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"http to https",
			"http://api.nasa.gov/neo/rest/v1/feed?start=2024-01-15",
			"https://api.nasa.gov/neo/rest/v1/feed?start=2024-01-15",
		},
		{
			"already https",
			"https://api.nasa.gov/neo/rest/v1/feed?start=2024-01-15",
			"https://api.nasa.gov/neo/rest/v1/feed?start=2024-01-15",
		},
		{
			"non-NASA URL",
			"http://example.com/some/path",
			"http://example.com/some/path",
		},
		{
			"empty string",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fixHTTPLinks(tt.input)
			if got != tt.want {
				t.Errorf("fixHTTPLinks(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
