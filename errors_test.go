package nasa

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockResponse builds an *http.Response with the given status code, body, and
// optional headers (key-value pairs).
func mockResponse(statusCode int, body string, headers ...string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	for i := 0; i+1 < len(headers); i += 2 {
		resp.Header.Set(headers[i], headers[i+1])
	}
	return resp
}

func TestErrorString(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		e := &Error{StatusCode: 429, Message: "rate limited"}
		got := e.Error()
		if got != "nasa: 429: rate limited" {
			t.Errorf("got %q, want %q", got, "nasa: 429: rate limited")
		}
	})

	t.Run("without message", func(t *testing.T) {
		e := &Error{StatusCode: 404}
		got := e.Error()
		if got != "nasa: 404: Not Found" {
			t.Errorf("got %q, want %q", got, "nasa: 404: Not Found")
		}
	})
}

func TestErrorsAs(t *testing.T) {
	t.Run("RateLimitError as Error", func(t *testing.T) {
		var err error = &RateLimitError{Err: &Error{StatusCode: 429, Message: "too many requests"}}
		var base *Error
		if !errors.As(err, &base) {
			t.Fatal("errors.As should match *Error from *RateLimitError")
		}
		if base.StatusCode != 429 {
			t.Errorf("StatusCode = %d, want 429", base.StatusCode)
		}
	})

	t.Run("AuthError as Error", func(t *testing.T) {
		var err error = &AuthError{Err: &Error{StatusCode: 401, Message: "unauthorized"}}
		var base *Error
		if !errors.As(err, &base) {
			t.Fatal("errors.As should match *Error from *AuthError")
		}
		if base.StatusCode != 401 {
			t.Errorf("StatusCode = %d, want 401", base.StatusCode)
		}
	})

	t.Run("NotFoundError as Error", func(t *testing.T) {
		var err error = &NotFoundError{Err: &Error{StatusCode: 404, Message: "not found"}}
		var base *Error
		if !errors.As(err, &base) {
			t.Fatal("errors.As should match *Error from *NotFoundError")
		}
		if base.StatusCode != 404 {
			t.Errorf("StatusCode = %d, want 404", base.StatusCode)
		}
	})

	t.Run("APIError as Error", func(t *testing.T) {
		var err error = &APIError{Err: &Error{StatusCode: 500, Message: "internal error"}}
		var base *Error
		if !errors.As(err, &base) {
			t.Fatal("errors.As should match *Error from *APIError")
		}
		if base.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", base.StatusCode)
		}
	})

	t.Run("RateLimitError as RateLimitError", func(t *testing.T) {
		var err error = &RateLimitError{Err: &Error{StatusCode: 429}}
		var rlErr *RateLimitError
		if !errors.As(err, &rlErr) {
			t.Fatal("errors.As should match *RateLimitError")
		}
	})

	t.Run("error string contains status code", func(t *testing.T) {
		errs := []error{
			&RateLimitError{Err: &Error{StatusCode: 429, Message: "limited"}},
			&AuthError{Err: &Error{StatusCode: 401, Message: "unauthorized"}},
			&NotFoundError{Err: &Error{StatusCode: 404, Message: "not found"}},
			&APIError{Err: &Error{StatusCode: 500, Message: "server error"}},
		}
		codes := []string{"429", "401", "404", "500"}
		for i, err := range errs {
			if !strings.Contains(err.Error(), codes[i]) {
				t.Errorf("error %q should contain status code %s", err.Error(), codes[i])
			}
		}
	})
}

