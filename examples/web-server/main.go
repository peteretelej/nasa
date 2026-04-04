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
	IsVideo            bool
	IsGIF              bool
	EmbedURL           string // video embed URL with query params stripped
	AutoReloadInterval int
	Stale              bool // true when serving stale cached data after fetch error
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
	mux.HandleFunc("/sw.js", handleServiceWorker)

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
	stale := false
	if img == nil || time.Since(updatedAt) > cacheRefreshInterval {
		apod, err := client.APOD.Today(r.Context())
		if err != nil {
			log.Printf("APOD fetch error: %v", err)
			if img != nil {
				// Serve stale cached data rather than an error page.
				stale = true
			} else {
				renderError(w, r)
				return
			}
		} else {
			todayCache.set(apod)
			img = apod
		}
	}

	renderTemplate(w, img, r, stale)
}

func handleRandom(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/random-apod/" {
		http.NotFound(w, r)
		return
	}

	img, updatedAt := randomCache.get()
	stale := false
	if img == nil || time.Since(updatedAt) > cacheRefreshInterval {
		imgs, err := client.APOD.Random(r.Context(), 1)
		if err != nil || len(imgs) == 0 {
			if err != nil {
				log.Printf("APOD random fetch error: %v", err)
			}
			if img != nil {
				stale = true
			} else {
				renderError(w, r)
				return
			}
		} else if imgs[0].URL != "" {
			randomCache.set(imgs[0])
			img = imgs[0]
		} else if img == nil {
			renderError(w, r)
			return
		}
	}

	renderTemplate(w, img, r, stale)
}

func handleServiceWorker(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, swJS)
}

func renderError(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Retry-After", "30")
	w.WriteHeader(http.StatusServiceUnavailable)
	fmt.Fprint(w, errorHTML)
}

func renderTemplate(w http.ResponseWriter, img *nasa.APODImage, r *http.Request, stale bool) {
	td := templateData{
		Title:      img.Title,
		Image:      img,
		SD:         r.URL.Query().Get("sd") != "",
		AutoReload: r.URL.Query().Get("auto") != "" || r.URL.Query().Get("interval") != "",
		Legacy:     r.URL.Query().Get("legacy") != "",
		Stale:      stale,
	}

	if i := r.URL.Query().Get("interval"); i != "" {
		if n, err := strconv.Atoi(i); err == nil {
			td.AutoReloadInterval = n
		}
	}
	if td.AutoReloadInterval < 1 {
		td.AutoReloadInterval = 5 * 60 // default reload every 5 minutes
	}

	// Detect video APODs using the MediaType field.
	if img.MediaType == nasa.MediaVideo {
		td.IsVideo = true
		td.EmbedURL = img.URL
		if i := strings.Index(td.EmbedURL, "?"); i > 1 {
			td.EmbedURL = td.EmbedURL[:i]
		}
	}

	// Detect GIF images to avoid loading huge HD GIFs.
	if !td.IsVideo && strings.HasSuffix(strings.ToLower(img.URL), ".gif") {
		td.IsGIF = true
	}

	if img.URL == "" && img.HDURL == "" {
		renderError(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, td); err != nil {
		log.Printf("template render error: %v", err)
	}
}

const tmplHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{with .Title}}{{.}}{{else}}NASA Astronomy Picture of the Day{{end}}</title>
<meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<meta name="description" content="{{with .Image}}{{.Title}} - {{end}}NASA Astronomy Picture of the Day">
<meta name="theme-color" content="#000">
{{- if .AutoReload}}{{if not .Legacy}}
<meta http-equiv="refresh" content="{{.AutoReloadInterval}}">
{{- end}}{{end}}
<style>
*,*::before,*::after{margin:0;padding:0;box-sizing:border-box}
:root{
  --bg:#000;
  --text:rgba(255,255,255,0.92);
  --text-muted:rgba(255,255,255,0.6);
  --overlay:rgba(0,0,0,0.55);
  --link:rgba(255,255,255,0.85);
}

