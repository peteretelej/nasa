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

func donkiCmd(args []string) {
	if len(args) == 0 {
		donkiUsage(os.Stderr)
		os.Exit(1)
	}
	switch args[0] {
	case "cme":
		donkiCMECmd(args[1:])
	case "cme-analysis":
		donkiCMEAnalysisCmd(args[1:])
	case "gst":
		donkiGSTCmd(args[1:])
	case "ips":
		donkiIPSCmd(args[1:])
	case "flr":
		donkiFLRCmd(args[1:])
	case "sep":
		donkiSEPCmd(args[1:])
	case "mpc":
		donkiMPCCmd(args[1:])
	case "rbe":
		donkiRBECmd(args[1:])
	case "hss":
		donkiHSSCmd(args[1:])
	case "wsa-enlil":
		donkiWSAEnlilCmd(args[1:])
	case "notifications":
		donkiNotificationsCmd(args[1:])
	case "--help", "-help", "-h":
		donkiUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "nasa donki: unknown command %q\n\n", args[0])
		donkiUsage(os.Stderr)
		os.Exit(1)
	}
}

func donkiUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage: nasa donki <command> [flags]

Commands:
  cme             Coronal Mass Ejections
  cme-analysis    CME Analysis
  gst             Geomagnetic Storms
  ips             Interplanetary Shocks
  flr             Solar Flares
  sep             Solar Energetic Particles
  mpc             Magnetopause Crossings
  rbe             Radiation Belt Enhancements
  hss             High Speed Streams
  wsa-enlil       WSA-Enlil Simulations
  notifications   DONKI Notifications

Common flags:
  --start     Start date YYYY-MM-DD (default: 30 days ago)
  --end       End date YYYY-MM-DD (default: today)
  --json      Output as JSON
  --api-key   API key override

