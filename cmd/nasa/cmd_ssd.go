package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

func ssdCmd(args []string) {
	if len(args) == 0 {
		ssdUsage(os.Stderr)
		os.Exit(1)
	}
	switch args[0] {
	case "cad":
		ssdCADCmd(args[1:])
	case "fireball":
		ssdFireballCmd(args[1:])
	case "nhats":
		ssdNHATSCmd(args[1:])
	case "scout":
		ssdScoutCmd(args[1:])
	case "sentry":
		ssdSentryCmd(args[1:])
	case "sbdb":
		ssdSBDBCmd(args[1:])
	case "--help", "-help", "-h":
		ssdUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "nasa ssd: unknown command %q\n\n", args[0])
		ssdUsage(os.Stderr)
		os.Exit(1)
	}
}

func ssdUsage(w io.Writer) {
	if _, err := fmt.Fprintf(w, `Usage: nasa ssd <command> [flags]

Commands:
  cad         Close Approach Data
  fireball    Fireball and Bolide Reports
  nhats       NHATS (Human-Accessible NEOs)
  scout       Scout (Impact Hazard Assessment)
  sentry      Sentry (Impact Monitoring)
  sbdb        Small-Body Database Lookup

Use "nasa ssd <command> --help" for more information about a command.
`); err != nil {
		fatal("failed to write output: %v", err)
	}
}

// ssdCADCmd handles "nasa ssd cad".
func ssdCADCmd(args []string) {
	fs := flag.NewFlagSet("ssd cad", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa ssd cad [flags]

Query close-approach data for asteroids and comets.

Flags:
`)
		fs.PrintDefaults()
	}

	dateMin := fs.String("date-min", "", "minimum date (YYYY-MM-DD)")
	dateMax := fs.String("date-max", "", "maximum date (YYYY-MM-DD)")
	distMax := fs.Float64("dist-max", 0, "maximum approach distance (AU)")
	body := fs.String("body", "", "close-approach body (e.g. Earth, Mars)")
	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	var opts []nasa.CADOption
	if *dateMin != "" {
		t, err := time.Parse("2006-01-02", *dateMin)
		if err != nil {
			fatal("ssd cad: invalid --date-min, expected YYYY-MM-DD")
		}
		opts = append(opts, nasa.CADWithDateMin(t))
	}
	if *dateMax != "" {
		t, err := time.Parse("2006-01-02", *dateMax)
		if err != nil {
			fatal("ssd cad: invalid --date-max, expected YYYY-MM-DD")
		}
		opts = append(opts, nasa.CADWithDateMax(t))
	}
	if *distMax > 0 {
		opts = append(opts, nasa.CADWithDistMax(*distMax))
	}
	if *body != "" {
		opts = append(opts, nasa.CADWithBody(*body))
	}

	client := nasa.NewClient()
	ctx := context.Background()

	data, err := client.SSD.CloseApproach(ctx, opts...)
	if err != nil {
		fatal("ssd cad: %v", err)
	}

	if *jsonFlag {
		printJSON(data)
		return
	}

	if len(data) == 0 {
		fmt.Println("No close-approach data found.")
		return
	}

	fmt.Printf("Close Approach Data (%d results)\n\n", len(data))

	tw := newTable("Designation", "Date", "Distance (AU)", "V_rel (km/s)", "H")
	for _, d := range data {
		tablePrintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			d.Designation,
			d.CloseDate,
			formatFloat(d.Distance, 4),
			formatFloat(d.VRelative, 2),
			formatFloat(d.AbsMagnitude, 1),
		)
	}
	flushTable(tw)
}

// ssdFireballCmd handles "nasa ssd fireball".
func ssdFireballCmd(args []string) {
	fs := flag.NewFlagSet("ssd fireball", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa ssd fireball [flags]

Query fireball and bolide event data.

Flags:
`)
		fs.PrintDefaults()
	}

	dateMin := fs.String("date-min", "", "minimum date (YYYY-MM-DD)")
	dateMax := fs.String("date-max", "", "maximum date (YYYY-MM-DD)")
	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	var opts []nasa.FireballOption
	if *dateMin != "" {
		t, err := time.Parse("2006-01-02", *dateMin)
		if err != nil {
			fatal("ssd fireball: invalid --date-min, expected YYYY-MM-DD")
		}
		opts = append(opts, nasa.FireballWithDateMin(t))
	}
	if *dateMax != "" {
		t, err := time.Parse("2006-01-02", *dateMax)
		if err != nil {
			fatal("ssd fireball: invalid --date-max, expected YYYY-MM-DD")
		}
		opts = append(opts, nasa.FireballWithDateMax(t))
	}

	client := nasa.NewClient()
	ctx := context.Background()

	events, err := client.SSD.Fireball(ctx, opts...)
	if err != nil {
		fatal("ssd fireball: %v", err)
	}

	if *jsonFlag {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No fireball events found.")
		return
	}

	fmt.Printf("Fireball Events (%d results)\n\n", len(events))

	tw := newTable("Date", "Latitude", "Longitude", "Energy (kt)", "Velocity (km/s)")
	for _, e := range events {
		lat := formatFireballCoord(e.Lat, e.LatDir)
		lon := formatFireballCoord(e.Lon, e.LonDir)
		tablePrintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			e.Date,
			lat,
			lon,
			formatFloat(e.Energy, 2),
			formatFloat(e.Velocity, 1),
		)
	}
	flushTable(tw)
}

// formatFireballCoord formats a coordinate value with its direction.
func formatFireballCoord(val float64, dir string) string {
	if dir == "" {
		return "-"
	}
	return fmt.Sprintf("%.1f %s", val, dir)
}

