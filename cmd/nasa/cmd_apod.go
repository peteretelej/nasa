package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

func apodCmd(args []string) {
	fs := flag.NewFlagSet("apod", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa apod [flags]

Retrieve NASA's Astronomy Picture of the Day.

Flags:
`)
		fs.PrintDefaults()
	}

	dateFlag := fs.String("date", "", "specific date (YYYY-MM-DD)")
	randomFlag := fs.Bool("random", false, "get random APOD(s)")
	countFlag := fs.Int("count", 1, "number of random APODs (use with --random)")
	startFlag := fs.String("start", "", "range start date (YYYY-MM-DD)")
	endFlag := fs.String("end", "", "range end date (YYYY-MM-DD)")
	jsonFlag := fs.Bool("json", false, "output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key override")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Validate mutually exclusive options.
	hasDate := *dateFlag != ""
	hasRange := *startFlag != "" || *endFlag != ""
	hasRandom := *randomFlag

	if hasDate && hasRange {
		fatal("--date and --start/--end are mutually exclusive")
	}
	if hasRandom && (hasDate || hasRange) {
		fatal("--random and --date/--start/--end are mutually exclusive")
	}
	if *countFlag != 1 && !hasRandom {
		fatal("--count can only be used with --random")
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	switch {
	case hasRandom:
		apods, err := client.APOD.Random(ctx, *countFlag)
		if err != nil {
			fatal("apod random: %v", err)
		}
		if *jsonFlag {
			printJSON(apodSliceToMaps(apods))
			return
		}
		if len(apods) == 1 {
			printSingleAPOD(apods[0])
			return
		}
		printAPODTable(apods)

	case hasRange:
		start, err := time.Parse("2006-01-02", *startFlag)
		if err != nil {
			fatal("invalid --start date: %v", err)
		}
		end, err := time.Parse("2006-01-02", *endFlag)
		if err != nil {
			fatal("invalid --end date: %v", err)
		}
		apods, err := client.APOD.GetRange(ctx, start, end)
		if err != nil {
			fatal("apod range: %v", err)
		}
		if *jsonFlag {
			printJSON(apodSliceToMaps(apods))
			return
		}
		if len(apods) == 1 {
			printSingleAPOD(apods[0])
			return
		}
		printAPODTable(apods)

	case hasDate:
		date, err := time.Parse("2006-01-02", *dateFlag)
		if err != nil {
			fatal("invalid --date: %v", err)
		}
		apod, err := client.APOD.Get(ctx, date)
		if err != nil {
			fatal("apod get: %v", err)
		}
		if *jsonFlag {
			printJSON(apodToMap(apod))
			return
		}
		printSingleAPOD(apod)

	default:
		apod, err := client.APOD.Today(ctx)
		if err != nil {
			fatal("apod today: %v", err)
		}
		if *jsonFlag {
			printJSON(apodToMap(apod))
			return
		}
		printSingleAPOD(apod)
	}
}

func printSingleAPOD(apod *nasa.APODImage) {
	printHeader("Astronomy Picture of the Day")
	fmt.Printf("Title:       %s\n", apod.Title)
	fmt.Printf("Date:        %s\n", formatDate(apod.Date))
	fmt.Printf("Type:        %s\n", apod.MediaType)
	fmt.Printf("URL:         %s\n", apod.URL)
	if apod.HDURL != "" {
		fmt.Printf("HD URL:      %s\n", apod.HDURL)
	}
	if apod.Copyright != "" {
		fmt.Printf("Copyright:   %s\n", apod.Copyright)
	}
	fmt.Println()
	fmt.Println(apod.Explanation)
}

func printAPODTable(apods []*nasa.APODImage) {
	tw := newTable("Title", "Date", "Type", "URL")
	for _, a := range apods {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			truncate(a.Title, 40),
			formatDate(a.Date),
			a.MediaType,
			a.URL,
		)
	}
	flushTable(tw)
}

// apodToMap converts an APODImage to a map that includes the Date field,
// which is excluded from standard JSON marshaling (json:"-").
func apodToMap(a *nasa.APODImage) map[string]any {
	m := map[string]any{
		"date":        formatDate(a.Date),
		"title":       a.Title,
		"explanation": a.Explanation,
		"url":         a.URL,
		"hdurl":       a.HDURL,
		"media_type":  string(a.MediaType),
	}
	if a.Copyright != "" {
		m["copyright"] = a.Copyright
	}
	if a.ThumbnailURL != "" {
		m["thumbnail_url"] = a.ThumbnailURL
	}
	return m
}

func apodSliceToMaps(apods []*nasa.APODImage) []map[string]any {
	result := make([]map[string]any, len(apods))
	for i, a := range apods {
		result[i] = apodToMap(a)
	}
	return result
}
