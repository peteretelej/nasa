package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"
)

// printJSON marshals v as indented JSON to stdout.
func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal("failed to marshal JSON: %v", err)
	}
	fmt.Println(string(data))
}

// printHeader prints a title with underline.
func printHeader(title string) {
	fmt.Println(title)
	for range len(title) {
		fmt.Print("=")
	}
	fmt.Println()
}

// newTable creates a tabwriter with the given column headers.
func newTable(headers ...string) *tabwriter.Writer {
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			mustWriteString(tw, "\t")
		}
		mustWriteString(tw, h)
	}
	mustWriteString(tw, "\n")
	return tw
}

// flushTable flushes the tabwriter.
func flushTable(tw *tabwriter.Writer) {
	if err := tw.Flush(); err != nil {
		fatal("failed to flush output: %v", err)
	}
}

func tablePrintf(tw *tabwriter.Writer, format string, args ...any) {
	if _, err := fmt.Fprintf(tw, format, args...); err != nil {
		fatal("failed to write output: %v", err)
	}
}

func mustWriteString(w io.Writer, s string) {
	if _, err := io.WriteString(w, s); err != nil {
		fatal("failed to write output: %v", err)
	}
}

// formatDate returns a date as YYYY-MM-DD.
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// formatBool returns trueVal or falseVal based on b.
func formatBool(b bool, trueVal, falseVal string) string {
	if b {
		return trueVal
	}
	return falseVal
}

// formatFloat formats a float64 with the given precision.
func formatFloat(f float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, f)
}

// fatal prints a formatted error to stderr and exits with code 1.
func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "nasa: "+format+"\n", args...)
	os.Exit(1)
}

// getAPIKey resolves the API key from flag value, environment, or DEMO_KEY.
func getAPIKey(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if key := os.Getenv("NASA_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("NASAKEY"); key != "" {
		return key
	}
	fmt.Fprintln(os.Stderr, "nasa: warning: no API key set, using DEMO_KEY (limited to 30 req/hr)")
	return "DEMO_KEY"
}

// truncate shortens s to maxLen chars, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
