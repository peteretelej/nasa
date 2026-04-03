# nasa

The comprehensive Go SDK for NASA APIs.

[![Go Reference](https://pkg.go.dev/badge/github.com/peteretelej/nasa.svg)](https://pkg.go.dev/github.com/peteretelej/nasa)
[![CI](https://github.com/peteretelej/nasa/actions/workflows/ci.yml/badge.svg)](https://github.com/peteretelej/nasa/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Covers 6 NASA APIs with typed Go responses, built-in rate limiting with multi-key rotation, a CLI for terminal usage, and an MCP server for AI assistants.

## Install

```bash
go get github.com/peteretelej/nasa
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient(nasa.WithAPIKey(os.Getenv("NASA_API_KEY")))

	apod, err := client.APOD.Today(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %s\n", apod.Title, apod.URL)
}
```

## Features

- **6 NASA APIs** - APOD, NeoWs, DONKI, NASA Image Library, SSD/CNEOS, EONET
- **Built-in rate limiting** - automatic 429 retry with header-based dynamic adjustment
- **Multi-key rotation** - distribute requests across multiple API keys
- **Typed responses** - strongly typed Go structs, no raw JSON
- **Context-aware** - supports cancellation and timeouts
- **CLI** - access all APIs from the terminal
- **MCP server** - expose NASA APIs as tools for AI assistants

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

The client automatically respects NASA API rate limits. With a registered key, the default is 1000 requests/hour. If a key is rate-limited, the client retries with the next available key.

```go
// Single key
client := nasa.NewClient(nasa.WithAPIKey(os.Getenv("NASA_API_KEY")))

// Multiple keys for automatic rotation
client = nasa.NewClient(nasa.WithAPIKeys([]string{"key1", "key2", "key3"}))
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

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

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

The MCP server exposes 13 tools covering all 6 APIs. Works with Claude Desktop, Claude Code, Cursor, and VS Code Copilot. See the [documentation](https://peteretelej.github.io/nasa) for other client configurations.

## Documentation

Full documentation at [peteretelej.github.io/nasa](https://peteretelej.github.io/nasa).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT License - see [LICENSE](LICENSE).
