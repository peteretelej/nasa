# Authentication

NASA APIs use API keys for access control. The SDK supports two authentication modes.

## DEMO_KEY (Default)

Without any configuration, the SDK uses NASA's `DEMO_KEY`. This is fine for testing but has strict limits:

- 30 requests per hour
- 50 requests per day
- Per IP address

```go
// No configuration needed - DEMO_KEY is used automatically
client := nasa.NewClient()
```

## API Key

Register for a free key at [api.nasa.gov](https://api.nasa.gov/) and configure it via environment variable or code.

**Environment variable** (recommended):

```bash
export NASA_API_KEY="your-registered-key"
```

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
	client := nasa.NewClient(
		nasa.WithAPIKey(os.Getenv("NASA_API_KEY")),
	)

	apod, err := client.APOD.Today(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(apod.Title)
}
```

A registered key allows 1,000 requests per hour.

## Which APIs Need Keys?

| API | Requires Key | Base URL |
|-----|-------------|----------|
| APOD | Yes (api.nasa.gov) | api.nasa.gov |
| NEO (NeoWs) | Yes (api.nasa.gov) | api.nasa.gov |
| DONKI | Yes (api.nasa.gov) | api.nasa.gov |
| Image Library | No | images-api.nasa.gov |
| SSD/CNEOS | No | ssd-api.jpl.nasa.gov |
| EONET | No | eonet.gsfc.nasa.gov |

The Image Library, SSD, and EONET services work without any API key configuration.
