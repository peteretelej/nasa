package nasa

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestRateLimit_MultiKeyFailover_ServiceLevel verifies that when key1 gets a
// 429, the client rotates to key2 and APOD.Today succeeds.
func TestRateLimit_MultiKeyFailover_ServiceLevel(t *testing.T) {
	var mu sync.Mutex
	seenKeys := make(map[string]int)

	fixture := loadFixture(t, "apod_single.json")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("api_key")
		mu.Lock()
		seenKeys[key]++
		mu.Unlock()

		if key == "key1" {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKeys([]string{"key1", "key2"}),
		WithRateLimit(1000),
	)
	// Force deterministic key order for testing.
	c.rateLimiter.keys.keys = []string{"key1", "key2"}
	c.rateLimiter.keys.current = 0

	img, err := c.APOD.Today(context.Background())
	if err != nil {
		t.Fatalf("expected success after failover, got error: %v", err)
	}
	if img == nil {
		t.Fatal("expected non-nil APOD image")
	}
	if img.Title != "Star Cluster IC 348 from Webb" {
		t.Errorf("expected title 'Star Cluster IC 348 from Webb', got %q", img.Title)
	}

	mu.Lock()
	defer mu.Unlock()
	if seenKeys["key2"] == 0 {
		t.Error("key2 was never used; expected failover to key2")
	}
}

// TestRateLimit_AllKeysExhausted_ServiceLevel verifies that when all keys get
// 429, APOD.Today returns *RateLimitError.
func TestRateLimit_AllKeysExhausted_ServiceLevel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKeys([]string{"key1", "key2"}),
		WithRateLimit(1000),
	)

	_, err := c.APOD.Today(context.Background())
	if err == nil {
		t.Fatal("expected error when all keys exhausted, got nil")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
}

// TestRateLimit_RetryAfterHeader_Respected verifies that the Retry-After header
// controls how long a key stays in cooldown. A single key receives a 429 with
// Retry-After:1, causing the first call to fail. After sleeping >= 1s (letting
// the cooldown expire), the second call succeeds, confirming the Retry-After
// duration was applied to the cooldown.
func TestRateLimit_RetryAfterHeader_Respected(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	fixture := loadFixture(t, "apod_single.json")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		n := callCount
		mu.Unlock()

		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("single-key"),
		WithRateLimit(1000),
	)

	// First call: gets 429, marks key limited for 1s.
	start := time.Now()
	_, err := c.APOD.Today(context.Background())
	if err == nil {
		t.Fatal("expected error on first call (429), got nil")
	}
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}

	// Wait for the Retry-After cooldown to expire.
	time.Sleep(1100 * time.Millisecond)
	elapsed := time.Since(start)

	// Second call: key is available again, returns 200.
	img, err := c.APOD.Today(context.Background())
	if err != nil {
		t.Fatalf("expected success after cooldown, got error: %v", err)
	}
	if img == nil {
		t.Fatal("expected non-nil APOD image")
	}

	if elapsed < 1*time.Second {
		t.Errorf("expected elapsed >= 1s (Retry-After respected), got %v", elapsed)
	}

	mu.Lock()
	defer mu.Unlock()
	if callCount != 2 {
		t.Errorf("expected exactly 2 requests, got %d", callCount)
	}
}
