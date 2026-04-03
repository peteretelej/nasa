// Command update-testdata fetches real NASA API responses and writes them to
// testdata/ as fixture files. It refreshes the 27 primary fixtures while
// leaving the 13 hand-crafted edge-case fixtures untouched.
//
// Usage:
//
//	NASA_API_KEY=your-key go run ./cmd/update-testdata
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	apiKey := loadAPIKey()
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: NASA_API_KEY not set (checked env and .env file)")
		os.Exit(1)
	}
	fmt.Printf("Using API key: %s...%s\n", apiKey[:4], apiKey[len(apiKey)-4:])

	testdataDir := "testdata"

	// Phase A: static fixtures with hardcoded URLs.
	fmt.Println("\n=== Phase A: static fixtures ===")

	// APOD
	apodSingle := mustFetch(apiKey, "apod_single.json",
		"https://api.nasa.gov/planetary/apod?date=2024-01-15&thumbs=true&api_key="+apiKey, testdataDir)
	writeFixture(testdataDir, "apod_single.json", apodSingle)

	apodVideo := fetchVideoAPOD(apiKey)
	writeFixture(testdataDir, "apod_single_video.json", apodVideo)

	apodRange := mustFetch(apiKey, "apod_range.json",
		"https://api.nasa.gov/planetary/apod?start_date=2024-01-10&end_date=2024-01-12&thumbs=true&api_key="+apiKey, testdataDir)
	writeFixture(testdataDir, "apod_range.json", apodRange)

	// NEO
	neoFeed := mustFetch(apiKey, "neo_feed.json",
		"https://api.nasa.gov/neo/rest/v1/feed?start_date=2024-01-15&end_date=2024-01-15&api_key="+apiKey, testdataDir)
	writeFixture(testdataDir, "neo_feed.json", neoFeed)

	neoBrowse := mustFetch(apiKey, "neo_browse.json",
		"https://api.nasa.gov/neo/rest/v1/neo/browse?page=0&size=2&api_key="+apiKey, testdataDir)
	writeFixture(testdataDir, "neo_browse.json", neoBrowse)

	// EONET (no API key needed)
	eonetEvents := mustFetch(apiKey, "eonet_events.json",
		"https://eonet.gsfc.nasa.gov/api/v3/events?limit=5", testdataDir)
	writeFixture(testdataDir, "eonet_events.json", eonetEvents)

	eonetCategories := mustFetch(apiKey, "eonet_categories.json",
		"https://eonet.gsfc.nasa.gov/api/v3/categories", testdataDir)
	writeFixture(testdataDir, "eonet_categories.json", eonetCategories)

	eonetSources := mustFetch(apiKey, "eonet_sources.json",
		"https://eonet.gsfc.nasa.gov/api/v3/sources", testdataDir)
	writeFixture(testdataDir, "eonet_sources.json", eonetSources)

	// DONKI
	donkiCME := mustFetchTruncateArray(apiKey, "donki_cme.json",
		"https://api.nasa.gov/DONKI/CME?startDate=2024-01-10&endDate=2024-01-15&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_cme.json", donkiCME)

	donkiCMEAnalysis := mustFetchTruncateArray(apiKey, "donki_cme_analysis.json",
		"https://api.nasa.gov/DONKI/CMEAnalysis?startDate=2024-01-10&endDate=2024-01-15&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_cme_analysis.json", donkiCMEAnalysis)

	donkiGST := mustFetchTruncateArray(apiKey, "donki_gst.json",
		"https://api.nasa.gov/DONKI/GST?startDate=2024-01-01&endDate=2024-01-31&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_gst.json", donkiGST)

	donkiIPS := mustFetchTruncateArray(apiKey, "donki_ips.json",
		"https://api.nasa.gov/DONKI/IPS?startDate=2024-01-10&endDate=2024-01-20&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_ips.json", donkiIPS)

	donkiFLR := mustFetchTruncateArray(apiKey, "donki_flr.json",
		"https://api.nasa.gov/DONKI/FLR?startDate=2024-01-10&endDate=2024-01-15&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_flr.json", donkiFLR)

	donkiSEP := mustFetchTruncateArray(apiKey, "donki_sep.json",
		"https://api.nasa.gov/DONKI/SEP?startDate=2024-01-01&endDate=2024-01-31&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_sep.json", donkiSEP)

	donkiMPC := mustFetchTruncateArray(apiKey, "donki_mpc.json",
		"https://api.nasa.gov/DONKI/MPC?startDate=2024-01-01&endDate=2024-01-31&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_mpc.json", donkiMPC)

	donkiRBE := mustFetchTruncateArray(apiKey, "donki_rbe.json",
		"https://api.nasa.gov/DONKI/RBE?startDate=2024-01-01&endDate=2024-01-31&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_rbe.json", donkiRBE)

	donkiHSS := mustFetchTruncateArray(apiKey, "donki_hss.json",
		"https://api.nasa.gov/DONKI/HSS?startDate=2024-01-01&endDate=2024-01-31&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_hss.json", donkiHSS)

	donkiWSA := mustFetchTruncateArray(apiKey, "donki_wsa_enlil.json",
		"https://api.nasa.gov/DONKI/WSAEnlilSimulations?startDate=2024-01-01&endDate=2024-01-31&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_wsa_enlil.json", donkiWSA)

	donkiNotif := mustFetchTruncateArray(apiKey, "donki_notifications.json",
		"https://api.nasa.gov/DONKI/notifications?startDate=2024-01-10&endDate=2024-01-15&api_key="+apiKey, 3, testdataDir)
	writeFixture(testdataDir, "donki_notifications.json", donkiNotif)

	// Images
	imagesSearch := mustFetch(apiKey, "images_search.json",
		"https://images-api.nasa.gov/search?q=apollo+11&page_size=2", testdataDir)
	writeFixture(testdataDir, "images_search.json", imagesSearch)

	// SSD/CNEOS
	ssdCAD := mustFetch(apiKey, "ssd_cad.json",
		"https://ssd-api.jpl.nasa.gov/cad.api?limit=3", testdataDir)
	ssdCAD = truncateColumnarData(ssdCAD, 3)
	writeFixture(testdataDir, "ssd_cad.json", ssdCAD)

	ssdFireball := mustFetch(apiKey, "ssd_fireball.json",
		"https://ssd-api.jpl.nasa.gov/fireball.api?req-loc=true&limit=3", testdataDir)
	ssdFireball = truncateColumnarData(ssdFireball, 3)
	writeFixture(testdataDir, "ssd_fireball.json", ssdFireball)

	ssdSentry := mustFetch(apiKey, "ssd_sentry.json",
		"https://ssd-api.jpl.nasa.gov/sentry.api", testdataDir)
	ssdSentry = truncateSentryData(ssdSentry, 3)
	writeFixture(testdataDir, "ssd_sentry.json", ssdSentry)

	ssdSBDB := mustFetch(apiKey, "ssd_sbdb.json",
		"https://ssd-api.jpl.nasa.gov/sbdb.api?des=99942", testdataDir)
	writeFixture(testdataDir, "ssd_sbdb.json", ssdSBDB)

	// Phase B: dynamic fixtures needing IDs from Phase A.
	fmt.Println("\n=== Phase B: dynamic fixtures ===")

	// NEO single: extract first asteroid ID from feed
	neoID := extractNEOID(neoFeed)
	fmt.Printf("  Extracted NEO ID: %s\n", neoID)
	neoSingle := mustFetch(apiKey, "neo_single.json",
		"https://api.nasa.gov/neo/rest/v1/neo/"+neoID+"?api_key="+apiKey, testdataDir)
	writeFixture(testdataDir, "neo_single.json", neoSingle)

	// EONET single event: extract first event ID from events
	eonetID := extractEONETEventID(eonetEvents)
	fmt.Printf("  Extracted EONET event ID: %s\n", eonetID)
	eonetSingle := mustFetch(apiKey, "eonet_event_single.json",
		"https://eonet.gsfc.nasa.gov/api/v3/events/"+eonetID, testdataDir)
	writeFixture(testdataDir, "eonet_event_single.json", eonetSingle)

	// Images asset: extract first NASA ID from search
	nasaID := extractNASAImageID(imagesSearch)
	fmt.Printf("  Extracted NASA Image ID: %s\n", nasaID)
	imagesAsset := mustFetch(apiKey, "images_asset.json",
		"https://images-api.nasa.gov/asset/"+nasaID, testdataDir)
	writeFixture(testdataDir, "images_asset.json", imagesAsset)

	// Suppress unused variable warnings for fixtures that are only written.
	_ = apodSingle
	_ = apodRange
	_ = neoBrowse
	_ = eonetCategories
	_ = eonetSources
	_ = donkiCME
	_ = donkiCMEAnalysis
	_ = donkiGST
	_ = donkiIPS
	_ = donkiFLR
	_ = donkiSEP
	_ = donkiMPC
	_ = donkiRBE
	_ = donkiHSS
	_ = donkiWSA
	_ = donkiNotif
	_ = ssdCAD
	_ = ssdFireball
	_ = ssdSentry
	_ = ssdSBDB

	fmt.Println("\nDone! All 27 primary fixtures updated.")
}

