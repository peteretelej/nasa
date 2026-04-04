// Package main demonstrates the EONET (Earth Observatory Natural Event
// Tracker) service with filtering, categories, and sources.
//
// Set the NASA_API_KEY environment variable before running, or it will
// fall back to DEMO_KEY (rate-limited).
//
// Usage:
//
//	go run ./examples/earth-events
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()
	ctx := context.Background()

	// Fetch recent open events.
	events, err := client.EONET.Events(ctx, nasa.EONETWithLimit(10), nasa.EONETWithStatus("open"))
	if err != nil {
		log.Fatal(err)
	}

	// Group events by category.
	byCategory := make(map[string][]*nasa.EONETEvent)
	for _, ev := range events {
		for _, cat := range ev.Categories {
			byCategory[cat.Title] = append(byCategory[cat.Title], ev)
		}
	}

	fmt.Println("=== Recent Open Events by Category ===")
	for catTitle, catEvents := range byCategory {
		fmt.Printf("\n%s (%d events)\n", catTitle, len(catEvents))
		for _, ev := range catEvents {
			fmt.Printf("  Title: %s\n", ev.Title)
			if len(ev.Geometry) > 0 {
				g := ev.Geometry[0]
				fmt.Printf("    Date: %s\n", g.Date.Format("2006-01-02 15:04"))
				coords, err := g.PointCoordinates()
				if err == nil {
					fmt.Printf("    Coordinates: %.4f, %.4f\n", coords[0], coords[1])
				}
				if g.MagnitudeValue != nil {
					fmt.Printf("    Magnitude: %.2f %s\n", *g.MagnitudeValue, g.MagnitudeUnit)
				}
			}
		}
	}

	// Fetch details for the first event.
	if len(events) == 0 {
		fmt.Println("\nNo events found.")
		return
	}
	first := events[0]
	detail, err := client.EONET.Event(ctx, first.ID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n=== Event Detail: %s ===\n", detail.Title)
	if detail.Description != "" {
		fmt.Printf("  Description: %s\n", detail.Description)
	}
	fmt.Printf("  Geometry entries: %d\n", len(detail.Geometry))
	for i, g := range detail.Geometry {
		fmt.Printf("    [%d] %s type=%s\n", i, g.Date.Format("2006-01-02"), g.Type)
	}
	fmt.Printf("  Sources: %d\n", len(detail.Sources))
	for _, src := range detail.Sources {
		fmt.Printf("    %s: %s\n", src.ID, src.URL)
	}

	// List all categories.
	categories, err := client.EONET.Categories(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n=== EONET Categories (%d) ===\n", len(categories))
	for _, cat := range categories {
		fmt.Printf("  %s: %s\n", cat.ID, cat.Title)
	}

	// List sources (validates Phase 1 EONETSource bug fix).
	sources, err := client.EONET.Sources(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n=== EONET Sources (%d total) ===\n", len(sources))
	limit := 5
	if len(sources) < limit {
		limit = len(sources)
	}
	for _, src := range sources[:limit] {
		fmt.Printf("  ID: %-10s Title: %-25s URL: %s\n", src.ID, src.Title, src.URL)
		if src.Link != "" {
			fmt.Printf("               Link: %s\n", src.Link)
		}
	}

	// Filter events by days and category.
	wildfire, err := client.EONET.Events(ctx, nasa.EONETWithDays(7), nasa.EONETWithCategory("wildfires"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n=== Wildfires in Last 7 Days: %d events ===\n", len(wildfire))
}
