package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

// subcommands and flags
var (
	apodCommand = flag.NewFlagSet("apod", flag.ExitOnError)
	apodDate    = apodCommand.String("date", "", "APOD on a particular date YYYY-MM-DD")

	neoCommand = flag.NewFlagSet("neo", flag.ExitOnError)
	neoStart   = neoCommand.String("start", "", "NEO start date YYYY-MM-DD")
	neoEnd     = neoCommand.String("end", "", "NEO end date YYYY-MM-DD")

	webCommand = flag.NewFlagSet("web", flag.ExitOnError)
	webListen  = webCommand.String("listen", ":8080", "http web server address")
)

func init() {
	if os.Getenv("NASAKEY") == "" {
		fmt.Println(nasa.APIKEYMissing)
	}
}

func main() {
	flag.Parse()
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "apod")
	}

	switch os.Args[1] {
	case "apod":
		t := time.Now()
		if len(os.Args) > 2 {
			_ = apodCommand.Parse(os.Args[2:]) // exits on error
		}
		if *apodDate != "" {
			var err error
			t, err = time.Parse("2006-01-02", *apodDate)
			if err != nil {
				fmt.Printf("nasa apod: invalid -date, should use format YYYY-MM-DD\n")
				os.Exit(1)
				os.Exit(1)
			}
		}
		apod, err := nasa.ApodImage(t)
		if err != nil {
			fmt.Printf("unable to get apod: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(apod)
	case "neo":
		if len(os.Args) > 2 {
			_ = neoCommand.Parse(os.Args[2:]) //exits on error
		}
		start, end := *neoStart, *neoEnd
		today := time.Now().Format("2006-01-02")
		if start == "" {
			start = today
		}
		if end == "" {
			end = today
		}
		st, err := time.Parse("2006-01-02", start)
		if err != nil {
			fmt.Printf("nasa neo: invalid -start date, should be YYYY-MM-DD\n")
			os.Exit(1)
		}
		et, err := time.Parse("2006-01-02", end)
		if err != nil {
			fmt.Printf("nasa neo: invalid -end date, should be YYYY-MM-DD\n")
			os.Exit(1)
		}
		nl, err := nasa.NeoFeed(st, et)
		if err != nil {
			fmt.Printf("nasa neo: %v", err)
			os.Exit(1)
		}
		fmt.Println(nl)
	case "web":
		if len(os.Args) > 2 {
			_ = webCommand.Parse(os.Args[2:]) //exits on error
		}
		svr, err := nasa.NewServer(*webListen)
		if err != nil {
			fmt.Printf("failed to launch webserver: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("nasa web: launching http server at %s\n", svr.Addr)
		if err := svr.ListenAndServe(); err != nil {
			log.Fatalf("server crashed: %v", err)
		}
	}
}
