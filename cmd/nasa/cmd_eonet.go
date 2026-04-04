package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/peteretelej/nasa"
)

func eonetCmd(args []string) {
	if len(args) == 0 {
		eonetUsage(os.Stderr)
		os.Exit(1)
	}
	switch args[0] {
	case "events":
		eonetEventsCmd(args[1:])
	case "event":
		eonetEventCmd(args[1:])
	case "categories":
		eonetCategoriesCmd(args[1:])
	case "sources":
		eonetSourcesCmd(args[1:])
	case "--help", "-help", "-h":
		eonetUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "nasa eonet: unknown command %q\n\n", args[0])
		eonetUsage(os.Stderr)
		os.Exit(1)
	}
}

func eonetUsage(w io.Writer) {
	if _, err := fmt.Fprintf(w, `Usage: nasa eonet <command> [flags]

Commands:
  events       List natural events
  event        Get details for a specific event
  categories   List event categories
  sources      List event sources

Use "nasa eonet <command> --help" for more information about a command.
`); err != nil {
		fatal("failed to write output: %v", err)
	}
}

// eonetEventsCmd handles "nasa eonet events".
func eonetEventsCmd(args []string) {
	fs := flag.NewFlagSet("eonet events", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa eonet events [flags]

List natural events from EONET.

Flags:
`)
		fs.PrintDefaults()
	}

	status := fs.String("status", "", "event status (open, closed)")
	category := fs.String("category", "", "category filter")
	days := fs.Int("days", 0, "events within last N days")
	limit := fs.Int("limit", 0, "maximum number of events")
	source := fs.String("source", "", "source filter")
	startFlag := fs.String("start", "", "start date (YYYY-MM-DD)")
	endFlag := fs.String("end", "", "end date (YYYY-MM-DD)")
	jsonFlag := fs.Bool("json", false, "output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key override")

	_ = fs.Parse(args)
	if fs.NArg() > 0 {
		fatal("eonet events: unexpected arguments")
	}

	var opts []nasa.EONETOption
	if *status != "" {
		opts = append(opts, nasa.EONETWithStatus(*status))
	}
	if *category != "" {
		opts = append(opts, nasa.EONETWithCategory(*category))
	}
	if *days > 0 {
		opts = append(opts, nasa.EONETWithDays(*days))
	} else if *days < 0 {
		fatal("eonet events: --days must not be negative")
	}
	if *limit > 0 {
		opts = append(opts, nasa.EONETWithLimit(*limit))
	} else if *limit < 0 {
		fatal("eonet events: --limit must not be negative")
	}
	if *source != "" {
		opts = append(opts, nasa.EONETWithSource(*source))
	}
	if *startFlag != "" {
		t, err := time.Parse("2006-01-02", *startFlag)
		if err != nil {
			fatal("eonet events: invalid --start date, expected YYYY-MM-DD")
		}
		opts = append(opts, nasa.EONETWithStart(t))
	}
	if *endFlag != "" {
		t, err := time.Parse("2006-01-02", *endFlag)
		if err != nil {
			fatal("eonet events: invalid --end date, expected YYYY-MM-DD")
		}
		opts = append(opts, nasa.EONETWithEnd(t))
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.EONET.Events(ctx, opts...)
	if err != nil {
		fatal("eonet events: %v", err)
	}

	if *jsonFlag {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No EONET events found.")
		return
	}

	fmt.Printf("EONET Events (%d results)\n\n", len(events))

	tw := newTable("ID", "Title", "Category", "Status", "Date")
	for _, e := range events {
		cat := ""
		if len(e.Categories) > 0 {
			cat = e.Categories[0].Title
		}
		eventStatus := "open"
		if e.Closed != nil {
			eventStatus = "closed"
		}
		date := ""
		if len(e.Geometry) > 0 {
			date = formatDate(e.Geometry[0].Date)
		}
		tablePrintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			e.ID,
			truncate(e.Title, 30),
			cat,
			eventStatus,
			date,
		)
	}
	flushTable(tw)
}

// eonetEventCmd handles "nasa eonet event <event-id>".
func eonetEventCmd(args []string) {
	fs := flag.NewFlagSet("eonet event", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa eonet event <event-id> [flags]

Get details for a specific EONET event.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key override")

	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fatal("eonet event: event ID is required\n\nUsage: nasa eonet event <event-id> [flags]")
	}
	if fs.NArg() > 1 {
		fatal("eonet event: unexpected arguments")
	}
	eventID := fs.Arg(0)

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	event, err := client.EONET.Event(ctx, eventID)
	if err != nil {
		fatal("eonet event: %v", err)
	}

	if *jsonFlag {
		printJSON(event)
		return
	}

	printHeader(fmt.Sprintf("EONET Event: %s", event.ID))
	fmt.Printf("Title:       %s\n", event.Title)

	eventStatus := "open"
	if event.Closed != nil {
		eventStatus = "closed"
	}
	fmt.Printf("Status:      %s\n", eventStatus)

	if len(event.Categories) > 0 {
		cats := make([]string, len(event.Categories))
		for i, c := range event.Categories {
			cats[i] = c.Title
		}
		fmt.Printf("Categories:  %s\n", strings.Join(cats, ", "))
	}

	if len(event.Sources) > 0 {
		srcs := make([]string, len(event.Sources))
		for i, s := range event.Sources {
			srcs[i] = s.ID
		}
		fmt.Printf("Sources:     %s\n", strings.Join(srcs, ", "))
	}

	if len(event.Geometry) > 0 {
		fmt.Println("\nGeometry:")
		tw := newTable("Date", "Type", "Coordinates")
		for _, g := range event.Geometry {
			coords := formatEONETCoords(g)
			tablePrintf(tw, "%s\t%s\t%s\n",
				g.Date.Format(time.RFC3339),
				g.Type,
				coords,
			)
		}
		flushTable(tw)
	}
}

// formatEONETCoords extracts coordinates from EONET geometry.
func formatEONETCoords(g *nasa.EONETGeometry) string {
	if g.Type == "Point" {
		coords, err := g.PointCoordinates()
		if err != nil {
			return "-"
		}
		return fmt.Sprintf("%.2f, %.2f", coords[0], coords[1])
	}
	// For polygons, just indicate the type.
	return fmt.Sprintf("[%s]", g.Type)
}

// eonetCategoriesCmd handles "nasa eonet categories".
func eonetCategoriesCmd(args []string) {
	fs := flag.NewFlagSet("eonet categories", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa eonet categories [flags]

List EONET event categories.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	_ = fs.Parse(args)
	if fs.NArg() > 0 {
		fatal("eonet categories: unexpected arguments")
	}

	client := nasa.NewClient()
	ctx := context.Background()

	categories, err := client.EONET.Categories(ctx)
	if err != nil {
		fatal("eonet categories: %v", err)
	}

	if *jsonFlag {
		printJSON(categories)
		return
	}

	if len(categories) == 0 {
		fmt.Println("No EONET categories found.")
		return
	}

	fmt.Println("EONET Categories")
	fmt.Println()

	tw := newTable("ID", "Title", "Description")
	for _, c := range categories {
		tablePrintf(tw, "%s\t%s\t%s\n",
			c.ID,
			c.Title,
			truncate(c.Description, 50),
		)
	}
	flushTable(tw)
}

// eonetSourcesCmd handles "nasa eonet sources".
func eonetSourcesCmd(args []string) {
	fs := flag.NewFlagSet("eonet sources", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa eonet sources [flags]

List EONET event sources.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	_ = fs.Parse(args)
	if fs.NArg() > 0 {
		fatal("eonet sources: unexpected arguments")
	}

	client := nasa.NewClient()
	ctx := context.Background()

	sources, err := client.EONET.Sources(ctx)
	if err != nil {
		fatal("eonet sources: %v", err)
	}

	if *jsonFlag {
		printJSON(sources)
		return
	}

	if len(sources) == 0 {
		fmt.Println("No EONET sources found.")
		return
	}

	fmt.Println("EONET Sources")
	fmt.Println()

	tw := newTable("ID", "URL")
	for _, s := range sources {
		tablePrintf(tw, "%s\t%s\n",
			s.ID,
			s.URL,
		)
	}
	flushTable(tw)
}
