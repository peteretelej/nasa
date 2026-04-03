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

func TestDeprecatedHandleIndex(t *testing.T) {
	testList := []httpTestList{
		{"GET", "/", http.StatusOK, "#explanation"},
		{"GET", "/abcd", http.StatusNotFound, ""},
	}

	var err error
	deprecatedTmpl, err = template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range testList {
		req, err := http.NewRequest(v.method, v.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(deprecatedHandleIndex)
		handler.ServeHTTP(rr, req)
		if rr.Code != v.code {
			t.Errorf("deprecatedHandleIndex returned wrong status got %d, want %d", rr.Code, v.code)
		}
		if !strings.Contains(rr.Body.String(), v.contains) {
			t.Errorf("deprecatedHandleIndex returned body missing wanted text: %s", v.contains)
		}
	}
}

func TestDeprecatedRandomHandler(t *testing.T) {
	testList := []httpTestList{
		{"GET", "/abcd", http.StatusNotFound, ""},
		{"GET", "/random-apod/abcd", http.StatusNotFound, ""},
		{"GET", "/random-apod/", http.StatusOK, "#explanation"},
	}

	var err error
	deprecatedTmpl, err = template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
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
			tmpl:       deprecatedTmpl,
		}
		handler.ServeHTTP(rr, req)
		if rr.Code != v.code {
			t.Errorf("deprecatedRandomHandler returned wrong status got %d, want %d", rr.Code, v.code)
		}
		if !strings.Contains(rr.Body.String(), v.contains) {
			t.Errorf("deprecatedRandomHandler missing expected text in returned body: %s", v.contains)
		}
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
