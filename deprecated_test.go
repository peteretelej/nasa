package nasa

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type httpTestList struct {
	method   string
	path     string
	code     int
	contains string
}

// apodMockHandler returns an http.HandlerFunc that serves different fixtures
// depending on whether the request looks like a Today call (single object) or
// a Random call (array, triggered by the "count" query param).
func apodMockHandler(t *testing.T, todayFixture, randomFixture string) http.HandlerFunc {
	t.Helper()
	todayData := loadFixture(t, todayFixture)
	randomData := loadFixture(t, randomFixture)
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/planetary/apod") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("count") != "" {
			_, _ = w.Write(randomData)
		} else {
			_, _ = w.Write(todayData)
		}
	}
}

func TestDeprecatedHandleIndex(t *testing.T) {
	// Wire up a mock client so APODToday() does not hit the real API.
	c := setupMockServer(t, apodMockHandler(t, "apod_single.json", "apod_range.json"))
	setDefaultClientForTest(t, c)

	var err error
	tmpl, err := template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	testList := []httpTestList{
		{"GET", "/", http.StatusOK, "#explanation"},
		{"GET", "/abcd", http.StatusNotFound, ""},
	}

	for _, v := range testList {
		req, err := http.NewRequest(v.method, v.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler := &deprecatedIndexHandler{tmpl: tmpl}
		handler.ServeHTTP(rr, req)
		if rr.Code != v.code {
			t.Errorf("deprecatedIndexHandler %s returned wrong status got %d, want %d", v.path, rr.Code, v.code)
		}
		if v.contains != "" && !strings.Contains(rr.Body.String(), v.contains) {
			t.Errorf("deprecatedIndexHandler %s missing expected text %q", v.path, v.contains)
		}
	}
}

func TestDeprecatedHandleIndex_Fallback(t *testing.T) {
	// Mock server that always fails, forcing the fallback path.
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":500,"msg":"internal"}`, http.StatusInternalServerError)
	})
	setDefaultClientForTest(t, c)

	var err error
	tmpl, err := template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := &deprecatedIndexHandler{tmpl: tmpl}
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("fallback: got status %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "temporarily unavailable") {
		t.Error("fallback: body should contain fallback explanation text")
	}
}

func TestDeprecatedRandomHandler(t *testing.T) {
	// Wire up a mock client so RandomAPOD() does not hit the real API.
	c := setupMockServer(t, apodMockHandler(t, "apod_single.json", "apod_range.json"))
	setDefaultClientForTest(t, c)

	tmpl, err := template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	testList := []httpTestList{
		{"GET", "/abcd", http.StatusNotFound, ""},
		{"GET", "/random-apod/abcd", http.StatusNotFound, ""},
		{"GET", "/random-apod/", http.StatusOK, "#explanation"},
	}

	for _, v := range testList {
		req, err := http.NewRequest(v.method, v.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler := &deprecatedRandomHandler{
			lastUpdate: time.Now().Add(-10 * time.Hour),
			cachedApod: &Image{},
			tmpl:       tmpl,
		}
		handler.ServeHTTP(rr, req)
		if rr.Code != v.code {
			t.Errorf("deprecatedRandomHandler %s returned wrong status got %d, want %d", v.path, rr.Code, v.code)
		}
		if v.contains != "" && !strings.Contains(rr.Body.String(), v.contains) {
			t.Errorf("deprecatedRandomHandler %s missing expected text %q", v.path, v.contains)
		}
	}
}

func TestDeprecatedRandomHandler_Fallback(t *testing.T) {
	// Mock server that always fails, forcing the fallback path.
	c := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":500,"msg":"internal"}`, http.StatusInternalServerError)
	})
	setDefaultClientForTest(t, c)

	tmpl, err := template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/random-apod/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := &deprecatedRandomHandler{
		lastUpdate: time.Now().Add(-10 * time.Hour),
		cachedApod: &Image{}, // empty, so fallback will be used
		tmpl:       tmpl,
	}
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("fallback: got status %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "temporarily unavailable") {
		t.Error("fallback: body should contain fallback explanation text")
	}
}

// Compile-time assertion: NewServer signature must remain stable.
func TestNewServer_SignatureCompiles(t *testing.T) {
	assertType[func(string) (*http.Server, error)](NewServer)
}

// Compile-time assertion: TmplData must remain an exported type with Render.
func TestTmplData_SignatureCompiles(t *testing.T) {
	var td TmplData
	td.Apod = Image{}
	_ = td.Page
	_ = td.Title
	_ = td.SD
	_ = td.AutoReload
	_ = td.Legacy
	_ = td.IsYoutube
	_ = td.AutoReloadInterval

	assertType[func(http.ResponseWriter)](td.Render)
}