// loadAPIKey reads the API key from the NASA_API_KEY environment variable,
// falling back to parsing a .env file in the current directory.
func loadAPIKey() string {
	if key := os.Getenv("NASA_API_KEY"); key != "" {
		return key
	}
	f, err := os.Open(".env")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			if strings.TrimSpace(k) == "NASA_API_KEY" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

// mustFetch fetches a URL and returns the raw JSON bytes. It exits on error.
func mustFetch(_ string, name, url, _ string) []byte {
	rateLimitSleep()
	fmt.Printf("  Fetching %s ...", name)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nerror fetching %s: %v\n", name, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nerror reading %s: %v\n", name, err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "\nerror: %s returned HTTP %d: %s\n", name, resp.StatusCode, string(body[:min(200, len(body))]))
		os.Exit(1)
	}

	fmt.Printf(" OK (%d bytes)\n", len(body))
	return body
}

// mustFetchTruncateArray fetches a URL expecting a JSON array, truncates to
// maxItems elements if the array is longer, and returns the JSON bytes.
func mustFetchTruncateArray(apiKey, name, url string, maxItems int, testdataDir string) []byte {
	raw := mustFetch(apiKey, name, url, testdataDir)

	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		// Not an array (could be empty or error). Return as-is.
		fmt.Printf("    warning: %s is not a JSON array, keeping as-is\n", name)
		return raw
	}

	if len(arr) > maxItems {
		fmt.Printf("    Truncating %s from %d to %d items\n", name, len(arr), maxItems)
		arr = arr[:maxItems]
		out, err := json.Marshal(arr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling truncated %s: %v\n", name, err)
			os.Exit(1)
		}
		return out
	}
	return raw
}

