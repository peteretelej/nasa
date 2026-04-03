//go:build integration

package nasa

import (
	"context"
	"errors"
	"testing"
)

func TestIntegration_EONET_Events(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	events, err := client.EONET.Events(ctx,
		EONETWithStatus("open"),
		EONETWithLimit(5),
	)
	if err != nil {
		// EONET API has been returning HTTP 500 errors as of April 2026.
		// Skip rather than hard-fail when the API is unavailable.
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
			t.Skipf("EONET API is unavailable (HTTP %d)", apiErr.StatusCode)
		}
		t.Fatalf("EONET.Events() error: %v", err)
	}

	if len(events) == 0 {
		t.Skip("EONET.Events() returned 0 open events; API may be degraded")
	}

	first := events[0]
	if first.ID == "" {
		t.Error("first event has empty ID")
	}
	if first.Title == "" {
		t.Error("first event has empty Title")
	}
	if len(first.Categories) == 0 {
		t.Error("first event has no Categories")
	}
	if len(first.Geometry) == 0 {
		t.Error("first event has no Geometry")
	}

	sleepBetweenCalls(t)
}

func TestIntegration_EONET_Categories(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	cats, err := client.EONET.Categories(ctx)
	if err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 500 {
			t.Skipf("EONET API is unavailable (HTTP %d)", apiErr.StatusCode)
		}
		t.Fatalf("EONET.Categories() error: %v", err)
	}

	if len(cats) == 0 {
		t.Fatal("EONET.Categories() returned 0 categories, want > 0")
	}

	first := cats[0]
	if first.ID == "" {
		t.Error("first category has empty ID")
	}
	if first.Title == "" {
		t.Error("first category has empty Title")
	}

	sleepBetweenCalls(t)
}
