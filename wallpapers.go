package nasa

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/disintegration/imaging"
)

// CmdString is as speficied by -cmd flag
var CmdString string

// CmdDefault is as speficied by -cmdDefault flag
var CmdDefault string

var tmpfile string

// InitWallpapers sets up global values for use by nasa-wallpapers
func InitWallpapers() {
	tmp, err := ioutil.TempFile("", "nasawallpaperspic")
	if err != nil {
		log.Fatalf("unable to get tempfile to work with: %v", err)
	}
	tmpfile = tmp.Name()
	if err := tmp.Close(); err != nil {
		log.Fatalf("unable to use tempfile %v", err)
	}

	switch runtime.GOOS {
	case "windows":
	default:
		realCmdString := getCmdString()
		if realCmdString == "" {
			log.Fatal("wallpapers change command not found, set custom one with -cmd")
		}
		cmds = strings.Split(fmt.Sprintf(realCmdString, tmpfile), " ")
	}

}

// CleanUpWallpapers runs tears down nasa-wallpapers global values
func CleanUpWallpapers() {
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

// UpdateWallpaper updates the wallpaper
func UpdateWallpaper(dat []byte) error {
	if !IsImage(dat) {
		return fmt.Errorf("APOD is not an image")
	}
	if err := updateWallpaperFile(dat); err != nil {
		return fmt.Errorf("unable to update wallpaper file: %v", err)
	}
	if runtime.GOOS == "windows" {
		return WindowsChangeWallpaper(tmpfile)
	}
	return LinuxChangeWallpaper(tmpfile)
}

// IsImage checks if the byte slice is a valid image
func IsImage(dat []byte) bool {
	if len(dat) < 512 {
		return false
	}
	switch http.DetectContentType(dat[:512]) {
	case "image/jpg", "image/jpeg", "image/png", "image/gif":
		return true
	}
	return false
}

func updateWallpaperFile(dat []byte) error {
	if runtime.GOOS != "windows" {
		return ioutil.WriteFile(tmpfile, dat, 0645)
	}
	img, err := imaging.Decode(bytes.NewReader(dat))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	wr := bufio.NewWriter(&buf)
	if err := imaging.Encode(wr, img, imaging.BMP); err != nil {
		return fmt.Errorf("unable to convert apod image: %v", err)
	}

	return ioutil.WriteFile(tmpfile, buf.Bytes(), 0644)
}

// WindowsChangeWallpaper changes wallpaper for Windows OS
func WindowsChangeWallpaper(filename string) error {
	if _, err := os.Stat(filename); err != nil {
		return errors.New("file for new wallpaper does not exist")
	}
	winCmds := [][]string{
		{`reg`, `add`, `"HKCU\control panel\desktop"`, `/v`, `wallpaper`, `/t`, `REG_SZ`, `/d`, `""`, `/f`},
		{`reg`, `add`, `"HKCU\control panel\desktop"`, `/v`, `wallpaper`, `/t`, `REG_SZ`, `/d`, filename, `/f`},
		{`reg`, `delete`, `"HKCU\Software\Microsoft\Internet Explorer\Desktop\General"`, `/v`, `WallpaperStyle`, `/f`},
		{`reg`, `add`, `"HKCU\control panel\desktop"`, `/v`, `WallpaperStyle`, `/t`, `REG_SZ`, `/d`, `2`, `/f`},
		{`RUNDLL32.EXE`, `user32.dll,UpdatePerUserSystemParameters`},
	}
	for _, cmds := range winCmds {
		out, err := exec.Command(cmds[0], cmds[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("unable to change windows background: %v %s", err, out)
		}
	}
	return nil
}

// linux wallpaper commands
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

func getCmdString() string {
	if CmdString != "" {
		return CmdString
	}
	if cmd, ok := cmdDefaults[CmdDefault]; ok {
		return cmd
	}
	// in case no -cmdDefaults is defined, autodetect based on env variables
	// Based on:
	// https://askubuntu.com/questions/72549/how-to-determine-which-window-manager-is-running
	// More cases on other distributions are welcome
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

// LinuxChangeWallpaper changes
func LinuxChangeWallpaper(filename string) error {
	if _, err := os.Stat(filename); err != nil {
		return errors.New("file for new wallpaper does not exist")
	}
	_, err := exec.Command(cmds[0], cmds[1:]...).Output()
	return err
}