html,body{height:100%;overflow:hidden;background:var(--bg);color:var(--text);font-family:system-ui,-apple-system,sans-serif}

/* Full-viewport media container */
.media{position:fixed;inset:0;z-index:0}
.media img{
  display:block;width:100%;height:100%;
  object-fit:contain;
  opacity:0;transition:opacity 0.4s ease-out;
}
.media img.loaded{opacity:1}
.media iframe{border:0;width:100%;height:100%}

/* Info overlay - bottom of screen */
.info{
  position:fixed;bottom:0;left:0;right:0;z-index:10;
  padding:16px 20px calc(16px + env(safe-area-inset-bottom,0px));
  background:linear-gradient(transparent,var(--overlay) 30%);
  transform:translateY(calc(100% - 64px - env(safe-area-inset-bottom,0px)));
  transition:transform 0.3s cubic-bezier(0.16,1,0.3,1);
}
.info:hover,.info:focus-within,.info.open{
  transform:translateY(0);
}
.info-title{
  font-size:15px;font-weight:600;letter-spacing:-0.01em;
  white-space:nowrap;overflow:hidden;text-overflow:ellipsis;
  cursor:pointer;
  padding-bottom:4px;
}
.info-body{
  max-height:40vh;overflow-y:auto;
  -webkit-overflow-scrolling:touch;
  padding-top:8px;
  opacity:0;transition:opacity 0.25s ease;
}
.info:hover .info-body,.info:focus-within .info-body,.info.open .info-body{opacity:1}
.info-body p{font-size:13px;line-height:1.5;color:var(--text-muted);margin-bottom:8px}
.info-body a{color:var(--link);text-decoration:underline;text-underline-offset:2px}
.info-meta{
  font-size:12px;color:var(--text-muted);
  display:flex;flex-wrap:wrap;gap:8px 16px;align-items:center;
  padding-top:8px;border-top:1px solid rgba(255,255,255,0.1);
}
.info-meta a{color:var(--link);text-decoration:none}
.info-meta a:hover{text-decoration:underline}

/* Stale data indicator */
.stale-badge{
  display:inline-block;font-size:11px;
  padding:2px 6px;border-radius:3px;
  background:rgba(255,180,0,0.2);color:rgba(255,200,80,0.9);
}

/* Loading state */
.loading{
  position:fixed;inset:0;z-index:1;
  display:flex;align-items:center;justify-content:center;
}
.loading::after{
  content:"";width:28px;height:28px;
  border:2px solid rgba(255,255,255,0.15);
  border-top-color:rgba(255,255,255,0.7);
  border-radius:50%;
  animation:spin 0.8s linear infinite;
}
@keyframes spin{to{transform:rotate(360deg)}}

/* Reduced motion */
@media(prefers-reduced-motion:reduce){
  *,*::before,*::after{
    animation-duration:0.01ms!important;
    transition-duration:0.01ms!important;
  }
  .media img{opacity:1}
}

/* Mobile: make overlay tappable */
@media(max-width:600px){
  .info{padding:12px 16px calc(12px + env(safe-area-inset-bottom,0px));transform:translateY(calc(100% - 56px - env(safe-area-inset-bottom,0px)))}
  .info-title{font-size:14px}
  .info-body p{font-size:12px}
}
</style>
</head>
{{- if and .AutoReload .Legacy}}
<script>
function TimedRefresh(t){setTimeout(function(){location.reload(true)},t)}
</script>
{{- end}}
<body{{if and .AutoReload .Legacy}} onload="TimedRefresh({{.AutoReloadInterval}}*1000)"{{end}}>

