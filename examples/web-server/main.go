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
	"log"
	"os"

	"github.com/peteretelej/nasa"
	"github.com/peteretelej/nasa/internal/webdemo"
)

func main() {
	listenAddr := flag.String("listen", ":8080", "HTTP listen address")
	flag.Parse()

	apiKey := os.Getenv("NASA_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("NASAKEY")
	}
	if apiKey == "" {
		log.Println("WARNING: NASA_API_KEY not set, using DEMO_KEY (limited to 30 req/hr)")
		apiKey = "DEMO_KEY"
	}

	client := nasa.NewClient(nasa.WithAPIKey(apiKey))

	if err := webdemo.ListenAndServe(*listenAddr, client); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