// fetchVideoAPOD tries candidate dates known to have video APODs. If none
// work, it fetches random APODs until finding one with media_type "video".
func fetchVideoAPOD(apiKey string) []byte {
	candidates := []string{"2024-04-09", "2024-03-10", "2024-01-01", "2023-12-31"}

	for _, date := range candidates {
		rateLimitSleep()
		fmt.Printf("  Trying video APOD date %s ...", date)

		url := fmt.Sprintf("https://api.nasa.gov/planetary/apod?date=%s&thumbs=true&api_key=%s", date, apiKey)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf(" error: %v\n", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf(" HTTP %d\n", resp.StatusCode)
			continue
		}

		var apod map[string]interface{}
		if err := json.Unmarshal(body, &apod); err != nil {
			fmt.Printf(" parse error\n")
			continue
		}

		if mt, ok := apod["media_type"].(string); ok && mt == "video" {
			fmt.Printf(" OK (video found)\n")
			return body
		}
		fmt.Printf(" not a video (media_type=%v)\n", apod["media_type"])
	}

	// Fallback: fetch random APODs by requesting count=20
	fmt.Println("  Trying random APOD batch for video...")
	rateLimitSleep()
	url := fmt.Sprintf("https://api.nasa.gov/planetary/apod?count=20&thumbs=true&api_key=%s", apiKey)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching random APODs: %v\n", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var apods []map[string]interface{}
	if err := json.Unmarshal(body, &apods); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing random APODs: %v\n", err)
		os.Exit(1)
	}

	for _, apod := range apods {
		if mt, ok := apod["media_type"].(string); ok && mt == "video" {
			out, _ := json.Marshal(apod)
			fmt.Printf("  Found video APOD: %s\n", apod["date"])
			return out
		}
	}

	fmt.Fprintln(os.Stderr, "error: could not find any video APOD after exhaustive search")
	os.Exit(1)
	return nil
}

