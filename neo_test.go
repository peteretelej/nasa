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

func TestNEOServiceFeedViaFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/neo_feed.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	nl, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nl == nil {
		t.Fatal("expected non-nil NeoList")
	}
	if nl.ElementCount < 1 {
		t.Errorf("expected ElementCount >= 1, got %d", nl.ElementCount)
	}
	if !strings.Contains(nl.String(), "Near Earth Objects From:") {
		t.Error("NeoList.String() should contain 'Near Earth Objects From:'")
	}
}
