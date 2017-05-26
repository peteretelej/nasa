package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

// nasa-wallpapers Flags
var (
	random   = flag.Bool("random", true, "use random pictures, if false will only display today's APOD")
	interval = flag.Duration("interval", time.Minute*10, "interval to change wallpaper")

	cmdString  = flag.String("cmd", "", "command string to change the wallpaper")
	cmdDefault = flag.String("cmdDefault", "", "use a default command to set the wallpaper")
)

func init() {
	if os.Getenv("NASAKEY") == "" {
		fmt.Println(nasa.APIKEYMissing)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	nasa.CmdString = *cmdString
	nasa.CmdDefault = *cmdDefault
	if !*random {
		if err := todaysAPOD(); err != nil {
			log.Fatalf("nasa-wallpapers: %v\n", err)
		}
	}

	if err := randomAPOD(*interval); err != nil {
		log.Fatalf("nasa-wallpapers: %v\n", err)
	}
}

func todaysAPOD() error {
	nasa.InitWallpapers()
	defer nasa.CleanUpWallpapers()
	// TODO: display today's APOD as wallpaper
	return errors.New("not implemented")
}

func randomAPOD(interval time.Duration) error {
	nasa.InitWallpapers()
	defer nasa.CleanUpWallpapers()

	if interval < time.Second {
		return errors.New("interval set is too low")
	}

	fmt.Printf("nasa-wallpapers: resetting wallpaper to a random NASA APOD picture every %s\n", interval)
	for {
		var err error
		for i := 0; i < 3; i++ {
			err = updateRandom()
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Printf("nasa-wallpapers: unable to fetch wallpapers: %v", err)
		}
		time.Sleep(interval)
	}
}

func updateRandom() error {
	apod, err := nasa.RandomAPOD()
	if err != nil {
		return err
	}
	if apod.HDURL == "" {
		return errors.New("invalid response from NASA API")
	}
	req, err := http.NewRequest("GET", apod.HDURL, nil)
	if err != nil {
		return err
	}
	cl := &http.Client{Timeout: time.Second * 40}

	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NASA API Response not OK: %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	defer func() { _ = resp.Body.Close() }()
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nasa.UpdateWallpaper(dat)
}
