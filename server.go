package nasa

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// NewServer a http web-server that serves APOD pictures
//     / - today's APOD
//     /random-apod - returns a random APOD
//     TODO: /apod/YYYY-MM-DD - returns apod for specified date
func NewServer(listenAddr string) (*http.Server, error) {
	var err error
	tmpl, err = template.New("tmpl").Parse(tmplHTML)
	if err != nil {
		return nil, fmt.Errorf("unable to parse template: %v", err)
	}
	rh := &randomHandler{
		lastUpdate: time.Now().Add(-10 * time.Hour),
		cachedApod: &Image{},
		tmpl:       tmpl,
	}
	http.HandleFunc("/", handleIndex)
	http.Handle("/random-apod/", rh)

	return &http.Server{
		Addr:           listenAddr,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}, nil

}
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	apod, err := APODToday()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	TmplData{Apod: *apod}.Render(w)
}

type randomHandler struct {
	tmpl *template.Template

	mu         sync.RWMutex // protects the values below
	lastUpdate time.Time
	cachedApod *Image
}

func (h *randomHandler) last() time.Time {
	h.mu.RLock()
	last := h.lastUpdate
	h.mu.RUnlock()
	return last
}
func (h *randomHandler) apod() Image {
	h.mu.RLock()
	apod := *h.cachedApod
	h.mu.RUnlock()
	return apod
}
func (h *randomHandler) update(apod Image, t time.Time) {
	h.mu.Lock()
	h.cachedApod = &apod
	h.lastUpdate = time.Now()
	h.mu.Unlock()

}

func (h *randomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/random-apod/" {
		http.NotFound(w, r)
		return
	}
	apod := h.apod()

	// Update if cached apod is older than a second
	if time.Now().Sub(h.last()) > time.Second {
		if newApod, err := RandomAPOD(); err == nil {
			if newApod.URL != "" {
				apod = *newApod
			}
		}
		h.update(apod, time.Now())
	}

	td := TmplData{
		Apod:       h.apod(),
		SD:         r.URL.Query().Get("sd") != "",
		AutoReload: r.URL.Query().Get("auto") != "" || r.URL.Query().Get("interval") != "",
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
	td.Render(w)
}

var tmpl *template.Template

// TmplData defines the data used to render the html template (tmpl)
type TmplData struct {
	Page               string
	Title              string
	Apod               Image
	SD                 bool // Standard definition display
	AutoReload         bool
	AutoReloadInterval int
}

// Render returns an html to the responsewriter based on the template data
func (td TmplData) Render(wr http.ResponseWriter) {
	if td.Apod.URL == "" && td.Apod.HDURL == "" {
		http.Error(wr, "NASA API currently unavailable, it's experiencing downtime :(", http.StatusServiceUnavailable)
		return
	}
	if err := tmpl.Execute(wr, td); err != nil {
		log.Print(err)
	}
}

const tmplHTML = `<!DOCTYPE html>
<html lang="en">
<meta charset="UTF-8">
<title>{{with .Title}}{{.}}{{else}}NASA Astronomy Picture of the Day{{end}}</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
{{if .AutoReload -}}
<meta http-equiv="refresh" content="{{.AutoReloadInterval}}" >
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
<body>
<div id="imgwrap"><img src="{{if .SD}}{{.Apod.URL}}{{else}}{{.Apod.HDURL}}{{end}}" id="bg" alt="{{.Apod.Title}}" /></div>
<div id="apod">
<div id="explanation">
<h4>{{.Apod.Title}}</h4>
<p>{{.Apod.Explanation}}</p>
<p>NASA Astronomy Picture of the Day {{.Apod.Date}} <a href="{{.Apod.HDURL}}" style="display:inline-block; color:#efefef"><i>Open Image in HD</i></a> </p>
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
