# Web Server Example

A complete Go web server that serves today's APOD as HTML and provides a JSON API for near-Earth objects.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient(
		nasa.WithAPIKey(os.Getenv("NASA_API_KEY")),
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		apod, err := client.APOD.Today(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>%s</title></head><body>
<h1>%s</h1>
<p>%s</p>
<img src="%s" style="max-width:800px">
</body></html>`, apod.Title, apod.Title, apod.Explanation, apod.URL)
	})

	http.HandleFunc("/api/neo", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now()
		feed, err := client.NEO.Feed(ctx, now, now.AddDate(0, 0, 7))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(feed); err != nil {
			log.Printf("encode error: %v", err)
		}
	})

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Endpoints

| Path | Method | Description |
|------|--------|-------------|
| `/` | GET | Today's APOD rendered as HTML |
| `/api/neo` | GET | Near-Earth objects for the next 7 days (JSON) |

## Running

```bash
export NASA_API_KEY="your-key"
go run main.go
```

Open `http://localhost:8080` for the APOD page, or `http://localhost:8080/api/neo` for NEO JSON data.
