//go:build integration

package nasa

import (
	"bufio"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// loadAPIKey returns a NASA API key for integration tests.
// It checks NASA_API_KEY, then NASAKEY env var, then .env file fallback.
// Skips the test if no key is found.
func loadAPIKey(t *testing.T) string {
	t.Helper()
	if key := os.Getenv("NASA_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("NASAKEY"); key != "" {
		return key
	}
	// Try .env file fallback.
	if f, err := os.Open(".env"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				if k == "NASA_API_KEY" && v != "" {
					return v
				}
			}
		}
	}
	t.Skip("NASA_API_KEY not set")
	return ""
}

// sleepBetweenCalls pauses between API calls to respect rate limits.
func sleepBetweenCalls(t *testing.T) {
	t.Helper()
	t.Log("sleeping 2s between API calls to respect rate limits")
	time.Sleep(2 * time.Second)
}

func TestIntegration_APOD_Today(t *testing.T) {
	key := loadAPIKey(t)
	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	apod, err := client.APOD.Today(ctx)
	if err != nil {
		t.Fatalf("APOD.Today() error: %v", err)
	}
	if apod.Title == "" {
		t.Error("APOD.Today() returned empty Title")
	}
	if apod.URL == "" {
		t.Error("APOD.Today() returned empty URL")
	}
	// Verify date is within the last 24 hours (avoids timezone flakes).
	now := time.Now().UTC()
	diff := now.Sub(apod.Date)
	if diff < -24*time.Hour || diff > 48*time.Hour {
		t.Errorf("APOD.Today() date %v is not within expected range of %v", apod.Date, now)
	}
}

func TestIntegration_APOD_Get(t *testing.T) {
	key := loadAPIKey(t)
	sleepBetweenCalls(t)

	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	apod, err := client.APOD.Get(ctx, date)
	if err != nil {
		t.Fatalf("APOD.Get(2024-01-01) error: %v", err)
	}
	if apod.Title != "NGC 1232: A Grand Design Spiral Galaxy" {
		t.Errorf("APOD.Get(2024-01-01) title = %q, want %q", apod.Title, "NGC 1232: A Grand Design Spiral Galaxy")
	}
}

func TestIntegration_APOD_GetRange(t *testing.T) {
	key := loadAPIKey(t)
	sleepBetweenCalls(t)

	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	imgs, err := client.APOD.GetRange(ctx, start, end)
	if err != nil {
		t.Fatalf("APOD.GetRange() error: %v", err)
	}
	if len(imgs) < 1 {
		t.Error("APOD.GetRange() returned 0 results, want >= 1")
	}
}

func TestIntegration_NEO_Feed(t *testing.T) {
	key := loadAPIKey(t)
	sleepBetweenCalls(t)

	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	// Use a recent 3-day range.
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -2)
	feed, err := client.NEO.Feed(ctx, start, end)
	if err != nil {
		t.Fatalf("NEO.Feed() error: %v", err)
	}
	if feed.ElementCount <= 0 {
		t.Error("NEO.Feed() ElementCount <= 0")
	}
	if len(feed.NearEarthObjects) == 0 {
		t.Error("NEO.Feed() NearEarthObjects map is empty")
	}
}

func TestIntegration_NEO_Feed_StringNumbers(t *testing.T) {
	key := loadAPIKey(t)
	sleepBetweenCalls(t)

	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -2)
	feed, err := client.NEO.Feed(ctx, start, end)
	if err != nil {
		t.Fatalf("NEO.Feed() error: %v", err)
	}

	// Pick the first asteroid from any date.
	var found bool
	for _, neos := range feed.NearEarthObjects {
		for _, neo := range neos {
			if len(neo.CloseApproachData) > 0 {
				vel := float64(neo.CloseApproachData[0].RelativeVelocity.KilometersPerSecond)
				if vel == 0 {
					t.Error("RelativeVelocity.KilometersPerSecond is 0, expected non-zero float64")
				}
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Error("no asteroids with close approach data found in feed")
	}
}

func TestIntegration_NEO_Feed_HTTPSLinks(t *testing.T) {
	key := loadAPIKey(t)
	sleepBetweenCalls(t)

	client := NewClient(WithAPIKey(key))
	ctx := context.Background()

	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -2)
	feed, err := client.NEO.Feed(ctx, start, end)
	if err != nil {
		t.Fatalf("NEO.Feed() error: %v", err)
	}

	if !strings.HasPrefix(feed.Links.Self, "https://") {
		t.Errorf("feed.Links.Self = %q, want prefix \"https://\"", feed.Links.Self)
	}
}
