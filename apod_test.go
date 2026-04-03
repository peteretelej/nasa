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

func TestAPODServiceGetViaFixture(t *testing.T) {
	fixture, err := os.ReadFile("testdata/apod_single.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	img, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img == nil {
		t.Fatal("expected non-nil Image")
	}
	if !strings.Contains(img.String(), "Title:") {
		t.Error("Image.String() should contain 'Title:'")
	}
	if img.Title != "Test APOD Title" {
		t.Errorf("got title %q, want %q", img.Title, "Test APOD Title")
	}
}
