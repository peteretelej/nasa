// Package main demonstrates the NEO (Near-Earth Object) service by fetching
// this week's asteroid data, looking up orbital details, and browsing the
// full catalog.
//
// Set the NASA_API_KEY environment variable before running, or it will
// fall back to DEMO_KEY (rate-limited).
//
// Usage:
//
//	go run ./examples/neo-watch
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()
	ctx := context.Background()

	// Fetch this week's near-earth objects.
	today := time.Now()
	weekAgo := today.AddDate(0, 0, -7)
	feed, err := client.NEO.Feed(ctx, weekAgo, today)
	if err != nil {
		log.Fatal(err)
	}

	// Collect all objects into a flat slice.
	var objects []*nasa.NearEarthObject
	for _, neos := range feed.NearEarthObjects {
		objects = append(objects, neos...)
	}

	// Sort by closest approach distance (ascending).
	sort.Slice(objects, func(i, j int) bool {
		di, dj := math.Inf(1), math.Inf(1)
		if len(objects[i].CloseApproachData) > 0 {
			di = float64(objects[i].CloseApproachData[0].MissDistance.Kilometers)
		}
		if len(objects[j].CloseApproachData) > 0 {
			dj = float64(objects[j].CloseApproachData[0].MissDistance.Kilometers)
		}
		return di < dj
	})

	fmt.Printf("=== Near-Earth Objects This Week (%d total) ===\n", len(objects))
	fmt.Printf("%-30s %15s %12s %12s %12s %s\n",
		"Name", "Diameter (km)", "Vel (km/s)", "Miss (lunar)", "Miss (km)", "Hazardous")
	for _, obj := range objects {
		dMin := obj.EstimatedDiameter.Kilometers.Min
		dMax := obj.EstimatedDiameter.Kilometers.Max

		var vel, missLunar, missKm float64
		if len(obj.CloseApproachData) > 0 {
			ca := obj.CloseApproachData[0]
			vel = float64(ca.RelativeVelocity.KilometersPerSecond)
			missLunar = float64(ca.MissDistance.Lunar)
			missKm = float64(ca.MissDistance.Kilometers)
		}

		hazard := " "
		if obj.IsPotentiallyHazardous {
			hazard = "YES"
		}
		fmt.Printf("%-30s %6.3f - %.3f %12.1f %12.2f %12.0f %s\n",
			obj.Name, dMin, dMax, vel, missLunar, missKm, hazard)
	}

	// Look up orbital details for the closest asteroid.
	if len(objects) == 0 {
		fmt.Println("\nNo asteroids found this week.")
		return
	}
	closest := objects[0]
	fmt.Printf("\n=== Orbital Data: %s ===\n", closest.Name)

	detail, err := client.NEO.Get(ctx, closest.NeoReferenceID)
	if err != nil {
		log.Fatal(err)
	}
	if detail.OrbitalData != nil {
		od := detail.OrbitalData
		fmt.Printf("  Orbit ID:       %s\n", od.OrbitID)
		fmt.Printf("  Eccentricity:   %s\n", od.Eccentricity)
		fmt.Printf("  Semi-Major Axis: %s AU\n", od.SemiMajorAxis)
		fmt.Printf("  Inclination:    %s deg\n", od.Inclination)
		fmt.Printf("  Orbital Period: %s days\n", od.OrbitalPeriod)
	}

	// Browse the full NEO catalog.
	browse, err := client.NEO.Browse(ctx, 0, 2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n=== NEO Catalog ===\n")
	fmt.Printf("  Total Objects: %d\n", browse.Page.TotalElements)
	fmt.Printf("  Total Pages:   %d\n", browse.Page.TotalPages)
	fmt.Println("  First 2 entries:")
	for _, obj := range browse.NearEarthObjects {
		fmt.Printf("    - %s\n", obj.Name)
	}
}
