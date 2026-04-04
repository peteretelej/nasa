package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/peteretelej/nasa"
)

// --- Shared helpers ---
//
// These functions provide common argument extraction, date parsing, and result
// formatting used across all tool handlers.

// formatLine returns a "Label: value\n" string for building text output.
func formatLine(label, value string) string {
	return label + ": " + value + "\n"
}

// errorResult creates a ToolResult with isError set to true. The format string
// and args follow fmt.Sprintf conventions.
func errorResult(format string, args ...any) *ToolResult {
	return &ToolResult{
		Content: []Content{{Type: "text", Text: "Error: " + fmt.Sprintf(format, args...)}},
		IsError: true,
	}
}

// textResult creates a successful ToolResult containing a single text block.
func textResult(text string) *ToolResult {
	return &ToolResult{
		Content: []Content{{Type: "text", Text: text}},
	}
}

// parseDate parses a YYYY-MM-DD string into a time.Time value.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// getStringArg extracts a string value from the args map. Returns an empty
// string if the key is missing or the value is not a string.
func getStringArg(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// getIntArg extracts an integer from the args map. JSON numbers arrive as
// float64, so this validates the value is finite, has no fractional part,
// and fits within int range before converting. Returns (0, false) if the
// key is missing, not a number, or the value is invalid.
func getIntArg(args map[string]any, key string) (int, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	if !ok {
		return 0, false
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, false
	}
	if f != math.Trunc(f) {
		return 0, false
	}
	if f < math.MinInt || f > math.MaxInt {
		return 0, false
	}
	return int(f), true
}

// getFloatArg extracts a float64 from the args map. Returns (0, false) if the
// key is missing or the value is not a number.
func getFloatArg(args map[string]any, key string) (float64, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	if !ok {
		return 0, false
	}
	return f, true
}

// --- APOD formatting ---

// formatAPOD formats an APODImage into a human-readable text block with title,
// date, explanation, and image URLs.
func formatAPOD(img *nasa.APODImage) string {
	var b strings.Builder
	b.WriteString(formatLine("Title", img.Title))
	b.WriteString(formatLine("Date", img.Date.Format("2006-01-02")))
	b.WriteString(formatLine("Explanation", img.Explanation))
	b.WriteString(formatLine("Image URL", img.URL))
	if img.HDURL != "" {
		b.WriteString(formatLine("HD URL", img.HDURL))
	}
	return b.String()
}

// --- Handler functions ---

// handleAPOD handles the nasa_apod tool. It supports three modes: random
// (when count is provided), specific date, or today's APOD (the default).
func handleAPOD(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	if count, ok := getIntArg(args, "count"); ok {
		imgs, err := client.APOD.Random(ctx, count)
		if err != nil {
			return errorResult("%v", err)
		}
		var b strings.Builder
		for i, img := range imgs {
			if i > 0 {
				b.WriteString("---\n")
			}
			b.WriteString(formatAPOD(img))
		}
		return textResult(b.String())
	}

	if dateStr := getStringArg(args, "date"); dateStr != "" {
		date, err := parseDate(dateStr)
		if err != nil {
			return errorResult("invalid date format: %v", err)
		}
		img, err := client.APOD.Get(ctx, date)
		if err != nil {
			return errorResult("%v", err)
		}
		return textResult(formatAPOD(img))
	}

	img, err := client.APOD.Today(ctx)
	if err != nil {
		return errorResult("%v", err)
	}
	return textResult(formatAPOD(img))
}

// handleAPODRange handles the nasa_apod_range tool. Both start_date and
// end_date are required.
func handleAPODRange(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	startStr := getStringArg(args, "start_date")
	endStr := getStringArg(args, "end_date")
	if startStr == "" || endStr == "" {
		return errorResult("start_date and end_date are required")
	}

	start, err := parseDate(startStr)
	if err != nil {
		return errorResult("invalid start_date: %v", err)
	}
	end, err := parseDate(endStr)
	if err != nil {
		return errorResult("invalid end_date: %v", err)
	}

	imgs, err := client.APOD.GetRange(ctx, start, end)
	if err != nil {
		return errorResult("%v", err)
	}

	var b strings.Builder
	for i, img := range imgs {
		fmt.Fprintf(&b, "%d. %s\n", i+1, img.Title)
		b.WriteString(formatLine("   Date", img.Date.Format("2006-01-02")))
		b.WriteString(formatLine("   Explanation", img.Explanation))
		b.WriteString(formatLine("   Image URL", img.URL))
		b.WriteString("\n")
	}
	return textResult(b.String())
}

// handleNEOFeed handles the nasa_neo_feed tool. Returns near-Earth objects
// for a date range of up to 7 days.
func handleNEOFeed(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	startStr := getStringArg(args, "start_date")
	if startStr == "" {
		return errorResult("start_date is required")
	}
	start, err := parseDate(startStr)
	if err != nil {
		return errorResult("invalid start_date: %v", err)
	}

	var end time.Time
	if endStr := getStringArg(args, "end_date"); endStr != "" {
		end, err = parseDate(endStr)
		if err != nil {
			return errorResult("invalid end_date: %v", err)
		}
	} else {
		end = start.AddDate(0, 0, 6)
	}

	feed, err := client.NEO.Feed(ctx, start, end)
	if err != nil {
		return errorResult("%v", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Near-Earth Objects: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Fprintf(&b, "Total: %d\n\n", feed.ElementCount)

	// Sort dates for consistent output.
	dates := make([]string, 0, len(feed.NearEarthObjects))
	for d := range feed.NearEarthObjects {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	for _, d := range dates {
		neos := feed.NearEarthObjects[d]
		b.WriteString(d + ":\n")
		for _, neo := range neos {
			hazardous := "no"
			if neo.IsPotentiallyHazardous {
				hazardous = "yes"
			}
			missKm := ""
			if len(neo.CloseApproachData) > 0 {
				missKm = fmt.Sprintf("%.2f", float64(neo.CloseApproachData[0].MissDistance.Kilometers))
			}
			fmt.Fprintf(&b, "  - %s (ID: %s, Hazardous: %s, Miss Distance: %s km)\n",
				neo.Name, neo.ID, hazardous, missKm)
		}
	}

	return textResult(b.String())
}

// handleNEOGet handles the nasa_neo_get tool. Returns detailed information
// about a specific asteroid identified by its SPK-ID.
func handleNEOGet(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	id := getStringArg(args, "asteroid_id")
	if id == "" {
		return errorResult("asteroid_id is required")
	}

	neo, err := client.NEO.Get(ctx, id)
	if err != nil {
		return errorResult("%v", err)
	}

	hazardous := "no"
	if neo.IsPotentiallyHazardous {
		hazardous = "yes"
	}

	var b strings.Builder
	b.WriteString(formatLine("Asteroid", neo.Name))
	b.WriteString(formatLine("ID", neo.ID))
	b.WriteString(formatLine("Hazardous", hazardous))
	b.WriteString(formatLine("Magnitude", fmt.Sprintf("%.2f", neo.AbsoluteMagnitude)))
	b.WriteString(formatLine("Diameter",
		fmt.Sprintf("%.4f - %.4f km (estimated)",
			neo.EstimatedDiameter.Kilometers.Min,
			neo.EstimatedDiameter.Kilometers.Max)))

	if len(neo.CloseApproachData) > 0 {
		b.WriteString("Close Approaches:\n")
		for _, ca := range neo.CloseApproachData {
			fmt.Fprintf(&b, "  - %s: %.0f km miss distance, %.2f km/s velocity\n",
				ca.CloseApproachDate,
				float64(ca.MissDistance.Kilometers),
				float64(ca.RelativeVelocity.KilometersPerSecond))
		}
	}

	return textResult(b.String())
}

// handleDONKICME handles the nasa_donki_cme tool. Returns coronal mass
// ejection events for an optional date range.
func handleDONKICME(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	start, end, err := parseDONKIDates(args)
	if err != nil {
		return errorResult("%v", err)
	}

	events, err := client.DONKI.CME(ctx, start, end)
	if err != nil {
		return errorResult("%v", err)
	}

	if len(events) == 0 {
		return textResult("No CME events found for the given date range.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Coronal Mass Ejections: %d events\n\n", len(events))
	for _, e := range events {
		b.WriteString(formatLine("Activity ID", e.ActivityID))
		if !e.StartTime.IsZero() {
			b.WriteString(formatLine("Start Time", e.StartTime.Format(time.RFC3339)))
		}
		b.WriteString(formatLine("Source Location", e.SourceLocation))
		if e.Note != "" {
			b.WriteString(formatLine("Note", e.Note))
		}
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleDONKIGST handles the nasa_donki_gst tool. Returns geomagnetic storm
// events with Kp index data.
func handleDONKIGST(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	start, end, err := parseDONKIDates(args)
	if err != nil {
		return errorResult("%v", err)
	}

	events, err := client.DONKI.GST(ctx, start, end)
	if err != nil {
		return errorResult("%v", err)
	}

	if len(events) == 0 {
		return textResult("No geomagnetic storm events found for the given date range.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Geomagnetic Storms: %d events\n\n", len(events))
	for _, e := range events {
		b.WriteString(formatLine("Storm ID", e.GstID))
		if !e.StartTime.IsZero() {
			b.WriteString(formatLine("Start Time", e.StartTime.Format(time.RFC3339)))
		}
		if len(e.AllKpIndex) > 0 {
			b.WriteString("Kp Index:\n")
			for _, kp := range e.AllKpIndex {
				fmt.Fprintf(&b, "  - %s: Kp=%.1f (source: %s)\n",
					kp.ObservedTime.Format(time.RFC3339), kp.KpIndex, kp.Source)
			}
		}
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleDONKIFLR handles the nasa_donki_flr tool. Returns solar flare events
// with class, timing, and source location.
func handleDONKIFLR(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	start, end, err := parseDONKIDates(args)
	if err != nil {
		return errorResult("%v", err)
	}

	events, err := client.DONKI.FLR(ctx, start, end)
	if err != nil {
		return errorResult("%v", err)
	}

	if len(events) == 0 {
		return textResult("No solar flare events found for the given date range.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Solar Flares: %d events\n\n", len(events))
	for _, e := range events {
		b.WriteString(formatLine("Flare ID", e.FlrID))
		if !e.BeginTime.IsZero() {
			b.WriteString(formatLine("Begin Time", e.BeginTime.Format(time.RFC3339)))
		}
		if !e.PeakTime.IsZero() {
			b.WriteString(formatLine("Peak Time", e.PeakTime.Format(time.RFC3339)))
		}
		if !e.EndTime.IsZero() {
			b.WriteString(formatLine("End Time", e.EndTime.Format(time.RFC3339)))
		}
		b.WriteString(formatLine("Class", e.ClassType))
		b.WriteString(formatLine("Source Location", e.SourceLocation))
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleDONKISEP handles the nasa_donki_sep tool. Returns solar energetic
// particle events with instrument and linked event data.
func handleDONKISEP(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	start, end, err := parseDONKIDates(args)
	if err != nil {
		return errorResult("%v", err)
	}

	events, err := client.DONKI.SEP(ctx, start, end)
	if err != nil {
		return errorResult("%v", err)
	}

	if len(events) == 0 {
		return textResult("No solar energetic particle events found for the given date range.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Solar Energetic Particles: %d events\n\n", len(events))
	for _, e := range events {
		b.WriteString(formatLine("SEP ID", e.SepID))
		if !e.EventTime.IsZero() {
			b.WriteString(formatLine("Event Time", e.EventTime.Format(time.RFC3339)))
		}
		if len(e.Instruments) > 0 {
			names := make([]string, len(e.Instruments))
			for i, inst := range e.Instruments {
				names[i] = inst.DisplayName
			}
			b.WriteString(formatLine("Instruments", strings.Join(names, ", ")))
		}
		if len(e.LinkedEvents) > 0 {
			ids := make([]string, len(e.LinkedEvents))
			for i, le := range e.LinkedEvents {
				ids[i] = le.ActivityID
			}
			b.WriteString(formatLine("Linked Events", strings.Join(ids, ", ")))
		}
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleDONKINotifications handles the nasa_donki_notifications tool. Returns
// space weather notifications, optionally filtered by type.
func handleDONKINotifications(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	start, end, err := parseDONKIDates(args)
	if err != nil {
		return errorResult("%v", err)
	}
	notifType := getStringArg(args, "type")

	notifications, err := client.DONKI.Notifications(ctx, start, end, notifType)
	if err != nil {
		return errorResult("%v", err)
	}

	if len(notifications) == 0 {
		return textResult("No DONKI notifications found for the given parameters.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "DONKI Notifications: %d\n\n", len(notifications))
	for _, n := range notifications {
		b.WriteString(formatLine("Message Type", n.MessageType))
		if !n.MessageIssueTime.IsZero() {
			b.WriteString(formatLine("Issue Time", n.MessageIssueTime.Format(time.RFC3339)))
		}
		body := n.MessageBody
		if len(body) > 2000 {
			body = body[:2000] + "... (truncated)"
		}
		b.WriteString(formatLine("Body", body))
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleImagesSearch handles the nasa_images_search tool. Searches the NASA
// Image and Video Library with optional media type and year filters.
func handleImagesSearch(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	q := getStringArg(args, "q")
	if q == "" {
		return errorResult("q (query) is required")
	}

	var opts []nasa.ImageSearchOption
	if mt := getStringArg(args, "media_type"); mt != "" {
		opts = append(opts, nasa.ImageWithMediaType(mt))
	}
	if ys, ok := getIntArg(args, "year_start"); ok {
		opts = append(opts, nasa.ImageWithYearStart(ys))
	}
	if ye, ok := getIntArg(args, "year_end"); ok {
		opts = append(opts, nasa.ImageWithYearEnd(ye))
	}

	coll, err := client.Images.Search(ctx, q, opts...)
	if err != nil {
		return errorResult("%v", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Results for '%s':\n\n", q)
	for i, item := range coll.Items {
		fmt.Fprintf(&b, "%d. %s\n", i+1, item.Title)
		b.WriteString(formatLine("   NASA ID", item.NasaID))
		if !item.DateCreated.IsZero() {
			b.WriteString(formatLine("   Date", item.DateCreated.Format("2006-01-02")))
		}
		if item.Description != "" {
			desc := item.Description
			if len(desc) > 500 {
				desc = desc[:500] + "..."
			}
			b.WriteString(formatLine("   Description", desc))
		}
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleSSDCAD handles the nasa_ssd_cad tool. Returns asteroid/comet close
// approach data from JPL's SSD/CNEOS database.
func handleSSDCAD(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	var opts []nasa.CADOption

	if dmin := getStringArg(args, "date_min"); dmin != "" {
		t, err := parseDate(dmin)
		if err != nil {
			return errorResult("invalid date_min: %v", err)
		}
		opts = append(opts, nasa.CADWithDateMin(t))
	}
	if dmax := getStringArg(args, "date_max"); dmax != "" {
		t, err := parseDate(dmax)
		if err != nil {
			return errorResult("invalid date_max: %v", err)
		}
		opts = append(opts, nasa.CADWithDateMax(t))
	}
	if distStr := getStringArg(args, "dist_max"); distStr != "" {
		dist, err := strconv.ParseFloat(distStr, 64)
		if err != nil {
			return errorResult("invalid dist_max: %v", err)
		}
		opts = append(opts, nasa.CADWithDistMax(dist))
	}
	if body := getStringArg(args, "body"); body != "" {
		opts = append(opts, nasa.CADWithBody(body))
	}

	data, err := client.SSD.CloseApproach(ctx, opts...)
	if err != nil {
		return errorResult("%v", err)
	}

	if len(data) == 0 {
		return textResult("No close approach data found for the given parameters.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Close Approach Data: %d objects\n\n", len(data))
	for _, d := range data {
		b.WriteString(formatLine("Object", d.Designation))
		b.WriteString(formatLine("Close Approach Date", d.CloseDate))
		b.WriteString(formatLine("Distance", fmt.Sprintf("%.6f AU (%.0f km)", d.Distance, d.Distance*1.496e8)))
		b.WriteString(formatLine("Relative Velocity", fmt.Sprintf("%.2f km/s", d.VRelative)))
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleSSDFireball handles the nasa_ssd_fireball tool. Returns fireball/bolide
// event data with optional client-side energy filtering.
func handleSSDFireball(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	var opts []nasa.FireballOption

	if dmin := getStringArg(args, "date_min"); dmin != "" {
		t, err := parseDate(dmin)
		if err != nil {
			return errorResult("invalid date_min: %v", err)
		}
		opts = append(opts, nasa.FireballWithDateMin(t))
	}
	if dmax := getStringArg(args, "date_max"); dmax != "" {
		t, err := parseDate(dmax)
		if err != nil {
			return errorResult("invalid date_max: %v", err)
		}
		opts = append(opts, nasa.FireballWithDateMax(t))
	}

	energyMin, hasEnergyMin := getFloatArg(args, "energy_min")

	events, err := client.SSD.Fireball(ctx, opts...)
	if err != nil {
		return errorResult("%v", err)
	}

	// Client-side energy filter since no SDK option exists for this.
	if hasEnergyMin {
		filtered := make([]*nasa.FireballEvent, 0, len(events))
		for _, e := range events {
			if e.Energy >= energyMin {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	if len(events) == 0 {
		return textResult("No fireball events found for the given parameters.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Fireball Events: %d\n\n", len(events))
	for _, e := range events {
		b.WriteString(formatLine("Date", e.Date))
		b.WriteString(formatLine("Energy", fmt.Sprintf("%.2f kt", e.Energy)))
		latDir := e.LatDir
		if latDir == "" {
			latDir = "N"
		}
		lonDir := e.LonDir
		if lonDir == "" {
			lonDir = "E"
		}
		b.WriteString(formatLine("Location", fmt.Sprintf("%.1f%s, %.1f%s", e.Lat, latDir, e.Lon, lonDir)))
		if e.Velocity > 0 {
			b.WriteString(formatLine("Velocity", fmt.Sprintf("%.1f km/s", e.Velocity)))
		}
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// handleEONETEvents handles the nasa_eonet_events tool. Returns natural events
// from NASA's Earth Observatory Natural Event Tracker.
func handleEONETEvents(ctx context.Context, args map[string]any, client *nasa.Client) *ToolResult {
	var opts []nasa.EONETOption

	if status := getStringArg(args, "status"); status != "" {
		opts = append(opts, nasa.EONETWithStatus(status))
	}
	if limit, ok := getIntArg(args, "limit"); ok {
		opts = append(opts, nasa.EONETWithLimit(limit))
	}
	if days, ok := getIntArg(args, "days"); ok {
		opts = append(opts, nasa.EONETWithDays(days))
	}
	if cat := getStringArg(args, "category"); cat != "" {
		opts = append(opts, nasa.EONETWithCategory(cat))
	}

	events, err := client.EONET.Events(ctx, opts...)
	if err != nil {
		return errorResult("%v", err)
	}

	if len(events) == 0 {
		return textResult("No EONET events found for the given parameters.")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "EONET Events: %d\n\n", len(events))
	for _, e := range events {
		b.WriteString(formatLine("Title", e.Title))
		if len(e.Categories) > 0 {
			b.WriteString(formatLine("Category", e.Categories[0].Title))
		}
		if len(e.Geometry) > 0 {
			g := e.Geometry[len(e.Geometry)-1]
			b.WriteString(formatLine("Date", g.Date.Format("2006-01-02")))
			if g.Type == "Point" {
				if coords, err := g.PointCoordinates(); err == nil {
					b.WriteString(formatLine("Coordinates", fmt.Sprintf("%.4f, %.4f", coords[1], coords[0])))
				}
			}
		}
		b.WriteString("\n")
	}

	return textResult(b.String())
}

// --- DONKI date helper ---

// parseDONKIDates extracts optional start_date and end_date from args. Returns
// an error when a non-empty date string fails to parse. Zero-value times for
// missing dates are interpreted by the DONKI SDK as "use API defaults".
func parseDONKIDates(args map[string]any) (start, end time.Time, err error) {
	if s := getStringArg(args, "start_date"); s != "" {
		start, err = parseDate(s)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date %q: expected YYYY-MM-DD", s)
		}
	}
	if s := getStringArg(args, "end_date"); s != "" {
		end, err = parseDate(s)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date %q: expected YYYY-MM-DD", s)
		}
	}
	return start, end, nil
}

// --- Tool definitions ---

// tools is the single source of truth for all MCP tools exposed by the server.
// Each entry pairs a Tool (name, description, JSON Schema) with its handler
// function. This slice is used by handleToolsList to report available tools
// and by handleToolsCall to dispatch incoming tool requests.
var tools = []ToolDef{
	{
		Tool: Tool{
			Name:        "nasa_apod",
			Description: "Get NASA Astronomy Picture of the Day. Returns the APOD for today, a specific date, or random entries.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date": map[string]any{
						"type":        "string",
						"description": "Date in YYYY-MM-DD format (default: today)",
					},
					"count": map[string]any{
						"type":        "integer",
						"description": "Number of random APOD entries to return",
					},
				},
			},
		},
		Handler: handleAPOD,
	},
	{
		Tool: Tool{
			Name:        "nasa_apod_range",
			Description: "Get Astronomy Picture of the Day for a date range.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format",
					},
					"end_date": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format",
					},
				},
				"required": []string{"start_date", "end_date"},
			},
		},
		Handler: handleAPODRange,
	},
	{
		Tool: Tool{
			Name:        "nasa_neo_feed",
			Description: "Get near-Earth objects for a date range (max 7 days).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format",
					},
					"end_date": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format (default: start + 6 days)",
					},
				},
				"required": []string{"start_date"},
			},
		},
		Handler: handleNEOFeed,
	},
	{
		Tool: Tool{
			Name:        "nasa_neo_get",
			Description: "Get details for a specific near-Earth asteroid by ID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"asteroid_id": map[string]any{
						"type":        "string",
						"description": "The asteroid SPK-ID (e.g., 3542519)",
					},
				},
				"required": []string{"asteroid_id"},
			},
		},
		Handler: handleNEOGet,
	},
	{
		Tool: Tool{
			Name:        "nasa_donki_cme",
			Description: "Get coronal mass ejection events from DONKI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format (default: 30 days ago)",
					},
					"end_date": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format (default: today)",
					},
				},
			},
		},
		Handler: handleDONKICME,
	},
	{
		Tool: Tool{
			Name:        "nasa_donki_gst",
			Description: "Get geomagnetic storm events from DONKI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format (default: 30 days ago)",
					},
					"end_date": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format (default: today)",
					},
				},
			},
		},
		Handler: handleDONKIGST,
	},
	{
		Tool: Tool{
			Name:        "nasa_donki_flr",
			Description: "Get solar flare events from DONKI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format (default: 30 days ago)",
					},
					"end_date": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format (default: today)",
					},
				},
			},
		},
		Handler: handleDONKIFLR,
	},
	{
		Tool: Tool{
			Name:        "nasa_donki_sep",
			Description: "Get solar energetic particle events from DONKI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format (default: 30 days ago)",
					},
					"end_date": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format (default: today)",
					},
				},
			},
		},
		Handler: handleDONKISEP,
	},
	{
		Tool: Tool{
			Name:        "nasa_donki_notifications",
			Description: "Get DONKI space weather notifications.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format (default: 30 days ago)",
					},
					"end_date": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format (default: today)",
					},
					"type": map[string]any{
						"type":        "string",
						"description": "Notification type filter",
						"enum":        []string{"all", "FLR", "SEP", "CME", "IPS", "MPC", "GST", "RBE", "report"},
					},
				},
			},
		},
		Handler: handleDONKINotifications,
	},
	{
		Tool: Tool{
			Name:        "nasa_images_search",
			Description: "Search the NASA Image and Video Library.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"q": map[string]any{
						"type":        "string",
						"description": "Search query (e.g., 'mars rover', 'apollo 11')",
					},
					"media_type": map[string]any{
						"type":        "string",
						"description": "Filter by media type: image, video, or audio",
					},
					"year_start": map[string]any{
						"type":        "integer",
						"description": "Filter results to items from this year or later",
					},
					"year_end": map[string]any{
						"type":        "integer",
						"description": "Filter results to items from this year or earlier",
					},
				},
				"required": []string{"q"},
			},
		},
		Handler: handleImagesSearch,
	},
	{
		Tool: Tool{
			Name:        "nasa_ssd_cad",
			Description: "Get asteroid/comet close approach data from JPL SSD/CNEOS.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date_min": map[string]any{
						"type":        "string",
						"description": "Minimum close-approach date in YYYY-MM-DD format",
					},
					"date_max": map[string]any{
						"type":        "string",
						"description": "Maximum close-approach date in YYYY-MM-DD format",
					},
					"dist_max": map[string]any{
						"type":        "string",
						"description": "Maximum approach distance in AU (e.g., '0.05')",
					},
					"body": map[string]any{
						"type":        "string",
						"description": "Filter by close-approach body (e.g., 'Earth', 'Mars')",
					},
				},
			},
		},
		Handler: handleSSDCAD,
	},
	{
		Tool: Tool{
			Name:        "nasa_ssd_fireball",
			Description: "Get fireball/bolide event data from JPL SSD/CNEOS.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date_min": map[string]any{
						"type":        "string",
						"description": "Minimum event date in YYYY-MM-DD format",
					},
					"date_max": map[string]any{
						"type":        "string",
						"description": "Maximum event date in YYYY-MM-DD format",
					},
					"energy_min": map[string]any{
						"type":        "number",
						"description": "Minimum energy in kilotons (client-side filter)",
					},
				},
			},
		},
		Handler: handleSSDFireball,
	},
	{
		Tool: Tool{
			Name:        "nasa_eonet_events",
			Description: "Get natural events from the Earth Observatory Natural Event Tracker.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{
						"type":        "string",
						"description": "Filter by status: 'open' or 'closed'",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of events to return",
					},
					"days": map[string]any{
						"type":        "integer",
						"description": "Filter to events within the last n days",
					},
					"category": map[string]any{
						"type":        "string",
						"description": "Filter by category identifier",
					},
				},
			},
		},
		Handler: handleEONETEvents,
	},
}