Use "nasa donki <command> --help" for more information about a command.
`)
}

// parseDonkiFlags creates a FlagSet with the 4 common DONKI flags, parses
// args, and returns the resolved start/end times, JSON flag, and API key.
// The returned FlagSet is available for sub-commands that need extra flags -
// those sub-commands should define their own FlagSet instead of using this.
func parseDonkiFlags(name string, args []string) (start, end time.Time, jsonOut bool, apiKey string) {
	fs := flag.NewFlagSet("donki "+name, flag.ExitOnError)
	startStr := fs.String("start", "", "Start date YYYY-MM-DD (default: 30 days ago)")
	endStr := fs.String("end", "", "End date YYYY-MM-DD (default: today)")
	jsonFlag := fs.Bool("json", false, "Output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key (default: NASA_API_KEY env)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	end = time.Now().Truncate(24 * time.Hour)
	start = end.AddDate(0, 0, -30)

	if *startStr != "" {
		var err error
		start, err = time.Parse("2006-01-02", *startStr)
		if err != nil {
			fatal("donki %s: invalid --start date, expected YYYY-MM-DD", name)
		}
	}
	if *endStr != "" {
		var err error
		end, err = time.Parse("2006-01-02", *endStr)
		if err != nil {
			fatal("donki %s: invalid --end date, expected YYYY-MM-DD", name)
		}
	}

	return start, end, *jsonFlag, *apiKeyFlag
}

// formatDONKITime formats a DONKITime to a readable "YYYY-MM-DD HH:MM" string.
func formatDONKITime(t nasa.DONKITime) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// donkiCMECmd handles "nasa donki cme".
func donkiCMECmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("cme", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.CME(ctx, start, end)
	if err != nil {
		fatal("donki cme: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No CME events found.")
		return
	}

	tw := newTable("ACTIVITY ID", "START TIME", "SOURCE LOCATION", "NOTE")
	for _, e := range events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			e.ActivityID,
			formatDONKITime(e.StartTime),
			e.SourceLocation,
			truncate(e.Note, 60),
		)
	}
	flushTable(tw)
}

// donkiCMEAnalysisCmd handles "nasa donki cme-analysis".
func donkiCMEAnalysisCmd(args []string) {
	fs := flag.NewFlagSet("donki cme-analysis", flag.ExitOnError)
	startStr := fs.String("start", "", "Start date YYYY-MM-DD (default: 30 days ago)")
	endStr := fs.String("end", "", "End date YYYY-MM-DD (default: today)")
	jsonFlag := fs.Bool("json", false, "Output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key (default: NASA_API_KEY env)")
	mostAccurate := fs.Bool("most-accurate", false, "Filter to most accurate only")
	completeEntry := fs.Bool("complete-entry", false, "Filter to complete entries only")
	speedFlag := fs.Float64("speed", 0, "Minimum speed (km/s)")
	halfAngleFlag := fs.Float64("half-angle", 0, "Minimum half-angle (degrees)")
	catalogFlag := fs.String("catalog", "", "Catalog filter (ALL, M2M_CATALOG, SWPC_ANNEX_CME_CATALOG, SEPVAL_CME_CATALOG)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	end := time.Now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -30)

	if *startStr != "" {
		var err error
		start, err = time.Parse("2006-01-02", *startStr)
		if err != nil {
			fatal("donki cme-analysis: invalid --start date, expected YYYY-MM-DD")
		}
	}
	if *endStr != "" {
		var err error
		end, err = time.Parse("2006-01-02", *endStr)
		if err != nil {
			fatal("donki cme-analysis: invalid --end date, expected YYYY-MM-DD")
		}
	}

	var opts []nasa.CMEAnalysisOption
	if *mostAccurate {
		opts = append(opts, nasa.WithMostAccurateOnly(true))
	}
	if *completeEntry {
		opts = append(opts, nasa.WithCompleteEntryOnly(true))
	}
	if *speedFlag > 0 {
		opts = append(opts, nasa.WithSpeed(*speedFlag))
	}
	if *halfAngleFlag > 0 {
		opts = append(opts, nasa.WithHalfAngle(*halfAngleFlag))
	}
	if *catalogFlag != "" {
		opts = append(opts, nasa.WithCatalog(*catalogFlag))
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	analyses, err := client.DONKI.CMEAnalysis(ctx, start, end, opts...)
	if err != nil {
		fatal("donki cme-analysis: %v", err)
	}

	if *jsonFlag {
		printJSON(analyses)
		return
	}

	if len(analyses) == 0 {
		fmt.Println("No CME analysis results found.")
		return
	}

	tw := newTable("TIME 21.5", "LATITUDE", "LONGITUDE", "SPEED (KM/S)", "HALF ANGLE", "TYPE", "MOST ACCURATE")
	for _, a := range analyses {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			formatDONKITime(a.Time21_5),
			formatFloat(a.Latitude, 1),
			formatFloat(a.Longitude, 1),
			formatFloat(a.Speed, 1),
			formatFloat(a.HalfAngle, 1),
			a.Type,
			formatBool(a.IsMostAccurate, "Yes", "No"),
		)
	}
	flushTable(tw)
}

// donkiGSTCmd handles "nasa donki gst".
func donkiGSTCmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("gst", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.GST(ctx, start, end)
	if err != nil {
		fatal("donki gst: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No geomagnetic storm events found.")
		return
	}

	tw := newTable("GST ID", "START TIME", "KP VALUES")
	for _, e := range events {
		kpValues := make([]string, len(e.AllKpIndex))
		for i, kp := range e.AllKpIndex {
			kpValues[i] = formatFloat(kp.KpIndex, 1)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			e.GstID,
			formatDONKITime(e.StartTime),
			strings.Join(kpValues, ", "),
		)
	}
	flushTable(tw)
}

// donkiIPSCmd handles "nasa donki ips".
func donkiIPSCmd(args []string) {
	fs := flag.NewFlagSet("donki ips", flag.ExitOnError)
	startStr := fs.String("start", "", "Start date YYYY-MM-DD (default: 30 days ago)")
	endStr := fs.String("end", "", "End date YYYY-MM-DD (default: today)")
	jsonFlag := fs.Bool("json", false, "Output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key (default: NASA_API_KEY env)")
	locationFlag := fs.String("location", "", "Location filter (Earth, Mars, MESSENGER, STEREO A, STEREO B)")
	catalogFlag := fs.String("catalog", "", "Catalog filter (ALL, M2M_CATALOG, WINSLOW_MESSENGER_ICME_CATALOG)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	end := time.Now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -30)

	if *startStr != "" {
		var err error
		start, err = time.Parse("2006-01-02", *startStr)
		if err != nil {
			fatal("donki ips: invalid --start date, expected YYYY-MM-DD")
		}
	}
	if *endStr != "" {
		var err error
		end, err = time.Parse("2006-01-02", *endStr)
		if err != nil {
			fatal("donki ips: invalid --end date, expected YYYY-MM-DD")
		}
	}

	var opts []nasa.IPSOption
	if *locationFlag != "" {
		opts = append(opts, nasa.WithLocation(*locationFlag))
	}
	if *catalogFlag != "" {
		opts = append(opts, nasa.WithIPSCatalog(*catalogFlag))
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.IPS(ctx, start, end, opts...)
	if err != nil {
		fatal("donki ips: %v", err)
	}

	if *jsonFlag {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No interplanetary shock events found.")
		return
	}

	tw := newTable("ACTIVITY ID", "EVENT TIME", "LOCATION", "CATALOG")
	for _, e := range events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			e.ActivityID,
			formatDONKITime(e.EventTime),
			e.Location,
			e.Catalog,
		)
	}
	flushTable(tw)
}

// donkiFLRCmd handles "nasa donki flr".
func donkiFLRCmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("flr", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.FLR(ctx, start, end)
	if err != nil {
		fatal("donki flr: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No solar flare events found.")
		return
	}

	tw := newTable("FLR ID", "BEGIN TIME", "PEAK TIME", "END TIME", "CLASS", "SOURCE LOCATION")
	for _, e := range events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			e.FlrID,
			formatDONKITime(e.BeginTime),
			formatDONKITime(e.PeakTime),
			formatDONKITime(e.EndTime),
			e.ClassType,
			e.SourceLocation,
		)
	}
	flushTable(tw)
}

// donkiSEPCmd handles "nasa donki sep".
func donkiSEPCmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("sep", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.SEP(ctx, start, end)
	if err != nil {
		fatal("donki sep: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No solar energetic particle events found.")
		return
	}

	tw := newTable("SEP ID", "EVENT TIME", "INSTRUMENTS")
	for _, e := range events {
		names := make([]string, len(e.Instruments))
		for i, inst := range e.Instruments {
			names[i] = inst.DisplayName
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			e.SepID,
			formatDONKITime(e.EventTime),
			strings.Join(names, ", "),
		)
	}
	flushTable(tw)
}

// donkiMPCCmd handles "nasa donki mpc".
func donkiMPCCmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("mpc", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.MPC(ctx, start, end)
	if err != nil {
		fatal("donki mpc: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No magnetopause crossing events found.")
		return
	}

	tw := newTable("MPC ID", "EVENT TIME", "INSTRUMENTS")
	for _, e := range events {
		names := make([]string, len(e.Instruments))
		for i, inst := range e.Instruments {
			names[i] = inst.DisplayName
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			e.MpcID,
			formatDONKITime(e.EventTime),
			strings.Join(names, ", "),
		)
	}
	flushTable(tw)
}

// donkiRBECmd handles "nasa donki rbe".
func donkiRBECmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("rbe", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.RBE(ctx, start, end)
	if err != nil {
		fatal("donki rbe: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No radiation belt enhancement events found.")
		return
	}

	tw := newTable("RBE ID", "EVENT TIME", "INSTRUMENTS")
	for _, e := range events {
		names := make([]string, len(e.Instruments))
		for i, inst := range e.Instruments {
			names[i] = inst.DisplayName
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			e.RbeID,
			formatDONKITime(e.EventTime),
			strings.Join(names, ", "),
		)
	}
	flushTable(tw)
}

// donkiHSSCmd handles "nasa donki hss".
func donkiHSSCmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("hss", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.HSS(ctx, start, end)
	if err != nil {
		fatal("donki hss: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No high speed stream events found.")
		return
	}

	tw := newTable("HSS ID", "EVENT TIME", "INSTRUMENTS")
	for _, e := range events {
		names := make([]string, len(e.Instruments))
		for i, inst := range e.Instruments {
			names[i] = inst.DisplayName
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			e.HssID,
			formatDONKITime(e.EventTime),
			strings.Join(names, ", "),
		)
	}
	flushTable(tw)
}

// donkiWSAEnlilCmd handles "nasa donki wsa-enlil".
func donkiWSAEnlilCmd(args []string) {
	start, end, jsonOut, apiKeyFlag := parseDonkiFlags("wsa-enlil", args)
	key := getAPIKey(apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	events, err := client.DONKI.WSAEnlil(ctx, start, end)
	if err != nil {
		fatal("donki wsa-enlil: %v", err)
	}

	if jsonOut {
		printJSON(events)
		return
	}

	if len(events) == 0 {
		fmt.Println("No WSA-Enlil simulation events found.")
		return
	}

	tw := newTable("SIMULATION ID", "MODEL COMPLETION", "SHOCK ARRIVAL", "DURATION (HRS)", "CME INPUTS")
	for _, e := range events {
		duration := ""
		if e.EstimatedDuration != nil {
			duration = formatFloat(*e.EstimatedDuration, 1)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n",
			e.SimulationID,
			formatDONKITime(e.ModelCompletionTime),
			formatDONKITime(e.EstimatedShockArrivalTime),
			duration,
			len(e.CMEInputs),
		)
	}
	flushTable(tw)
}

// donkiNotificationsCmd handles "nasa donki notifications".
func donkiNotificationsCmd(args []string) {
	fs := flag.NewFlagSet("donki notifications", flag.ExitOnError)
	startStr := fs.String("start", "", "Start date YYYY-MM-DD (default: 30 days ago)")
	endStr := fs.String("end", "", "End date YYYY-MM-DD (default: today)")
	jsonFlag := fs.Bool("json", false, "Output as JSON")
	apiKeyFlag := fs.String("api-key", "", "API key (default: NASA_API_KEY env)")
	typeFlag := fs.String("type", "", "Notification type filter (FLR, SEP, CME, IPS, MPC, GST, RBE, report)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	end := time.Now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -30)

	if *startStr != "" {
		var err error
		start, err = time.Parse("2006-01-02", *startStr)
		if err != nil {
			fatal("donki notifications: invalid --start date, expected YYYY-MM-DD")
		}
	}
	if *endStr != "" {
		var err error
		end, err = time.Parse("2006-01-02", *endStr)
		if err != nil {
			fatal("donki notifications: invalid --end date, expected YYYY-MM-DD")
		}
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))
	ctx := context.Background()

	notifications, err := client.DONKI.Notifications(ctx, start, end, *typeFlag)
	if err != nil {
		fatal("donki notifications: %v", err)
	}

	if *jsonFlag {
		printJSON(notifications)
		return
	}

	if len(notifications) == 0 {
		fmt.Println("No notifications found.")
		return
	}

	tw := newTable("MESSAGE TYPE", "ISSUE TIME", "BODY")
	for _, n := range notifications {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			n.MessageType,
			formatDONKITime(n.MessageIssueTime),
			truncate(strings.ReplaceAll(n.MessageBody, "\n", " "), 80),
		)
	}
	flushTable(tw)
}
