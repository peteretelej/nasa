package nasa

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// setupDONKITestServer creates an httptest server that serves the given fixture
// file and returns a Client configured to use it. The handler captures the last
// request for parameter verification.
func setupDONKITestServer(t *testing.T, fixturePath string) (*Client, **http.Request) {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixturePath, err)
	}
	var captured *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing DONKI fixture: %v", err)
		}
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	return c, &captured
}

// setupDONKIErrorServer creates an httptest server that returns the given status
// code and body, and returns a Client configured to use it.
func setupDONKIErrorServer(t *testing.T, statusCode int, body string) *Client {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("writing DONKI error response: %v", err)
		}
	}))
	t.Cleanup(ts.Close)
	return NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
}

func TestParseHeliographicCoord(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    HeliographicCoord
		wantErr bool
	}{
		{"north east", "N10E80", HeliographicCoord{Latitude: 10, Longitude: 80}, false},
		{"south west", "S25W60", HeliographicCoord{Latitude: -25, Longitude: -60}, false},
		{"zero zero", "N00E00", HeliographicCoord{Latitude: 0, Longitude: 0}, false},
		{"extreme values", "S90W180", HeliographicCoord{Latitude: -90, Longitude: -180}, false},
		{"empty string", "", HeliographicCoord{}, true},
		{"backside", "backside", HeliographicCoord{}, true},
		{"incomplete", "N10", HeliographicCoord{}, true},
		{"invalid direction", "X10E80", HeliographicCoord{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHeliographicCoord(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDONKITime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, dt DONKITime)
	}{
		{
			"no seconds",
			`"2024-01-15T12:00Z"`,
			false,
			func(t *testing.T, dt DONKITime) {
				want := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
				if !dt.Equal(want) {
					t.Errorf("got %v, want %v", dt, want)
				}
			},
		},
		{
			"with seconds",
			`"2024-01-15T12:00:00Z"`,
			false,
			func(t *testing.T, dt DONKITime) {
				want := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
				if !dt.Equal(want) {
					t.Errorf("got %v, want %v", dt, want)
				}
			},
		},
		{
			"no timezone",
			`"2024-01-15T12:00:00"`,
			false,
			func(t *testing.T, dt DONKITime) {
				if dt.Year() != 2024 || dt.Month() != 1 || dt.Day() != 15 {
					t.Errorf("unexpected date: %v", dt)
				}
				if dt.Hour() != 12 || dt.Minute() != 0 {
					t.Errorf("unexpected time: %v", dt)
				}
			},
		},
		{
			"no seconds no timezone",
			`"2024-01-15T12:00"`,
			false,
			func(t *testing.T, dt DONKITime) {
				if dt.Year() != 2024 || dt.Hour() != 12 {
					t.Errorf("unexpected parse: %v", dt)
				}
			},
		},
		{
			"with offset",
			`"2024-01-15T12:00:00-07:00"`,
			false,
			func(t *testing.T, dt DONKITime) {
				// Should parse to 12:00 in -07:00 timezone.
				if dt.Hour() != 12 {
					t.Errorf("hour = %d, want 12", dt.Hour())
				}
			},
		},
		{
			"null value",
			`null`,
			false,
			func(t *testing.T, dt DONKITime) {
				if !dt.IsZero() {
					t.Errorf("expected zero time for null, got %v", dt)
				}
			},
		},
		{
			"invalid string",
			`"not-a-time"`,
			true,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dt DONKITime
			err := json.Unmarshal([]byte(tt.input), &dt)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, dt)
			}
		})
	}
}

// --- Per-endpoint fixture tests ---

