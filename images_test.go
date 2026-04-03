package nasa

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// setupImageTestServer creates an httptest server that serves the given fixture
// file and returns a Client configured to use it. The handler captures the last
// request for parameter verification.
func setupImageTestServer(t *testing.T, fixturePath string) (*Client, *ImageService, **http.Request) {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", fixturePath, err)
	}
	var captured *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("TEST_KEY"))
	svc := &ImageService{client: c, baseURL: ts.URL}
	return c, svc, &captured
}

// setupImageErrorServer creates an httptest server that returns the given
// status code and body.
func setupImageErrorServer(t *testing.T, statusCode int, body string) *ImageService {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("TEST_KEY"))
	return &ImageService{client: c, baseURL: ts.URL}
}

// setupImageTextServer creates an httptest server that returns plain text.
func setupImageTextServer(t *testing.T, text string) *ImageService {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(text))
	}))
	t.Cleanup(ts.Close)
	c := NewClient(WithAPIKey("TEST_KEY"))
	return &ImageService{client: c, baseURL: ts.URL}
}

func TestImageSearch(t *testing.T) {
	_, svc, _ := setupImageTestServer(t, "testdata/images_search.json")

	coll, err := svc.Search(context.Background(), "apollo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if coll.TotalHits != 1542 {
		t.Errorf("expected TotalHits=1542, got %d", coll.TotalHits)
	}
	if len(coll.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(coll.Items))
	}

	// Check first item.
	item := coll.Items[0]
	if item.NasaID != "PIA12345" {
		t.Errorf("expected NasaID=PIA12345, got %s", item.NasaID)
	}
	if item.Title != "Apollo 11 Lunar Module" {
		t.Errorf("expected title 'Apollo 11 Lunar Module', got %s", item.Title)
	}
	if item.MediaType != "image" {
		t.Errorf("expected media_type=image, got %s", item.MediaType)
	}
	if item.Center != "JSC" {
		t.Errorf("expected center=JSC, got %s", item.Center)
	}
	if item.DateCreated.Year() != 1969 {
		t.Errorf("expected year 1969, got %d", item.DateCreated.Year())
	}
	if len(item.Keywords) != 3 {
		t.Errorf("expected 3 keywords, got %d", len(item.Keywords))
	}
	if item.Photographer != "NASA" {
		t.Errorf("expected photographer=NASA, got %s", item.Photographer)
	}
	if item.Location != "Moon" {
		t.Errorf("expected location=Moon, got %s", item.Location)
	}
	if !strings.Contains(item.PreviewURL, "thumb.jpg") {
		t.Errorf("expected preview URL with thumb.jpg, got %s", item.PreviewURL)
	}
	if !strings.Contains(item.AssetURL, "collection.json") {
		t.Errorf("expected asset URL with collection.json, got %s", item.AssetURL)
	}

	// Check second item (video).
	item2 := coll.Items[1]
	if item2.MediaType != "video" {
		t.Errorf("expected media_type=video, got %s", item2.MediaType)
	}
}

func TestImageSearchEmpty(t *testing.T) {
	_, svc, _ := setupImageTestServer(t, "testdata/images_empty.json")

	coll, err := svc.Search(context.Background(), "nonexistent-query-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if coll.TotalHits != 0 {
		t.Errorf("expected TotalHits=0, got %d", coll.TotalHits)
	}
	if coll.Items == nil {
		t.Error("expected non-nil Items slice for empty results")
	}
	if len(coll.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(coll.Items))
	}
	if coll.NextPage != "" {
		t.Errorf("expected empty NextPage, got %s", coll.NextPage)
	}
	if coll.PrevPage != "" {
		t.Errorf("expected empty PrevPage, got %s", coll.PrevPage)
	}
}

