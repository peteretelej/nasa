package nasa

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAPODGet_NoCopyrightField(t *testing.T) {
	fixture := loadFixture(t, "apod_no_copyright.json")

	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	})

	date := time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC)
	img, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	if img.Copyright != "" {
		t.Errorf("Copyright = %q, want empty string for public domain entry", img.Copyright)
	}
	if img.Title != "Public Domain APOD" {
		t.Errorf("Title = %q, want %q", img.Title, "Public Domain APOD")
	}
	if img.HDURL == "" {
		t.Error("HDURL should be populated for image entry without copyright")
	}
}

func TestAPODGet_SingleObjectVsArrayResponse(t *testing.T) {
	// Use the same data for both single and array responses to test
	// the decoding branching logic.
	singleFixture := loadFixture(t, "apod_single.json")

	// Wrap the single object in an array for GetRange.
	arrayFixture, err := json.Marshal([]json.RawMessage{json.RawMessage(singleFixture)})
	if err != nil {
		t.Fatalf("marshaling array fixture: %v", err)
	}

	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		// If count or start_date param present, serve array; otherwise single.
		q := r.URL.Query()
		if q.Get("start_date") != "" || q.Get("count") != "" {
			w.Write(arrayFixture)
		} else {
			w.Write(singleFixture)
		}
	})

	t.Run("single object via Get", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		img, err := c.APOD.Get(context.Background(), date)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if img.Title != "Test APOD Title" {
			t.Errorf("Title = %q, want %q", img.Title, "Test APOD Title")
		}
	})

	t.Run("array via GetRange", func(t *testing.T) {
		start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		imgs, err := c.APOD.GetRange(context.Background(), start, end)
		if err != nil {
			t.Fatalf("GetRange() error: %v", err)
		}
		if len(imgs) != 1 {
			t.Fatalf("got %d images, want 1", len(imgs))
		}
		if imgs[0].Title != "Test APOD Title" {
			t.Errorf("Title = %q, want %q", imgs[0].Title, "Test APOD Title")
		}
	})
}

func TestAPODGetRange_EmptyRange(t *testing.T) {
	// When start == end, the API returns a single-element array.
	singleFixture := loadFixture(t, "apod_single.json")
	arrayFixture, err := json.Marshal([]json.RawMessage{json.RawMessage(singleFixture)})
	if err != nil {
		t.Fatalf("marshaling array fixture: %v", err)
	}

	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("start_date") != q.Get("end_date") {
			t.Errorf("start_date=%q != end_date=%q, expected equal", q.Get("start_date"), q.Get("end_date"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(arrayFixture)
	})

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	imgs, err := c.APOD.GetRange(context.Background(), date, date)
	if err != nil {
		t.Fatalf("GetRange() error: %v", err)
	}
	if len(imgs) != 1 {
		t.Fatalf("got %d images, want 1 for start==end range", len(imgs))
	}
	if imgs[0].Title != "Test APOD Title" {
		t.Errorf("Title = %q, want %q", imgs[0].Title, "Test APOD Title")
	}
}

func TestAPODRandom_CountZero(t *testing.T) {
	// Random(ctx, 0) should return an error since count < 1.
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for count=0")
	})

	_, err := c.APOD.Random(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for Random(ctx, 0), got nil")
	}
	if !strings.Contains(err.Error(), "count must be at least 1") {
		t.Errorf("error = %q, should mention count constraint", err.Error())
	}
}
