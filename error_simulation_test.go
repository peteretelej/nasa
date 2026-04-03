package nasa

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestServiceError_APODGet_ServerError verifies that a 500 from the APOD
// endpoint propagates as *APIError through APOD.Get.
func TestServiceError_APODGet_ServerError(t *testing.T) {
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal failure"}}`))
	})

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err := c.APOD.Get(context.Background(), date)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Err.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.Err.StatusCode)
	}
	if !strings.Contains(apiErr.Error(), "internal failure") {
		t.Errorf("expected error to contain 'internal failure', got %q", apiErr.Error())
	}
}

// TestServiceError_NEOFeed_RateLimit verifies that a 429 from the NEO endpoint
// propagates as *RateLimitError through NEO.Feed, including correct Retry-After.
func TestServiceError_NEOFeed_RateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	t.Cleanup(ts.Close)

	// Single key so the client cannot retry with another key.
	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("only-key"), WithRateLimit(1000))

	start := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 4, 3, 0, 0, 0, 0, time.UTC)
	_, err := c.NEO.Feed(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rlErr.Err.RetryAfter != 30*time.Second {
		t.Errorf("expected Retry-After 30s, got %v", rlErr.Err.RetryAfter)
	}
}

// TestServiceError_DONKINotFound verifies that a 404 from a DONKI endpoint
// propagates as *NotFoundError.
func TestServiceError_DONKINotFound(t *testing.T) {
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"no data found"}}`))
	})

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	_, err := c.DONKI.CME(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
	}
	if nfErr.Err.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", nfErr.Err.StatusCode)
	}
}

// TestServiceError_MalformedJSON verifies that a 200 with invalid JSON from
// APOD returns a descriptive parse error (not nil).
func TestServiceError_MalformedJSON(t *testing.T) {
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not json`))
	})

	_, err := c.APOD.Today(context.Background())
	if err == nil {
		t.Fatal("expected JSON parse error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(strings.ToLower(errMsg), "json") &&
		!strings.Contains(strings.ToLower(errMsg), "unmarshal") &&
		!strings.Contains(strings.ToLower(errMsg), "invalid character") {
		t.Errorf("expected error mentioning JSON/unmarshal, got %q", errMsg)
	}
}

// TestServiceError_ConnectionRefused verifies behavior when the base URL points
// to a closed port (no server); expects a network-level error.
func TestServiceError_ConnectionRefused(t *testing.T) {
	c := NewClient(
		WithBaseURL("http://127.0.0.1:1"),
		WithAPIKey("test"),
		WithRateLimit(1000),
	)

	_, err := c.APOD.Today(context.Background())
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// redirectRoundTripper intercepts outgoing requests and redirects them to a
// local httptest server. Used to test services with unexported baseURL fields
// (e.g., SSD) without modifying the SDK.
type redirectRoundTripper struct {
	target *httptest.Server
}

func (rt *redirectRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point at the test server, preserving path and query.
	u := *req.URL
	targetURL := rt.target.URL
	parsed, _ := http.NewRequest(req.Method, targetURL+u.Path+"?"+u.RawQuery, req.Body)
	parsed.Header = req.Header
	return http.DefaultTransport.RoundTrip(parsed)
}

// TestServiceError_EmptyBody200 verifies that a 200 with completely empty body
// from SSD returns a descriptive error (not a nil/zero struct).
// SSD uses getExternal() with an unexported baseURL field, so we use
// WithHTTPClient with a custom RoundTripper to intercept requests.
func TestServiceError_EmptyBody200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Write nothing - completely empty body.
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithAPIKey("test"),
		WithHTTPClient(&http.Client{
			Transport: &redirectRoundTripper{target: ts},
		}),
		WithRateLimit(1000),
	)

	_, err := c.SSD.CloseApproach(context.Background())
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(strings.ToLower(errMsg), "json") &&
		!strings.Contains(strings.ToLower(errMsg), "unmarshal") &&
		!strings.Contains(strings.ToLower(errMsg), "unexpected end") &&
		!strings.Contains(strings.ToLower(errMsg), "eof") {
		t.Errorf("expected JSON/unmarshal error, got %q", errMsg)
	}
}
