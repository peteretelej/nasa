# NEO - Near-Earth Objects

The `NEOService` provides access to NASA's Near-Earth Object Web Service (NeoWs). Requires an API key (uses `DEMO_KEY` by default).

## Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Feed` | `Feed(ctx context.Context, start, end time.Time) (*NEOFeed, error)` | List NEOs approaching Earth in a date range (max 7 days) |
| `Get` | `Get(ctx context.Context, asteroidID string) (*NearEarthObject, error)` | Get details for a specific asteroid |
| `Browse` | `Browse(ctx context.Context, page, size int) (*NEOBrowse, error)` | Browse the full NEO catalog (paginated) |

## Response Types

### NearEarthObject

```go
type NearEarthObject struct {
    ID                     string
    NeoReferenceID         string
    Name                   string
    JPLURL                 string
    AbsoluteMagnitude      float64
    EstimatedDiameter      EstimatedDiameter
    IsPotentiallyHazardous bool
    CloseApproachData      []CloseApproach
    OrbitalData            *OrbitalData
    IsSentryObject         bool
}
```

### NEOFeed

```go
type NEOFeed struct {
    Links            PaginationLinks
    ElementCount     int
    NearEarthObjects map[string][]*NearEarthObject // keyed by date string
}
```

### NEOBrowse

```go
type NEOBrowse struct {
    Links            PaginationLinks
    Page             PageInfo
    NearEarthObjects []*NearEarthObject
}
```

### CloseApproach

```go
type CloseApproach struct {
    CloseApproachDate      string
    CloseApproachDateFull  time.Time
    EpochDateCloseApproach MillisecondEpoch
    RelativeVelocity       VelocityData
    MissDistance            DistanceData
    OrbitingBody            string
}
```

## Examples

### Iterate Near-Earth Objects from Feed

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient(
		nasa.WithAPIKey(os.Getenv("NASA_API_KEY")),
	)

	start := time.Now()
	end := start.AddDate(0, 0, 7)

	feed, err := client.NEO.Feed(context.Background(), start, end)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total objects: %d\n\n", feed.ElementCount)

	// Sort dates for consistent output
	dates := make([]string, 0, len(feed.NearEarthObjects))
	for date := range feed.NearEarthObjects {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	for _, date := range dates {
		neos := feed.NearEarthObjects[date]
		fmt.Printf("%s (%d objects)\n", date, len(neos))
		for _, neo := range neos {
			hazardous := ""
			if neo.IsPotentiallyHazardous {
				hazardous = " [HAZARDOUS]"
			}
			fmt.Printf("  - %s%s\n", neo.Name, hazardous)
		}
	}
}
```

### Get Asteroid Details

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

	neo, err := client.NEO.Get(context.Background(), "3542519")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\n", neo.Name)
	fmt.Printf("Hazardous: %v\n", neo.IsPotentiallyHazardous)
	fmt.Printf("Diameter: %.2f - %.2f km\n",
		neo.EstimatedDiameter.Kilometers.Min,
		neo.EstimatedDiameter.Kilometers.Max,
	)

	for _, ca := range neo.CloseApproachData {
		fmt.Printf("Close approach: %s (%.2f km)\n",
			ca.CloseApproachDate,
			float64(ca.MissDistance.Kilometers),
		)
	}
}
```

### Browse the NEO Catalog

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

	result, err := client.NEO.Browse(context.Background(), 0, 10)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Page %d of %d (total: %d)\n\n",
		result.Page.Number, result.Page.TotalPages, result.Page.TotalElements,
	)

	for _, neo := range result.NearEarthObjects {
		fmt.Printf("%s (magnitude: %.1f)\n", neo.Name, neo.AbsoluteMagnitude)
	}
}
```

> **Note:** The legacy `Asteroid` type in `neo.go` exists for backward compatibility. New code should use `NearEarthObject` and the `NEOService` methods.
