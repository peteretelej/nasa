package nasa

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewClientDefaults(t *testing.T) {
	// Clear env vars so we get DEMO_KEY fallback.
	t.Setenv("NASA_API_KEY", "")
	t.Setenv("NASAKEY", "")

	c := NewClient()

	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", c.httpClient.Timeout)
	}
	if c.baseURL != "https://api.nasa.gov" {
		t.Errorf("expected https://api.nasa.gov, got %s", c.baseURL)
	}
	if c.userAgent != "go-nasa-sdk" {
		t.Errorf("expected go-nasa-sdk user agent, got %s", c.userAgent)
	}
	if len(c.apiKeys) != 1 || c.apiKeys[0] != "DEMO_KEY" {
		t.Errorf("expected DEMO_KEY, got %v", c.apiKeys)
	}
	if c.rateLimiter == nil {
		t.Fatal("expected rateLimiter to be set")
	}
}

func TestNewClientWithAPIKey(t *testing.T) {
	c := NewClient(WithAPIKey("test-key"))

	if len(c.apiKeys) != 1 || c.apiKeys[0] != "test-key" {
		t.Errorf("expected [test-key], got %v", c.apiKeys)
	}
}

func TestNewClientWithAPIKeys(t *testing.T) {
	c := NewClient(WithAPIKeys([]string{"k1", "k2", "k3"}))

	if len(c.apiKeys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(c.apiKeys))
	}
}

func TestNewClientWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 60 * time.Second}
	c := NewClient(WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("expected custom HTTP client")
	}
	if c.httpClient.Timeout != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", c.httpClient.Timeout)
	}
}

func TestNewClientWithBaseURL(t *testing.T) {
	c := NewClient(WithBaseURL("http://localhost:8080"))

	if c.baseURL != "http://localhost:8080" {
		t.Errorf("expected http://localhost:8080, got %s", c.baseURL)
	}
}

func TestNewClientWithRateLimit(t *testing.T) {
	c := NewClient(WithAPIKey("test"), WithRateLimit(10.0))

	if c.rps != 10.0 {
		t.Errorf("expected rps 10.0, got %f", c.rps)
	}
	if !c.rpsSet {
		t.Error("expected rpsSet to be true")
	}
}

func TestDemoKeyDoesNotOverrideExplicitRateLimit(t *testing.T) {
	t.Setenv("NASA_API_KEY", "")
	t.Setenv("NASAKEY", "")

	c := NewClient(WithAPIKey("DEMO_KEY"), WithRateLimit(1.0))

	if c.rps != 1.0 {
		t.Errorf("expected rps 1.0, got %f (DEMO_KEY auto-rate should not override explicit rate)", c.rps)
	}
}

func TestDemoKeyAutoRate(t *testing.T) {
	t.Setenv("NASA_API_KEY", "")
	t.Setenv("NASAKEY", "")

	c := NewClient() // Falls back to DEMO_KEY, no explicit rate.

	expectedRPS := 30.0 / 3600.0
	if c.rps != expectedRPS {
		t.Errorf("expected DEMO_KEY rps %f, got %f", expectedRPS, c.rps)
	}
}

func TestEnvVarResolutionOrder(t *testing.T) {
	// NASA_API_KEY takes priority over NASAKEY.
	t.Setenv("NASA_API_KEY", "from-nasa-api-key")
	t.Setenv("NASAKEY", "from-nasakey")

	c := NewClient()
	if len(c.apiKeys) != 1 || c.apiKeys[0] != "from-nasa-api-key" {
		t.Errorf("expected from-nasa-api-key, got %v", c.apiKeys)
	}
}

func TestEnvVarFallbackToNASAKEY(t *testing.T) {
	t.Setenv("NASA_API_KEY", "")
	t.Setenv("NASAKEY", "from-nasakey")

	c := NewClient()
	if len(c.apiKeys) != 1 || c.apiKeys[0] != "from-nasakey" {
		t.Errorf("expected from-nasakey, got %v", c.apiKeys)
	}
}

func TestEnvVarFallbackToDemoKey(t *testing.T) {
	t.Setenv("NASA_API_KEY", "")
	t.Setenv("NASAKEY", "")

	c := NewClient()
	if len(c.apiKeys) != 1 || c.apiKeys[0] != "DEMO_KEY" {
		t.Errorf("expected DEMO_KEY, got %v", c.apiKeys)
	}
}

