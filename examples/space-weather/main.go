// Package main demonstrates all 11 DONKI space weather event types over a
// 30-day window, printing a summary of each event category.
//
// Set the NASA_API_KEY environment variable before running, or it will
// fall back to DEMO_KEY (rate-limited).
//
// Usage:
//
//	go run ./examples/space-weather
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/peteretelej/nasa"
)

func printSummary(name string, count int, detail string) {
	fmt.Printf("  %-25s %d events%s\n", name+":", count, detail)
}

func main() {
	client := nasa.NewClient()
	ctx := context.Background()

	start := time.Now().AddDate(0, 0, -30)
	end := time.Now()

	fmt.Println("=== DONKI Space Weather Summary (last 30 days) ===")

	// 1. Coronal Mass Ejections
	cmes, err := client.DONKI.CME(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail := ""
	if len(cmes) > 0 {
		detail = fmt.Sprintf(" - first: %s at %s",
			cmes[0].StartTime.Format("2006-01-02 15:04"), cmes[0].SourceLocation)
	}
	printSummary("CME", len(cmes), detail)

	// 2. CME Analysis
	analyses, err := client.DONKI.CMEAnalysis(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(analyses) > 0 {
		detail = fmt.Sprintf(" - first: speed=%.0f km/s, type=%s",
			analyses[0].Speed, analyses[0].Type)
	}
	printSummary("CME Analysis", len(analyses), detail)

	// 3. Geomagnetic Storms
	gsts, err := client.DONKI.GST(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(gsts) > 0 && len(gsts[0].AllKpIndex) > 0 {
		maxKp := gsts[0].AllKpIndex[0].KpIndex
		for _, kp := range gsts[0].AllKpIndex[1:] {
			if kp.KpIndex > maxKp {
				maxKp = kp.KpIndex
			}
		}
		detail = fmt.Sprintf(" - first event max Kp=%.1f", maxKp)
	}
	printSummary("Geomagnetic Storm", len(gsts), detail)

	// 4. Interplanetary Shocks
	ips, err := client.DONKI.IPS(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(ips) > 0 {
		detail = fmt.Sprintf(" - first: %s at %s",
			ips[0].EventTime.Format("2006-01-02 15:04"), ips[0].Location)
	}
	printSummary("Interplanetary Shock", len(ips), detail)

	// 5. Solar Flares
	flrs, err := client.DONKI.FLR(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(flrs) > 0 {
		detail = fmt.Sprintf(" - first: class %s, peak %s",
			flrs[0].ClassType, flrs[0].PeakTime.Format("2006-01-02 15:04"))
	}
	printSummary("Solar Flare", len(flrs), detail)

	// 6. Solar Energetic Particles
	seps, err := client.DONKI.SEP(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(seps) > 0 {
		detail = fmt.Sprintf(" - first: %s", seps[0].EventTime.Format("2006-01-02 15:04"))
	}
	printSummary("Solar Energetic Particle", len(seps), detail)

	// 7. Magnetopause Crossings
	mpcs, err := client.DONKI.MPC(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(mpcs) > 0 {
		detail = fmt.Sprintf(" - first: %s", mpcs[0].EventTime.Format("2006-01-02 15:04"))
	}
	printSummary("Magnetopause Crossing", len(mpcs), detail)

	// 8. Radiation Belt Enhancements
	rbes, err := client.DONKI.RBE(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(rbes) > 0 {
		detail = fmt.Sprintf(" - first: %s", rbes[0].EventTime.Format("2006-01-02 15:04"))
	}
	printSummary("Radiation Belt Enhancement", len(rbes), detail)

	// 9. High Speed Streams
	hsss, err := client.DONKI.HSS(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(hsss) > 0 {
		detail = fmt.Sprintf(" - first: %s", hsss[0].EventTime.Format("2006-01-02 15:04"))
	}
	printSummary("High Speed Stream", len(hsss), detail)

	// 10. WSA-Enlil Simulations
	enlils, err := client.DONKI.WSAEnlil(ctx, start, end)
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(enlils) > 0 {
		detail = fmt.Sprintf(" - first sim: %s", enlils[0].SimulationID)
	}
	printSummary("WSA-Enlil Simulation", len(enlils), detail)

	// 11. Notifications
	notifs, err := client.DONKI.Notifications(ctx, start, end, "all")
	if err != nil {
		log.Fatal(err)
	}
	detail = ""
	if len(notifs) > 0 {
		detail = fmt.Sprintf(" - first type: %s", notifs[0].MessageType)
	}
	printSummary("Notification", len(notifs), detail)
}
