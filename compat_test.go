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

// TestAPODGet_Wrapper tests the APODGet convenience wrapper by testing
// the underlying APOD.Get code path via a client with WithBaseURL.
func TestAPODGet_Wrapper(t *testing.T) {
	data, err := os.ReadFile("testdata/apod_single.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	img, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("APOD.Get error: %v", err)
	}
	if img.Title != "Test APOD Title" {
		t.Errorf("Title = %q, want %q", img.Title, "Test APOD Title")
	}
	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !img.Date.Equal(expectedDate) {
		t.Errorf("Date = %v, want %v", img.Date, expectedDate)
	}
}

// TestAPODTodayWithKey_Wrapper tests the APODTodayWithKey convenience wrapper
// by testing the underlying APOD.Today code path.
func TestAPODTodayWithKey_Wrapper(t *testing.T) {
	data, err := os.ReadFile("testdata/apod_single.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	img, err := c.APOD.Today(context.Background())
	if err != nil {
		t.Fatalf("APOD.Today error: %v", err)
	}
	if img.Title != "Test APOD Title" {
		t.Errorf("Title = %q, want %q", img.Title, "Test APOD Title")
	}
}

// TestNEOFeedWithKey_Wrapper tests the NEOFeedWithKey convenience wrapper
// by testing the underlying NEO.Feed code path.
func TestNEOFeedWithKey_Wrapper(t *testing.T) {
	data, err := os.ReadFile("testdata/neo_feed.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/neo/rest/v1/feed") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	feed, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		t.Fatalf("NEO.Feed error: %v", err)
	}
	if feed.ElementCount != 1 {
		t.Errorf("ElementCount = %d, want 1", feed.ElementCount)
	}
}

// TestNeoToAsteroid_OrbitalData tests the neoToAsteroid conversion with orbital data.
func TestNeoToAsteroid_OrbitalData(t *testing.T) {
	neo := &NearEarthObject{
		NeoReferenceID:         "12345",
		Name:                   "Test Asteroid",
		JPLURL:                 "https://ssd.jpl.nasa.gov/tools/sbdb_lookup.html#/?sstr=12345",
		AbsoluteMagnitude:      22.5,
		IsPotentiallyHazardous: true,
		OrbitalData: &OrbitalData{
			OrbitID:                   "42",
			OrbitDeterminationDate:    "2024-01-15",
			OrbitUncertainty:          "0",
			MinimumOrbitIntersection:  "0.02",
			JupiterTisserandInvariant: "5.1",
			EpochOsculation:           "2460400.5",
			Eccentricity:              "0.3",
			Inclination:               "12.5",
			SemiMajorAxis:             "1.2",
			AscendingNodeLongitude:    "45.0",
			OrbitalPeriod:             "400",
			PerihelionDistance:        "0.84",
			PerihelionArgument:        "120.5",
			AphelionDistance:          "1.56",
			PerihelionTime:            "2460300.5",
			MeanAnomaly:               "90.0",
			MeanMotion:                "0.9",
			Equinox:                   "J2000",
		},
	}

	a := neoToAsteroid(neo)

	if a.ID != "12345" {
		t.Errorf("ID = %q, want %q", a.ID, "12345")
	}
	if a.Name != "Test Asteroid" {
		t.Errorf("Name = %q, want %q", a.Name, "Test Asteroid")
	}
	if !a.PotentiallyHazardous {
		t.Error("PotentiallyHazardous = false, want true")
	}
	// Verify orbital data maps correctly, including typo preservation
	if a.OrbitalData.OrbitID != "42" {
		t.Errorf("OrbitID = %q, want %q", a.OrbitalData.OrbitID, "42")
	}
	if a.OrbitalData.OrbitUncertainity != "0" {
		t.Errorf("OrbitUncertainity = %q, want %q", a.OrbitalData.OrbitUncertainity, "0")
	}
	if a.OrbitalData.Eccentricity != "0.3" {
		t.Errorf("Eccentricity = %q, want %q", a.OrbitalData.Eccentricity, "0.3")
	}
	if a.OrbitalData.Equinox != "J2000" {
		t.Errorf("Equinox = %q, want %q", a.OrbitalData.Equinox, "J2000")
	}
}

// TestNeoToAsteroid_NilOrbitalData verifies no panic when OrbitalData is nil.
func TestNeoToAsteroid_NilOrbitalData(t *testing.T) {
	neo := &NearEarthObject{
		NeoReferenceID: "99999",
		Name:           "No Orbital",
	}
	a := neoToAsteroid(neo)
	if a.ID != "99999" {
		t.Errorf("ID = %q, want %q", a.ID, "99999")
	}
	// OrbitalData should be zero-valued, not panic
	if a.OrbitalData.OrbitID != "" {
		t.Errorf("OrbitID = %q, want empty", a.OrbitalData.OrbitID)
	}
}

// TestFormatSF verifies StringFloat64 to string conversion.
func TestFormatSF(t *testing.T) {
	tests := []struct {
		name  string
		input StringFloat64
		want  string
	}{
		{"zero", 0, "0"},
		{"integer", 42, "42"},
		{"decimal", 15.23, "15.23"},
		{"large", 1234567.89, "1234567.89"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSF(tt.input)
			if got != tt.want {
				t.Errorf("formatSF(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestApodImageToImage verifies the APODImage to Image conversion.
func TestApodImageToImage(t *testing.T) {
	apod := &APODImage{
		Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Title:       "Test Title",
		Explanation: "Test explanation",
		URL:         "https://example.com/image.jpg",
		HDURL:       "https://example.com/image_hd.jpg",
		MediaType:   MediaImage,
		Copyright:   "Photographer",
	}

	img := apodImageToImage(apod)

	if img.Date != "2024-01-15" {
		t.Errorf("Date = %q, want %q", img.Date, "2024-01-15")
	}
	if img.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", img.Title, "Test Title")
	}
	if img.URL != "https://example.com/image.jpg" {
		t.Errorf("URL = %q, want full URL", img.URL)
	}
	if img.HDURL != "https://example.com/image_hd.jpg" {
		t.Errorf("HDURL = %q, want full HD URL", img.HDURL)
	}
	if img.Explanation != "Test explanation" {
		t.Errorf("Explanation = %q, want %q", img.Explanation, "Test explanation")
	}
	if !img.ApodDate.Equal(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("ApodDate = %v, want 2024-01-15", img.ApodDate)
	}
}

// Compile-time type assertions for legacy function signatures.

func TestExistingAPODToday_Compiles(t *testing.T) {
	var f func() (*Image, error) = APODToday
	_ = f
}

func TestExistingApodImage_Compiles(t *testing.T) {
	var f func(time.Time) (*Image, error) = ApodImage
	_ = f
}

func TestExistingRandomAPOD_Compiles(t *testing.T) {
	var f func() (*Image, error) = RandomAPOD
	_ = f
}

func TestExistingNeoFeed_Compiles(t *testing.T) {
	var f func(time.Time, time.Time) (*NeoList, error) = NeoFeed
	_ = f
}

// Compile-time type assertions for new wrapper function signatures.

func TestAPODGet_SignatureCompiles(t *testing.T) {
	var f func(string, string) (*APODImage, error) = APODGet
	_ = f
}

func TestAPODTodayWithKey_SignatureCompiles(t *testing.T) {
	var f func(string) (*APODImage, error) = APODTodayWithKey
	_ = f
}

func TestNEOFeedWithKey_SignatureCompiles(t *testing.T) {
	var f func(string, string, string) (*NEOFeed, error) = NEOFeedWithKey
	_ = f
}