func TestGetWithHTTPTest(t *testing.T) {
	var gotAPIKey string
	var gotUserAgent string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.URL.Query().Get("api_key")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"title": "test"})
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("test-key-123"),
		WithRateLimit(1000), // high rate to avoid blocking
	)

	params := url.Values{}
	params.Set("date", "2024-01-01")

	resp, err := c.get(context.Background(), "/planetary/apod", params, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closeResponseBody(resp.Body)

	if gotAPIKey != "test-key-123" {
		t.Errorf("expected api_key=test-key-123, got %s", gotAPIKey)
	}
	if gotUserAgent != "go-nasa-sdk" {
		t.Errorf("expected User-Agent go-nasa-sdk, got %s", gotUserAgent)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetUnauthenticated(t *testing.T) {
	var gotAPIKey string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.URL.Query().Get("api_key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("should-not-appear"),
	)

	resp, err := c.get(context.Background(), "/test", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closeResponseBody(resp.Body)

	if gotAPIKey != "" {
		t.Errorf("expected no api_key for unauthenticated request, got %s", gotAPIKey)
	}
}

func TestGet429RetryWithKeyRotation(t *testing.T) {
	var requestCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		if count == 1 {
			// First request: return 429.
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		// Second request: return 200.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"title":"success"}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKeys([]string{"key1", "key2"}),
		WithRateLimit(1000),
	)

	resp, err := c.get(context.Background(), "/test", nil, true)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	defer closeResponseBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount.Load() != 2 {
		t.Errorf("expected 2 requests (1 retry), got %d", requestCount.Load())
	}
}

func TestGet429AllKeysLimited(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("only-key"),
		WithRateLimit(1000),
	)

	_, err := c.get(context.Background(), "/test", nil, true)
	if err == nil {
		t.Fatal("expected error when all keys are limited")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestGet429RetryReEntersRateLimiter(t *testing.T) {
	var requestKeys []string
	var requestCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestKeys = append(requestKeys, r.URL.Query().Get("api_key"))

		if requestCount.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"title":"success"}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKeys([]string{"key1", "key2"}),
		WithRateLimit(1000),
	)
	c.rateLimiter.keys.keys = []string{"key1", "key2"}
	c.rateLimiter.keys.current = 0
	c.rateLimiter.limiter = rate.NewLimiter(rate.Every(250*time.Millisecond), 1)
	if !c.rateLimiter.limiter.Allow() {
		t.Fatal("expected to consume initial burst token")
	}

	start := time.Now()
	resp, err := c.get(context.Background(), "/test", nil, true)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	defer closeResponseBody(resp.Body)

	if requestCount.Load() != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount.Load())
	}
	if requestKeys[0] != "key1" || requestKeys[1] != "key2" {
		t.Fatalf("unexpected key order: %v", requestKeys)
	}

	elapsed := time.Since(start)
	if elapsed < 450*time.Millisecond {
		t.Fatalf("total time = %v, want two limiter waits across retry", elapsed)
	}
}

func TestNon2xxAuthError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("bad-key"),
		WithRateLimit(1000),
	)

	_, err := c.get(context.Background(), "/test", nil, true)
	if err == nil {
		t.Fatal("expected error for 401")
	}

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestNon2xxNotFoundError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("test"),
		WithRateLimit(1000),
	)

	_, err := c.get(context.Background(), "/test", nil, true)
	if err == nil {
		t.Fatal("expected error for 404")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestNon2xxAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"server error"}}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("test"),
		WithRateLimit(1000),
	)

	_, err := c.get(context.Background(), "/test", nil, true)
	if err == nil {
		t.Fatal("expected error for 500")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("expected APIError, got %T: %v", err, err)
	}
}

func TestContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow handler - should not complete.
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("test"),
		WithRateLimit(1000),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := c.get(ctx, "/test", nil, true)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestCloneValues(t *testing.T) {
	original := url.Values{
		"key1": {"val1"},
		"key2": {"val2a", "val2b"},
	}

	cloned := cloneValues(original)

	// Modify clone should not affect original.
	cloned.Set("key1", "modified")
	cloned.Add("key3", "val3")

	if original.Get("key1") != "val1" {
		t.Errorf("original was modified: key1 = %s", original.Get("key1"))
	}
	if original.Get("key3") != "" {
		t.Error("original gained unexpected key3")
	}
}

func TestCloneValuesNil(t *testing.T) {
	cloned := cloneValues(nil)
	if cloned == nil {
		t.Fatal("expected non-nil url.Values for nil input")
	}
	if len(cloned) != 0 {
		t.Errorf("expected empty values, got %v", cloned)
	}
}

func TestGetPreservesExistingParams(t *testing.T) {
	var gotDate string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDate = r.URL.Query().Get("date")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(
		WithBaseURL(ts.URL),
		WithAPIKey("test"),
		WithRateLimit(1000),
	)

	params := url.Values{}
	params.Set("date", "2024-01-15")

	resp, err := c.get(context.Background(), "/planetary/apod", params, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closeResponseBody(resp.Body)

	if gotDate != "2024-01-15" {
		t.Errorf("expected date=2024-01-15, got %s", gotDate)
	}
}

func TestGetExternal_CustomBaseURL(t *testing.T) {
	var gotPath string
	var gotQuery string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("test-key"))

	params := url.Values{}
	params.Set("q", "apollo")

	resp, err := c.getExternal(context.Background(), ts.URL, "search", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closeResponseBody(resp.Body)

	if gotPath != "/search" {
		t.Errorf("expected path /search, got %s", gotPath)
	}
	if !strings.Contains(gotQuery, "q=apollo") {
		t.Errorf("expected query to contain q=apollo, got %s", gotQuery)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetExternal_NoAPIKey(t *testing.T) {
	var gotAPIKey string
	var gotUserAgent string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.URL.Query().Get("api_key")
		gotUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("should-not-appear"))

	resp, err := c.getExternal(context.Background(), ts.URL, "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closeResponseBody(resp.Body)

	if gotAPIKey != "" {
		t.Errorf("expected no api_key for external request, got %s", gotAPIKey)
	}
	if gotUserAgent != "go-nasa-sdk" {
		t.Errorf("expected User-Agent go-nasa-sdk, got %s", gotUserAgent)
	}
}

func TestGetExternal_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("test"))

	_, err := c.getExternal(context.Background(), ts.URL, "missing", nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestNewClient_ServicesInitialized(t *testing.T) {
	c := NewClient(WithAPIKey("test"))

	if c.APOD == nil {
		t.Error("expected APOD service to be initialized")
	}
	if c.NEO == nil {
		t.Error("expected NEO service to be initialized")
	}
	if c.DONKI == nil {
		t.Error("expected DONKI service to be initialized")
	}
	if c.Images == nil {
		t.Error("expected Images service to be initialized")
	}
	if c.SSD == nil {
		t.Error("expected SSD service to be initialized")
	}
	if c.EONET == nil {
		t.Error("expected EONET service to be initialized")
	}
}
