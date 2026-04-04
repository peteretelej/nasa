# EONET - Earth Observatory Natural Event Tracker

The `EONETService` provides access to NASA's Earth Observatory Natural Event Tracker. No API key is required, though one can be provided.

> **Note:** The EONET API was returning HTTP 500 errors during April 2026 testing. Some endpoints may be intermittently unavailable.

## Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Events` | `Events(ctx context.Context, opts ...EONETOption) ([]*EONETEvent, error)` | List natural events with filtering |
| `Event` | `Event(ctx context.Context, eventID string) (*EONETEvent, error)` | Get a single event by ID |
| `Categories` | `Categories(ctx context.Context) ([]*EONETCategory, error)` | List event categories |
| `Sources` | `Sources(ctx context.Context) ([]*EONETSource, error)` | List event sources |

## Event Filter Options

```go
nasa.EONETWithStatus("open")      // "open" or "closed"
nasa.EONETWithLimit(10)           // max number of events
nasa.EONETWithDays(30)            // events within last N days
nasa.EONETWithSource("InciWeb")   // filter by source
nasa.EONETWithStart(start)        // start date
nasa.EONETWithEnd(end)            // end date
nasa.EONETWithCategory("wildfires") // filter by category
```

## Response Types

### EONETEvent

```go
type EONETEvent struct {
    ID          string
    Title       string
    Description string
    Link        string
    Closed      *time.Time       // nil if event is still open
    Categories  []*EONETCategory
    Sources     []*EONETSource
    Geometry    []*EONETGeometry
}
```

### EONETGeometry

```go
type EONETGeometry struct {
    Date           time.Time
    Type           string          // "Point" or "Polygon"
    Coordinates    json.RawMessage // raw coordinate data
    MagnitudeValue *float64
    MagnitudeUnit  string
}
```

Use `PointCoordinates()` or `PolygonCoordinates()` to parse coordinates:

```go
coords, err := geo.PointCoordinates()   // returns [2]float64{lon, lat}
poly, err := geo.PolygonCoordinates()   // returns [][][2]float64
```

### EONETCategory

```go
type EONETCategory struct {
    ID    string
    Title string
    Link  string
}
```

## Example: Fetch Active Events

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

	events, err := client.EONET.Events(
		context.Background(),
		nasa.EONETWithStatus("open"),
		nasa.EONETWithLimit(10),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Active events: %d\n\n", len(events))
	for _, e := range events {
		status := "open"
		if e.Closed != nil {
			status = "closed"
		}
		fmt.Printf("[%s] %s (%s)\n", status, e.Title, e.ID)

		for _, g := range e.Geometry {
			if g.Type == "Point" {
				coords, err := g.PointCoordinates()
				if err == nil {
					fmt.Printf("  Location: %.2f, %.2f\n", coords[1], coords[0])
				}
			}
		}
	}
}
```

## Example: List Categories

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

	categories, err := client.EONET.Categories(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, cat := range categories {
		fmt.Printf("%s: %s\n", cat.ID, cat.Title)
	}
}
```

## Rate Limiting

EONET has its own rate limit pool separate from api.nasa.gov. The SDK handles this automatically.