<div class="media">
{{- if .IsVideo}}
  <iframe src="{{.EmbedURL}}?autoplay=1&mute=1&controls=0&showinfo=0&rel=0&modestbranding=1&loop=1" allow="autoplay;encrypted-media" allowfullscreen loading="lazy"></iframe>
{{- else}}
  <div class="loading" id="loader"></div>
  <img
    src="{{if .SD}}{{.Image.URL}}{{else if .IsGIF}}{{.Image.URL}}{{else if .Image.HDURL}}{{.Image.HDURL}}{{else}}{{.Image.URL}}{{end}}"
    alt="{{.Image.Title}}"
    id="bg"
    onload="this.classList.add('loaded');var l=document.getElementById('loader');if(l)l.remove()"
    onerror="this.onerror=null;{{if and (not .SD) .Image.URL .Image.HDURL}}this.src='{{.Image.URL}}';{{end}}var l=document.getElementById('loader');if(l)l.remove();this.classList.add('loaded')"
  />
{{- end}}
</div>

<div class="info" tabindex="0" role="complementary" aria-label="Image details">
  <div class="info-title" onclick="this.parentElement.classList.toggle('open')">
    {{.Image.Title}}
  </div>
  <div class="info-body">
    <p>{{.Image.Explanation}}</p>
    <div class="info-meta">
      <span>{{.Image.Date | formatDate}}</span>
      {{- if .Stale}}<span class="stale-badge">cached</span>{{end}}
      {{- if .Image.Copyright}}<span>{{.Image.Copyright}}</span>{{end}}
      {{- if .Image.HDURL}}<a href="{{.Image.HDURL}}" target="_blank" rel="noopener">Open HD</a>{{end}}
      <a href="/random-apod/?auto=1&interval=60">Random (1m)</a>
      <a href="/random-apod/?auto=1&interval=600">Random (10m)</a>
      <a href="https://github.com/peteretelej/nasa" target="_blank" rel="noopener">&#9733; Star on GitHub</a>
    </div>
  </div>
</div>

<script>
if('serviceWorker' in navigator){navigator.serviceWorker.register('/sw.js').catch(function(){})}
</script>
</body>
</html>`

// errorHTML is a self-contained error page shown when no cached data is available.
const errorHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>NASA APOD - Temporarily Unavailable</title>
<meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<meta http-equiv="refresh" content="30">
<style>
*{margin:0;padding:0;box-sizing:border-box}
html,body{height:100%;background:#000;color:rgba(255,255,255,0.85);font-family:system-ui,-apple-system,sans-serif}
body{display:flex;align-items:center;justify-content:center;text-align:center;padding:20px}
h1{font-size:18px;font-weight:600;margin-bottom:8px}
p{font-size:14px;color:rgba(255,255,255,0.5);line-height:1.5}
</style>
</head>
<body>
<div>
<h1>Temporarily unavailable</h1>
<p>Could not load today's image from NASA. This page will retry automatically.</p>
</div>
</body>
</html>`

// swJS is a lightweight service worker that caches image and page responses.
const swJS = `var CACHE='nasa-apod-v1';
self.addEventListener('install',function(e){self.skipWaiting()});
self.addEventListener('activate',function(e){
  e.waitUntil(caches.keys().then(function(names){
    return Promise.all(names.filter(function(n){return n!==CACHE}).map(function(n){return caches.delete(n)}))
  }).then(function(){return self.clients.claim()}))
});
self.addEventListener('fetch',function(e){
  var req=e.request;
  if(req.method!=='GET')return;
  var url=new URL(req.url);
  var isImage=/\.(jpg|jpeg|png|gif|webp)(\?.*)?$/i.test(url.pathname);
  var isNASA=url.hostname.indexOf('nasa.gov')!==-1;
  var isSelf=url.origin===self.location.origin;
  if(!isImage&&!isNASA&&!isSelf)return;
  e.respondWith(
    fetch(req).then(function(resp){
      if(resp&&resp.status===200){
        var clone=resp.clone();
        caches.open(CACHE).then(function(c){c.put(req,clone)});
      }
      return resp;
    }).catch(function(){return caches.match(req)})
  );
});`
