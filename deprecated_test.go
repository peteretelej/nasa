package nasa

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeprecatedHandleIndex(t *testing.T) {
	var err error
	deprecatedTmpl, err = template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("success renders template", func(t *testing.T) {
		prev := deprecatedAPODToday
		deprecatedAPODToday = func() (*Image, error) {
			return &Image{URL: "https://example.com/apod.jpg", HDURL: "https://example.com/apod.jpg"}, nil
		}
		defer func() { deprecatedAPODToday = prev }()

		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		http.HandlerFunc(deprecatedHandleIndex).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("deprecatedHandleIndex returned wrong status got %d, want %d", rr.Code, http.StatusOK)
		}
		if !strings.Contains(rr.Body.String(), "#explanation") {
			t.Fatalf("deprecatedHandleIndex returned body missing wanted text: %s", "#explanation")
		}
	})

	t.Run("fetch failure returns 503", func(t *testing.T) {
		prev := deprecatedAPODToday
		deprecatedAPODToday = func() (*Image, error) {
			return nil, errors.New("NASA API currently unavailable")
		}
		defer func() { deprecatedAPODToday = prev }()

		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		http.HandlerFunc(deprecatedHandleIndex).ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("deprecatedHandleIndex returned wrong status got %d, want %d", rr.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("non-root returns 404", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/abcd", nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		http.HandlerFunc(deprecatedHandleIndex).ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("deprecatedHandleIndex returned wrong status got %d, want %d", rr.Code, http.StatusNotFound)
		}
	})
}

func TestDeprecatedRandomHandler(t *testing.T) {
	var err error
	deprecatedTmpl, err = template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("success renders template", func(t *testing.T) {
		prev := deprecatedRandomAPOD
		deprecatedRandomAPOD = func() (*Image, error) {
			return &Image{URL: "https://example.com/random.jpg", HDURL: "https://example.com/random.jpg"}, nil
		}
		defer func() { deprecatedRandomAPOD = prev }()

		req, err := http.NewRequest(http.MethodGet, "/random-apod/", nil)
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

		if rr.Code != http.StatusOK {
			t.Fatalf("deprecatedRandomHandler returned wrong status got %d, want %d", rr.Code, http.StatusOK)
		}
		if !strings.Contains(rr.Body.String(), "#explanation") {
			t.Fatalf("deprecatedRandomHandler missing expected text in returned body: %s", "#explanation")
		}
	})

	t.Run("empty-cache fetch failure returns 503", func(t *testing.T) {
		prev := deprecatedRandomAPOD
		deprecatedRandomAPOD = func() (*Image, error) {
			return nil, errors.New("NASA API currently unavailable")
		}
		defer func() { deprecatedRandomAPOD = prev }()

		req, err := http.NewRequest(http.MethodGet, "/random-apod/", nil)
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

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("deprecatedRandomHandler returned wrong status got %d, want %d", rr.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("invalid paths return 404", func(t *testing.T) {
		for _, path := range []string{"/abcd", "/random-apod/abcd"} {
			req, err := http.NewRequest(http.MethodGet, path, nil)
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
			if rr.Code != http.StatusNotFound {
				t.Fatalf("deprecatedRandomHandler returned wrong status got %d, want %d", rr.Code, http.StatusNotFound)
			}
		}
	})
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
