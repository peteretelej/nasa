// Typed error hierarchy for NASA API responses.

package nasa

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Error is the base error type for all NASA API errors. It carries the HTTP
// status code, a human-readable message extracted from the response body, and
// an optional RetryAfter duration for rate-limited responses.
type Error struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("nasa: %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("nasa: %d: %s", e.StatusCode, http.StatusText(e.StatusCode))
}

// RateLimitError is returned when the API responds with HTTP 429 (Too Many
// Requests). The embedded [Error] includes a RetryAfter hint when the server
// provides one. Use errors.As to check for this type.
type RateLimitError struct {
	Err *Error
}

func (e *RateLimitError) Error() string { return e.Err.Error() }
func (e *RateLimitError) Unwrap() error { return e.Err }

// AuthError is returned when the API responds with HTTP 401 (Unauthorized) or
// 403 (Forbidden), typically indicating an invalid or missing API key.
type AuthError struct {
	Err *Error
}

func (e *AuthError) Error() string { return e.Err.Error() }
func (e *AuthError) Unwrap() error { return e.Err }

// NotFoundError is returned when the API responds with HTTP 404, indicating
// the requested resource does not exist (e.g., invalid date for APOD).
type NotFoundError struct {
	Err *Error
}

func (e *NotFoundError) Error() string { return e.Err.Error() }
func (e *NotFoundError) Unwrap() error { return e.Err }

// APIError is returned for non-2xx HTTP responses that do not match the more
// specific error types (not 401, 403, 404, or 429).
type APIError struct {
	Err *Error
}

func (e *APIError) Error() string { return e.Err.Error() }
func (e *APIError) Unwrap() error { return e.Err }

// newError reads the HTTP response and returns an appropriate typed error.
// The response body is read up to 1MB to prevent unbounded memory usage.
// The response body is closed by this function.
func newError(resp *http.Response) error {
	defer resp.Body.Close()

	// Read body up to 1MB.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	msg := ""
	// Try to extract message from JSON response body.
	var apiResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Msg string `json:"msg"`
	}
	if json.Unmarshal(body, &apiResp) == nil {
		if apiResp.Error.Message != "" {
			msg = apiResp.Error.Message
		} else if apiResp.Msg != "" {
			msg = apiResp.Msg
		}
	}
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}

	base := &Error{
		StatusCode: resp.StatusCode,
		Message:    msg,
	}

	// Parse Retry-After header for 429 responses.
	if resp.StatusCode == http.StatusTooManyRequests {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil {
				base.RetryAfter = time.Duration(seconds) * time.Second
			}
		}
		return &RateLimitError{Err: base}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &AuthError{Err: base}
	case http.StatusNotFound:
		return &NotFoundError{Err: base}
	default:
		return &APIError{Err: base}
	}
}
