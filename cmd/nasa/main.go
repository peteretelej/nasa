package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

// subcommands and flags
var (
	apodCommand = flag.NewFlagSet("apod", flag.ExitOnError)
	apodDate    = apodCommand.String("date", "", "APOD on a particular date YYYY-MM-DD")

	neoCommand = flag.NewFlagSet("neo", flag.ExitOnError)
	neoStart   = flag.String("start", "", "NEO start date YYYY-MM-DD")
	neoEnd     = flag.String("end", "", "NEO end date YYYY-MM-DD")
)

func main() {
	flag.Parse()
	nasaKey := os.Getenv("NASAKEY")
	if nasaKey == "" {
		nasaKey = "DEMO_KEY"
		defer fmt.Println("You are using the demo API Key DEMO_KEY." +
			" Apply for an API key at https://api.nasa.gov/index.html#apply-for-an-api-key")
	}

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "apod")
	}

	switch os.Args[1] {
	case "apod":
		t := time.Now()
		if len(os.Args) > 2 {
			apodCommand.Parse(os.Args[2:])
		}
		if *apodDate != "" {
			var err error
			t, err = time.Parse("2006-01-02", *apodDate)
			if err != nil {
				errors.New("invalid -date; should use format YYYY-MM-DD")
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
			neoCommand.Parse(os.Args[2:])
		}
	}
}
