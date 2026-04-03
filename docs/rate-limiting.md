# Rate Limiting

The SDK includes a built-in rate limiter that prevents you from exceeding NASA's API quotas.

## How It Works

The rate limiter uses a token bucket algorithm (from `golang.org/x/time/rate`) that controls request throughput:

1. **Default rate**: Requests are paced to stay within NASA's limits
2. **Dynamic adjustment**: The limiter reads `X-RateLimit-Remaining` response headers and throttles automatically when the remaining quota is low
3. **429 retry**: If a request receives HTTP 429 (Too Many Requests), the SDK retries with the next available API key (when using multi-key rotation)

## Custom Rate Limit

Override the default requests-per-second rate:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient(
		nasa.WithRateLimit(5.0), // 5 requests per second
	)

	apod, err := client.APOD.Today(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(apod.Title)
}
```

## Multi-Key Rotation

Multi-key rotation multiplies your effective throughput. When one key's rate limit is exhausted, the SDK immediately fails over to the next key:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/peteretelej/nasa"
)

func main() {
	keys := strings.Split(os.Getenv("NASA_API_KEYS"), ",")

	client := nasa.NewClient(
		nasa.WithAPIKeys(keys),
		nasa.WithRateLimit(10.0),
	)

	apod, err := client.APOD.Today(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(apod.Title)
}
```

With 3 keys at 1,000 requests/hour each, you get an effective 3,000 requests/hour.

## Error Handling

When rate limits are exhausted (and no additional keys are available), the SDK returns a `*nasa.RateLimitError`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()

	_, err := client.APOD.Today(context.Background())
	if err != nil {
		var rateLimitErr *nasa.RateLimitError
		if errors.As(err, &rateLimitErr) {
			fmt.Printf("Rate limited: %s\n", rateLimitErr)
			return
		}
		log.Fatal(err)
	}
}
```

## Rate Limit Scope

Each base URL has its own rate limit pool:

| Service | Base URL | Rate Limited |
|---------|----------|-------------|
| APOD, NEO, DONKI | api.nasa.gov | Yes (shared pool) |
| Image Library | images-api.nasa.gov | Separate pool |
| SSD/CNEOS | ssd-api.jpl.nasa.gov | Separate pool |
| EONET | eonet.gsfc.nasa.gov | Separate pool |

APOD, NEO, and DONKI share the same api.nasa.gov rate limit. The other services have their own independent limits.
