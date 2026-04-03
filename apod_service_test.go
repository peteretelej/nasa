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

func newTestAPODServer(t *testing.T) *httptest.Server {
	t.Helper()
	fixture, err := os.ReadFile("testdata/apod_single.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
}

func TestAPODServiceGet(t *testing.T) {
	ts := newTestAPODServer(t)
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	img, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("APOD.Get returned error: %v", err)
	}

	if img.Title != "Test APOD Title" {
		t.Errorf("got title %q, want %q", img.Title, "Test APOD Title")
	}
	if img.Date != "2024-01-15" {
		t.Errorf("got date %q, want %q", img.Date, "2024-01-15")
	}
	if img.URL == "" {
		t.Error("expected non-empty URL")
	}
	if img.HDURL == "" {
		t.Error("expected non-empty HDURL")
	}
	if img.ApodDate.IsZero() {
		t.Error("expected ApodDate to be parsed")
	}
}

func TestAPODServiceToday(t *testing.T) {
	ts := newTestAPODServer(t)
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))

	img, err := c.APOD.Today(context.Background())
	if err != nil {
		t.Fatalf("APOD.Today returned error: %v", err)
	}

	if img == nil {
		t.Fatal("expected non-nil Image")
	}
	if img.Title == "" {
		t.Error("expected non-empty title")
	}
}

func TestAPODServiceRandom(t *testing.T) {
	ts := newTestAPODServer(t)
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))

	img, err := c.APOD.Random(context.Background())
	if err != nil {
		t.Fatalf("APOD.Random returned error: %v", err)
	}

	if img == nil {
		t.Fatal("expected non-nil Image")
	}
	if img.URL == "" && img.HDURL == "" {
		t.Error("expected at least one image URL")
	}
}

func TestAPODServiceGetNoImageURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"date":"2024-01-15","title":"No Image","url":"","hdurl":""}`))
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST"))
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	_, err := c.APOD.Get(context.Background(), date)
	if err == nil {
		t.Fatal("expected error for response with no image URL")
	}
	if !strings.Contains(err.Error(), "no image URL") {
		t.Errorf("error should mention 'no image URL', got: %v", err)
	}
}