func TestNewErrorTypes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantType   string
	}{
		{"429 returns RateLimitError", http.StatusTooManyRequests, "RateLimitError"},
		{"401 returns AuthError", http.StatusUnauthorized, "AuthError"},
		{"403 returns AuthError", http.StatusForbidden, "AuthError"},
		{"404 returns NotFoundError", http.StatusNotFound, "NotFoundError"},
		{"500 returns APIError", http.StatusInternalServerError, "APIError"},
		{"502 returns APIError", http.StatusBadGateway, "APIError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mockResponse(tt.statusCode, "")
			err := newError(resp)

			switch tt.wantType {
			case "RateLimitError":
				var target *RateLimitError
				if !errors.As(err, &target) {
					t.Errorf("expected *RateLimitError, got %T", err)
				}
			case "AuthError":
				var target *AuthError
				if !errors.As(err, &target) {
					t.Errorf("expected *AuthError, got %T", err)
				}
			case "NotFoundError":
				var target *NotFoundError
				if !errors.As(err, &target) {
					t.Errorf("expected *NotFoundError, got %T", err)
				}
			case "APIError":
				var target *APIError
				if !errors.As(err, &target) {
					t.Errorf("expected *APIError, got %T", err)
				}
			}
		})
	}
}

func TestNewErrorRetryAfter(t *testing.T) {
	t.Run("429 with Retry-After header", func(t *testing.T) {
		resp := mockResponse(http.StatusTooManyRequests, "", "Retry-After", "60")
		err := newError(resp)

		var rlErr *RateLimitError
		if !errors.As(err, &rlErr) {
			t.Fatal("expected *RateLimitError")
		}
		want := 60 * time.Second
		if rlErr.Err.RetryAfter != want {
			t.Errorf("RetryAfter = %v, want %v", rlErr.Err.RetryAfter, want)
		}
	})

	t.Run("429 without Retry-After header", func(t *testing.T) {
		resp := mockResponse(http.StatusTooManyRequests, "")
		err := newError(resp)

		var rlErr *RateLimitError
		if !errors.As(err, &rlErr) {
			t.Fatal("expected *RateLimitError")
		}
		if rlErr.Err.RetryAfter != 0 {
			t.Errorf("RetryAfter = %v, want 0", rlErr.Err.RetryAfter)
		}
	})

	t.Run("429 with non-numeric Retry-After", func(t *testing.T) {
		resp := mockResponse(http.StatusTooManyRequests, "", "Retry-After", "not-a-number")
		err := newError(resp)

		var rlErr *RateLimitError
		if !errors.As(err, &rlErr) {
			t.Fatal("expected *RateLimitError")
		}
		if rlErr.Err.RetryAfter != 0 {
			t.Errorf("RetryAfter = %v, want 0 for non-numeric header", rlErr.Err.RetryAfter)
		}
	})
}

func TestNewErrorBodyParsing(t *testing.T) {
	t.Run("JSON error.message field", func(t *testing.T) {
		body := `{"error":{"message":"rate limited"}}`
		resp := mockResponse(http.StatusTooManyRequests, body)
		err := newError(resp)

		if !strings.Contains(err.Error(), "rate limited") {
			t.Errorf("error %q should contain 'rate limited'", err.Error())
		}
	})

	t.Run("JSON msg field", func(t *testing.T) {
		body := `{"msg":"api key invalid"}`
		resp := mockResponse(http.StatusForbidden, body)
		err := newError(resp)

		if !strings.Contains(err.Error(), "api key invalid") {
			t.Errorf("error %q should contain 'api key invalid'", err.Error())
		}
	})

	t.Run("empty body falls back to status text", func(t *testing.T) {
		resp := mockResponse(http.StatusInternalServerError, "")
		err := newError(resp)

		if !strings.Contains(err.Error(), "Internal Server Error") {
			t.Errorf("error %q should contain 'Internal Server Error'", err.Error())
		}
	})

	t.Run("non-JSON body falls back to status text", func(t *testing.T) {
		resp := mockResponse(http.StatusBadGateway, "<html>Bad Gateway</html>")
		err := newError(resp)

		if !strings.Contains(err.Error(), "Bad Gateway") {
			t.Errorf("error %q should contain 'Bad Gateway'", err.Error())
		}
	})

	t.Run("JSON with neither error.message nor msg", func(t *testing.T) {
		body := `{"code":500,"detail":"something"}`
		resp := mockResponse(http.StatusInternalServerError, body)
		err := newError(resp)

		if !strings.Contains(err.Error(), "Internal Server Error") {
			t.Errorf("error %q should fall back to status text", err.Error())
		}
	})
}
