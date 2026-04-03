package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/peteretelej/nasa"
)

func neoCmd(args []string) {
	if len(args) == 0 {
		neoUsage(os.Stderr)
		os.Exit(1)
	}
	switch args[0] {
	case "feed":
		neoFeedCmd(args[1:])
	case "get":
		neoGetCmd(args[1:])
	case "browse":
		neoBrowseCmd(args[1:])
	case "--help", "-help", "-h":
		neoUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "nasa neo: unknown command %q\n\n", args[0])
		neoUsage(os.Stderr)
		os.Exit(1)
	}
}

func neoUsage(w io.Writer) {
	if _, err := fmt.Fprintf(w, `Usage: nasa neo <command> [flags]

Commands:
  feed      List near-Earth objects by date range
  get       Get details for a specific asteroid
  browse    Browse the full asteroid catalog

Use "nasa neo <command> --help" for more information about a command.
`); err != nil {
		fatal("failed to write output: %v", err)
	}
}

// neoFeedCmd handles the "nasa neo feed" command.
func neoFeedCmd(args []string) {
	fs := flag.NewFlagSet("neo feed", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa neo feed [flags]

List near-Earth objects approaching Earth within a date range (max 7 days).

Flags:
`)
		fs.PrintDefaults()
	}

	startFlag := fs.String("start", "", "start date (YYYY-MM-DD, default: today)")
	endFlag := fs.String("end", "", "end date (YYYY-MM-DD, default: start+7d)")
	jsonFlag := fs.Bool("json", false, "output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key override")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	now := time.Now().Truncate(24 * time.Hour)
	start := now
	if *startFlag != "" {
		var err error
		start, err = time.Parse("2006-01-02", *startFlag)
		if err != nil {
			fatal("neo feed: invalid --start date, expected YYYY-MM-DD")
		}
	}
	end := start.Add(7 * 24 * time.Hour)
	if *endFlag != "" {
		var err error
		end, err = time.Parse("2006-01-02", *endFlag)
		if err != nil {
			fatal("neo feed: invalid --end date, expected YYYY-MM-DD")
		}
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	feed, err := client.NEO.Feed(ctx, start, end)
	if err != nil {
		fatal("neo feed: %v", err)
	}

	if *jsonFlag {
		printJSON(feed)
		return
	}

	// Sort date keys for consistent output.
	dates := make([]string, 0, len(feed.NearEarthObjects))
	for d := range feed.NearEarthObjects {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	fmt.Printf("Near-Earth Objects: %s to %s (%d objects)\n\n",
		formatDate(start), formatDate(end), feed.ElementCount)

	tw := newTable("Name", "Date", "Distance (km)", "Velocity (km/s)", "Hazardous")
	for _, d := range dates {
		neos := feed.NearEarthObjects[d]
		for _, neo := range neos {
			for _, ca := range neo.CloseApproachData {
				tablePrintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					neo.Name,
					ca.CloseApproachDate,
					formatFloat(float64(ca.MissDistance.Kilometers), 2),
					formatFloat(float64(ca.RelativeVelocity.KilometersPerSecond), 2),
					formatBool(neo.IsPotentiallyHazardous, "Yes", "No"),
				)
			}
		}
	}
	flushTable(tw)
}

// neoGetCmd handles the "nasa neo get <asteroid-id>" command.
func neoGetCmd(args []string) {
	fs := flag.NewFlagSet("neo get", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa neo get <asteroid-id> [flags]

Get detailed information about a specific near-Earth object.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key override")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fatal("neo get: asteroid ID is required\n\nUsage: nasa neo get <asteroid-id> [flags]")
	}
	asteroidID := fs.Arg(0)

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	neo, err := client.NEO.Get(ctx, asteroidID)
	if err != nil {
		fatal("neo get: %v", err)
	}

	if *jsonFlag {
		printJSON(neoToMap(neo))
		return
	}

	printDetailedNEO(neo)
}

// neoBrowseCmd handles the "nasa neo browse" command.
func neoBrowseCmd(args []string) {
	fs := flag.NewFlagSet("neo browse", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa neo browse [flags]

Browse the full near-Earth object catalog.

Flags:
`)
		fs.PrintDefaults()
	}

	pageFlag := fs.Int("page", 0, "page number (0-indexed)")
	sizeFlag := fs.Int("size", 20, "results per page")
	jsonFlag := fs.Bool("json", false, "output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key override")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	browse, err := client.NEO.Browse(ctx, *pageFlag, *sizeFlag)
	if err != nil {
		fatal("neo browse: %v", err)
	}

	if *jsonFlag {
		printJSON(browse)
		return
	}

	fmt.Printf("Near-Earth Objects: page %d of %d (%d total)\n\n",
		browse.Page.Number, browse.Page.TotalPages, browse.Page.TotalElements)

	tw := newTable("Name", "Magnitude", "Hazardous", "Sentry")
	for _, neo := range browse.NearEarthObjects {
		tablePrintf(tw, "%s\t%s\t%s\t%s\n",
			neo.Name,
			formatFloat(neo.AbsoluteMagnitude, 1),
			formatBool(neo.IsPotentiallyHazardous, "Yes", "No"),
			formatBool(neo.IsSentryObject, "Yes", "No"),
		)
	}
	flushTable(tw)
}

