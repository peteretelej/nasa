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
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing APOD fixture: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	img, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("APOD.Get error: %v", err)
	}
	if img.Title != "Star Cluster IC 348 from Webb" {
		t.Errorf("Title = %q, want %q", img.Title, "Star Cluster IC 348 from Webb")
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
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing APOD today fixture: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	img, err := c.APOD.Today(context.Background())
	if err != nil {
		t.Fatalf("APOD.Today error: %v", err)
	}
	if img.Title != "Star Cluster IC 348 from Webb" {
		t.Errorf("Title = %q, want %q", img.Title, "Star Cluster IC 348 from Webb")
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
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing NEO fixture: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithBaseURL(ts.URL), WithAPIKey("TEST_KEY"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	feed, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		t.Fatalf("NEO.Feed error: %v", err)
	}
	if feed.ElementCount != 19 {
		t.Errorf("ElementCount = %d, want 19", feed.ElementCount)
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

func assertType[T any](T) {}

func TestExistingAPODToday_Compiles(t *testing.T) {
	assertType[func() (*Image, error)](APODToday)
}

func TestExistingApodImage_Compiles(t *testing.T) {
	assertType[func(time.Time) (*Image, error)](ApodImage)
}

func TestExistingRandomAPOD_Compiles(t *testing.T) {
	assertType[func() (*Image, error)](RandomAPOD)
}

func TestExistingNeoFeed_Compiles(t *testing.T) {
	assertType[func(time.Time, time.Time) (*NeoList, error)](NeoFeed)
}

// Compile-time type assertions for new wrapper function signatures.

func TestAPODGet_SignatureCompiles(t *testing.T) {
	assertType[func(string, string) (*APODImage, error)](APODGet)
}

func TestAPODTodayWithKey_SignatureCompiles(t *testing.T) {
	assertType[func(string) (*APODImage, error)](APODTodayWithKey)
}

func TestNEOFeedWithKey_SignatureCompiles(t *testing.T) {
	assertType[func(string, string, string) (*NEOFeed, error)](NEOFeedWithKey)
}

// Compile-time struct field type assertions for legacy and new types.

// TestCompat_ImageStructFields verifies the legacy Image struct preserves
// its field types. All user-facing string fields must remain string; ApodDate
// must remain time.Time.
func TestCompat_ImageStructFields(t *testing.T) {
	assertType[string](Image{}.Date)
	assertType[string](Image{}.Title)
	assertType[string](Image{}.URL)
	assertType[string](Image{}.HDURL)
	assertType[string](Image{}.Explanation)
	assertType[time.Time](Image{}.ApodDate)
}

// TestCompat_APODImageStructFields verifies the new APODImage struct field types.
func TestCompat_APODImageStructFields(t *testing.T) {
	assertType[string](APODImage{}.Copyright)
	assertType[MediaType](APODImage{}.MediaType)
	assertType[string](APODImage{}.ServiceVersion)
	assertType[string](APODImage{}.ThumbnailURL)
}

// TestCompat_AsteroidStructFields verifies the legacy Asteroid struct preserves
// its field types, including all orbital data fields which must remain string.
func TestCompat_AsteroidStructFields(t *testing.T) {
	assertType[string](Asteroid{}.ID)
	assertType[string](Asteroid{}.Name)
	assertType[string](Asteroid{}.JPLURL)
	assertType[float64](Asteroid{}.AbsoluteMagnitude)
	assertType[bool](Asteroid{}.PotentiallyHazardous)

	// Orbital data fields must all remain string
	var a Asteroid
	assertType[string](a.OrbitalData.OrbitID)
	assertType[string](a.OrbitalData.OrbitDeterminationDate)
	assertType[string](a.OrbitalData.OrbitUncertainity)
	assertType[string](a.OrbitalData.MinimumOrbitIntersection)
	assertType[string](a.OrbitalData.JupiterTisserandInvariant)
	assertType[string](a.OrbitalData.EpochOsculation)
	assertType[string](a.OrbitalData.Eccentricity)
	assertType[string](a.OrbitalData.Inclination)
	assertType[string](a.OrbitalData.SemiMajorAxis)
	assertType[string](a.OrbitalData.AscendingNodeLongitude)
	assertType[string](a.OrbitalData.OrbitalPeriod)
	assertType[string](a.OrbitalData.PerihelionDistance)
	assertType[string](a.OrbitalData.PerihelionArgument)
	assertType[string](a.OrbitalData.AphelionDistance)
	assertType[string](a.OrbitalData.PerihelionTime)
	assertType[string](a.OrbitalData.MeanAnomaly)
	assertType[string](a.OrbitalData.MeanMotion)
	assertType[string](a.OrbitalData.Equinox)
}

// TestCompat_NeoListStructFields verifies the legacy NeoList struct preserves
// its field types.
func TestCompat_NeoListStructFields(t *testing.T) {
	assertType[int64](NeoList{}.ElementCount)
	assertType[map[string][]Asteroid](NeoList{}.NearEarthObjects)
}

// TestCompat_APODConversion_ResponseFormat verifies the APOD conversion path:
// call APOD.Get through a mock server, convert the APODImage to an Image via
// apodImageToImage, and verify the returned Image has Date as a string in
// YYYY-MM-DD format and ApodDate as time.Time.
func TestCompat_APODConversion_ResponseFormat(t *testing.T) {
	data := loadFixture(t, "apod_single.json")
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing APOD conversion fixture: %v", err)
		}
	})

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	apodImg, err := c.APOD.Get(context.Background(), date)
	if err != nil {
		t.Fatalf("APOD.Get error: %v", err)
	}

	img := apodImageToImage(apodImg)

	// Date must be a YYYY-MM-DD string
	if img.Date != "2024-01-15" {
		t.Errorf("Image.Date = %q, want %q", img.Date, "2024-01-15")
	}

	// ApodDate must be a time.Time matching the date
	expectedTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !img.ApodDate.Equal(expectedTime) {
		t.Errorf("Image.ApodDate = %v, want %v", img.ApodDate, expectedTime)
	}

	// Verify other fields carried over
	if img.Title != "Star Cluster IC 348 from Webb" {
		t.Errorf("Image.Title = %q, want %q", img.Title, "Star Cluster IC 348 from Webb")
	}
	if img.URL == "" {
		t.Error("Image.URL is empty, want non-empty")
	}
	if img.HDURL == "" {
		t.Error("Image.HDURL is empty, want non-empty")
	}
	if img.Explanation == "" {
		t.Error("Image.Explanation is empty, want non-empty")
	}
}

