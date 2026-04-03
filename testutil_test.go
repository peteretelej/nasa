package nasa

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// setupMockServer creates an httptest.Server with the given handler and returns
// a Client configured to use it. The server is automatically closed when the
// test completes.
func setupMockServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"), WithRateLimit(1000))
}

// loadFixture reads a file from testdata/ and returns its contents.
// It calls t.Fatalf if the file cannot be read.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("loadFixture(%q): %v", name, err)
	}
	return data
}

// assertErrorType asserts that err is of type T using errors.As.
// It fails the test if err is nil or not the expected type.
func assertErrorType[T any](t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var target T
	if !errors.As(err, &target) {
		t.Errorf("error type = %T, want %T", err, target)
	}
}

func TestAssertErrorType(t *testing.T) {
	t.Run("matches correct type", func(t *testing.T) {
		err := &RateLimitError{Err: &Error{StatusCode: 429, Message: "rate limited"}}
		assertErrorType[*RateLimitError](t, err)
	})

	t.Run("matches wrapped type", func(t *testing.T) {
		err := &NotFoundError{Err: &Error{StatusCode: 404, Message: "not found"}}
		assertErrorType[*NotFoundError](t, err)
	})
}
