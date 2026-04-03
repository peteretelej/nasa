# Getting Started

## Installation

```bash
go get github.com/peteretelej/nasa
```

## Your First API Call

Fetch today's Astronomy Picture of the Day:

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

	fmt.Println(apod.Title)
	fmt.Println(apod.URL)
}
```

This uses the default `DEMO_KEY` (limited to 30 requests/hour). For production use, [set up an API key](authentication.md).

## Error Handling

The SDK returns typed errors that you can match with `errors.As`:

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

	apod, err := client.APOD.Today(context.Background())
	if err != nil {
		var rateLimitErr *nasa.RateLimitError
		var authErr *nasa.AuthError
		var notFoundErr *nasa.NotFoundError

		switch {
		case errors.As(err, &rateLimitErr):
			fmt.Println("Rate limited, try again later")
		case errors.As(err, &authErr):
			fmt.Println("Invalid or missing API key")
		case errors.As(err, &notFoundErr):
			fmt.Println("Resource not found")
		default:
			log.Fatal(err)
		}
		return
	}

	fmt.Println(apod.Title)
}
```

## Using Context for Timeouts

All service methods accept a `context.Context` as their first argument, so you can set timeouts and cancellation:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apod, err := client.APOD.Today(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(apod.Title)
}
```

## Next Steps

- [Authentication](authentication.md) - configure API keys for higher rate limits
- [Rate Limiting](rate-limiting.md) - understand the built-in rate limiter
- [API Reference](apis/apod.md) - explore all available NASA APIs
