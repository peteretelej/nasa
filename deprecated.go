// Deprecated types and functions retained for backward compatibility.
//
// Everything in this file is scheduled for removal in a future major version.
// New code should use [Client] and its sub-services instead.

package nasa

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const deprecatedFallbackImageURL = "https://apod.nasa.gov/apod/image/1901/IC405_Abolfath_3952.jpg"

// NewServer creates an HTTP web server that serves APOD pictures.
//
//	/ - today's APOD
//	/random-apod - returns a random APOD
//
// Deprecated: Use [Client].APOD with your own HTTP server instead.
// This function will be removed in a future major version.
func NewServer(listenAddr string) (*http.Server, error) {
	tmpl, err := template.New("tmpl").Parse(deprecatedTmplHTML)
	if err != nil {
		return nil, fmt.Errorf("unable to parse template: %v", err)
	}
	rh := &deprecatedRandomHandler{
		lastUpdate: time.Now().Add(-10 * time.Hour),
		cachedApod: &Image{},
		tmpl:       tmpl,
	}
	ih := &deprecatedIndexHandler{tmpl: tmpl}

	mux := http.NewServeMux()
	mux.Handle("/", ih)
	mux.Handle("/random-apod/", rh)

	return &http.Server{
		Addr:           listenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		IdleTimeout:    5 * time.Minute,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}, nil
}

type deprecatedIndexHandler struct {
	tmpl *template.Template
}

func (h *deprecatedIndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	apod, err := APODToday()
	if err != nil {
		log.Printf("deprecated index APOD fetch error: %v", err)
		apod = deprecatedFallbackImage()
	}
	td := TmplData{Apod: *apod}
	td.renderWith(w, h.tmpl)
}

type deprecatedRandomHandler struct {
	tmpl *template.Template

	mu         sync.RWMutex // protects the values below
	lastUpdate time.Time
	cachedApod *Image
}

func (h *deprecatedRandomHandler) last() time.Time {
	h.mu.RLock()
	last := h.lastUpdate
	h.mu.RUnlock()
	return last
}
func (h *deprecatedRandomHandler) apod() Image {
	h.mu.RLock()
	apod := *h.cachedApod
	h.mu.RUnlock()
	return apod
}
func (h *deprecatedRandomHandler) update(apod Image) {
	h.mu.Lock()
	h.cachedApod = &apod
	h.lastUpdate = time.Now()
	h.mu.Unlock()
}

func (h *deprecatedRandomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/random-apod/" {
		http.NotFound(w, r)
		return
	}
	apod := h.apod()

	// Update if cached apod is older than a second
	if time.Since(h.last()) > time.Second {
		if newApod, err := RandomAPOD(); err == nil {
			if newApod.URL != "" {
				apod = *newApod
			}
		} else {
			log.Printf("deprecated random APOD fetch error: %v", err)
			if apod.URL == "" && apod.HDURL == "" {
				apod = *deprecatedFallbackImage()
			}
		}
		h.update(apod)
	}

	td := TmplData{
		Apod:       h.apod(),
		SD:         r.URL.Query().Get("sd") != "",
		AutoReload: r.URL.Query().Get("auto") != "" || r.URL.Query().Get("interval") != "",
		Legacy:     r.URL.Query().Get("legacy") != "",
	}
	i := r.URL.Query().Get("interval")
	if i != "" {
		if n, err := strconv.Atoi(i); err == nil {
			td.AutoReloadInterval = n
		}
	}
	if td.AutoReloadInterval < 1 {
		td.AutoReloadInterval = 5 * 60 // default reload every 5 minutes
	}
	td.renderWith(w, h.tmpl)
}

var (
	deprecatedTmpl     *template.Template
	deprecatedTmplOnce sync.Once
)

// deprecatedEnsureTmpl lazily parses the template for callers of the public
// Render method that may not have gone through NewServer.
func deprecatedEnsureTmpl() *template.Template {
	deprecatedTmplOnce.Do(func() {
		deprecatedTmpl = template.Must(template.New("tmpl").Parse(deprecatedTmplHTML))
	})
	return deprecatedTmpl
}

func deprecatedFallbackImage() *Image {
	return &Image{
		Date:        time.Now().Format(dateFormat),
		Title:       "NASA Astronomy Picture of the Day",
		URL:         deprecatedFallbackImageURL,
		HDURL:       deprecatedFallbackImageURL,
		Explanation: "NASA APOD is temporarily unavailable. This placeholder keeps the legacy web view responsive until the next refresh succeeds.",
		ApodDate:    time.Now(),
	}
}

