package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/peteretelej/nasa"
)

var (
	random   = flag.Bool("random", true, "use random pictures, if false will only display today's APOD")
	interval = flag.Duration("interval", time.Minute*10, "interval to change wallpaper")
)

func main() {
	flag.Parse()
	nasaKey := os.Getenv("NASAKEY")
	if nasaKey == "" {
		nasaKey = "DEMO_KEY"
		fmt.Println(nasa.APIKEYMissing)
	}

	if !*random {
		if err := todaysAPOD(); err != nil {
			fmt.Printf("nasa-wallpapers: %v\n", err)
			os.Exit(1)
		}
		return // this return is just for-show, displayAPOD is long running
	}

	if err := randomAPOD(*interval); err != nil {
		fmt.Printf("nasa-wallpapers: %v\n", err)
		os.Exit(1)
	}
}

func todaysAPOD() error {
	defer cleanUp()
	return errors.New("TODO")
}

func randomAPOD(interval time.Duration) error {
	defer cleanUp()
	if interval < time.Second {
		return errors.New("interval set is too low")
	}

	fmt.Printf("nasa-wallpapers: resetting wallpaper to a random NASA APOD picture every %s\n", interval)
	for {
		fails := 0
		var err error
		for i := 0; i < 3; i++ {
			err = updateRandom()
			if err == nil {
				fails = 0
				break
			}
			fails++
		}
		if fails != 0 {
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
	if len(dat) < 512 {
		return errors.New("invalid response from APOD image url")
	}
	switch http.DetectContentType(dat[:512]) {
	case "image/jpg", "image/jpeg", "image/png", "image/gif":
	default:
		return errors.New("Apod returned is not a valid image mimetype")
	}
	err = ioutil.WriteFile(tmpfile, dat, 0644)
	if err != nil {
		return err
	}
	cmdString := fmt.Sprintf("gsettings set org.gnome.desktop.background picture-uri file://%s", tmpfile)
	cmds := strings.Split(cmdString, " ")
	_, err = exec.Command(cmds[0], cmds[1:]...).Output()
	return err
}

var tmpfile string

func init() {
	tmp, err := ioutil.TempFile("", "nasa-wallpapers-pic.jpg")
	if err != nil {
		log.Fatalf("unable to get tempfile to work with: %v", err)
	}
	tmpfile = tmp.Name()
	if err := tmp.Close(); err != nil {
		log.Fatalf("unable to use tempfile %v", err)
	}
}
func cleanUp() {
	if tmpfile == "" {
		return
	}
	if _, err := os.Stat(tmpfile); err != nil {
		return
	}
	if err := os.Remove(tmpfile); err != nil {
		log.Printf("unable to clean up: %v", err)
	}
}
