# nasa

A Go client library for NASA's open APIs, covering APOD, NeoWs, DONKI, the Image Library, SSD/CNEOS, and EONET.

[![Go Reference](https://pkg.go.dev/badge/github.com/peteretelej/nasa.svg)](https://pkg.go.dev/github.com/peteretelej/nasa)
[![CI](https://github.com/peteretelej/nasa/actions/workflows/ci.yml/badge.svg)](https://github.com/peteretelej/nasa/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/peteretelej/nasa)](https://goreportcard.com/report/github.com/peteretelej/nasa)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Docs](https://img.shields.io/badge/docs-peteretelej.github.io%2Fnasa-blue)](https://peteretelej.github.io/nasa)
[![Demo](https://img.shields.io/badge/demo-nasa.etelej.com-orange)](https://nasa.etelej.com)

Query NASA's public data APIs from Go - astronomy photos, near-Earth objects, space weather events, satellite imagery, and more. Responses are returned as typed Go structs with rate limiting handled automatically. A CLI and an MCP server are also included for terminal and AI assistant usage.

- **6 NASA APIs** - APOD, NeoWs, DONKI, NASA Image Library, SSD/CNEOS, EONET
- **Built-in rate limiting** - automatic 429 retry with header-based dynamic adjustment
- **Typed responses** - strongly typed Go structs, no raw JSON
- **Context-aware** - supports cancellation and timeouts
- **CLI** - access all APIs from the terminal
- **MCP server** - expose NASA APIs as tools for AI assistants

## Install

```bash
go get github.com/peteretelej/nasa
```

## API Key

Get a free API key at [api.nasa.gov](https://api.nasa.gov), then export it:

```bash
export NASA_API_KEY=your_key_here
```

Or specify it directly when creating the client:

```go
client := nasa.NewClient(nasa.WithAPIKey("your_key_here"))
```

The SDK works without a key using NASA's DEMO_KEY, but it is heavily rate-limited.

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()

	apod, err := client.APOD.Today(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %s\n", apod.Title, apod.URL)
}
```

## API Coverage

| API | Description | Service |
|-----|-------------|---------|
| APOD | Astronomy Picture of the Day | `client.APOD` |
| NeoWs | Near-Earth Object Web Service | `client.NEO` |
| DONKI | Space Weather Database and Notification System | `client.DONKI` |
| NASA Image Library | Image and video search across NASA media | `client.Images` |
| SSD/CNEOS | Close approaches, fireballs, small bodies | `client.SSD` |
| EONET | Earth Observatory Natural Event Tracker | `client.EONET` |

## Rate Limiting

The client automatically respects NASA API rate limits. With a registered key, the default is 1000 requests/hour.

```go
client := nasa.NewClient()
```

## CLI

```bash
go install github.com/peteretelej/nasa/cmd/nasa@latest
```

```bash
nasa apod                          # today's Astronomy Picture of the Day
nasa apod -date 2024-01-01        # APOD for a specific date
nasa neo feed                      # near-Earth objects for today
nasa donki cme                     # coronal mass ejections
nasa images search "apollo 11"     # search NASA image library
nasa ssd cad                       # close approach data
```

## MCP Server

```bash
go install github.com/peteretelej/nasa/cmd/nasa-mcp@latest
```

Add to your MCP client config:

```json
{
  "mcpServers": {
    "nasa": {
      "command": "nasa-mcp",
      "env": { "NASA_API_KEY": "your-key" }
    }
  }
}
```

The MCP server exposes 13 tools covering all 6 APIs. See the [documentation](https://peteretelej.github.io/nasa) for details.

## Documentation

Full documentation at [peteretelej.github.io/nasa](https://peteretelej.github.io/nasa).

## Examples

Runnable programs covering every API are in [`examples/`](examples/). Each is a single `main.go` with no external dependencies beyond the SDK.

### Daily Briefing

Combines all six APIs into one summary - APOD, near-Earth objects, space weather, and active Earth events.

```bash
go run ./examples/daily-briefing
```

```
=== Today's APOD ===
  Title: Hello World
  URL:   https://apod.nasa.gov/apod/image/2604/art002e000192_1050.jpg

=== Near-Earth Objects This Week ===
  105 objects tracked, 1 potentially hazardous

=== Solar Activity (7 days) ===
  Coronal mass ejections: 43
  Solar flares: 13

=== Active Earth Events ===
  [Wildfires] rx-12_ 13_ 14 Prescribed Fire, Burnett, Wisconsin
  [Wildfires] Rx Houston 3041 Prescribed Fire, Houston, Texas
  [Severe Storms] Tropical Cyclone Indusa
  [Wildfires] RX PCS Beauchamp North Prescribed Fire, Scott, Arkansas
  [Wildfires] Julie Pond Wildfire, Dorchester, Maryland
```

### APOD Gallery

Fetches today's Astronomy Picture of the Day plus a recent gallery.

```bash
go run ./examples/apod-gallery
```

### NEO Watch

Tracks near-Earth objects for the current week, with details on close approaches and potentially hazardous asteroids.

```bash
go run ./examples/neo-watch
```

### Space Weather

Queries DONKI for recent coronal mass ejections, solar flares, geomagnetic storms, and other space weather events.

```bash
go run ./examples/space-weather
```

### Image Search

Searches the NASA Image and Video Library for media, with metadata and download links.

```bash
go run ./examples/image-search
```

### Close Approaches

Queries SSD/CNEOS for recent fireballs, upcoming close approaches, and small body lookups.

```bash
go run ./examples/close-approaches
```

### Earth Events

Lists active natural events from EONET - wildfires, storms, volcanic activity, and more.

```bash
go run ./examples/earth-events
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
