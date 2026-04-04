// Package main combines multiple NASA services into a daily space summary,
// demonstrating cross-service client usage with shared rate limiting.
//
// Set the NASA_API_KEY environment variable before running, or it will
// fall back to DEMO_KEY (rate-limited).
//
// Usage:
//
//	go run ./examples/daily-briefing
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

	today := time.Now()
	weekAgo := today.AddDate(0, 0, -7)

	// APOD section.
	fmt.Println("=== Today's APOD ===")
	apod, err := client.APOD.Today(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Title: %s\n", apod.Title)
	fmt.Printf("  URL:   %s\n", apod.URL)

	// NEO section.
	fmt.Println("\n=== Near-Earth Objects This Week ===")
	feed, err := client.NEO.Feed(ctx, weekAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	var totalObjects, hazardous int
	for _, neos := range feed.NearEarthObjects {
		totalObjects += len(neos)
		for _, obj := range neos {
			if obj.IsPotentiallyHazardous {
				hazardous++
			}
		}
	}
	fmt.Printf("  %d objects tracked, %d potentially hazardous\n", totalObjects, hazardous)

	// Solar activity section.
	fmt.Println("\n=== Solar Activity (7 days) ===")
	cmes, err := client.DONKI.CME(ctx, weekAgo, today)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Coronal mass ejections: %d\n", len(cmes))

	flares, err := client.DONKI.FLR(ctx, weekAgo, today)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Solar flares: %d\n", len(flares))

	// Earth events section.
	fmt.Println("\n=== Active Earth Events ===")
	events, err := client.EONET.Events(ctx, nasa.EONETWithLimit(5), nasa.EONETWithStatus("open"))
	if err != nil {
		log.Fatal(err)
	}

	for _, ev := range events {
		catTitle := "Uncategorized"
		if len(ev.Categories) > 0 {
			catTitle = ev.Categories[0].Title
		}
		fmt.Printf("  [%s] %s\n", catTitle, ev.Title)
	}
}
