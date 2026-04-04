# APOD - Astronomy Picture of the Day

The `APODService` provides access to NASA's Astronomy Picture of the Day. Requires an API key (uses `DEMO_KEY` by default).

## Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Today` | `Today(ctx context.Context) (*APODImage, error)` | Get today's APOD |
| `Get` | `Get(ctx context.Context, date time.Time) (*APODImage, error)` | Get APOD for a specific date |
| `GetRange` | `GetRange(ctx context.Context, start, end time.Time) ([]*APODImage, error)` | Get APODs for a date range |
| `Random` | `Random(ctx context.Context, count int) ([]*APODImage, error)` | Get random APODs |

## Response Type

```go
type APODImage struct {
    Date           time.Time
    Title          string
    Explanation    string
    URL            string
    HDURL          string
    MediaType      MediaType // "image" or "video"
    Copyright      string
    ThumbnailURL   string
    ServiceVersion string
}
```

## Examples

### Get Today's APOD

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

	fmt.Printf("%s (%s)\n", apod.Title, apod.Date.Format("2006-01-02"))
	fmt.Println(apod.URL)
}
```

### Get APOD by Date

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

	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	apod, err := client.APOD.Get(context.Background(), date)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n%s\n", apod.Title, apod.Explanation)
}
```

### Get Random APODs

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

	images, err := client.APOD.Random(context.Background(), 3)
	if err != nil {
		log.Fatal(err)
	}

	for _, img := range images {
		fmt.Printf("%s (%s)\n", img.Title, img.Date.Format("2006-01-02"))
	}
}
```