func printDetailedNEO(neo *nasa.NearEarthObject) {
	printHeader(fmt.Sprintf("Near-Earth Object: %s", neo.Name))
	fmt.Printf("ID:                    %s\n", neo.ID)
	fmt.Printf("Name:                  %s\n", neo.Name)
	fmt.Printf("Absolute Magnitude:    %s\n", formatFloat(neo.AbsoluteMagnitude, 1))
	fmt.Printf("Hazardous:             %s\n", formatBool(neo.IsPotentiallyHazardous, "Yes", "No"))
	fmt.Printf("Sentry Object:         %s\n", formatBool(neo.IsSentryObject, "Yes", "No"))
	if neo.JPLURL != "" {
		fmt.Printf("JPL URL:               %s\n", neo.JPLURL)
	}

	// Estimated diameter.
	d := neo.EstimatedDiameter.Kilometers
	fmt.Printf("\nEstimated Diameter (km): %s - %s\n",
		formatFloat(d.Min, 3), formatFloat(d.Max, 3))

	// Close approaches.
	if len(neo.CloseApproachData) > 0 {
		fmt.Println("\nClose Approaches:")
		tw := newTable("Date", "Distance (km)", "Velocity (km/s)", "Orbiting Body")
		for _, ca := range neo.CloseApproachData {
			tablePrintf(tw, "%s\t%s\t%s\t%s\n",
				ca.CloseApproachDate,
				formatFloat(float64(ca.MissDistance.Kilometers), 2),
				formatFloat(float64(ca.RelativeVelocity.KilometersPerSecond), 2),
				ca.OrbitingBody,
			)
		}
		flushTable(tw)
	}

	// Orbital data.
	if neo.OrbitalData != nil {
		od := neo.OrbitalData
		fmt.Println("\nOrbital Data:")
		fmt.Printf("  Orbit ID:              %s\n", od.OrbitID)
		fmt.Printf("  Eccentricity:          %s\n", od.Eccentricity)
		fmt.Printf("  Semi-Major Axis:       %s\n", od.SemiMajorAxis)
		fmt.Printf("  Inclination:           %s\n", od.Inclination)
		fmt.Printf("  Orbital Period:        %s\n", od.OrbitalPeriod)
		fmt.Printf("  Perihelion Distance:   %s\n", od.PerihelionDistance)
		fmt.Printf("  Aphelion Distance:     %s\n", od.AphelionDistance)
		fmt.Printf("  Orbit Uncertainty:     %s\n", od.OrbitUncertainty)
		if od.OrbitClass != nil {
			fmt.Printf("  Orbit Class:           %s\n", od.OrbitClass.Type)
		}
	}
}

// neoToMap builds a JSON-friendly map for a NearEarthObject, including
// fields that have json:"-" tags (like CloseApproachDateFull).
func neoToMap(neo *nasa.NearEarthObject) map[string]any {
	m := map[string]any{
		"id":                                neo.ID,
		"neo_reference_id":                  neo.NeoReferenceID,
		"name":                              neo.Name,
		"nasa_jpl_url":                      neo.JPLURL,
		"absolute_magnitude_h":              neo.AbsoluteMagnitude,
		"is_potentially_hazardous_asteroid": neo.IsPotentiallyHazardous,
		"is_sentry_object":                  neo.IsSentryObject,
		"estimated_diameter":                neo.EstimatedDiameter,
	}

	if len(neo.CloseApproachData) > 0 {
		cas := make([]map[string]any, len(neo.CloseApproachData))
		for i, ca := range neo.CloseApproachData {
			cas[i] = map[string]any{
				"close_approach_date":       ca.CloseApproachDate,
				"close_approach_date_full":  ca.CloseApproachDateFull.Format("2006-Jan-02 15:04"),
				"epoch_date_close_approach": ca.EpochDateCloseApproach.Time().UnixMilli(),
				"relative_velocity":         ca.RelativeVelocity,
				"miss_distance":             ca.MissDistance,
				"orbiting_body":             ca.OrbitingBody,
			}
		}
		m["close_approach_data"] = cas
	}

	if neo.OrbitalData != nil {
		m["orbital_data"] = neo.OrbitalData
	}

	return m
}
