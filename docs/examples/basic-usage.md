# Basic Usage Examples

Common patterns for working with the NASA Go SDK.

## Error Handling with Type Switches

```go
package main

import (
	"context"
	"errors"
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
		var rateLimitErr *nasa.RateLimitError
		var authErr *nasa.AuthError
		var notFoundErr *nasa.NotFoundError
		var apiErr *nasa.APIError

		switch {
		case errors.As(err, &rateLimitErr):
			fmt.Printf("Rate limited: %s\n", rateLimitErr)
		case errors.As(err, &authErr):
			fmt.Printf("Auth error: %s\n", authErr)
		case errors.As(err, &notFoundErr):
			fmt.Printf("Not found: %s\n", notFoundErr)
		case errors.As(err, &apiErr):
			fmt.Printf("API error: %s\n", apiErr)
		default:
			log.Fatal(err)
		}
		return
	}

	fmt.Println(apod.Title)
}
```

## Context with Timeout

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient(
		nasa.WithAPIKey(os.Getenv("NASA_API_KEY")),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apod, err := client.APOD.Today(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(apod.Title)
}
```

## Concurrent API Calls

Use goroutines to call multiple APIs in parallel:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient(
		nasa.WithAPIKey(os.Getenv("NASA_API_KEY")),
	)

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(2)

	var apod *nasa.APODImage
	var feed *nasa.NEOFeed
	var apodErr, neoErr error

	go func() {
		defer wg.Done()
		apod, apodErr = client.APOD.Today(ctx)
	}()

	go func() {
		defer wg.Done()
		now := time.Now()
		feed, neoErr = client.NEO.Feed(ctx, now, now.AddDate(0, 0, 7))
	}()

	wg.Wait()

	if apodErr != nil {
		log.Printf("APOD error: %v", apodErr)
	} else {
		fmt.Printf("APOD: %s\n", apod.Title)
	}

	if neoErr != nil {
		log.Printf("NEO error: %v", neoErr)
	} else {
		fmt.Printf("NEOs this week: %d\n", feed.ElementCount)
	}
}
```

## Legacy Convenience Functions

The package provides backward-compatible convenience functions that use a default client:

```go
package main

import (
	"fmt"
	"log"

	"github.com/peteretelej/nasa"
)

func main() {
	// Get today's APOD (uses DEMO_KEY)
	img, err := nasa.APODToday()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %s\n", img.Title, img.URL)

	// Get a random APOD
	random, err := nasa.RandomAPOD()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Random: %s\n", random.Title)
}
```

These return the legacy `*Image` type. For full features, use `NewClient` with service methods that return `*APODImage`.

## Using a Registered API Key

```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	// Convenience functions with an API key
	apod, err := nasa.APODTodayWithKey(os.Getenv("NASA_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(apod.Title)

	// NEO feed with an API key
	now := time.Now()
	feed, err := nasa.NEOFeedWithKey(
		os.Getenv("NASA_API_KEY"),
		now.Format("2006-01-02"),
		now.AddDate(0, 0, 7).Format("2006-01-02"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("NEOs: %d\n", feed.ElementCount)
}
```