// truncateColumnarData truncates the "data" array in a columnar SSD response
// (like CAD or Fireball) to maxRows rows.
func truncateColumnarData(raw []byte, maxRows int) []byte {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw
	}

	dataRaw, ok := obj["data"]
	if !ok {
		return raw
	}

	var data []json.RawMessage
	if err := json.Unmarshal(dataRaw, &data); err != nil {
		return raw
	}

	if len(data) > maxRows {
		fmt.Printf("    Truncating columnar data from %d to %d rows\n", len(data), maxRows)
		data = data[:maxRows]
		truncated, _ := json.Marshal(data)
		obj["data"] = truncated

		// Update count field if present
		if _, hasCount := obj["count"]; hasCount {
			countStr := fmt.Sprintf(`"%d"`, maxRows)
			obj["count"] = json.RawMessage(countStr)
		}
	}

	out, _ := json.Marshal(obj)
	return out
}

// truncateSentryData truncates the "data" array in a Sentry API response
// (object array format) to maxItems elements.
func truncateSentryData(raw []byte, maxItems int) []byte {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw
	}

	dataRaw, ok := obj["data"]
	if !ok {
		return raw
	}

	var data []json.RawMessage
	if err := json.Unmarshal(dataRaw, &data); err != nil {
		return raw
	}

	if len(data) > maxItems {
		fmt.Printf("    Truncating sentry data from %d to %d items\n", len(data), maxItems)
		data = data[:maxItems]
		truncated, _ := json.Marshal(data)
		obj["data"] = truncated

		// Update count field
		if _, hasCount := obj["count"]; hasCount {
			obj["count"] = json.RawMessage(fmt.Sprintf("%d", maxItems))
		}
	}

	out, _ := json.Marshal(obj)
	return out
}

// extractNEOID extracts the first asteroid ID from a NEO feed response.
func extractNEOID(raw []byte) string {
	var feed struct {
		NearEarthObjects map[string][]struct {
			ID string `json:"id"`
		} `json:"near_earth_objects"`
	}
	if err := json.Unmarshal(raw, &feed); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing NEO feed for ID extraction: %v\n", err)
		os.Exit(1)
	}
	for _, objects := range feed.NearEarthObjects {
		if len(objects) > 0 {
			return objects[0].ID
		}
	}
	fmt.Fprintln(os.Stderr, "error: no asteroids found in NEO feed")
	os.Exit(1)
	return ""
}

// extractEONETEventID extracts the first event ID from an EONET events response.
func extractEONETEventID(raw []byte) string {
	var resp struct {
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing EONET events for ID extraction: %v\n", err)
		os.Exit(1)
	}
	if len(resp.Events) == 0 {
		fmt.Fprintln(os.Stderr, "error: no events found in EONET response")
		os.Exit(1)
	}
	return resp.Events[0].ID
}

// extractNASAImageID extracts the first nasa_id from an Images search response.
func extractNASAImageID(raw []byte) string {
	var resp struct {
		Collection struct {
			Items []struct {
				Data []struct {
					NasaID string `json:"nasa_id"`
				} `json:"data"`
			} `json:"items"`
		} `json:"collection"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing Images search for ID extraction: %v\n", err)
		os.Exit(1)
	}
	if len(resp.Collection.Items) > 0 && len(resp.Collection.Items[0].Data) > 0 {
		return resp.Collection.Items[0].Data[0].NasaID
	}
	fmt.Fprintln(os.Stderr, "error: no images found in search response")
	os.Exit(1)
	return ""
}

// writeFixture pretty-prints JSON with 2-space indent and writes to testdata/.
// It scrubs any embedded API key from the output to avoid leaking secrets into
// committed fixture files.
func writeFixture(dir, name string, raw []byte) {
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s contains invalid JSON: %v\n", name, err)
		os.Exit(1)
	}
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error pretty-printing %s: %v\n", name, err)
		os.Exit(1)
	}

	// Scrub the real API key from any embedded URLs (e.g. NEO self/next links).
	apiKey := loadAPIKey()
	if apiKey != "" && apiKey != "DEMO_KEY" {
		pretty = []byte(strings.ReplaceAll(string(pretty), apiKey, "DEMO_KEY"))
	}

	// Ensure trailing newline.
	pretty = append(pretty, '\n')

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, pretty, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("  Wrote %s (%d bytes)\n", path, len(pretty))
}

var lastRequest time.Time

// rateLimitSleep enforces a 500ms gap between HTTP requests.
func rateLimitSleep() {
	if !lastRequest.IsZero() {
		elapsed := time.Since(lastRequest)
		if elapsed < 500*time.Millisecond {
			time.Sleep(500*time.Millisecond - elapsed)
		}
	}
	lastRequest = time.Now()
}
