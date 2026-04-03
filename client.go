package nasa

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client is a NASA API client with rate limiting and multi-key support.
type Client struct {
	httpClient  *http.Client
	apiKeys     []string
	baseURL     string
	rateLimiter *rateLimiter
	userAgent   string
	rps         float64 // rate in requests per second, used during construction
	rpsSet      bool    // true if WithRateLimit was explicitly called

	// Services
	APOD  *APODService
	NEO   *NEOService
	DONKI *DONKIService
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets a single API key.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKeys = []string{key} }
}

// WithAPIKeys sets multiple API keys for rotation.
func WithAPIKeys(keys []string) Option {
	return func(c *Client) { c.apiKeys = keys }
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithBaseURL overrides the base URL (useful for testing).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithRateLimit sets the rate limit in requests per second.
func WithRateLimit(rps float64) Option {
	return func(c *Client) {
		c.rps = rps
		c.rpsSet = true
	}
}

// NewClient creates a new NASA API client with the given options.
// If no API key is provided, it reads from NASA_API_KEY or NASAKEY environment
// variables, falling back to DEMO_KEY.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.nasa.gov",
		userAgent:  "go-nasa-sdk",
		rps:        1000.0 / 3600.0, // ~0.278 rps, registered key default
	}

	for _, o := range opts {
		o(c)
	}

	// Resolve API key if none provided.
	if len(c.apiKeys) == 0 {
		if key := os.Getenv("NASA_API_KEY"); key != "" {
			c.apiKeys = []string{key}
		} else if key := os.Getenv("NASAKEY"); key != "" {
			c.apiKeys = []string{key}
		} else {
			c.apiKeys = []string{"DEMO_KEY"}
		}
	}

	// Auto-adjust rate for DEMO_KEY (only if user did not explicitly set a rate).
	if len(c.apiKeys) == 1 && c.apiKeys[0] == "DEMO_KEY" && !c.rpsSet {
		c.rps = 30.0 / 3600.0 // ~0.0083 rps
	}

	// Create rate limiter with final keys and rate.
	c.rateLimiter = newRateLimiter(c.rps, c.apiKeys)

	// Wire services.
	c.APOD = &APODService{client: c}
	c.NEO = &NEOService{client: c}
	c.DONKI = &DONKIService{client: c}

	return c
}

// get performs a GET request to the given API path with optional query params.
// If authenticated is true, the request waits for the rate limiter and injects
// an API key. On 429 responses, it retries with the next available key.
func (c *Client) get(ctx context.Context, path string, params url.Values, authenticated bool) (*http.Response, error) {
	if authenticated {
		if err := c.rateLimiter.wait(ctx); err != nil {
			return nil, err
		}
	}

	// Try each key (for authenticated requests).
	attempts := 1
	if authenticated && len(c.apiKeys) > 0 {
		attempts = len(c.apiKeys)
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		reqParams := cloneValues(params)

		if authenticated {
			key, ok := c.rateLimiter.keys.next()
			if !ok {
				retryAfter := c.shortestCooldown()
				return nil, &RateLimitError{Err: &Error{
					StatusCode: http.StatusTooManyRequests,
					Message:    "all API keys are rate limited",
					RetryAfter: retryAfter,
				}}
			}
			reqParams.Set("api_key", key)
		}

		fullURL := strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
		if encoded := reqParams.Encode(); encoded != "" {
			fullURL += "?" + encoded
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.do(req)
		if err != nil {
			var rlErr *RateLimitError
			if errors.As(err, &rlErr) && authenticated {
				key := reqParams.Get("api_key")
				cooldown := cooldownRateLimit
				if rlErr.Err.RetryAfter > 0 {
					cooldown = rlErr.Err.RetryAfter
				}
				c.rateLimiter.keys.markLimited(key, cooldown)
				lastErr = err
				continue
			}
			return nil, err
		}

		// Adjust rate from response headers for authenticated requests.
		if authenticated {
			c.rateLimiter.adjustFromHeaders(resp.Header)
		}
		return resp, nil
	}

	return nil, lastErr
}

// do executes an HTTP request, setting the User-Agent header. On success (2xx),
// the response is returned and the caller owns the body. On non-2xx, a typed
// error is returned and the body is consumed and closed by newError.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}

	return nil, newError(resp)
}

// shortestCooldown returns the shortest remaining cooldown across all limited keys.
func (c *Client) shortestCooldown() time.Duration {
	return c.rateLimiter.keys.shortestCooldown()
}

// cloneValues returns a shallow copy of the provided URL values. If v is nil,
// an empty url.Values is returned.
func cloneValues(v url.Values) url.Values {
	if v == nil {
		return url.Values{}
	}
	c := make(url.Values, len(v))
	for k, vs := range v {
		c[k] = append([]string(nil), vs...)
	}
	return c
}