func TestDONKI_CME(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_cme.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.CME(context.Background(), start, end)
	if err != nil {
		t.Fatalf("CME() error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	e := events[0]
	if e.ActivityID != "2024-01-10T00:12:00-CME-001" {
		t.Errorf("ActivityID = %q, want %q", e.ActivityID, "2024-01-10T00:12:00-CME-001")
	}
	if e.SourceLocation != "N15E80" {
		t.Errorf("SourceLocation = %q, want %q", e.SourceLocation, "N15E80")
	}
	if e.LinkedEvents != nil {
		t.Errorf("LinkedEvents = %v, want nil for null", e.LinkedEvents)
	}
	if len(e.CMEAnalyses) != 1 {
		t.Errorf("CMEAnalyses count = %d, want 1", len(e.CMEAnalyses))
	}
	if e.CMEAnalyses[0].Speed != 402 {
		t.Errorf("CMEAnalyses[0].Speed = %f, want 402", e.CMEAnalyses[0].Speed)
	}

	// Verify request path and params.
	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/CME") {
		t.Errorf("path = %q, want prefix /DONKI/CME", req.URL.Path)
	}
	q := req.URL.Query()
	if q.Get("startDate") != "2024-01-15" {
		t.Errorf("startDate = %q, want 2024-01-15", q.Get("startDate"))
	}
	if q.Get("endDate") != "2024-01-17" {
		t.Errorf("endDate = %q, want 2024-01-17", q.Get("endDate"))
	}
	if q.Get("api_key") != "TEST_KEY" {
		t.Errorf("api_key = %q, want TEST_KEY", q.Get("api_key"))
	}
}

func TestDONKI_CMEAnalysis(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_cme_analysis.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.CMEAnalysis(context.Background(), start, end)
	if err != nil {
		t.Fatalf("CMEAnalysis() error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	e := events[0]
	if e.AssociatedCMEID != "2024-01-10T00:12:00-CME-001" {
		t.Errorf("AssociatedCMEID = %q, want %q", e.AssociatedCMEID, "2024-01-10T00:12:00-CME-001")
	}
	if e.Speed != 402 {
		t.Errorf("Speed = %f, want 402", e.Speed)
	}
	if !e.IsMostAccurate {
		t.Error("expected IsMostAccurate to be true")
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/CMEAnalysis") {
		t.Errorf("path = %q, want prefix /DONKI/CMEAnalysis", req.URL.Path)
	}
}

func TestDONKI_GST(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_gst.json")

	start := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.GST(context.Background(), start, end)
	if err != nil {
		t.Fatalf("GST() error: %v", err)
	}

	// Fixture returned empty array for the chosen date range.
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/GST") {
		t.Errorf("path = %q, want prefix /DONKI/GST", req.URL.Path)
	}
}

func TestDONKI_IPS(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_ips.json")

	start := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.IPS(context.Background(), start, end)
	if err != nil {
		t.Fatalf("IPS() error: %v", err)
	}

	// Fixture returned empty array for the chosen date range.
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/IPS") {
		t.Errorf("path = %q, want prefix /DONKI/IPS", req.URL.Path)
	}
}

func TestDONKI_FLR(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_flr.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.FLR(context.Background(), start, end)
	if err != nil {
		t.Fatalf("FLR() error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	e := events[0]
	if e.FlrID != "2024-01-10T12:39:00-FLR-001" {
		t.Errorf("FlrID = %q, want %q", e.FlrID, "2024-01-10T12:39:00-FLR-001")
	}
	if e.ClassType != "M1.4" {
		t.Errorf("ClassType = %q, want M1.4", e.ClassType)
	}
	if e.BeginTime.IsZero() {
		t.Error("expected BeginTime to be non-zero")
	}
	if e.PeakTime.IsZero() {
		t.Error("expected PeakTime to be non-zero")
	}
	if e.EndTime.IsZero() {
		t.Error("expected EndTime to be non-zero")
	}
	if e.LinkedEvents != nil {
		t.Errorf("LinkedEvents = %v, want nil for null", e.LinkedEvents)
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/FLR") {
		t.Errorf("path = %q, want prefix /DONKI/FLR", req.URL.Path)
	}
}

func TestDONKI_SEP(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_sep.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.SEP(context.Background(), start, end)
	if err != nil {
		t.Fatalf("SEP() error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	e := events[0]
	if e.SepID != "2024-01-02T23:22:00-SEP-001" {
		t.Errorf("SepID = %q, want %q", e.SepID, "2024-01-02T23:22:00-SEP-001")
	}
	if len(e.LinkedEvents) != 2 {
		t.Errorf("LinkedEvents count = %d, want 2", len(e.LinkedEvents))
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/SEP") {
		t.Errorf("path = %q, want prefix /DONKI/SEP", req.URL.Path)
	}
}

func TestDONKI_MPC(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_mpc.json")

	start := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.MPC(context.Background(), start, end)
	if err != nil {
		t.Fatalf("MPC() error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	e := events[0]
	if e.MpcID != "2024-01-22T21:45:00-MPC-001" {
		t.Errorf("MpcID = %q, want %q", e.MpcID, "2024-01-22T21:45:00-MPC-001")
	}
	// linkedEvents is null in fixture; should be nil.
	if e.LinkedEvents != nil {
		t.Errorf("LinkedEvents = %v, want nil for null", e.LinkedEvents)
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/MPC") {
		t.Errorf("path = %q, want prefix /DONKI/MPC", req.URL.Path)
	}
}

func TestDONKI_RBE(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_rbe.json")

	start := time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.RBE(context.Background(), start, end)
	if err != nil {
		t.Fatalf("RBE() error: %v", err)
	}

	// Fixture returned empty array for the chosen date range.
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/RBE") {
		t.Errorf("path = %q, want prefix /DONKI/RBE", req.URL.Path)
	}
}

func TestDONKI_HSS(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_hss.json")

	start := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.HSS(context.Background(), start, end)
	if err != nil {
		t.Fatalf("HSS() error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	e := events[0]
	if e.HssID != "2024-01-29T12:52:00-HSS-001" {
		t.Errorf("HssID = %q, want %q", e.HssID, "2024-01-29T12:52:00-HSS-001")
	}
	if len(e.Instruments) != 3 {
		t.Errorf("Instruments count = %d, want 3", len(e.Instruments))
	}
	if e.LinkedEvents != nil {
		t.Errorf("LinkedEvents = %v, want nil for null", e.LinkedEvents)
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/HSS") {
		t.Errorf("path = %q, want prefix /DONKI/HSS", req.URL.Path)
	}
}

func TestDONKI_WSAEnlil(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_wsa_enlil.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.WSAEnlil(context.Background(), start, end)
	if err != nil {
		t.Fatalf("WSAEnlil() error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	e := events[0]
	if e.SimulationID != "WSA-ENLIL/28426/1" {
		t.Errorf("SimulationID = %q, want %q", e.SimulationID, "WSA-ENLIL/28426/1")
	}
	if len(e.ImpactList) != 1 {
		t.Errorf("ImpactList count = %d, want 1", len(e.ImpactList))
	}
	if e.ImpactList[0].Location != "Mars" {
		t.Errorf("ImpactList[0].Location = %q, want Mars", e.ImpactList[0].Location)
	}
	if len(e.CMEInputs) != 1 {
		t.Errorf("CMEInputs count = %d, want 1", len(e.CMEInputs))
	}
	if e.EstimatedDuration != nil {
		t.Errorf("EstimatedDuration = %v, want nil", e.EstimatedDuration)
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/WSAEnlilSimulations") {
		t.Errorf("path = %q, want prefix /DONKI/WSAEnlilSimulations", req.URL.Path)
	}
}

func TestDONKI_Notifications(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_notifications.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.Notifications(context.Background(), start, end, "")
	if err != nil {
		t.Fatalf("Notifications() error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	e := events[0]
	if e.MessageType != "CME" {
		t.Errorf("MessageType = %q, want CME", e.MessageType)
	}
	if e.MessageID != "20240114-AL-001" {
		t.Errorf("MessageID = %q, want 20240114-AL-001", e.MessageID)
	}
	if e.MessageBody == "" {
		t.Error("expected non-empty MessageBody")
	}
	// Notifications in this fixture don't include linkedEvents.
	if e.LinkedEvents != nil {
		t.Errorf("LinkedEvents = %v, want nil", e.LinkedEvents)
	}

	req := *captured
	if !strings.HasPrefix(req.URL.Path, "/DONKI/notifications") {
		t.Errorf("path = %q, want prefix /DONKI/notifications", req.URL.Path)
	}
}

// --- Option parameter tests ---

func TestDONKI_CMEAnalysis_Options(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_cme_analysis.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	_, err := c.DONKI.CMEAnalysis(context.Background(), start, end,
		WithMostAccurateOnly(true),
		WithSpeed(500),
		WithHalfAngle(10),
		WithCatalog("ALL"),
	)
	if err != nil {
		t.Fatalf("CMEAnalysis() error: %v", err)
	}

	req := *captured
	q := req.URL.Query()
	if q.Get("mostAccurateOnly") != "true" {
		t.Errorf("mostAccurateOnly = %q, want true", q.Get("mostAccurateOnly"))
	}
	if q.Get("speed") != "500" {
		t.Errorf("speed = %q, want 500", q.Get("speed"))
	}
	if q.Get("halfAngle") != "10" {
		t.Errorf("halfAngle = %q, want 10", q.Get("halfAngle"))
	}
	if q.Get("catalog") != "ALL" {
		t.Errorf("catalog = %q, want ALL", q.Get("catalog"))
	}
}

func TestDONKI_IPS_Options(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_ips.json")

	start := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC)

	_, err := c.DONKI.IPS(context.Background(), start, end,
		WithLocation("Earth"),
		WithIPSCatalog("SWPC_CATALOG"),
	)
	if err != nil {
		t.Fatalf("IPS() error: %v", err)
	}

	req := *captured
	q := req.URL.Query()
	if q.Get("location") != "Earth" {
		t.Errorf("location = %q, want Earth", q.Get("location"))
	}
	if q.Get("catalog") != "SWPC_CATALOG" {
		t.Errorf("catalog = %q, want SWPC_CATALOG", q.Get("catalog"))
	}
}

func TestDONKI_Notifications_Type(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_notifications.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

	_, err := c.DONKI.Notifications(context.Background(), start, end, "all")
	if err != nil {
		t.Fatalf("Notifications() error: %v", err)
	}

	req := *captured
	q := req.URL.Query()
	if q.Get("type") != "all" {
		t.Errorf("type = %q, want all", q.Get("type"))
	}
}

// --- Edge case tests ---

func TestDONKI_EmptyResponse(t *testing.T) {
	c, _ := setupDONKITestServer(t, "testdata/donki_empty.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.CME(context.Background(), start, end)
	if err != nil {
		t.Fatalf("CME() error: %v", err)
	}

	if events == nil {
		t.Fatal("expected non-nil slice for empty response")
	}
	if len(events) != 0 {
		t.Errorf("got %d events, want 0", len(events))
	}
}

func TestDONKI_DefaultDateRange(t *testing.T) {
	c, captured := setupDONKITestServer(t, "testdata/donki_empty.json")

	// Pass zero-value times to trigger the default date range.
	_, err := c.DONKI.CME(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("CME() error: %v", err)
	}

	req := *captured
	q := req.URL.Query()
	startDate := q.Get("startDate")
	if startDate == "" {
		t.Fatal("expected startDate param to be set")
	}

	// Parse the startDate and verify it's approximately 30 days ago.
	parsed, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		t.Fatalf("parsing startDate %q: %v", startDate, err)
	}
	daysAgo := time.Since(parsed).Hours() / 24
	if math.Abs(daysAgo-30) > 1 {
		t.Errorf("startDate %q is %.0f days ago, want ~30", startDate, daysAgo)
	}

	// endDate should not be set when end is zero.
	if q.Get("endDate") != "" {
		t.Errorf("endDate = %q, want empty for zero end time", q.Get("endDate"))
	}
}

func TestDONKI_RateLimitError(t *testing.T) {
	c := setupDONKIErrorServer(t, http.StatusTooManyRequests,
		`{"error":{"message":"rate limit exceeded"}}`)

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	_, err := c.DONKI.CME(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Errorf("error type = %T, want *RateLimitError", err)
	}
}

func TestDONKI_AuthError(t *testing.T) {
	c := setupDONKIErrorServer(t, http.StatusForbidden,
		`{"error":{"message":"API key invalid"}}`)

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	_, err := c.DONKI.CME(context.Background(), start, end)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}

	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Errorf("error type = %T, want *AuthError", err)
	}
}

func TestDONKIParamsInjectableTime(t *testing.T) {
	fixed := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	t.Run("zero start uses now minus 30 days", func(t *testing.T) {
		params := donkiParams(time.Time{}, time.Time{}, fixed)
		got := params.Get("startDate")
		want := "2025-05-16"
		if got != want {
			t.Errorf("startDate = %q, want %q", got, want)
		}
		if params.Get("endDate") != "" {
			t.Errorf("endDate = %q, want empty for zero end", params.Get("endDate"))
		}
	})

	t.Run("explicit start and end", func(t *testing.T) {
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		params := donkiParams(start, end, fixed)
		if params.Get("startDate") != "2025-01-01" {
			t.Errorf("startDate = %q, want %q", params.Get("startDate"), "2025-01-01")
		}
		if params.Get("endDate") != "2025-01-31" {
			t.Errorf("endDate = %q, want %q", params.Get("endDate"), "2025-01-31")
		}
	})
}
