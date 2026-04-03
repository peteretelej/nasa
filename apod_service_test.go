package nasa

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// setupAPODTestClient creates an httptest server from the given handler and
// returns a Client configured to use it.
func setupAPODTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
}

// apodFixtureHandler returns a handler that serves the given fixture file
// for any request to /planetary/apod.
func apodFixtureHandler(t *testing.T, fixturePath string) http.HandlerFunc {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixturePath, err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing fixture response: %v", err)
		}
	}
}

func TestAPODService_Today(t *testing.T) {
	c := setupAPODTestClient(t, apodFixtureHandler(t, "testdata/apod_single.json"))

	img, err := c.APOD.Today(context.Background())
	if err != nil {
		t.Fatalf("Today() error: %v", err)
	}

	if img.Title != "Test APOD Title" {
		t.Errorf("Title = %q, want %q", img.Title, "Test APOD Title")
	}
	if img.Explanation != "Test explanation for APOD." {
		t.Errorf("Explanation = %q, want %q", img.Explanation, "Test explanation for APOD.")
	}
	if img.URL != "https://apod.nasa.gov/apod/image/2401/test.jpg" {
		t.Errorf("URL = %q, want full URL", img.URL)
	}
	if img.HDURL != "https://apod.nasa.gov/apod/image/2401/test_hd.jpg" {
		t.Errorf("HDURL = %q, want full HD URL", img.HDURL)
	}
	if img.MediaType != MediaImage {
		t.Errorf("MediaType = %q, want %q", img.MediaType, MediaImage)
	}
	if img.Copyright != "Test Photographer" {
		t.Errorf("Copyright = %q, want %q", img.Copyright, "Test Photographer")
	}
	if img.ServiceVersion != "v1" {
		t.Errorf("ServiceVersion = %q, want %q", img.ServiceVersion, "v1")
	}
	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !img.Date.Equal(expectedDate) {
		t.Errorf("Date = %v, want %v", img.Date, expectedDate)
	}
}

func TestAPODService_Get(t *testing.T) {
	var capturedQuery string
	c := setupAPODTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		data, _ := os.ReadFile("testdata/apod_single.json")
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing get response: %v", err)
		}
	})

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	img, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if img.Title != "Test APOD Title" {
		t.Errorf("Title = %q, want %q", img.Title, "Test APOD Title")
	}

	// Verify date and thumbs params were sent
	if !strings.Contains(capturedQuery, "date=2024-01-15") {
		t.Errorf("query %q should contain date=2024-01-15", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "thumbs=true") {
		t.Errorf("query %q should contain thumbs=true", capturedQuery)
	}
}

func TestAPODService_Get_VideoType(t *testing.T) {
	c := setupAPODTestClient(t, apodFixtureHandler(t, "testdata/apod_single_video.json"))

	date := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	img, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if img.MediaType != MediaVideo {
		t.Errorf("MediaType = %q, want %q", img.MediaType, MediaVideo)
	}
	if img.HDURL != "" {
		t.Errorf("HDURL = %q, want empty for video", img.HDURL)
	}
	if img.ThumbnailURL == "" {
		t.Error("ThumbnailURL should be populated for video entries")
	}
	if img.ThumbnailURL != "https://img.youtube.com/vi/abc123/0.jpg" {
		t.Errorf("ThumbnailURL = %q, want youtube thumbnail", img.ThumbnailURL)
	}
}

func TestAPODService_GetRange(t *testing.T) {
	c := setupAPODTestClient(t, apodFixtureHandler(t, "testdata/apod_range.json"))

	start := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)
	imgs, err := c.APOD.GetRange(context.Background(), start, end)
	if err != nil {
		t.Fatalf("GetRange() error: %v", err)
	}

	if len(imgs) != 3 {
		t.Fatalf("got %d images, want 3", len(imgs))
	}

	if imgs[0].Title != "Range Image One" {
		t.Errorf("imgs[0].Title = %q, want %q", imgs[0].Title, "Range Image One")
	}
	if imgs[1].MediaType != MediaVideo {
		t.Errorf("imgs[1].MediaType = %q, want %q", imgs[1].MediaType, MediaVideo)
	}
	if imgs[2].Copyright != "Photographer Three" {
		t.Errorf("imgs[2].Copyright = %q, want %q", imgs[2].Copyright, "Photographer Three")
	}
}

func TestAPODService_Random(t *testing.T) {
	var capturedQuery string
	c := setupAPODTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		data, _ := os.ReadFile("testdata/apod_range.json")
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing random response: %v", err)
		}
	})

	imgs, err := c.APOD.Random(context.Background(), 3)
	if err != nil {
		t.Fatalf("Random() error: %v", err)
	}

	if len(imgs) != 3 {
		t.Fatalf("got %d images, want 3", len(imgs))
	}

	// Verify count param was sent
	if !strings.Contains(capturedQuery, "count=3") {
		t.Errorf("query %q should contain count=3", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "thumbs=true") {
		t.Errorf("query %q should contain thumbs=true", capturedQuery)
	}
}

func TestAPODService_Get_InvalidDate(t *testing.T) {
	c := NewClient(WithAPIKey("TEST_KEY"))

	// Date before earliest APOD (1995-06-16)
	date := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := c.APOD.Get(context.Background(), date)
	if err == nil {
		t.Fatal("expected error for date before earliest APOD")
	}
	if !strings.Contains(err.Error(), "before the earliest APOD") {
		t.Errorf("error = %q, should mention earliest APOD", err.Error())
	}
}

func TestAPODService_Get_FutureDate(t *testing.T) {
	c := NewClient(WithAPIKey("TEST_KEY"))

	future := time.Now().AddDate(0, 0, 7)
	_, err := c.APOD.Get(context.Background(), future)
	if err == nil {
		t.Fatal("expected error for future date")
	}
	if !strings.Contains(err.Error(), "in the future") {
		t.Errorf("error = %q, should mention 'in the future'", err.Error())
	}
}

func TestAPODService_GetRange_InvalidRange(t *testing.T) {
	c := NewClient(WithAPIKey("TEST_KEY"))

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err := c.APOD.GetRange(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error when start > end")
	}
	if !strings.Contains(err.Error(), "after end date") {
		t.Errorf("error = %q, should mention start after end", err.Error())
	}
}

func TestAPODService_Get_APIError(t *testing.T) {
	c := setupAPODTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`)); err != nil {
			t.Fatalf("writing api error response: %v", err)
		}
	})

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := c.APOD.Get(context.Background(), date)
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Errorf("error type = %T, want *RateLimitError", err)
	}
}

func TestAPODService_Get_NotFound(t *testing.T) {
	data, err := os.ReadFile("testdata/apod_error_404.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	c := setupAPODTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing not found response: %v", err)
		}
	})

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err = c.APOD.Get(context.Background(), date)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("error type = %T, want *NotFoundError", err)
	}
}
