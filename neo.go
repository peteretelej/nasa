// Legacy NeoWs types, convenience functions, and conversion helpers.

package nasa

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NeoEndpoint is the NASA NeoWs Feed API endpoint URL.
var NeoEndpoint = "https://api.nasa.gov/neo/rest/v1/feed"

type diameter struct {
	Min float64 `json:"estimated_diameter_min"`
	Max float64 `json:"estimated_diameter_max"`
}
type closeApproachData struct {
	CloseApproachDate      string `json:"close_approach_date"`
	EpochDateCloseApproach int64  `json:"epoch_date_close_approach"`
	RelativeVelocity       struct {
		KilometersPerSecond string `json:"kilometers_per_second"`
		KilometersPerHour   string `json:"kilometers_per_hour"`
		MilesPerHour        string `json:"miles_per_hour"`
	} `json:"relative_velocity"`
	MissDistance struct {
		Astronomical, Lunar, Kilometers,
		Miles string
	}
	OrbitingBody string `json:"orbiting_body"`
}

// Asteroid is the legacy type representing a near-earth object. New code
// should use [NearEarthObject] via the [Client].NEO service instead.
type Asteroid struct {
	Links             struct{ Self string }
	ID                string `json:"neo_reference_id"`
	Name              string
	JPLURL            string  `json:"nasa_jpl_url"`
	AbsoluteMagnitude float64 `json:"absolute_magnitude_h"`
	EstimatedDiameter struct {
		Kilometers, Meters, Miles, Feet diameter
	} `json:"estimated_diameter"`
	PotentiallyHazardous bool                `json:"is_potentially_hazardous_asteroid"`
	CloseApproachData    []closeApproachData `json:"close_approach_data"`
	OrbitalData          struct {
		OrbitID                   string `json:"orbit_id"`
		OrbitDeterminationDate    string `json:"orbit_determination_date"`
		OrbitUncertainity         string `json:"orbit_uncertainty"`
		MinimumOrbitIntersection  string `json:"minimum_orbit_intersection"`
		JupiterTisserandInvariant string `json:"jupiter_tisserand_invariant"`
		EpochOsculation           string `json:"epoch_osculation"`
		Eccentricity, Inclination string
		SemiMajorAxis             string `json:"semi_major_axis"`
		AscendingNodeLongitude    string `json:"ascending_node_longitude"`
		OrbitalPeriod             string `json:"orbital_period"`
		PerihelionDistance        string `json:"perihelion_distance"`
		PerihelionArgument        string `json:"perihelion_argument"`
		AphelionDistance          string `json:"aphelion_distance"`
		PerihelionTime            string `json:"perihelion_time"`
		MeanAnomaly               string `json:"mean_anomaly"`
		MeanMotion                string `json:"mean_motion"`
		Equinox                   string
	} `json:"orbital_data"`
}

// NeoList is the legacy response type for the NeoWs Feed endpoint. New code
// should use [NEOFeed] via the [Client].NEO service instead.
type NeoList struct {
	Links struct {
		Self, Next, Prev string
	}
	Start            string                `json:"-"`
	End              string                `json:"-"`
	ElementCount     int64                 `json:"element_count"`
	NearEarthObjects map[string][]Asteroid `json:"near_earth_objects"`
}

func (nl NeoList) String() string {
	var neos string
	for k, val := range nl.NearEarthObjects {
		neos += fmt.Sprintf("%s: %d objects\n", k, len(val))
		var objs []string
		for _, each := range val {
			objs = append(objs, each.Name)
		}
		neos += fmt.Sprintf("Objects: %s\n", strings.Join(objs, ","))
	}
	return fmt.Sprintf(`Near Earth Objects From: %s to %s
Number: %d
Link: %s
%s`,
		nl.Start, nl.End, nl.ElementCount, nl.Links.Self, neos)
}

// NeoFeed returns near-earth objects for the given date range. It uses a
// shared internal client that reads the API key from the environment. For
// full control, use [Client].NEO.Feed.
func NeoFeed(start, end time.Time) (*NeoList, error) {
	c := getDefaultClient()
	feed, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		return nil, err
	}
	return neoFeedToNeoList(feed, start.Format(dateFormat), end.Format(dateFormat)), nil
}

