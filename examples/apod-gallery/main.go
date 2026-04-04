// Package main demonstrates the APOD (Astronomy Picture of the Day) service
// by fetching yesterday's picture details and a week's worth of entries.
//
// Set the NASA_API_KEY environment variable before running, or it will
// fall back to DEMO_KEY (rate-limited).
//
// Usage:
//
//	go run ./examples/apod-gallery
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()
	ctx := context.Background()

	// Fetch yesterday's APOD.
	yesterday := time.Now().AddDate(0, 0, -1)
	apod, err := client.APOD.Get(ctx, yesterday)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Yesterday's Astronomy Picture of the Day ===")
	fmt.Printf("Title:      %s\n", apod.Title)
	fmt.Printf("Date:       %s\n", apod.Date.Format("2006-01-02"))
	fmt.Printf("Media Type: %s\n", apod.MediaType)

	explanation := apod.Explanation
	if len(explanation) > 100 {
		explanation = explanation[:100] + "..."
	}
	fmt.Printf("Explanation: %s\n", explanation)

	fmt.Printf("URL:        %s\n", apod.URL)
	if apod.HDURL != "" {
		fmt.Printf("HD URL:     %s\n", apod.HDURL)
	}
	if apod.Copyright != "" {
		fmt.Printf("Copyright:  %s\n", apod.Copyright)
	}
	if apod.ThumbnailURL != "" {
		fmt.Printf("Thumbnail:  %s\n", apod.ThumbnailURL)
	}

	// Fetch the past week of APOD entries.
	today := time.Now()
	weekAgo := today.AddDate(0, 0, -7)
	entries, err := client.APOD.GetRange(ctx, weekAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n=== APOD Gallery: Last 7 Days (%d entries) ===\n", len(entries))
	for _, e := range entries {
		mediaLabel := "IMG"
		if e.MediaType == nasa.MediaVideo {
			mediaLabel = "VID"
		}
		fmt.Printf("  [%s] %s - %s\n", mediaLabel, e.Date.Format("2006-01-02"), e.Title)
	}
}
