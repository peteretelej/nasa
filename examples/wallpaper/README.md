# NASA Wallpaper Changer

Periodically changes your desktop wallpaper to a random NASA Astronomy Picture
of the Day (APOD) image.

## Build

```bash
go build -o nasa-wallpapers ./examples/wallpaper/
```

## Usage

```bash
# Set API key (optional, falls back to DEMO_KEY with 30 req/hr limit)
export NASA_API_KEY=your_key_here

# Random wallpaper every 10 minutes (default)
./nasa-wallpapers

# Today's APOD only (no rotation)
./nasa-wallpapers -random=false

# Custom interval
./nasa-wallpapers -interval=30m

# Specify desktop environment command
./nasa-wallpapers -cmdDefault=gnome
./nasa-wallpapers -cmdDefault=kde
./nasa-wallpapers -cmdDefault=xfce

# Custom wallpaper-setting command (%s is replaced with the image path)
./nasa-wallpapers -cmd="feh --bg-scale %s"
```

## Supported Desktop Environments

The following `-cmdDefault` values are available:

| Value     | Command |
|-----------|---------|
| `gnome`   | gsettings (GNOME 3+, Unity, Cinnamon) |
| `gnome2`  | gconftool-2 (GNOME 2) |
| `kde`     | dcop (KDE) |
| `xfce`    | xfconf-query (XFCE) |
| `mate`    | dconf (MATE) |
| `lxde`    | pcmanfm (LXDE) |
| `feh`     | feh |
| `setroot` | setroot |

If no `-cmdDefault` or `-cmd` is provided, the program attempts to auto-detect
the desktop environment from `XDG_CURRENT_DESKTOP` and `GDM_DESKTOP` environment
variables.

## Systemd Service

A sample systemd unit file is included at `nasa-wallpapers.service`. Edit
`ExecStart` and `Environment` to match your setup, then:

```bash
cp ./examples/wallpaper/nasa-wallpapers.service ~/.config/systemd/user/
systemctl --user enable nasa-wallpapers
systemctl --user start nasa-wallpapers
```
