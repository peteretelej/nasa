// Command web-server serves NASA Astronomy Picture of the Day via HTTP.
//
// It demonstrates using the nasa.Client API to fetch APOD data and
// render it in a full-screen HTML view.
//
// Usage:
//
//	NASA_API_KEY=your-key go run ./examples/web-server
//	NASA_API_KEY=your-key go run ./examples/web-server -listen :9090
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/peteretelej/nasa"
)

var listenAddr = flag.String("listen", ":8080", "HTTP listen address")

// templateData holds the data passed to the HTML template.
type templateData struct {
	Title              string
	Image              *nasa.APODImage
	SD                 bool
	AutoReload         bool
	Legacy             bool
	IsYoutube          bool
	EmbedURL           string // YouTube URL with query params stripped
	AutoReloadInterval int
}

// cachedAPOD holds a cached APOD result protected by a RWMutex.
type cachedAPOD struct {
	mu        sync.RWMutex
	image     *nasa.APODImage
	updatedAt time.Time
}

func (c *cachedAPOD) get() (*nasa.APODImage, time.Time) {
	c.mu.RLock()
	img, t := c.image, c.updatedAt
	c.mu.RUnlock()
	return img, t
}

func (c *cachedAPOD) set(img *nasa.APODImage) {
	c.mu.Lock()
	c.image = img
	c.updatedAt = time.Now()
	c.mu.Unlock()
}

var (
	todayCache  = &cachedAPOD{}
	randomCache = &cachedAPOD{}
	tmpl        *template.Template
	client      *nasa.Client
)

const cacheRefreshInterval = 30 * time.Second

func main() {
	flag.Parse()

	// Resolve API key.
	apiKey := os.Getenv("NASA_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("NASAKEY")
	}
	if apiKey == "" {
		log.Println("WARNING: NASA_API_KEY not set, using DEMO_KEY (limited to 30 req/hr)")
		apiKey = "DEMO_KEY"
	}

	client = nasa.NewClient(nasa.WithAPIKey(apiKey))

	var err error
	tmpl, err = template.New("apod").Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
	}).Parse(tmplHTML)
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/random-apod/", handleRandom)

	svr := &http.Server{
		Addr:           *listenAddr,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		IdleTimeout:    5 * time.Minute,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Printf("nasa web-server: listening on %s\n", *listenAddr)
	if err := svr.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	img, updatedAt := todayCache.get()
	if img == nil || time.Since(updatedAt) > cacheRefreshInterval {
		apod, err := client.APOD.Today(r.Context())
		if err != nil {
			log.Printf("APOD fetch error: %v", err)
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		todayCache.set(apod)
		img = apod
	}

	renderTemplate(w, img, r)
}

func handleRandom(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/random-apod/" {
		http.NotFound(w, r)
		return
	}

	img, updatedAt := randomCache.get()
	if img == nil || time.Since(updatedAt) > cacheRefreshInterval {
		imgs, err := client.APOD.Random(r.Context(), 1)
		if err != nil || len(imgs) == 0 {
			if err != nil {
				log.Printf("APOD random fetch error: %v", err)
			}
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if imgs[0].URL != "" {
			randomCache.set(imgs[0])
			img = imgs[0]
		} else if img == nil {
			http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
	}

	renderTemplate(w, img, r)
}

func renderTemplate(w http.ResponseWriter, img *nasa.APODImage, r *http.Request) {
	td := templateData{
		Image:      img,
		SD:         r.URL.Query().Get("sd") != "",
		AutoReload: r.URL.Query().Get("auto") != "" || r.URL.Query().Get("interval") != "",
		Legacy:     r.URL.Query().Get("legacy") != "",
	}

	if i := r.URL.Query().Get("interval"); i != "" {
		if n, err := strconv.Atoi(i); err == nil {
			td.AutoReloadInterval = n
		}
	}
	if td.AutoReloadInterval < 1 {
		td.AutoReloadInterval = 5 * 60 // default reload every 5 minutes
	}

	// Detect YouTube embeds and strip query params for embedding.
	if strings.Contains(img.URL, "youtube") {
		td.IsYoutube = true
		td.EmbedURL = img.URL
		if i := strings.Index(td.EmbedURL, "?"); i > 1 {
			td.EmbedURL = td.EmbedURL[:i]
		}
	}

	if img.URL == "" && img.HDURL == "" {
		http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := tmpl.Execute(w, td); err != nil {
		log.Printf("template render error: %v", err)
	}
}

const tmplHTML = `<!DOCTYPE html>
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
src="{{.EmbedURL}}?autoplay=1&controls=0&showinfo=0&autohide=1">
</iframe>
</div>
{{else}}
<img src="{{if .SD}}{{.Image.URL}}{{else}}{{.Image.HDURL}}{{end}}" id="bg" alt="{{.Image.Title}}" />
{{end}}
</div>
<div id="apod">
<div id="explanation">
<h4>{{.Image.Title}}</h4>
<p>{{.Image.Explanation}}</p>
<p>NASA Astronomy Picture of the Day {{.Image.Date | formatDate}} <a href="{{.Image.HDURL}}" style="display:inline-block; color:#efefef"><i>Open Image in HD</i></a> </p>
</div>
<h4>{{.Image.Title}}</h4>
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