func TestImageSearchPagination(t *testing.T) {
	_, svc, _ := setupImageTestServer(t, "testdata/images_search.json")

	coll, err := svc.Search(context.Background(), "apollo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The fixture has an http:// next link that should be fixed to https://.
	if coll.NextPage == "" {
		t.Fatal("expected non-empty NextPage")
	}
	if strings.HasPrefix(coll.NextPage, "http://") {
		t.Errorf("expected http:// fixed to https://, got %s", coll.NextPage)
	}
	if !strings.HasPrefix(coll.NextPage, "https://") {
		t.Errorf("expected NextPage to start with https://, got %s", coll.NextPage)
	}

	// First page has no prev link.
	if coll.PrevPage != "" {
		t.Errorf("expected empty PrevPage for first page, got %s", coll.PrevPage)
	}
}

func TestImageSearchOptions(t *testing.T) {
	_, svc, captured := setupImageTestServer(t, "testdata/images_empty.json")

	_, err := svc.Search(context.Background(), "mars",
		ImageWithMediaType("image"),
		ImageWithYearStart(2020),
		ImageWithYearEnd(2025),
		ImageWithCenter("JPL"),
		ImageWithKeywords("Mars", "Rover"),
		ImageWithPage(3),
		ImageWithPageSize(50),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := *captured
	if req == nil {
		t.Fatal("expected captured request")
	}

	q := req.URL.Query()
	if got := q.Get("q"); got != "mars" {
		t.Errorf("expected q=mars, got %s", got)
	}
	if got := q.Get("media_type"); got != "image" {
		t.Errorf("expected media_type=image, got %s", got)
	}
	if got := q.Get("year_start"); got != "2020" {
		t.Errorf("expected year_start=2020, got %s", got)
	}
	if got := q.Get("year_end"); got != "2025" {
		t.Errorf("expected year_end=2025, got %s", got)
	}
	if got := q.Get("center"); got != "JPL" {
		t.Errorf("expected center=JPL, got %s", got)
	}
	if got := q.Get("keywords"); got != "Mars,Rover" {
		t.Errorf("expected keywords=Mars,Rover, got %s", got)
	}
	if got := q.Get("page"); got != "3" {
		t.Errorf("expected page=3, got %s", got)
	}
	if got := q.Get("page_size"); got != "50" {
		t.Errorf("expected page_size=50, got %s", got)
	}
}

func TestImageAsset(t *testing.T) {
	_, svc, _ := setupImageTestServer(t, "testdata/images_asset.json")

	manifest, err := svc.Asset(context.Background(), "PIA12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Items) != 6 {
		t.Fatalf("expected 6 asset items, got %d", len(manifest.Items))
	}

	// Verify first and last hrefs.
	if !strings.Contains(manifest.Items[0].Href, "orig.jpg") {
		t.Errorf("expected first item to be orig.jpg, got %s", manifest.Items[0].Href)
	}
	if !strings.Contains(manifest.Items[5].Href, "metadata.json") {
		t.Errorf("expected last item to be metadata.json, got %s", manifest.Items[5].Href)
	}
}

func TestImageMetadata(t *testing.T) {
	metadataJSON := `{"AVAIL:Title":"Apollo 11","EXIF:ImageWidth":4096,"EXIF:ImageHeight":3072,"XMP:Creator":"NASA"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(metadataJSON))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("TEST_KEY"))
	svc := &ImageService{client: c, baseURL: ts.URL}

	meta, err := svc.Metadata(context.Background(), "PIA12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta == nil {
		t.Fatal("expected non-nil metadata")
	}
	if title, ok := meta["AVAIL:Title"].(string); !ok || title != "Apollo 11" {
		t.Errorf("expected AVAIL:Title='Apollo 11', got %v", meta["AVAIL:Title"])
	}
	// JSON numbers decode as float64 by default.
	if width, ok := meta["EXIF:ImageWidth"].(float64); !ok || width != 4096 {
		t.Errorf("expected EXIF:ImageWidth=4096, got %v", meta["EXIF:ImageWidth"])
	}
}

func TestImageCaptions(t *testing.T) {
	srtContent := "1\n00:00:00,000 --> 00:00:05,000\nWelcome to the NASA video.\n\n2\n00:00:05,000 --> 00:00:10,000\nThis is a caption test.\n"

	svc := setupImageTextServer(t, srtContent)

	captions, err := svc.Captions(context.Background(), "KSC-VIDEO-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captions != srtContent {
		t.Errorf("expected caption text to match, got %q", captions)
	}
}

func TestImageSearch_Error404(t *testing.T) {
	svc := setupImageErrorServer(t, http.StatusNotFound, `{"error":{"message":"not found"}}`)

	_, err := svc.Search(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestImageAsset_Error404(t *testing.T) {
	svc := setupImageErrorServer(t, http.StatusNotFound, `{"error":{"message":"not found"}}`)

	_, err := svc.Asset(context.Background(), "INVALID-ID")
	if err == nil {
		t.Fatal("expected error for 404")
	}

	var nfErr *NotFoundError
	if !errors.As(err, &nfErr) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestImageSearch_NoAuth(t *testing.T) {
	var gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.URL.Query().Get("api_key")
		w.Header().Set("Content-Type", "application/json")
		data, _ := os.ReadFile("testdata/images_empty.json")
		w.Write(data)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("should-not-appear"))
	svc := &ImageService{client: c, baseURL: ts.URL}

	_, err := svc.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAPIKey != "" {
		t.Errorf("expected no api_key in request, got %s", gotAPIKey)
	}
}

func TestImageAsset_NoAuth(t *testing.T) {
	var gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.URL.Query().Get("api_key")
		w.Header().Set("Content-Type", "application/json")
		data, _ := os.ReadFile("testdata/images_asset.json")
		w.Write(data)
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("should-not-appear"))
	svc := &ImageService{client: c, baseURL: ts.URL}

	_, err := svc.Asset(context.Background(), "PIA12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAPIKey != "" {
		t.Errorf("expected no api_key in request, got %s", gotAPIKey)
	}
}

func TestImageMetadata_NoAuth(t *testing.T) {
	var gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.URL.Query().Get("api_key")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"key":"value"}`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("should-not-appear"))
	svc := &ImageService{client: c, baseURL: ts.URL}

	_, err := svc.Metadata(context.Background(), "PIA12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAPIKey != "" {
		t.Errorf("expected no api_key in request, got %s", gotAPIKey)
	}
}

func TestImageCaptions_NoAuth(t *testing.T) {
	var gotAPIKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.URL.Query().Get("api_key")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("captions"))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("should-not-appear"))
	svc := &ImageService{client: c, baseURL: ts.URL}

	_, err := svc.Captions(context.Background(), "VIDEO-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAPIKey != "" {
		t.Errorf("expected no api_key in request, got %s", gotAPIKey)
	}
}