// TmplData defines the data used to render the HTML template for the
// deprecated web server.
//
// Deprecated: Use [Client].APOD with your own HTTP server and templates
// instead. This type will be removed in a future major version.
type TmplData struct {
	Page               string
	Title              string
	Apod               Image
	SD                 bool // Standard definition display
	AutoReload         bool
	Legacy             bool // legacy browser does not support new reload
	IsYoutube          bool
	AutoReloadInterval int
}

// Render writes an HTML page to the ResponseWriter based on the template data.
//
// Deprecated: Use [Client].APOD with your own HTTP server and templates
// instead. This method will be removed in a future major version.
func (td TmplData) Render(wr http.ResponseWriter) {
	td.renderWith(wr, deprecatedEnsureTmpl())
}

// renderWith is the internal render method that accepts an explicit template,
// avoiding reliance on the package-level variable.
func (td TmplData) renderWith(wr http.ResponseWriter, tmpl *template.Template) {
	if strings.Contains(td.Apod.URL, "youtube") {
		td.IsYoutube = true
		if i := strings.Index(td.Apod.URL, "?"); i > 1 {
			td.Apod.URL = td.Apod.URL[:i]
		}
	}
	if td.Apod.URL == "" && td.Apod.HDURL == "" {
		http.Error(wr, "NASA API currently unavailable, it's experiencing downtime :(", http.StatusServiceUnavailable)
		return
	}
	if err := tmpl.Execute(wr, td); err != nil {
		log.Print(err)
	}
}

const deprecatedTmplHTML = `<!DOCTYPE html>
<html lang="en">
<meta charset="UTF-8">
<title>{{with .Title}}{{.}}{{else}}NASA Astronomy Picture of the Day{{end}}</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
{{if .AutoReload -}}
{{if not .Legacy }}
<meta http-equiv="refresh" content="{{.AutoReloadInterval}}" >
{{end}}
{{end -}}
<style>html,body{ margin:0; padding:0}
body{background-color:#000;color:#fff}
#imgwrap{
	position:fixed; top:0;left:0;
	min-width:100%; max-width:100%;
	overflow:hidden;
}
#imgwrap img{
	display:block; margin:0 auto;padding:0;
	max-width:100%; max-height:100%;
}

#apod{ display:block; position:fixed; bottom:0; left:30px; right:30px;}
#explanation {
	display:none;
	background-color:rgba(50,50,50,0.5); 
	padding:10px; border-radius:10px;
}
#apod:hover #explanation{display:block}
@media screen and (max-width:600px){
	#explanation{display:none;}
}
</style>
{{if and .AutoReload .Legacy}}
<script type='text/javascript'>
function TimedRefresh( t ) { setTimeout("location.reload(true);", t); }
</script>
{{end}}
<body {{if and .AutoReload .Legacy}}onload="javascript:TimedRefresh({{.AutoReloadInterval}}*1000)"{{end}}>
<div id="imgwrap">
{{if .IsYoutube}}
<div style="position: fixed; z-index: -99; width: 100%; height: 100%">
<iframe frameborder="0" height="100%" width="100%" 
src="{{.Apod.URL}}?autoplay=1&controls=0&showinfo=0&autohide=1">
</iframe>
</div>
{{else}}
<img src="{{if .SD}}{{.Apod.URL}}{{else}}{{if .Apod.HDURL}}{{.Apod.HDURL}}{{else}}{{.Apod.URL}}{{end}}{{end}}" id="bg" alt="{{.Apod.Title}}" />
{{end}}
</div>
<div id="apod">
<div id="explanation">
<h4>{{.Apod.Title}}</h4>
<p>{{.Apod.Explanation}}</p>
<p>NASA Astronomy Picture of the Day {{.Apod.Date}} {{if .Apod.HDURL}}<a href="{{.Apod.HDURL}}" style="display:inline-block; color:#efefef"><i>Open Image in HD</i></a>{{end}} </p>
</div>
<h4>{{.Apod.Title}}</h4>
<p style="text-align:right">
View in fullscreen (F11) for best experience &#9786;.
<i>Reload random pic every <a href="/random-apod/?auto=1&interval=60" style="color:#fff">1 min</a>, <a href="/random-apod/?auto=1&interval=600" style="color:#fff">10 min</a></i>
<b>{{.Title}}</b>
<i>This project is on Github.</i>
<a class="github-button" href="https://github.com/peteretelej/nasa" data-icon="octicon-star" data-show-count="true" aria-label="Star peteretelej/nasa on GitHub">Star</a>
</p>
</div>
<script async defer src="https://buttons.github.io/buttons.js"></script>
</body>
</html>`
