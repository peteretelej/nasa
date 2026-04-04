// Command wallpaper periodically changes the desktop wallpaper to a random
// NASA Astronomy Picture of the Day image.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/peteretelej/nasa"
)

// Flags
var (
	random   = flag.Bool("random", true, "use random pictures, if false will only display today's APOD")
	interval = flag.Duration("interval", time.Minute*10, "interval to change wallpaper")

	cmdString  = flag.String("cmd", "", "command string to change the wallpaper")
	cmdDefault = flag.String("cmdDefault", "", "use a default command to set the wallpaper")
)

// Commands for changing wallpapers
var (
	cmds []string // actual command in use

	cmdDefaults = map[string]string{
		"gnome":   "gsettings set org.gnome.desktop.background picture-uri file://%s",
		"kde":     "dcop kdesktop KBackgroundIface setWallpaper %s 1",
		"gnome2":  "gconftool-2 --set /desktop/gnome/background/picture_filename --type=string %s",
		"xfce":    "xfconf-query -c xfce4-desktop -p /backdrop/screen0/monitor0/image-path -s %s",
		"mate":    `dconf write /org/mate/desktop/background/picture-filename "%s"`,
		"lxde":    "pcmanfm -w %s --wallpaper-mode=fit",
		"feh":     "feh --bg-scale %s",
		"setroot": "setroot %s",
	}
)

var (
	client  *nasa.Client
	tmpfile string
)

func init() {
	tmp, err := os.CreateTemp("", "nasa-wallpapers-pic.jpg")
	if err != nil {
		log.Fatalf("unable to get tempfile to work with: %v", err)
	}
	tmpfile = tmp.Name()
	if err := tmp.Close(); err != nil {
		log.Fatalf("unable to use tempfile %v", err)
	}
}

func main() {
	flag.Parse()

	// Warn if no API key is set.
	if os.Getenv("NASA_API_KEY") == "" && os.Getenv("NASAKEY") == "" {
		fmt.Fprintln(os.Stderr, "warning: no API key set (NASA_API_KEY or NASAKEY), using DEMO_KEY (limited to 30 req/hr)")
	}

	client = nasa.NewClient()

	realCmdString := getCmdString()
	if realCmdString == "" {
		log.Fatal("wallpapers change command not found, set custom one with -cmd")
	}
	cmds = strings.Split(fmt.Sprintf(realCmdString, tmpfile), " ")

	if !*random {
		if err := todaysAPOD(); err != nil {
			log.Fatalf("nasa-wallpapers: %v\n", err)
		}
		return
	}

	if err := randomAPOD(*interval); err != nil {
		log.Fatalf("nasa-wallpapers: %v\n", err)
	}
}

func todaysAPOD() error {
	ctx := context.Background()
	img, err := client.APOD.Today(ctx)
	if err != nil {
		return fmt.Errorf("fetching today's APOD: %w", err)
	}
	if img.HDURL == "" {
		return errors.New("today's APOD has no HD image URL")
	}
	return downloadAndSet(img.HDURL)
}

func randomAPOD(intv time.Duration) error {
	defer cleanUp()
	if intv < time.Second {
		return errors.New("interval set is too low")
	}

	fmt.Printf("nasa-wallpapers: resetting wallpaper to a random NASA APOD picture every %s\n", intv)
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
		time.Sleep(intv)
	}
}

func getCmdString() string {
	if *cmdString != "" {
		return *cmdString
	}
	if cmd, ok := cmdDefaults[*cmdDefault]; ok {
		return cmd
	}
	// Autodetect based on env variables.
	xdgDesktop := os.Getenv("XDG_CURRENT_DESKTOP")
	gdmDesktop := os.Getenv("GDM_DESKTOP")
	switch xdgDesktop {
	case "Unity":
		return cmdDefaults["gnome"]
	case "GNOME":
		switch gdmDesktop {
		case "gnome-shell", "gnome-classic", "gnome-fallback", "cinnamon", "gnome":
			return cmdDefaults["gnome"]
		}
	case "KDE":
		return cmdDefaults["kde"]
	case "XFCE":
		return cmdDefaults["xfce"]
	case "LXDE":
		return cmdDefaults["lxde"]
	case "X-Cinnamon":
		return cmdDefaults["gnome"]
	case "":
		switch gdmDesktop {
		case "kde-plasma":
			return cmdDefaults["kde"]
		}
	}
	return ""
}

func updateRandom() error {
	ctx := context.Background()
	imgs, err := client.APOD.Random(ctx, 1)
	if err != nil {
		return err
	}
	if len(imgs) == 0 {
		return errors.New("no images returned from NASA API")
	}
	apod := imgs[0]
	if apod.HDURL == "" {
		return errors.New("invalid response from NASA API")
	}
	return downloadAndSet(apod.HDURL)
}

func downloadAndSet(url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	cl := &http.Client{Timeout: time.Second * 40}

	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return fmt.Errorf("NASA API Response not OK: %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	defer func() { _ = resp.Body.Close() }()
	const maxDownloadSize = 50 << 20 // 50 MB
	dat, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize))
	if err != nil {
		return err
	}
	if len(dat) >= maxDownloadSize {
		return errors.New("download too large (exceeds 50 MB limit)")
	}
	if len(dat) < 512 {
		return errors.New("invalid response from APOD image url")
	}
	switch http.DetectContentType(dat[:512]) {
	case "image/jpg", "image/jpeg", "image/png", "image/gif":
	default:
		return errors.New("apod returned is not a valid image mimetype")
	}
	err = os.WriteFile(tmpfile, dat, 0644)
	if err != nil {
		return err
	}

	_, err = exec.Command(cmds[0], cmds[1:]...).Output()
	return err
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
