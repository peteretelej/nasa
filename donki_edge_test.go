package nasa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestDONKI_NullSourceLocation verifies that a CME event with
// "sourceLocation": null does not panic and results in an empty string.
func TestDONKI_NullSourceLocation(t *testing.T) {
	c, _ := setupDONKITestServer(t, "testdata/donki_cme_null_coords.json")

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	events, err := c.DONKI.CME(context.Background(), start, end)
	if err != nil {
		t.Fatalf("CME() error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	e := events[0]
	if e.ActivityID != "2024-01-15T06:12:00-CME-001" {
		t.Errorf("ActivityID = %q, want %q", e.ActivityID, "2024-01-15T06:12:00-CME-001")
	}

	// sourceLocation is null in the fixture; Go unmarshals null to the zero
	// value for string, which is "".
	if e.SourceLocation != "" {
		t.Errorf("SourceLocation = %q, want empty string for null", e.SourceLocation)
	}

	// Verify the rest of the event parsed correctly.
	if e.StartTime.IsZero() {
		t.Error("expected non-zero StartTime")
	}
	if e.ActiveRegionNum != 13536 {
		t.Errorf("ActiveRegionNum = %d, want 13536", e.ActiveRegionNum)
	}
	if len(e.Instruments) != 1 {
		t.Errorf("Instruments count = %d, want 1", len(e.Instruments))
	}
}

// TestDONKI_CrossReferenceActivityIDs verifies that LinkedEvents cross-
// references are correctly parsed across CME and FLR endpoints. The CME
// fixture links to FLR-001, and the FLR fixture links back to CME-001.
func TestDONKI_CrossReferenceActivityIDs(t *testing.T) {
	// Load both CME and FLR fixtures to serve from the same mock server.
	cmeData, err := os.ReadFile("testdata/donki_cme.json")
	if err != nil {
		t.Fatalf("reading CME fixture: %v", err)
	}
	flrData, err := os.ReadFile("testdata/donki_flr.json")
	if err != nil {
		t.Fatalf("reading FLR fixture: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/DONKI/FLR"):
			if _, err := w.Write(flrData); err != nil {
				t.Fatalf("writing FLR fixture: %v", err)
			}
		case strings.HasPrefix(r.URL.Path, "/DONKI/CME"):
			if _, err := w.Write(cmeData); err != nil {
				t.Fatalf("writing CME fixture: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

	// Fetch CME events.
	cmeEvents, err := c.DONKI.CME(context.Background(), start, end)
	if err != nil {
		t.Fatalf("CME() error: %v", err)
	}
	if len(cmeEvents) < 1 {
		t.Fatal("expected at least 1 CME event")
	}

	// First CME should link to the FLR event.
	cme := cmeEvents[0]
	if len(cme.LinkedEvents) != 1 {
		t.Fatalf("CME LinkedEvents count = %d, want 1", len(cme.LinkedEvents))
	}
	cmeLinkedID := cme.LinkedEvents[0].ActivityID
	if cmeLinkedID != "2024-01-15T05:00:00-FLR-001" {
		t.Errorf("CME linked event ID = %q, want %q", cmeLinkedID, "2024-01-15T05:00:00-FLR-001")
	}

	// Fetch FLR events.
	flrEvents, err := c.DONKI.FLR(context.Background(), start, end)
	if err != nil {
		t.Fatalf("FLR() error: %v", err)
	}
	if len(flrEvents) < 1 {
		t.Fatal("expected at least 1 FLR event")
	}

	// The FLR should link back to the CME event.
	flr := flrEvents[0]
	if len(flr.LinkedEvents) != 1 {
		t.Fatalf("FLR LinkedEvents count = %d, want 1", len(flr.LinkedEvents))
	}
	flrLinkedID := flr.LinkedEvents[0].ActivityID
	if flrLinkedID != "2024-01-15T06:12:00-CME-001" {
		t.Errorf("FLR linked event ID = %q, want %q", flrLinkedID, "2024-01-15T06:12:00-CME-001")
	}

	// Verify bidirectional cross-reference: CME->FLR ID matches FLR's own ID,
	// and FLR->CME ID matches CME's own ID.
	if cmeLinkedID != flr.FlrID {
		t.Errorf("CME links to %q but FLR's own ID is %q", cmeLinkedID, flr.FlrID)
	}
	if flrLinkedID != cme.ActivityID {
		t.Errorf("FLR links to %q but CME's own ID is %q", flrLinkedID, cme.ActivityID)
	}
}