// ssdNHATSCmd handles "nasa ssd nhats".
func ssdNHATSCmd(args []string) {
	fs := flag.NewFlagSet("ssd nhats", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa ssd nhats [flags]

List near-Earth objects accessible for human exploration (NHATS).

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	client := nasa.NewClient()
	ctx := context.Background()

	objects, err := client.SSD.NHATS(ctx)
	if err != nil {
		fatal("ssd nhats: %v", err)
	}

	if *jsonFlag {
		printJSON(objects)
		return
	}

	if len(objects) == 0 {
		fmt.Println("No NHATS objects found.")
		return
	}

	fmt.Printf("NHATS Objects (%d results)\n\n", len(objects))

	tw := newTable("Designation", "OCC", "Min DV (km/s)", "Min Duration (days)", "Viable Trajectories")
	for _, o := range objects {
		tablePrintf(tw, "%s\t%s\t%s\t%d\t%d\n",
			o.Designation,
			o.OCC,
			o.MinDV.DV,
			o.MinDuration.Dur,
			o.NViaTraj,
		)
	}
	flushTable(tw)
}

// ssdScoutCmd handles "nasa ssd scout".
func ssdScoutCmd(args []string) {
	fs := flag.NewFlagSet("ssd scout", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa ssd scout [flags]

List objects tracked by the Scout impact hazard assessment system.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	client := nasa.NewClient()
	ctx := context.Background()

	objects, err := client.SSD.Scout(ctx)
	if err != nil {
		fatal("ssd scout: %v", err)
	}

	if *jsonFlag {
		printJSON(objects)
		return
	}

	if len(objects) == 0 {
		fmt.Println("No Scout objects found.")
		return
	}

	fmt.Printf("Scout Objects (%d results)\n\n", len(objects))

	tw := newTable("Object Name", "Rating", "H", "NEO Score", "Observations")
	for _, o := range objects {
		rating := "-"
		if o.Rating != nil {
			rating = fmt.Sprintf("%d", *o.Rating)
		}
		tablePrintf(tw, "%s\t%s\t%s\t%d\t%d\n",
			o.ObjectName,
			rating,
			formatFloat(o.H, 1),
			o.NeoScore,
			o.NObs,
		)
	}
	flushTable(tw)
}

// ssdSentryCmd handles "nasa ssd sentry".
func ssdSentryCmd(args []string) {
	fs := flag.NewFlagSet("ssd sentry", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa ssd sentry [flags]

List objects monitored by the Sentry impact monitoring system.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	client := nasa.NewClient()
	ctx := context.Background()

	objects, err := client.SSD.Sentry(ctx)
	if err != nil {
		fatal("ssd sentry: %v", err)
	}

	if *jsonFlag {
		printJSON(objects)
		return
	}

	if len(objects) == 0 {
		fmt.Println("No Sentry objects found.")
		return
	}

	fmt.Printf("Sentry Objects (%d results)\n\n", len(objects))

	tw := newTable("Designation", "Diameter", "Impact Prob", "Palermo Scale", "Torino Scale", "Range")
	for _, o := range objects {
		tablePrintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			o.Designation,
			o.Diameter,
			o.IP,
			o.PS,
			o.TS,
			o.Range,
		)
	}
	flushTable(tw)
}

// ssdSBDBCmd handles "nasa ssd sbdb <designation>".
func ssdSBDBCmd(args []string) {
	fs := flag.NewFlagSet("ssd sbdb", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa ssd sbdb <designation> [flags]

Look up a small body in the JPL Small-Body Database.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fatal("ssd sbdb: designation is required\n\nUsage: nasa ssd sbdb <designation> [flags]")
	}
	designation := fs.Arg(0)

	client := nasa.NewClient()
	ctx := context.Background()

	sb, err := client.SSD.SBDB(ctx, designation)
	if err != nil {
		fatal("ssd sbdb: %v", err)
	}

	if *jsonFlag {
		printJSON(sb)
		return
	}

	printHeader(fmt.Sprintf("Small Body: %s", sb.Object.Des))
	fmt.Printf("Full Name:       %s\n", sb.Object.Fullname)
	fmt.Printf("Kind:            %s\n", sb.Object.Kind)
	fmt.Printf("Orbit Class:     %s\n", sb.Object.OrbitClass.Name)
	fmt.Printf("NEO:             %s\n", formatBool(sb.Object.NEO, "Yes", "No"))
	fmt.Printf("PHA:             %s\n", formatBool(sb.Object.PHA, "Yes", "No"))

	fmt.Println("\nOrbital Elements:")
	elements := []struct {
		name  string
		label string
		units string
	}{
		{"e", "Eccentricity", ""},
		{"a", "Semi-Major Axis", "AU"},
		{"i", "Inclination", "deg"},
		{"om", "Long. of Asc. Node", "deg"},
		{"w", "Arg. of Perihelion", "deg"},
		{"ma", "Mean Anomaly", "deg"},
		{"tp", "Time of Perihelion", "JED"},
		{"per", "Orbital Period", "days"},
		{"n", "Mean Motion", "deg/day"},
		{"ad", "Aphelion Distance", "AU"},
		{"q", "Perihelion Distance", "AU"},
	}
	for _, el := range elements {
		val := sb.Element(el.name)
		if val == "" {
			continue
		}
		if el.units != "" {
			fmt.Printf("  %-22s  %s %s\n", el.label+":", val, el.units)
		} else {
			fmt.Printf("  %-22s  %s\n", el.label+":", val)
		}
	}

	fmt.Println("\nOrbit Info:")
	fmt.Printf("  Condition Code:  %s\n", sb.Orbit.ConditionCode)
	fmt.Printf("  Data Arc:        %s days\n", sb.Orbit.DataArc)
	fmt.Printf("  Observations:    %d\n", sb.Orbit.NObs)
}
