//go:build integration

package nasa

import (
	"context"
	"testing"
	"time"
)

func TestIntegration_DONKI_CME(t *testing.T) {
	key := loadAPIKey(t)
	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	end := time.Now().UTC()
	start := end.AddDate(0, 0, -30)

	results, err := client.DONKI.CME(ctx, start, end)
	if err != nil {
		t.Fatalf("DONKI.CME() error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("DONKI.CME() returned 0 results, want > 0 for a 30-day window")
	}

	first := results[0]
	if first.ActivityID == "" {
		t.Error("first CME result has empty ActivityID")
	}
	if first.StartTime.IsZero() {
		t.Error("first CME result has zero StartTime")
	}

	sleepBetweenCalls(t)
}

func TestIntegration_DONKI_GST(t *testing.T) {
	key := loadAPIKey(t)
	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	end := time.Now().UTC()
	start := end.AddDate(0, 0, -90)

	results, err := client.DONKI.GST(ctx, start, end)
	if err != nil {
		t.Fatalf("DONKI.GST() error: %v", err)
	}

	// GSTs can be uncommon; a valid (possibly empty) slice is acceptable.
	if results == nil {
		t.Fatal("DONKI.GST() returned nil, want non-nil slice")
	}

	if len(results) > 0 {
		first := results[0]
		if first.GstID == "" {
			t.Error("first GST result has empty GstID")
		}
		if first.StartTime.IsZero() {
			t.Error("first GST result has zero StartTime")
		}
	}

	sleepBetweenCalls(t)
}

func TestIntegration_DONKI_FLR(t *testing.T) {
	key := loadAPIKey(t)
	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	end := time.Now().UTC()
	start := end.AddDate(0, 0, -30)

	results, err := client.DONKI.FLR(ctx, start, end)
	if err != nil {
		t.Fatalf("DONKI.FLR() error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("DONKI.FLR() returned 0 results, want > 0 for a 30-day window")
	}

	first := results[0]
	if first.FlrID == "" {
		t.Error("first FLR result has empty FlrID")
	}
	if first.ClassType == "" {
		t.Error("first FLR result has empty ClassType")
	}
	if first.BeginTime.IsZero() {
		t.Error("first FLR result has zero BeginTime")
	}

	sleepBetweenCalls(t)
}