// TestCompat_NeoFeedConversion_ResponseFormat verifies the NEO feed conversion
// path: call NEO.Feed through a mock server, convert the NEOFeed to a NeoList
// via neoFeedToNeoList, and verify the returned NeoList has ElementCount as
// int64 and NearEarthObjects as the expected old map type.
func TestCompat_NeoFeedConversion_ResponseFormat(t *testing.T) {
	data := loadFixture(t, "neo_feed.json")
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(data); err != nil {
			t.Fatalf("writing NEO conversion fixture: %v", err)
		}
	})

	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	feed, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		t.Fatalf("NEO.Feed error: %v", err)
	}

	nl := neoFeedToNeoList(feed, "2024-01-15", "2024-01-15")

	// ElementCount must be int64
	if nl.ElementCount != 19 {
		t.Errorf("NeoList.ElementCount = %d, want 19", nl.ElementCount)
	}

	// NearEarthObjects must be map[string][]Asteroid
	if nl.NearEarthObjects == nil {
		t.Fatal("NeoList.NearEarthObjects is nil")
	}

	asteroids, ok := nl.NearEarthObjects["2024-01-15"]
	if !ok {
		t.Fatal("NeoList.NearEarthObjects missing key \"2024-01-15\"")
	}
	if len(asteroids) != 19 {
		t.Fatalf("len(asteroids) = %d, want 19", len(asteroids))
	}

	a := asteroids[0]
	if a.ID != "3312035" {
		t.Errorf("Asteroid.ID = %q, want %q", a.ID, "3312035")
	}
	if a.Name != "(2006 AM8)" {
		t.Errorf("Asteroid.Name = %q, want %q", a.Name, "(2006 AM8)")
	}

	// Verify Start/End are set correctly
	if nl.Start != "2024-01-15" {
		t.Errorf("NeoList.Start = %q, want %q", nl.Start, "2024-01-15")
	}
	if nl.End != "2024-01-15" {
		t.Errorf("NeoList.End = %q, want %q", nl.End, "2024-01-15")
	}
}
