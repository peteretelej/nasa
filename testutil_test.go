package nasa

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

var oversizedResponseText = strings.Repeat("x", (1<<20)+1024)

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

func oversizedText() string {
	return oversizedResponseText
}

// setDefaultClientForTest replaces the package-level defaultClient singleton
// with c, so that APODToday / RandomAPOD / ApodImage use a test-controlled
// client. The original client and sync.Once are restored when the test ends.
//
// Must not be used with t.Parallel(); mutates package-level state.
func setDefaultClientForTest(t *testing.T, c *Client) {
	t.Helper()

	origClient := defaultClient
	origOnce := defaultClientOnce

	// Set the mock client and replace the Once with an exhausted instance
	// so getDefaultClient() returns c without re-initialising.
	defaultClient = c
	done := &sync.Once{}
	done.Do(func() {}) // exhaust it
	defaultClientOnce = done

	t.Cleanup(func() {
		defaultClient = origClient
		defaultClientOnce = origOnce
	})
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
