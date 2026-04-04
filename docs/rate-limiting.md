# Rate Limiting

The SDK includes a built-in rate limiter that prevents you from exceeding NASA's API quotas.

## How It Works

The rate limiter combines a token bucket with per-key cooldown tracking:

1. **Token bucket**: A `golang.org/x/time/rate` limiter paces outgoing requests to stay within the configured rate (default or custom). Every request waits for a token before executing, including retries.
2. **429 handling with key rotation**: When a request receives HTTP 429, the SDK marks that API key as limited and retries with the next available key. If you have N API keys configured, the client attempts up to N total requests (one per key) before giving up.
3. **`Retry-After` header**: If the 429 response includes a `Retry-After` header (in seconds), that value is used as the cooldown duration for the affected key. Without the header, the default cooldown is 30 minutes.
4. **When retries stop**: Once all API keys are in cooldown (or there is only one key), the SDK stops retrying and returns a `*nasa.RateLimitError` to the caller. The error's `RetryAfter` field contains the shortest remaining cooldown across all keys.

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

## Error Handling

When rate limits are exhausted, the SDK returns a `*nasa.RateLimitError`:

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
