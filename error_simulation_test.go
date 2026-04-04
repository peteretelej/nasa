package nasa

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	// Bind an ephemeral port, then close it to guarantee the port is unbound.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("binding ephemeral port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("closing listener: %v", err)
	}

	c := NewClient(
		WithBaseURL("http://"+addr),
		WithAPIKey("test"),
		WithRateLimit(1000),
	)

	_, err = c.APOD.Today(context.Background())
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

func assertExplicitBodyLimitError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected explicit size-limit error for oversized successful response body")
	}
	if err.Error() != "nasa: response body exceeds 10485760 bytes" {
		t.Fatalf("error = %q, want explicit size-limit error", err.Error())
	}
}

func TestOversizedSuccessBodies_APOD(t *testing.T) {
	single := fmt.Sprintf(`{"date":"2024-01-15","title":"%s","explanation":"ok","url":"https://example.com/image.jpg","media_type":"image","service_version":"v1"}`,
		oversizedText())
	rangeBody := "[" + single + "]"

	t.Run("Today", func(t *testing.T) {
		c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(single))
		})
		_, err := c.APOD.Today(context.Background())
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Get", func(t *testing.T) {
		c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(single))
		})
		_, err := c.APOD.Get(context.Background(), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("GetRange", func(t *testing.T) {
		c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(rangeBody))
		})
		_, err := c.APOD.GetRange(context.Background(), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Random", func(t *testing.T) {
		c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(rangeBody))
		})
		_, err := c.APOD.Random(context.Background(), 1)
		assertExplicitBodyLimitError(t, err)
	})
}

func TestOversizedSuccessBodies_NEO(t *testing.T) {
	neoObject := fmt.Sprintf(`{"id":"1","neo_reference_id":"1","name":"%s","nasa_jpl_url":"https://example.com/neo/1","absolute_magnitude_h":1,"estimated_diameter":{"kilometers":{"estimated_diameter_min":1,"estimated_diameter_max":2},"meters":{"estimated_diameter_min":1,"estimated_diameter_max":2},"miles":{"estimated_diameter_min":1,"estimated_diameter_max":2},"feet":{"estimated_diameter_min":1,"estimated_diameter_max":2}},"is_potentially_hazardous_asteroid":false,"close_approach_data":[],"is_sentry_object":false}`,
		oversizedText())
	feedBody := fmt.Sprintf(`{"links":{"self":"http://api.nasa.gov/neo/rest/v1/feed"},"element_count":1,"near_earth_objects":{"2024-01-15":[%s]}}`, neoObject)
	browseBody := fmt.Sprintf(`{"links":{"self":"http://api.nasa.gov/neo/rest/v1/neo/browse"},"page":{"size":1,"total_elements":1,"total_pages":1,"number":0},"near_earth_objects":[%s]}`, neoObject)

	t.Run("Feed", func(t *testing.T) {
		c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(feedBody))
		})
		_, err := c.NEO.Feed(context.Background(), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Get", func(t *testing.T) {
		c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(neoObject))
		})
		_, err := c.NEO.Get(context.Background(), "1")
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Browse", func(t *testing.T) {
		c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(browseBody))
		})
		_, err := c.NEO.Browse(context.Background(), 0, 1)
		assertExplicitBodyLimitError(t, err)
	})
}

func TestOversizedSuccessBodies_DONKI(t *testing.T) {
	body := fmt.Sprintf(`[{"activityID":"CME-1","catalog":"test","startTime":"2024-01-15T12:00Z","sourceLocation":"%s","link":"https://example.com/cme","instruments":[],"cmeAnalyses":[],"linkedEvents":[]}]`, oversizedText())
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})

	_, err := c.DONKI.CME(context.Background(), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	assertExplicitBodyLimitError(t, err)
}

func TestOversizedSuccessBodies_EONET(t *testing.T) {
	eventsBody := fmt.Sprintf(`{"title":"events","events":[{"id":"E1","title":"%s","categories":[],"sources":[],"geometry":[]}]}`,
		oversizedText())
	eventBody := fmt.Sprintf(`{"id":"E1","title":"%s","categories":[],"sources":[],"geometry":[]}`,
		oversizedText())

	t.Run("Events", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(eventsBody))
		}))
		t.Cleanup(ts.Close)
		svc := &EONETService{client: c, baseURL: ts.URL}
		_, err := svc.Events(context.Background())
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Event", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(eventBody))
		}))
		t.Cleanup(ts.Close)
		svc := &EONETService{client: c, baseURL: ts.URL}
		_, err := svc.Event(context.Background(), "E1")
		assertExplicitBodyLimitError(t, err)
	})
}

func TestOversizedSuccessBodies_Images(t *testing.T) {
	searchBody := fmt.Sprintf(`{"collection":{"items":[{"data":[{"nasa_id":"PIA12345","title":"%s","media_type":"image","date_created":"1969-07-20T00:00:00Z"}],"href":"https://example.com/collection.json","links":[{"href":"https://example.com/thumb.jpg","rel":"preview"}]}],"metadata":{"total_hits":1},"links":[]}}`, oversizedText())
	assetBody := fmt.Sprintf(`{"collection":{"items":[{"href":"https://example.com/%s.jpg"}],"metadata":{"total_hits":1},"links":[]}}`, oversizedText())
	metadataBody := fmt.Sprintf(`{"title":"%s"}`, oversizedText())

	t.Run("Search", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(searchBody))
		}))
		t.Cleanup(ts.Close)
		svc := &ImageService{client: c, baseURL: ts.URL}
		_, err := svc.Search(context.Background(), "apollo")
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Asset", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(assetBody))
		}))
		t.Cleanup(ts.Close)
		svc := &ImageService{client: c, baseURL: ts.URL}
		_, err := svc.Asset(context.Background(), "PIA12345")
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Metadata", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(metadataBody))
		}))
		t.Cleanup(ts.Close)
		svc := &ImageService{client: c, baseURL: ts.URL}
		_, err := svc.Metadata(context.Background(), "PIA12345")
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("Captions", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(oversizedText()))
		}))
		t.Cleanup(ts.Close)
		svc := &ImageService{client: c, baseURL: ts.URL}
		_, err := svc.Captions(context.Background(), "VIDEO-1")
		assertExplicitBodyLimitError(t, err)
	})
}

func TestOversizedSuccessBodies_SSD(t *testing.T) {
	columnarBody := fmt.Sprintf(`{"signature":{"source":"test","version":"1"},"count":1,"fields":["des","dist"],"data":[["%s","0.1"]]}`,
		oversizedText())
	listBody := fmt.Sprintf(`{"signature":{"source":"test","version":"1"},"count":1,"data":[{"des":"%s","fullname":"object","orbit_id":"1","h":"1","min_dv":{"dv":"1","dur":1},"min_dur":{"dv":"1","dur":1},"n_via_traj":1,"occ":"0","obs_flag":"Y"}]}`,
		oversizedText())

	t.Run("CloseApproach", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(columnarBody))
		}))
		t.Cleanup(ts.Close)
		svc := &SSDService{client: c, baseURL: ts.URL}
		_, err := svc.CloseApproach(context.Background())
		assertExplicitBodyLimitError(t, err)
	})

	t.Run("NHATS", func(t *testing.T) {
		c := NewClient(WithAPIKey("TEST_KEY"), WithRateLimit(1000))
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listBody))
		}))
		t.Cleanup(ts.Close)
		svc := &SSDService{client: c, baseURL: ts.URL}
		_, err := svc.NHATS(context.Background())
		assertExplicitBodyLimitError(t, err)
	})
}
