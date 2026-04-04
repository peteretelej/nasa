package nasa

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNEOFeed_EmptyStringFloatHandling verifies that the SDK handles empty
// string velocity/distance values through the full Feed path. The second
// asteroid in the fixture has all velocity and distance fields set to "".
func TestNEOFeed_EmptyStringFloatHandling(t *testing.T) {
	fixture, err := os.ReadFile("testdata/neo_feed_http_links.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/neo/rest/v1/feed") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(fixture); err != nil {
			t.Errorf("writing neo feed fixture: %v", err)
			return
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

	asteroids, ok := feed.NearEarthObjects["2024-01-15"]
	if !ok {
		t.Fatal("expected key 2024-01-15 in NearEarthObjects")
	}
	if len(asteroids) != 2 {
		t.Fatalf("expected 2 asteroids, got %d", len(asteroids))
	}

	// Second asteroid has empty string values for all velocity/distance fields.
	a := asteroids[1]
	if a.ID != "54321" {
		t.Fatalf("expected second asteroid ID 54321, got %s", a.ID)
	}
	if len(a.CloseApproachData) == 0 {
		t.Fatal("expected close approach data")
	}

	ca := a.CloseApproachData[0]

	// Empty strings should unmarshal to zero values.
	if got := float64(ca.RelativeVelocity.KilometersPerSecond); got != 0 {
		t.Errorf("KilometersPerSecond = %f, want 0 for empty string", got)
	}
	if got := float64(ca.RelativeVelocity.KilometersPerHour); got != 0 {
		t.Errorf("KilometersPerHour = %f, want 0 for empty string", got)
	}
	if got := float64(ca.RelativeVelocity.MilesPerHour); got != 0 {
		t.Errorf("MilesPerHour = %f, want 0 for empty string", got)
	}
	if got := float64(ca.MissDistance.Astronomical); got != 0 {
		t.Errorf("Astronomical = %f, want 0 for empty string", got)
	}
	if got := float64(ca.MissDistance.Lunar); got != 0 {
		t.Errorf("Lunar = %f, want 0 for empty string", got)
	}
	if got := float64(ca.MissDistance.Kilometers); got != 0 {
		t.Errorf("Kilometers = %f, want 0 for empty string", got)
	}
	if got := float64(ca.MissDistance.Miles); got != 0 {
		t.Errorf("Miles = %f, want 0 for empty string", got)
	}

	// First asteroid should still have real values.
	first := asteroids[0]
	if got := float64(first.CloseApproachData[0].RelativeVelocity.KilometersPerSecond); got != 15.23 {
		t.Errorf("first asteroid KilometersPerSecond = %f, want 15.23", got)
	}
}

// TestNEOFeed_HTTPLinkFixupNested verifies that top-level pagination links in
// a feed response are fixed from http:// to https://. This fixture has all
// links (self, next, prev) using http://api.nasa.gov.
func TestNEOFeed_HTTPLinkFixupNested(t *testing.T) {
	fixture, err := os.ReadFile("testdata/neo_feed_http_links.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/neo/rest/v1/feed") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(fixture); err != nil {
			t.Errorf("writing neo http-links fixture: %v", err)
			return
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

	// All top-level pagination links should be fixed to https.
	if !strings.HasPrefix(feed.Links.Self, "https://api.nasa.gov") {
		t.Errorf("Links.Self not fixed to HTTPS: %s", feed.Links.Self)
	}
	if !strings.HasPrefix(feed.Links.Next, "https://api.nasa.gov") {
		t.Errorf("Links.Next not fixed to HTTPS: %s", feed.Links.Next)
	}
	if !strings.HasPrefix(feed.Links.Prev, "https://api.nasa.gov") {
		t.Errorf("Links.Prev not fixed to HTTPS: %s", feed.Links.Prev)
	}

	// Verify no top-level link contains http://api.nasa.gov.
	for _, link := range []string{feed.Links.Self, feed.Links.Next, feed.Links.Prev} {
		if strings.Contains(link, "http://api.nasa.gov") {
			t.Errorf("link still contains http://api.nasa.gov: %s", link)
		}
	}

	// Verify the element count is correct despite nested http links.
	if feed.ElementCount != 2 {
		t.Errorf("ElementCount = %d, want 2", feed.ElementCount)
	}

	// Verify both asteroids were parsed correctly.
	asteroids := feed.NearEarthObjects["2024-01-15"]
	if len(asteroids) != 2 {
		t.Fatalf("expected 2 asteroids, got %d", len(asteroids))
	}

	// The NearEarthObject struct exposes JPLURL (nasa_jpl_url) which is not
	// processed by fixHTTPLinks (it only fixes api.nasa.gov links). The fixture
	// uses http://ssd.jpl.nasa.gov which is a different domain. This verifies
	// that the SDK doesn't corrupt non-api.nasa.gov URLs.
	for _, a := range asteroids {
		if a.JPLURL == "" {
			t.Errorf("asteroid %s: expected non-empty JPLURL", a.ID)
		}
	}
}

func TestNEOFeed_TimezoneEdgeUsesSameDatesForValidationAndWire(t *testing.T) {
	loc := time.FixedZone("east", 2*60*60)
	start := time.Date(2024, 1, 1, 0, 30, 0, 0, loc)
	end := time.Date(2024, 1, 7, 0, 30, 0, 0, loc)

	var gotQuery url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(loadFixture(t, "neo_feed.json"))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"), WithRateLimit(1000))
	if _, err := c.NEO.Feed(context.Background(), start, end); err != nil {
		t.Fatalf("Feed() should validate against the same dates it sends, got error: %v", err)
	}

	if gotQuery.Get("start_date") != "2024-01-01" {
		t.Fatalf("start_date = %q, want 2024-01-01", gotQuery.Get("start_date"))
	}
	// neoFeedEndDate truncates to start-of-day, so 2024-01-07 00:30 +02:00
	// becomes 2024-01-07 (the calendar date of the input).
	if gotQuery.Get("end_date") != "2024-01-07" {
		t.Fatalf("end_date = %q, want 2024-01-07", gotQuery.Get("end_date"))
	}
}

func TestNEOFeed_RejectsWireDateRangePastSevenDays(t *testing.T) {
	// Wire dates: 2024-01-01 to 2024-01-09 = 8 days, exceeds the 7-day limit.
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("HTTP request should not be made when wire dates exceed 7 days")
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"), WithRateLimit(1000))
	_, err := c.NEO.Feed(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error for wire date range > 7 days")
	}
	if !strings.Contains(err.Error(), "exceeds 7 days") {
		t.Fatalf("error = %q, want range validation message", err.Error())
	}
}

func TestNEOGet_EscapesCraftedAsteroidID(t *testing.T) {
	craftedID := "2024/AB?x=1#frag"
	var gotPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"2024/AB?x=1#frag","name":"crafted","close_approach_data":[]}`)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"), WithRateLimit(1000))
	if _, err := c.NEO.Get(context.Background(), craftedID); err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	wantPath := "/neo/rest/v1/neo/" + url.PathEscape(craftedID)
	if gotPath != wantPath {
		t.Fatalf("escaped path = %q, want %q", gotPath, wantPath)
	}
}