// neoFeedToNeoList converts a *NEOFeed to a *NeoList for backward compatibility.
func neoFeedToNeoList(f *NEOFeed, start, end string) *NeoList {
	nl := &NeoList{
		Start:        start,
		End:          end,
		ElementCount: int64(f.ElementCount),
	}
	nl.Links.Self = f.Links.Self
	nl.Links.Next = f.Links.Next
	nl.Links.Prev = f.Links.Prev
	nl.NearEarthObjects = make(map[string][]Asteroid)
	for date, neos := range f.NearEarthObjects {
		asteroids := make([]Asteroid, len(neos))
		for i, neo := range neos {
			asteroids[i] = neoToAsteroid(neo)
		}
		nl.NearEarthObjects[date] = asteroids
	}
	return nl
}

// neoToAsteroid converts a *NearEarthObject to the legacy Asteroid type.
// The conversion is lossy: the old type has fewer fields and different types.
func neoToAsteroid(neo *NearEarthObject) Asteroid {
	a := Asteroid{
		// Asteroid.ID json tag is "neo_reference_id", so map from NeoReferenceID
		ID:                   neo.NeoReferenceID,
		Name:                 neo.Name,
		JPLURL:               neo.JPLURL,
		AbsoluteMagnitude:    neo.AbsoluteMagnitude,
		PotentiallyHazardous: neo.IsPotentiallyHazardous,
	}
	a.Links.Self = neo.Links.Self

	// Diameter
	a.EstimatedDiameter.Kilometers = diameter(neo.EstimatedDiameter.Kilometers)
	a.EstimatedDiameter.Meters = diameter(neo.EstimatedDiameter.Meters)
	a.EstimatedDiameter.Miles = diameter(neo.EstimatedDiameter.Miles)
	a.EstimatedDiameter.Feet = diameter(neo.EstimatedDiameter.Feet)

	// Close approach data - use raw string values preserved during unmarshal
	for _, ca := range neo.CloseApproachData {
		cad := closeApproachData{
			CloseApproachDate:      ca.CloseApproachDate,
			EpochDateCloseApproach: ca.EpochDateCloseApproach.Time().UnixMilli(),
			OrbitingBody:           ca.OrbitingBody,
		}
		rv := ca.rawVelocity
		cad.RelativeVelocity.KilometersPerSecond = rv["kilometers_per_second"]
		cad.RelativeVelocity.KilometersPerHour = rv["kilometers_per_hour"]
		cad.RelativeVelocity.MilesPerHour = rv["miles_per_hour"]
		md := ca.rawDistance
		cad.MissDistance.Astronomical = md["astronomical"]
		cad.MissDistance.Lunar = md["lunar"]
		cad.MissDistance.Kilometers = md["kilometers"]
		cad.MissDistance.Miles = md["miles"]
		a.CloseApproachData = append(a.CloseApproachData, cad)
	}

	// Orbital data
	if neo.OrbitalData != nil {
		od := neo.OrbitalData
		a.OrbitalData.OrbitID = od.OrbitID
		a.OrbitalData.OrbitDeterminationDate = od.OrbitDeterminationDate
		// Old struct has "OrbitUncertainity" (typo) - must preserve
		a.OrbitalData.OrbitUncertainity = od.OrbitUncertainty
		a.OrbitalData.MinimumOrbitIntersection = od.MinimumOrbitIntersection
		a.OrbitalData.JupiterTisserandInvariant = od.JupiterTisserandInvariant
		a.OrbitalData.EpochOsculation = od.EpochOsculation
		a.OrbitalData.Eccentricity = od.Eccentricity
		a.OrbitalData.Inclination = od.Inclination
		a.OrbitalData.SemiMajorAxis = od.SemiMajorAxis
		a.OrbitalData.AscendingNodeLongitude = od.AscendingNodeLongitude
		a.OrbitalData.OrbitalPeriod = od.OrbitalPeriod
		a.OrbitalData.PerihelionDistance = od.PerihelionDistance
		a.OrbitalData.PerihelionArgument = od.PerihelionArgument
		a.OrbitalData.AphelionDistance = od.AphelionDistance
		a.OrbitalData.PerihelionTime = od.PerihelionTime
		a.OrbitalData.MeanAnomaly = od.MeanAnomaly
		a.OrbitalData.MeanMotion = od.MeanMotion
		a.OrbitalData.Equinox = od.Equinox
	}

	return a
}
