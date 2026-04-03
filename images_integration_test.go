//go:build integration

package nasa

import (
	"context"
	"testing"
)

func TestIntegration_Images_Search(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	results, err := client.Images.Search(ctx, "mars rover")
	if err != nil {
		t.Fatalf("Images.Search() error: %v", err)
	}
	if results.TotalHits <= 0 {
		t.Errorf("Images.Search() TotalHits = %d, want > 0", results.TotalHits)
	}
	if len(results.Items) == 0 {
		t.Fatal("Images.Search() returned 0 items, want > 0")
	}

	first := results.Items[0]
	if first.Title == "" {
		t.Error("first item has empty Title")
	}
	if first.NasaID == "" {
		t.Error("first item has empty NasaID")
	}
	if first.MediaType == "" {
		t.Error("first item has empty MediaType")
	}

	sleepBetweenCalls(t)
}

func TestIntegration_Images_Asset(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Use a well-known NASA ID.
	manifest, err := client.Images.Asset(ctx, "PIA06890")
	if err != nil {
		t.Fatalf("Images.Asset() error: %v", err)
	}
	if len(manifest.Items) == 0 {
		t.Fatal("Images.Asset() returned 0 items, want > 0")
	}

	sleepBetweenCalls(t)
}
