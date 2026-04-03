# DONKI - Space Weather Database

The `DONKIService` provides access to NASA's Space Weather Database of Notifications, Knowledge, Information (DONKI). Requires an API key (uses `DEMO_KEY` by default).

## Methods

All methods accept a date range (`start`, `end` as `time.Time`):

| Method | Returns | Description |
|--------|---------|-------------|
| `CME` | `[]*CMEEvent` | Coronal Mass Ejections |
| `CMEAnalysis` | `[]*CMEAnalysis` | CME analysis data (with optional filters) |
| `GST` | `[]*GSTEvent` | Geomagnetic Storms |
| `IPS` | `[]*IPSEvent` | Interplanetary Shocks (with optional filters) |
| `FLR` | `[]*FLREvent` | Solar Flares |
| `SEP` | `[]*SEPEvent` | Solar Energetic Particles |
| `MPC` | `[]*MPCEvent` | Magnetopause Crossings |
| `RBE` | `[]*RBEEvent` | Radiation Belt Enhancements |
| `HSS` | `[]*HSSEvent` | High Speed Streams |
| `WSAEnlil` | `[]*WSAEnlilEvent` | WSA-Enlil Simulations |
| `Notifications` | `[]*Notification` | DONKI notifications (with type filter) |

## Signatures

```go
func (s *DONKIService) CME(ctx context.Context, start, end time.Time) ([]*CMEEvent, error)
func (s *DONKIService) CMEAnalysis(ctx context.Context, start, end time.Time, opts ...CMEAnalysisOption) ([]*CMEAnalysis, error)
func (s *DONKIService) GST(ctx context.Context, start, end time.Time) ([]*GSTEvent, error)
func (s *DONKIService) IPS(ctx context.Context, start, end time.Time, opts ...IPSOption) ([]*IPSEvent, error)
func (s *DONKIService) FLR(ctx context.Context, start, end time.Time) ([]*FLREvent, error)
func (s *DONKIService) SEP(ctx context.Context, start, end time.Time) ([]*SEPEvent, error)
func (s *DONKIService) MPC(ctx context.Context, start, end time.Time) ([]*MPCEvent, error)
func (s *DONKIService) RBE(ctx context.Context, start, end time.Time) ([]*RBEEvent, error)
func (s *DONKIService) HSS(ctx context.Context, start, end time.Time) ([]*HSSEvent, error)
func (s *DONKIService) WSAEnlil(ctx context.Context, start, end time.Time) ([]*WSAEnlilEvent, error)
func (s *DONKIService) Notifications(ctx context.Context, start, end time.Time, notificationType string) ([]*Notification, error)
```

## CME Analysis Options

```go
nasa.WithMostAccurateOnly(true)   // filter to most accurate entries
nasa.WithCompleteEntryOnly(true)  // filter to complete entries
nasa.WithSpeed(min, max)          // filter by speed (km/s)
nasa.WithHalfAngle(min, max)      // filter by half-angle (degrees)
nasa.WithCatalog("M2M_CATALOG")   // filter by catalog
```

## IPS Options

```go
nasa.WithLocation("Earth")                  // location filter
nasa.WithIPSCatalog("WINSLOW_MESSENGER_ICME_CATALOG") // catalog filter
```

## Key Types

### DONKITime

All DONKI timestamps use `DONKITime`, which embeds `time.Time` and handles NASA's date format automatically.

### CMEEvent

```go
type CMEEvent struct {
    ActivityID      string
    Catalog         string
    StartTime       DONKITime
    SourceLocation  string
    ActiveRegionNum int
    Link            string
    Note            string
    Instruments     []CMEInstrument
    CMEAnalyses     []CMEAnalysisData
    LinkedEvents    []LinkedEvent
}
```

## Example: Fetch CMEs

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

	end := time.Now()
	start := end.AddDate(0, 0, -30)

	events, err := client.DONKI.CME(context.Background(), start, end)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("CME events in last 30 days: %d\n\n", len(events))
	for _, e := range events {
		fmt.Printf("ID: %s\n", e.ActivityID)
		fmt.Printf("Start: %s\n", e.StartTime.Format(time.RFC3339))
		fmt.Printf("Source: %s\n\n", e.SourceLocation)
	}
}
```

## Example: Filtered CME Analysis

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

	end := time.Now()
	start := end.AddDate(0, 0, -30)

	analyses, err := client.DONKI.CMEAnalysis(
		context.Background(), start, end,
		nasa.WithMostAccurateOnly(true),
		nasa.WithSpeed(500, 0),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, a := range analyses {
		fmt.Printf("Speed: %.0f km/s, Half-angle: %.0f deg\n", a.Speed, a.HalfAngle)
	}
}
```
