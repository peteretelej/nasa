// Package main demonstrates all SSD/CNEOS services for asteroid and fireball
// data, including close approaches, fireballs, NHATS targets, Scout objects,
// Sentry risk assessment, and the Small-Body Database.
//
// No API key is required (uses ssd-api.jpl.nasa.gov).
//
// Usage:
//
//	go run ./examples/close-approaches
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

	// --- Close Approach Data ---
	fmt.Println("=== Close Approach Data (upcoming Earth encounters) ===")

	approaches, err := client.SSD.CloseApproach(ctx)
	if err != nil {
		log.Fatal(err)
	}

	limit := 10
	if len(approaches) < limit {
		limit = len(approaches)
	}

	fmt.Printf("%-12s %-20s %10s %10s\n", "Designation", "Close Date", "Dist (AU)", "V_rel km/s")
	fmt.Println("---------------------------------------------------------------")
	for i := 0; i < limit; i++ {
		a := approaches[i]
		fmt.Printf("%-12s %-20s %10.4f %10.2f\n", a.Designation, a.CloseDate, a.Distance, a.VRelative)
	}

	// --- Fireball Data ---
	fmt.Println("\n=== Recent Fireball Events ===")

	fireballs, err := client.SSD.Fireball(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fbLimit := 5
	if len(fireballs) < fbLimit {
		fbLimit = len(fireballs)
	}

	for i := 0; i < fbLimit; i++ {
		fb := fireballs[i]
		loc := "unknown location"
		if fb.LatDir != "" && fb.LonDir != "" {
			loc = fmt.Sprintf("%.1f %s, %.1f %s", fb.Lat, fb.LatDir, fb.Lon, fb.LonDir)
		}
		fmt.Printf("  %s  Location: %s  Energy: %.1f J  Impact-E: %.2f kt  Velocity: %.1f km/s\n",
			fb.Date, loc, fb.Energy, fb.ImpactE, fb.Velocity)
	}

	// --- NHATS (Human-Accessible NEOs) ---
	fmt.Println("\n=== NHATS: Human-Accessible Near-Earth Objects (delta-v <= 6 km/s) ===")

	nhats, err := client.SSD.NHATS(ctx, nasa.NHATSWithDV(6))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total NHATS objects: %d\n", len(nhats))

	nhLimit := 3
	if len(nhats) < nhLimit {
		nhLimit = len(nhats)
	}
	for i := 0; i < nhLimit; i++ {
		obj := nhats[i]
		fmt.Printf("  %s (%s) - min delta-v: %s km/s\n", obj.Designation, obj.Fullname, obj.MinDV.DV)
	}

	// --- Scout (NEOCP objects) ---
	fmt.Println("\n=== Scout: Near-Earth Object Confirmation Page ===")

	scouts, err := client.SSD.Scout(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total Scout objects: %d\n", len(scouts))

	scLimit := 3
	if len(scouts) < scLimit {
		scLimit = len(scouts)
	}
	for i := 0; i < scLimit; i++ {
		obj := scouts[i]
		fmt.Printf("  %s - NEO score: %d, observations: %d\n", obj.ObjectName, obj.NeoScore, obj.NObs)
	}

	// --- Sentry (Impact Risk) ---
	fmt.Println("\n=== Sentry: Impact Risk Assessment ===")

	sentries, err := client.SSD.Sentry(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total Sentry objects: %d\n", len(sentries))

	seLimit := 3
	if len(sentries) < seLimit {
		seLimit = len(sentries)
	}
	for i := 0; i < seLimit; i++ {
		obj := sentries[i]
		fmt.Printf("  %s - impacts: %d, probability: %s\n", obj.Designation, obj.NImpacts, obj.IP)
	}

	// --- SBDB: Small-Body Database for Apophis ---
	fmt.Println("\n=== SBDB: Asteroid 99942 Apophis ===")

	apophis, err := client.SSD.SBDB(ctx, "99942")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name:   %s\n", apophis.Object.Fullname)
	fmt.Printf("Kind:   %s\n", apophis.Object.Kind)
	fmt.Printf("NEO:    %t\n", apophis.Object.NEO)
	fmt.Printf("PHA:    %t\n", apophis.Object.PHA)

	fmt.Println("\nOrbital Elements:")
	fmt.Printf("  Eccentricity (e):      %s\n", apophis.Element("e"))
	fmt.Printf("  Semi-major axis (a):   %s AU\n", apophis.Element("a"))
	fmt.Printf("  Inclination (i):       %s deg\n", apophis.Element("i"))
}
