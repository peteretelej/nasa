package nasa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

// setupSSDTestServer creates an httptest server that serves the given fixture
// file and returns an SSDService. The handler captures the last request for
// parameter verification.
func setupSSDTestServer(t *testing.T, fixturePath string) (*SSDService, **http.Request) {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixturePath, err)
	}
	var captured *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("TEST_KEY"))
	svc := &SSDService{client: c, baseURL: ts.URL}
	return svc, &captured
}

// --- zipColumns tests ---

func TestZipColumns(t *testing.T) {
	t.Run("basic field mapping", func(t *testing.T) {
		fields := []string{"des", "cd", "dist", "v_rel", "h"}
		data := [][]string{
			{"Apophis", "2029-Apr-13", "0.000254", "7.43", "19.7"},
		}

		result, err := zipColumns[CloseApproachData](fields, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		r := result[0]
		if r.Designation != "Apophis" {
			t.Errorf("Designation: got %q, want %q", r.Designation, "Apophis")
		}
		if r.CloseDate != "2029-Apr-13" {
			t.Errorf("CloseDate: got %q, want %q", r.CloseDate, "2029-Apr-13")
		}
		if r.Distance != 0.000254 {
			t.Errorf("Distance: got %v, want %v", r.Distance, 0.000254)
		}
		if r.VRelative != 7.43 {
			t.Errorf("VRelative: got %v, want %v", r.VRelative, 7.43)
		}
		if r.AbsMagnitude != 19.7 {
			t.Errorf("AbsMagnitude: got %v, want %v", r.AbsMagnitude, 19.7)
		}
	})

	t.Run("empty values produce zero values", func(t *testing.T) {
		fields := []string{"des", "dist", "h"}
		data := [][]string{
			{"test", "", ""},
		}

		result, err := zipColumns[CloseApproachData](fields, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result[0]
		if r.Designation != "test" {
			t.Errorf("Designation: got %q, want %q", r.Designation, "test")
		}
		if r.Distance != 0 {
			t.Errorf("Distance: expected zero, got %v", r.Distance)
		}
		if r.AbsMagnitude != 0 {
			t.Errorf("AbsMagnitude: expected zero, got %v", r.AbsMagnitude)
		}
	})

	t.Run("missing columns leave zero value", func(t *testing.T) {
		// Only provide "des"; other ssd-tagged fields should be zero.
		fields := []string{"des"}
		data := [][]string{
			{"test-only-des"},
		}

		result, err := zipColumns[CloseApproachData](fields, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result[0]
		if r.Designation != "test-only-des" {
			t.Errorf("Designation: got %q, want %q", r.Designation, "test-only-des")
		}
		if r.Distance != 0 || r.VRelative != 0 {
			t.Error("expected zero values for unmapped fields")
		}
	})

	t.Run("extra columns silently skipped", func(t *testing.T) {
		fields := []string{"des", "unknown_col", "dist"}
		data := [][]string{
			{"test", "extra-data", "1.5"},
		}

		result, err := zipColumns[CloseApproachData](fields, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result[0]
		if r.Designation != "test" {
			t.Errorf("Designation: got %q, want %q", r.Designation, "test")
		}
		if r.Distance != 1.5 {
			t.Errorf("Distance: got %v, want %v", r.Distance, 1.5)
		}
	})

	t.Run("short rows guard", func(t *testing.T) {
		fields := []string{"des", "dist", "v_rel"}
		data := [][]string{
			{"test"}, // row shorter than fields
		}

		result, err := zipColumns[CloseApproachData](fields, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result[0]
		if r.Designation != "test" {
			t.Errorf("Designation: got %q, want %q", r.Designation, "test")
		}
		if r.Distance != 0 {
			t.Errorf("Distance: expected zero for short row, got %v", r.Distance)
		}
	})

	t.Run("fireball with hyphen-case tags", func(t *testing.T) {
		fields := []string{"date", "energy", "impact-e", "lat", "lat-dir"}
		data := [][]string{
			{"2025-03-15 08:23:45", "1.2", "0.046", "36.5", "N"},
		}

		result, err := zipColumns[FireballEvent](fields, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := result[0]
		if r.Date != "2025-03-15 08:23:45" {
			t.Errorf("Date: got %q, want %q", r.Date, "2025-03-15 08:23:45")
		}
		if r.Energy != 1.2 {
			t.Errorf("Energy: got %v, want %v", r.Energy, 1.2)
		}
		if r.ImpactE != 0.046 {
			t.Errorf("ImpactE: got %v, want %v", r.ImpactE, 0.046)
		}
		if r.LatDir != "N" {
			t.Errorf("LatDir: got %q, want %q", r.LatDir, "N")
		}
	})
}

// --- parseCount tests ---

func TestParseCount(t *testing.T) {
	t.Run("number count", func(t *testing.T) {
		raw := json.RawMessage(`42`)
		n, err := parseCount(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 42 {
			t.Errorf("got %d, want 42", n)
		}
	})

	t.Run("string count", func(t *testing.T) {
		raw := json.RawMessage(`"25"`)
		n, err := parseCount(raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 25 {
			t.Errorf("got %d, want 25", n)
		}
	})

	t.Run("empty raw", func(t *testing.T) {
		n, err := parseCount(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 0 {
			t.Errorf("got %d, want 0", n)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		raw := json.RawMessage(`"abc"`)
		_, err := parseCount(raw)
		if err == nil {
			t.Error("expected error for malformed count")
		}
	})
}

// --- CloseApproach tests ---

func TestSSDCloseApproach(t *testing.T) {
	svc, req := setupSSDTestServer(t, "testdata/ssd_cad.json")

	results, err := svc.CloseApproach(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify first record.
	r := results[0]
	if r.Designation != "(2024 AA)" {
		t.Errorf("Designation: got %q, want %q", r.Designation, "(2024 AA)")
	}
	if r.Distance != 0.0412 {
		t.Errorf("Distance: got %v, want %v", r.Distance, 0.0412)
	}
	if r.VRelative != 12.34 {
		t.Errorf("VRelative: got %v, want %v", r.VRelative, 12.34)
	}

	// Verify Apophis record (last).
	r = results[2]
	if r.Designation != "99942 Apophis" {
		t.Errorf("Designation: got %q, want %q", r.Designation, "99942 Apophis")
	}
	if r.Distance != 0.000254 {
		t.Errorf("Distance: got %v, want %v", r.Distance, 0.000254)
	}
	if r.AbsMagnitude != 19.7 {
		t.Errorf("AbsMagnitude: got %v, want %v", r.AbsMagnitude, 19.7)
	}

	// Verify no api_key in request.
	if (*req).URL.Query().Get("api_key") != "" {
		t.Error("SSD requests must not include api_key")
	}
}

func TestSSDCloseApproachEmpty(t *testing.T) {
	svc, _ := setupSSDTestServer(t, "testdata/ssd_cad_empty.json")

	results, err := svc.CloseApproach(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSSDCloseApproachOptions(t *testing.T) {
	svc, req := setupSSDTestServer(t, "testdata/ssd_cad.json")

	_, err := svc.CloseApproach(context.Background(),
		CADWithDateMin(mustParseDate("2025-01-01")),
		CADWithDateMax(mustParseDate("2025-12-31")),
		CADWithDistMax(0.05),
		CADWithBody("Earth"),
		CADWithLimit(10),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := (*req).URL.Query()
	assertParam(t, q, "date-min", "2025-01-01")
	assertParam(t, q, "date-max", "2025-12-31")
	assertParam(t, q, "dist-max", "0.05")
	assertParam(t, q, "body", "Earth")
	assertParam(t, q, "limit", "10")
}

// --- Fireball tests ---

func TestSSDFireball(t *testing.T) {
	svc, req := setupSSDTestServer(t, "testdata/ssd_fireball.json")

	results, err := svc.Fireball(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Date != "2025-03-15 08:23:45" {
		t.Errorf("Date: got %q, want %q", r.Date, "2025-03-15 08:23:45")
	}
	if r.Energy != 1.2 {
		t.Errorf("Energy: got %v, want %v", r.Energy, 1.2)
	}
	if r.Lat != 36.5 {
		t.Errorf("Lat: got %v, want %v", r.Lat, 36.5)
	}
	if r.LatDir != "N" {
		t.Errorf("LatDir: got %q, want %q", r.LatDir, "N")
	}
	if r.Altitude != 25.0 {
		t.Errorf("Altitude: got %v, want %v", r.Altitude, 25.0)
	}
	if r.Velocity != 18.5 {
		t.Errorf("Velocity: got %v, want %v", r.Velocity, 18.5)
	}

	// Verify req-loc=true always sent.
	q := (*req).URL.Query()
	if q.Get("req-loc") != "true" {
		t.Error("Fireball must always send req-loc=true")
	}

	// Verify no api_key.
	if q.Get("api_key") != "" {
		t.Error("SSD requests must not include api_key")
	}
}

func TestSSDFireballOptions(t *testing.T) {
	svc, req := setupSSDTestServer(t, "testdata/ssd_fireball.json")

	_, err := svc.Fireball(context.Background(),
		FireballWithDateMin(mustParseDate("2025-01-01")),
		FireballWithDateMax(mustParseDate("2025-06-30")),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := (*req).URL.Query()
	assertParam(t, q, "date-min", "2025-01-01")
	assertParam(t, q, "date-max", "2025-06-30")
	assertParam(t, q, "req-loc", "true")
}

// --- NHATS tests ---

func TestSSDNHATS(t *testing.T) {
	svc, req := setupSSDTestServer(t, "testdata/ssd_nhats.json")

	results, err := svc.NHATS(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First object: all fields populated.
	r := results[0]
	if r.Designation != "2000 SG344" {
		t.Errorf("Designation: got %q, want %q", r.Designation, "2000 SG344")
	}
	if r.H != "24.7" {
		t.Errorf("H: got %q, want %q (should be string)", r.H, "24.7")
	}
	if r.OCC != "0" {
		t.Errorf("OCC: got %q, want %q (should be string)", r.OCC, "0")
	}

	// Nested NHATSDeltaV for min_dv.
	if r.MinDV.DV != "3.712" {
		t.Errorf("MinDV.DV: got %q, want %q", r.MinDV.DV, "3.712")
	}
	if r.MinDV.Dur != 350 {
		t.Errorf("MinDV.Dur: got %d, want %d", r.MinDV.Dur, 350)
	}

	// Nested NHATSDeltaV for min_dur.
	if r.MinDuration.DV != "5.100" {
		t.Errorf("MinDuration.DV: got %q, want %q", r.MinDuration.DV, "5.100")
	}
	if r.MinDuration.Dur != 120 {
		t.Errorf("MinDuration.Dur: got %d, want %d", r.MinDuration.Dur, 120)
	}

	if r.NViaTraj != 42 {
		t.Errorf("NViaTraj: got %d, want %d", r.NViaTraj, 42)
	}

	// String pointer fields.
	if r.ObsStart == nil || *r.ObsStart != "2000-09-29" {
		t.Errorf("ObsStart: got %v, want %q", r.ObsStart, "2000-09-29")
	}
	if r.ObsMag == nil || *r.ObsMag != "25.1" {
		t.Errorf("ObsMag: got %v, want %q (should be *string)", r.ObsMag, "25.1")
	}
	if r.MinSize == nil || *r.MinSize != "0.024" {
		t.Errorf("MinSize: got %v, want %q (should be *string)", r.MinSize, "0.024")
	}

	// Second object: nullable fields are null.
	r2 := results[1]
	if r2.ObsStart != nil {
		t.Errorf("ObsStart: expected nil, got %v", r2.ObsStart)
	}
	if r2.ObsMag != nil {
		t.Errorf("ObsMag: expected nil, got %v", r2.ObsMag)
	}
	if r2.MinSize != nil {
		t.Errorf("MinSize: expected nil, got %v", r2.MinSize)
	}
	if r2.OCC != "7" {
		t.Errorf("OCC: got %q, want %q", r2.OCC, "7")
	}

	// Verify no api_key.
	if (*req).URL.Query().Get("api_key") != "" {
		t.Error("SSD requests must not include api_key")
	}
}

// --- Scout tests ---

func TestSSDScout(t *testing.T) {
	svc, req := setupSSDTestServer(t, "testdata/ssd_scout.json")

	results, err := svc.Scout(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First object: non-null fields.
	r := results[0]
	if r.ObjectName != "P11yNcS" {
		t.Errorf("ObjectName: got %q, want %q", r.ObjectName, "P11yNcS")
	}
	if r.Rating == nil || *r.Rating != 0 {
		t.Errorf("Rating: expected 0, got %v", r.Rating)
	}
	if r.H != 25.3 {
		t.Errorf("H: got %v, want %v", r.H, 25.3)
	}
	if r.Vmag != 20.1 {
		t.Errorf("Vmag: got %v, want %v", r.Vmag, 20.1)
	}
	if r.NeoScore != 100 {
		t.Errorf("NeoScore: got %d, want %d", r.NeoScore, 100)
	}
	if r.CaDist == nil || *r.CaDist != "0.0045" {
		t.Errorf("CaDist: got %v, want %q", r.CaDist, "0.0045")
	}
	// VInf with ,string tag.
	if r.VInf == nil || *r.VInf != 20.7 {
		t.Errorf("VInf: got %v, want 20.7 (must use ,string tag)", r.VInf)
	}

	// Second object: nullable fields are null.
	r2 := results[1]
	if r2.Rating != nil {
		t.Errorf("Rating: expected nil, got %v", *r2.Rating)
	}
	if r2.CaDist != nil {
		t.Errorf("CaDist: expected nil, got %v", *r2.CaDist)
	}
	if r2.VInf != nil {
		t.Errorf("VInf: expected nil, got %v", *r2.VInf)
	}
	if r2.NObs != 5 {
		t.Errorf("NObs: got %d, want %d", r2.NObs, 5)
	}

	// Verify no api_key.
	if (*req).URL.Query().Get("api_key") != "" {
		t.Error("SSD requests must not include api_key")
	}
}

// --- Sentry tests ---

func TestSSDSentry(t *testing.T) {
	svc, _ := setupSSDTestServer(t, "testdata/ssd_sentry.json")

	results, err := svc.Sentry(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	r := results[0]
	if r.Designation != "99942" {
		t.Errorf("Designation: got %q, want %q", r.Designation, "99942")
	}
	if r.Fullname != "99942 Apophis (2004 MN4)" {
		t.Errorf("Fullname: got %q, want %q", r.Fullname, "99942 Apophis (2004 MN4)")
	}
	if r.NImpacts != 2 {
		t.Errorf("NImpacts: got %d, want %d", r.NImpacts, 2)
	}
	if r.IP != "4.4e-10" {
		t.Errorf("IP: got %q, want %q", r.IP, "4.4e-10")
	}
	if r.Diameter != "0.370" {
		t.Errorf("Diameter: got %q, want %q", r.Diameter, "0.370")
	}

	r2 := results[1]
	if r2.Designation != "2024 YR4" {
		t.Errorf("Designation: got %q, want %q", r2.Designation, "2024 YR4")
	}
	if r2.NImpacts != 1 {
		t.Errorf("NImpacts: got %d, want %d", r2.NImpacts, 1)
	}
}

// --- SBDB tests ---

func TestSSDSBDB(t *testing.T) {
	svc, req := setupSSDTestServer(t, "testdata/ssd_sbdb.json")

	sb, err := svc.SBDB(context.Background(), "99942")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Object identification.
	if sb.Object.Fullname != "99942 Apophis (2004 MN4)" {
		t.Errorf("Fullname: got %q", sb.Object.Fullname)
	}
	if sb.Object.Des != "99942" {
		t.Errorf("Des: got %q, want %q", sb.Object.Des, "99942")
	}
	if sb.Object.Kind != "an" {
		t.Errorf("Kind: got %q, want %q", sb.Object.Kind, "an")
	}
	if !sb.Object.NEO {
		t.Error("NEO: expected true")
	}
	if !sb.Object.PHA {
		t.Error("PHA: expected true")
	}
	if sb.Object.Prefix != nil {
		t.Errorf("Prefix: expected nil, got %v", sb.Object.Prefix)
	}

	// OrbitClass as nested object.
	if sb.Object.OrbitClass.Code != "ATE" {
		t.Errorf("OrbitClass.Code: got %q, want %q", sb.Object.OrbitClass.Code, "ATE")
	}
	if sb.Object.OrbitClass.Name != "Aten" {
		t.Errorf("OrbitClass.Name: got %q, want %q", sb.Object.OrbitClass.Name, "Aten")
	}

	// Orbital elements as array.
	if len(sb.Orbit.Elements) != 6 {
		t.Fatalf("expected 6 orbital elements, got %d", len(sb.Orbit.Elements))
	}
	e := sb.Orbit.Elements[0]
	if e.Name != "e" {
		t.Errorf("Element[0].Name: got %q, want %q", e.Name, "e")
	}
	if e.Title != "eccentricity" {
		t.Errorf("Element[0].Title: got %q, want %q", e.Title, "eccentricity")
	}
	if e.Value != "0.1914" {
		t.Errorf("Element[0].Value: got %q, want %q", e.Value, "0.1914")
	}
	if e.Units != nil {
		t.Errorf("Element[0].Units: expected nil (dimensionless), got %v", *e.Units)
	}

	// Element with units.
	a := sb.Orbit.Elements[1]
	if a.Units == nil || *a.Units != "au" {
		t.Errorf("Element[1].Units: expected %q, got %v", "au", a.Units)
	}

	// Orbit metadata.
	if sb.Orbit.Source != "JPL" {
		t.Errorf("Orbit.Source: got %q, want %q", sb.Orbit.Source, "JPL")
	}
	if sb.Orbit.NObs != 5134 {
		t.Errorf("Orbit.NObs: got %d, want %d", sb.Orbit.NObs, 5134)
	}
	if sb.Orbit.ConditionCode != "0" {
		t.Errorf("Orbit.ConditionCode: got %q, want %q", sb.Orbit.ConditionCode, "0")
	}

	// Element helper.
	if sb.Element("e") != "0.1914" {
		t.Errorf("Element('e'): got %q, want %q", sb.Element("e"), "0.1914")
	}
	if sb.Element("a") != "0.9224" {
		t.Errorf("Element('a'): got %q, want %q", sb.Element("a"), "0.9224")
	}
	if sb.Element("nonexistent") != "" {
		t.Errorf("Element('nonexistent'): expected empty, got %q", sb.Element("nonexistent"))
	}

	// Verify designation parameter.
	q := (*req).URL.Query()
	if q.Get("des") != "99942" {
		t.Errorf("des param: got %q, want %q", q.Get("des"), "99942")
	}

	// Verify no api_key.
	if q.Get("api_key") != "" {
		t.Error("SSD requests must not include api_key")
	}
}

// --- No auth verification ---

func TestSSDNoAuth(t *testing.T) {
	// All SSD endpoints use getExternal which does not inject api_key.
	// This test verifies across multiple endpoints.
	svc, req := setupSSDTestServer(t, "testdata/ssd_cad.json")
	_, _ = svc.CloseApproach(context.Background())
	if (*req).URL.Query().Get("api_key") != "" {
		t.Error("CloseApproach: api_key must not be sent")
	}

	svc2, req2 := setupSSDTestServer(t, "testdata/ssd_fireball.json")
	_, _ = svc2.Fireball(context.Background())
	if (*req2).URL.Query().Get("api_key") != "" {
		t.Error("Fireball: api_key must not be sent")
	}

	svc3, req3 := setupSSDTestServer(t, "testdata/ssd_sbdb.json")
	_, _ = svc3.SBDB(context.Background(), "99942")
	if (*req3).URL.Query().Get("api_key") != "" {
		t.Error("SBDB: api_key must not be sent")
	}
}

// --- Request path verification ---

func TestSSDRequestPaths(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		call    func(*SSDService) error
		want    string
	}{
		{
			name:    "CloseApproach",
			fixture: "testdata/ssd_cad.json",
			call:    func(s *SSDService) error { _, err := s.CloseApproach(context.Background()); return err },
			want:    "/cad.api",
		},
		{
			name:    "Fireball",
			fixture: "testdata/ssd_fireball.json",
			call:    func(s *SSDService) error { _, err := s.Fireball(context.Background()); return err },
			want:    "/fireball.api",
		},
		{
			name:    "NHATS",
			fixture: "testdata/ssd_nhats.json",
			call:    func(s *SSDService) error { _, err := s.NHATS(context.Background()); return err },
			want:    "/nhats.api",
		},
		{
			name:    "Scout",
			fixture: "testdata/ssd_scout.json",
			call:    func(s *SSDService) error { _, err := s.Scout(context.Background()); return err },
			want:    "/scout.api",
		},
		{
			name:    "Sentry",
			fixture: "testdata/ssd_sentry.json",
			call:    func(s *SSDService) error { _, err := s.Sentry(context.Background()); return err },
			want:    "/sentry.api",
		},
		{
			name:    "SBDB",
			fixture: "testdata/ssd_sbdb.json",
			call:    func(s *SSDService) error { _, err := s.SBDB(context.Background(), "99942"); return err },
			want:    "/sbdb.api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, req := setupSSDTestServer(t, tt.fixture)
			if err := tt.call(svc); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := (*req).URL.Path
			if got != tt.want {
				t.Errorf("path: got %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Helpers ---

func mustParseDate(s string) (t time.Time) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func assertParam(t *testing.T, q url.Values, key, want string) {
	t.Helper()
	got := q.Get(key)
	if got != want {
		t.Errorf("param %q: got %q, want %q", key, got, want)
	}
}
