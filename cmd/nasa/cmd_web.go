package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/peteretelej/nasa"
	"github.com/peteretelej/nasa/internal/webdemo"
)

func webCmd(args []string) {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa web [flags]

Launch a local web server that displays NASA's Astronomy Picture of the Day
in a full-screen HTML view with caching, auto-reload, and random mode.

See also: examples/web-server/ for a standalone example.

Flags:
`)
		fs.PrintDefaults()
	}

	listenFlag := fs.String("listen", ":8080", "HTTP listen address")
	apiKeyFlag := fs.String("api-key", "", "API key override")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	key := getAPIKey(*apiKeyFlag)
	client := nasa.NewClient(nasa.WithAPIKey(key))

	if err := webdemo.ListenAndServe(*listenFlag, client); err != nil {
		fatal("web: %v", err)
	}
}
