package nasa

import (
	"errors"
	"time"
)

// NeoEndpoint defines the API Endpoint for NASA Neo Web service
const NeoEndpoint = "https://api.nasa.gov/neo/rest/v1/feed"

type diameter struct {
	Min float64 `json:"estimated_diameter_min"`
	Max float64 `json:"estimated_diameter_max"`
}
type closeApproachData struct {
	CloseApproachDate      string
	EpochDateCloseApproach int64
	RelativeVelocity       struct {
		KilometersPerSecond, KilometersPerHour,
		MilesPerHour string
	}
	MissDistance struct {
		Astronomical, Lunar, Kilometers,
		Miles string
	}
	OrbitingBody string
}

// Asteroid defines the structure of NASA Asteroids
type Asteroid struct {
	ID                string `json:"neo_reference_id"`
	Name              string
	JPLURL            string  `json:"nasa_jpl_url"`
	AbsoluteMagnitude float64 `json:"absolute_magnitude_h"`
	EstimatedDiameter struct {
		Kilometers, Meters, Miles, Feet diameter
	} `json:"estimated_diameter"`
	PotentiallyHazardous bool `json:"is_potentially_hazardous_asteroid"`
	CloseApproachData    []closeApproachData
	OrbitalData          struct {
		OrbitID, OrbitDeterminationDate, OrbitUncertainity,
		MinimumOrbitIntersection, JupiterTisserandInvariant,
		EpochOsculation, Eccentricity, SemiMajorAxis, Inclination,
		AscendingNodeLongitude, OrbitalPeriod, PerihelionDistance,
		PerihelionArgument, AphelionDistance, PerihelionTime,
		MeanAnomaly, MeanMotion, Equinox string
	}
}

// NeoAsteroids returns a list of of asteroids based on their closest approach date to earth
// Limits time to start and end times specified
func NeoAsteroids(start, end time.Time) ([]Asteroid, error) {
	return nil, errors.New("TODO")
}
