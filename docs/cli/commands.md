<!-- Regenerate with: nasa [cmd] --help -->

# CLI Commands

All commands support `--json` for JSON output. Commands that call api.nasa.gov also support `--api-key` to override the environment variable.

## apod

Retrieve NASA's Astronomy Picture of the Day.

```bash
# Today's APOD
nasa apod

# Specific date
nasa apod --date 2024-01-01

# Random APOD
nasa apod --random

# Multiple random APODs
nasa apod --random --count 5

# Date range
nasa apod --start 2024-01-01 --end 2024-01-07

# JSON output
nasa apod --json
```

| Flag | Type | Description |
|------|------|-------------|
| `--date` | string | Specific date (YYYY-MM-DD) |
| `--random` | bool | Get random APOD(s) |
| `--count` | int | Number of random APODs (default: 1, use with --random) |
| `--start` | string | Range start date (YYYY-MM-DD) |
| `--end` | string | Range end date (YYYY-MM-DD) |
| `--json` | bool | Output as JSON |
| `--api-key` | string | API key override |

## neo

Near-Earth Object Web Service. Requires a subcommand.

### neo feed

List near-Earth objects approaching Earth within a date range (max 7 days).

```bash
# Default: next 7 days
nasa neo feed

# Custom date range
nasa neo feed --start 2024-06-01 --end 2024-06-07
```

| Flag | Type | Description |
|------|------|-------------|
| `--start` | string | Start date (YYYY-MM-DD, default: today) |
| `--end` | string | End date (YYYY-MM-DD, default: start+7d) |
| `--json` | bool | Output as JSON |
| `--api-key` | string | API key override |

### neo get

Get detailed information about a specific near-Earth object.

```bash
nasa neo get 3542519
```

| Flag | Type | Description |
|------|------|-------------|
| `--json` | bool | Output as JSON |
| `--api-key` | string | API key override |

### neo browse

Browse the full near-Earth object catalog.

```bash
nasa neo browse
nasa neo browse --page 2 --size 50
```

| Flag | Type | Description |
|------|------|-------------|
| `--page` | int | Page number (0-indexed, default: 0) |
| `--size` | int | Results per page (default: 20) |
| `--json` | bool | Output as JSON |
| `--api-key` | string | API key override |

## donki

Space Weather Database (DONKI). Requires a subcommand.

All DONKI subcommands share these flags:

| Flag | Type | Description |
|------|------|-------------|
| `--start` | string | Start date YYYY-MM-DD (default: 30 days ago) |
| `--end` | string | End date YYYY-MM-DD (default: today) |
| `--json` | bool | Output as JSON |
| `--api-key` | string | API key override |

### donki cme

```bash
nasa donki cme
nasa donki cme --start 2024-01-01 --end 2024-01-31
```

### donki cme-analysis

Additional flags:

| Flag | Type | Description |
|------|------|-------------|
| `--most-accurate` | bool | Filter to most accurate only |
| `--complete-entry` | bool | Filter to complete entries only |
| `--speed` | float64 | Minimum speed (km/s) |
| `--half-angle` | float64 | Minimum half-angle (degrees) |
| `--catalog` | string | Catalog filter |

```bash
nasa donki cme-analysis --most-accurate --speed 500
```

### donki gst

```bash
nasa donki gst
```

### donki ips

Additional flags: `--location`, `--catalog`

```bash
nasa donki ips --location Earth
```

### donki flr

```bash
nasa donki flr
```

### donki sep

```bash
nasa donki sep
```

### donki mpc

```bash
nasa donki mpc
```

### donki rbe

```bash
nasa donki rbe
```

### donki hss

```bash
nasa donki hss
```

### donki wsa-enlil

```bash
nasa donki wsa-enlil
```

### donki notifications

Additional flags:

| Flag | Type | Description |
|------|------|-------------|
| `--type` | string | Filter: FLR, SEP, CME, IPS, MPC, GST, RBE, report |

```bash
nasa donki notifications --type CME
```

## images

NASA Image and Video Library. Requires a subcommand. No API key needed.

### images search

```bash
nasa images search "apollo 11"
nasa images search "mars rover" --media-type image --year-start 2020
```

| Flag | Type | Description |
|------|------|-------------|
| `--media-type` | string | Filter: image, video, audio |
| `--year-start` | int | Results from this year or later |
| `--year-end` | int | Results up to this year |
| `--center` | string | NASA center filter |
| `--page` | int | Results page number |
| `--json` | bool | Output as JSON |

### images asset

```bash
nasa images asset as11-40-5874
```

### images metadata

```bash
nasa images metadata as11-40-5874
```

### images captions

```bash
nasa images captions "Space_Shuttle_Era"
```

## ssd

Small-Body Database (SSD/CNEOS). Requires a subcommand. No API key needed.

### ssd cad

Query close-approach data for asteroids and comets.

```bash
nasa ssd cad
nasa ssd cad --date-min 2024-06-01 --date-max 2024-06-30 --dist-max 0.05
```

| Flag | Type | Description |
|------|------|-------------|
| `--date-min` | string | Minimum date (YYYY-MM-DD) |
| `--date-max` | string | Maximum date (YYYY-MM-DD) |
| `--dist-max` | float64 | Maximum approach distance (AU) |
| `--body` | string | Close-approach body (e.g., Earth) |
| `--json` | bool | Output as JSON |

### ssd fireball

```bash
nasa ssd fireball
nasa ssd fireball --date-min 2024-01-01 --date-max 2024-06-30
```

| Flag | Type | Description |
|------|------|-------------|
| `--date-min` | string | Minimum date (YYYY-MM-DD) |
| `--date-max` | string | Maximum date (YYYY-MM-DD) |
| `--json` | bool | Output as JSON |

### ssd nhats

List near-Earth objects accessible for human exploration.

```bash
nasa ssd nhats
```

### ssd scout

List objects tracked by the Scout impact hazard assessment system.

```bash
nasa ssd scout
```

### ssd sentry

List objects monitored by the Sentry impact monitoring system.

```bash
nasa ssd sentry
```

### ssd sbdb

Look up a small body in the JPL Small-Body Database.

```bash
nasa ssd sbdb Ceres
nasa ssd sbdb "433 Eros"
```

## eonet

Earth Observatory Natural Event Tracker. Requires a subcommand.

### eonet events

```bash
nasa eonet events
nasa eonet events --status open --limit 10
nasa eonet events --category wildfires --days 30
```

| Flag | Type | Description |
|------|------|-------------|
| `--status` | string | Event status: open, closed |
| `--category` | string | Category filter |
| `--days` | int | Events within last N days |
| `--limit` | int | Maximum number of events |
| `--source` | string | Source filter |
| `--start` | string | Start date (YYYY-MM-DD) |
| `--end` | string | End date (YYYY-MM-DD) |
| `--json` | bool | Output as JSON |
| `--api-key` | string | API key override |

### eonet event

```bash
nasa eonet event EONET_6542
```

### eonet categories

```bash
nasa eonet categories
```

### eonet sources

```bash
nasa eonet sources
```
