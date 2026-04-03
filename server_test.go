package nasa

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestHandleIndex(t *testing.T) {
	// Prime cache so APODToday() returns immediately without HTTP call
	tAPOD.update(Image{
		URL:   "https://example.com/test.jpg",
		HDURL: "https://example.com/test_hd.jpg",
		Title: "Test APOD",
		Date:  time.Now().Format("2006-01-02"),
	})

	testList := []httpTestList{
		{"GET", "/", http.StatusOK, "#explanation"},
		{"GET", "/abcd", http.StatusNotFound, ""},
	}

	var err error
	tmpl, err = template.New("tmpl").Parse(tmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range testList {
		req, err := http.NewRequest(v.method, v.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handleIndex)
		handler.ServeHTTP(rr, req)
		if rr.Code != v.code {
			t.Errorf("handleIndex returned wrong status got %d, want %d", rr.Code, v.code)
		}
		if !strings.Contains(rr.Body.String(), v.contains) {
			t.Errorf("handleIndex returned body missing wanted text: %s", v.contains)
		}

	}
}

func TestRandomHandler(t *testing.T) {
	if os.Getenv("NASA_API_KEY") == "" && os.Getenv("NASAKEY") == "" {
		t.Skip("requires NASA_API_KEY or NASAKEY env var")
	}

	testList := []httpTestList{
		{"GET", "/abcd", http.StatusNotFound, ""},
		{"GET", "/random-apod/abcd", http.StatusNotFound, ""},
		{"GET", "/random-apod/", http.StatusOK, "#explanation"},
	}

	var err error
	tmpl, err = template.New("tmpl").Parse(tmplHTML)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range testList {
		req, err := http.NewRequest(v.method, v.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler := &randomHandler{
			lastUpdate: time.Now().Add(-10 * time.Hour),
			cachedApod: &Image{},
			tmpl:       tmpl,
		}
		handler.ServeHTTP(rr, req)
		if rr.Code != v.code {
			t.Errorf("randomHandler returned wrong status got %d, want %d", rr.Code, v.code)
		}
		if !strings.Contains(rr.Body.String(), v.contains) {
			t.Errorf("randomHandler missing expected text in returned body: %s", v.contains)
		}

	}

}
