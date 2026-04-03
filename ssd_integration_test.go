//go:build integration

package nasa

import (
	"context"
	"testing"
	"time"
)

func TestIntegration_SSD_CloseApproach(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	now := time.Now()
	results, err := client.SSD.CloseApproach(ctx,
		CADWithDateMin(now.AddDate(0, 0, -30)),
		CADWithDateMax(now),
	)
	if err != nil {
		t.Fatalf("SSD.CloseApproach() error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("SSD.CloseApproach() returned 0 results for a 30-day window, want > 0")
	}

	first := results[0]
	if first.Distance == 0 {
		t.Error("first result has Distance == 0, want > 0")
	}
	if first.CloseDate == "" {
		t.Error("first result has empty CloseDate")
	}
	if first.Designation == "" {
		t.Error("first result has empty Designation")
	}

	sleepBetweenCalls(t)
}

func TestIntegration_SSD_Fireball(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	now := time.Now()
	results, err := client.SSD.Fireball(ctx,
		FireballWithDateMin(now.AddDate(0, -6, 0)),
		FireballWithDateMax(now),
	)
	if err != nil {
		t.Fatalf("SSD.Fireball() error: %v", err)
	}
	// Fireballs can be sparse; a non-nil result is sufficient.
	if results == nil {
		t.Fatal("SSD.Fireball() returned nil, want non-nil slice")
	}
	if len(results) > 0 {
		first := results[0]
		if first.Date == "" {
			t.Error("first fireball has empty Date")
		}
	}

	sleepBetweenCalls(t)
}
