package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/peteretelej/nasa"
)

var (
	interval = flag.Duration("interval", time.Second, "interval to change the random image")
	listen   = flag.String("listen", ":8080", "http server listening address")
)

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := serve(*listen); err != nil {
		log.Fatalf("server crashed: %v", err)
	}
}

func serve(listenAddr string) error {
	tmpl, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("unable to parse template: %v", err)
	}
	apod, err := nasa.RandomAPOD()
	if err != nil {
		return fmt.Errorf("unable to fetch random apod: %v", err)
	}
	h := &handler{
		lastUpdate: time.Now(),
		cachedApod: apod,
		tmpl:       tmpl,
	}
	http.Handle("/", h)
	svr := &http.Server{
		Addr:           listenAddr,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Printf("launching http server at %s\n", listenAddr)
	return svr.ListenAndServe()
}

type handler struct {
	tmpl *template.Template

	mu         sync.RWMutex // protects the values below
	lastUpdate time.Time
	cachedApod *nasa.Image
}

func (h *handler) last() time.Time {
	h.mu.RLock()
	last := h.lastUpdate
	h.mu.RUnlock()
	return last
}
func (h *handler) apod() nasa.Image {
	h.mu.RLock()
	apod := *h.cachedApod
	h.mu.RUnlock()
	return apod
}
func (h *handler) update(apod nasa.Image, t time.Time) {
	h.mu.Lock()
	h.cachedApod = &apod
	h.lastUpdate = time.Now()
	h.mu.Unlock()

}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if time.Now().Sub(h.last()) < *interval {
		h.render(w, r)
		return
	}
	apod := h.apod()
	if newApod, err := nasa.RandomAPOD(); err == nil {
		apod = *newApod
	}
	h.update(apod, time.Now())
	h.render(w, r)
}
func (h *handler) render(w http.ResponseWriter, r *http.Request) {
	d := struct {
		Apod               nasa.Image
		HD                 bool
		AutoReload         bool
		AutoReloadInterval int // reload interval in seconds
	}{
		Apod:       h.apod(),
		HD:         r.URL.Query().Get("sd") == "",
		AutoReload: r.URL.Query().Get("auto") != "",
	}
	if d.AutoReload {
		i := r.URL.Query().Get("interval")
		if i != "" {
			if n, err := strconv.Atoi(i); err == nil {
				d.AutoReloadInterval = n
			}
		}
		if d.AutoReloadInterval < 1 {
			d.AutoReloadInterval = 5 * 60 // default reload every 5 minutes
		}
	}
	if err := h.tmpl.Execute(w, d); err != nil {
		log.Print(err)
	}
}

const tmpl = `<!DOCTYPE html>
<html lang="en">
<meta charset="UTF-8">
<title>Random NASA APOD</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
{{if .AutoReload -}}
<meta http-equiv="refresh" content="{{.AutoReloadInterval}}" >
{{end -}}
<style>html,body{ margin:0; padding:0}
body{background-color:#000;color:#fff}
{{if .Apod -}}
#imgwrap{
	position:fixed; top:0;left:0;
	min-width:100%; max-width:100%;
	overflow:hidden;
}
#imgwrap img{
	display:block; margin:0 auto;
	max-width:100%; max-height:100%;
}
{{end -}}
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
{{if not .Apod}}
<p>Unable to generate random APOD :(.</p>
{{end}}
{{if .Apod}}
<div id="imgwrap"><img src="{{.Apod.HDURL}}" id="bg" alt="{{.Apod.Title}}" /></div>
<div id="apod">
<div id="explanation">
	<h4>{{.Apod.Title}}</h4>
	<p>{{.Apod.Explanation}}</p>
	<p>NASA Astronomy Picture of the Day {{.Apod.Date}} <a href="{{.Apod.HDURL}}" style="display:inline-block; color:#efefef"><i>Open Image in HD</i></a> </p>
</div>
<h4>{{.Apod.Title}}</h4>
<p style="text-align:right">
View in fullscreen (F11) for best experience &#9786;.
<a href="?auto=1&interval=60" style="color:#fff">Click here to Auto-reload.</a>
<b>Random NASA Astronomy Picture of the Day.</b>
<i>This project is on Github.</i>
<a class="github-button" href="https://github.com/peteretelej/nasa" data-icon="octicon-star" data-show-count="true" aria-label="Star peteretelej/nasa on GitHub">Star</a>
</p>
</div>
<script async defer src="https://buttons.github.io/buttons.js"></script>
{{end}}
</body>
</html>`
