# SSD - Solar System Dynamics

The `SSDService` provides access to JPL's Solar System Dynamics and Center for Near-Earth Object Studies (CNEOS) APIs. No API key is required.

## Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `CloseApproach` | `CloseApproach(ctx context.Context, opts ...CADOption) ([]*CloseApproachData, error)` | Query close-approach data |
| `Fireball` | `Fireball(ctx context.Context, opts ...FireballOption) ([]*FireballEvent, error)` | Fireball and bolide reports |
| `NHATS` | `NHATS(ctx context.Context, opts ...NHATSOption) ([]*NHATSObject, error)` | Human-accessible near-Earth objects |
| `Scout` | `Scout(ctx context.Context, opts ...ScoutOption) ([]*ScoutObject, error)` | Impact hazard assessment |
| `Sentry` | `Sentry(ctx context.Context, opts ...SentryOption) ([]*SentryObject, error)` | Impact monitoring |
| `SBDB` | `SBDB(ctx context.Context, designation string) (*SmallBody, error)` | Small-body database lookup |

## Options

### Close Approach Options

```go
nasa.CADWithDateMin(time.Now())          // minimum date
nasa.CADWithDateMax(time.Now().AddDate(0, 0, 7)) // maximum date
nasa.CADWithDistMax(0.05)                // max distance in AU
nasa.CADWithBody("Earth")               // close-approach body
nasa.CADWithLimit(10)                    // max results
```

### Fireball Options

```go
nasa.FireballWithDateMin(start) // minimum date
nasa.FireballWithDateMax(end)   // maximum date
```

### Other Options

```go
nasa.NHATSWithDV(6.0)           // max delta-v in km/s
nasa.ScoutWithTDesMin(t)        // minimum designation time
nasa.SentryWithIP(1e-3)         // minimum impact probability
```

## Key Types

### CloseApproachData

```go
type CloseApproachData struct {
    Designation  string
    OrbitID      string
    JulianDate   string
    CloseDate    string
    Distance     float64 // AU
    DistanceMin  float64
    DistanceMax  float64
    VRelative    float64 // km/s
    VInfinity    float64 // km/s
    TSigma       string
    AbsMagnitude float64
}
```

### SmallBody

```go
type SmallBody struct {
    Object struct {
        Fullname   string
        ShortName  string
        Des        string
        Kind       string
        OrbitID    string
        OrbitClass SSDOrbitClass
        NEO        bool
        PHA        bool
    }
    Orbit struct {
        Elements      []OrbitalElement
        Epoch         string
        ConditionCode string
        // ... additional orbital data
    }
}
```

Use `sb.Element("e")` to look up an orbital element by name (e.g., eccentricity).

## Example: Close Approaches This Week

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

	now := time.Now()
	approaches, err := client.SSD.CloseApproach(
		context.Background(),
		nasa.CADWithDateMin(now),
		nasa.CADWithDateMax(now.AddDate(0, 0, 7)),
		nasa.CADWithDistMax(0.05),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Close approaches in next 7 days: %d\n\n", len(approaches))
	for _, ca := range approaches {
		fmt.Printf("%s on %s (%.4f AU, %.1f km/s)\n",
			ca.Designation, ca.CloseDate, ca.Distance, ca.VRelative,
		)
	}
}
```

## Example: Look Up a Small Body

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

	body, err := client.SSD.SBDB(context.Background(), "Ceres")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\n", body.Object.Fullname)
	fmt.Printf("Kind: %s\n", body.Object.Kind)
	fmt.Printf("Orbit class: %s\n", body.Object.OrbitClass.Name)
	fmt.Printf("NEO: %v\n", body.Object.NEO)
	fmt.Printf("Eccentricity: %s\n", body.Element("e"))
}
```
