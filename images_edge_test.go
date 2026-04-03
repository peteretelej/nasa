package nasa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestImages_Page2_PrevAndNextLinks verifies that a page 2 response has both
// PrevPage and NextPage populated with https-fixed URLs.
func TestImages_Page2_PrevAndNextLinks(t *testing.T) {
	_, svc, _ := setupImageTestServer(t, "testdata/images_page2.json")

	coll, err := svc.Search(context.Background(), "mars", ImageWithPage(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if coll.TotalHits != 250 {
		t.Errorf("TotalHits = %d, want 250", coll.TotalHits)
	}
	if len(coll.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(coll.Items))
	}

	// Verify the item parsed correctly.
	item := coll.Items[0]
	if item.NasaID != "PIA67890" {
		t.Errorf("NasaID = %q, want PIA67890", item.NasaID)
	}

	// Verify NextPage is present and https-fixed.
	if coll.NextPage == "" {
		t.Fatal("expected non-empty NextPage for page 2")
	}
	if strings.HasPrefix(coll.NextPage, "http://") {
		t.Errorf("NextPage not fixed to https://: %s", coll.NextPage)
	}
	if !strings.HasPrefix(coll.NextPage, "https://") {
		t.Errorf("NextPage should start with https://, got %s", coll.NextPage)
	}
	if !strings.Contains(coll.NextPage, "page=3") {
		t.Errorf("NextPage should contain page=3, got %s", coll.NextPage)
	}

	// Verify PrevPage is present and https-fixed.
	if coll.PrevPage == "" {
		t.Fatal("expected non-empty PrevPage for page 2")
	}
	if strings.HasPrefix(coll.PrevPage, "http://") {
		t.Errorf("PrevPage not fixed to https://: %s", coll.PrevPage)
	}
	if !strings.HasPrefix(coll.PrevPage, "https://") {
		t.Errorf("PrevPage should start with https://, got %s", coll.PrevPage)
	}
	if !strings.Contains(coll.PrevPage, "page=1") {
		t.Errorf("PrevPage should contain page=1, got %s", coll.PrevPage)
	}
}

// TestImages_MalformedResponse verifies that the SDK handles a non-Collection+JSON
// response gracefully. When the server returns {"bad": true} instead of a valid
// Collection+JSON envelope, the SDK returns an empty collection with zero hits
// rather than an error (since the JSON is syntactically valid, just structurally
// wrong).
func TestImages_MalformedResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"bad": true}`)); err != nil {
			t.Fatalf("writing malformed response: %v", err)
		}
	}))
	t.Cleanup(ts.Close)

	c := NewClient(WithAPIKey("TEST_KEY"))
	svc := &ImageService{client: c, baseURL: ts.URL}

	coll, err := svc.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The response is valid JSON but not Collection+JSON. The SDK unmarshals
	// it into an empty collectionJSON struct, resulting in zero-value fields.
	if coll.TotalHits != 0 {
		t.Errorf("TotalHits = %d, want 0 for malformed response", coll.TotalHits)
	}
	if len(coll.Items) != 0 {
		t.Errorf("Items length = %d, want 0 for malformed response", len(coll.Items))
	}
	if coll.NextPage != "" {
		t.Errorf("NextPage = %q, want empty for malformed response", coll.NextPage)
	}
	if coll.PrevPage != "" {
		t.Errorf("PrevPage = %q, want empty for malformed response", coll.PrevPage)
	}
}
