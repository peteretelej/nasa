package nasa

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NeoEndpoint defines the API Endpoint for NASA Neo Web service
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

// Asteroid defines the structure of NASA Asteroids
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

// NeoList is the structure of the response returned by NeoWs Feed
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

// NeoFeed returns a list of of asteroids based on their closest approach date to earth
// Limits time to start and end times specified
func NeoFeed(start, end time.Time) (*NeoList, error) {
	c := getDefaultClient()
	feed, err := c.NEO.Feed(context.Background(), start, end)
	if err != nil {
		return nil, err
	}
	return neoFeedToNeoList(feed, start, end), nil
}

// neoFeedToNeoList converts a *NEOFeed to a *NeoList for backward compatibility.
// This is a minimal inline conversion; Phase 3 will replace it with complete field mapping.
func neoFeedToNeoList(f *NEOFeed, start, end time.Time) *NeoList {
	nl := &NeoList{
		Start:        start.Format(dateFormat),
		End:          end.Format(dateFormat),
		ElementCount: int64(f.ElementCount),
	}
	nl.Links.Self = f.Links.Self
	nl.Links.Next = f.Links.Next
	nl.Links.Prev = f.Links.Prev

	nl.NearEarthObjects = make(map[string][]Asteroid, len(f.NearEarthObjects))
	for date, neos := range f.NearEarthObjects {
		asteroids := make([]Asteroid, 0, len(neos))
		for _, neo := range neos {
			a := Asteroid{
				ID:                   neo.NeoReferenceID,
				Name:                 neo.Name,
				JPLURL:               neo.JPLURL,
				AbsoluteMagnitude:    neo.AbsoluteMagnitude,
				PotentiallyHazardous: neo.IsPotentiallyHazardous,
			}
			// Map estimated diameter.
			a.EstimatedDiameter.Kilometers = diameter{Min: neo.EstimatedDiameter.Kilometers.Min, Max: neo.EstimatedDiameter.Kilometers.Max}
			a.EstimatedDiameter.Meters = diameter{Min: neo.EstimatedDiameter.Meters.Min, Max: neo.EstimatedDiameter.Meters.Max}
			a.EstimatedDiameter.Miles = diameter{Min: neo.EstimatedDiameter.Miles.Min, Max: neo.EstimatedDiameter.Miles.Max}
			a.EstimatedDiameter.Feet = diameter{Min: neo.EstimatedDiameter.Feet.Min, Max: neo.EstimatedDiameter.Feet.Max}

			// Map close approach data.
			for _, ca := range neo.CloseApproachData {
				cad := closeApproachData{
					CloseApproachDate:      ca.CloseApproachDate,
					EpochDateCloseApproach: ca.EpochDateCloseApproach.Time().UnixMilli(),
					OrbitingBody:           ca.OrbitingBody,
				}
				cad.RelativeVelocity.KilometersPerSecond = fmt.Sprintf("%g", float64(ca.RelativeVelocity.KilometersPerSecond))
				cad.RelativeVelocity.KilometersPerHour = fmt.Sprintf("%g", float64(ca.RelativeVelocity.KilometersPerHour))
				cad.RelativeVelocity.MilesPerHour = fmt.Sprintf("%g", float64(ca.RelativeVelocity.MilesPerHour))
				cad.MissDistance.Astronomical = fmt.Sprintf("%g", float64(ca.MissDistance.Astronomical))
				cad.MissDistance.Lunar = fmt.Sprintf("%g", float64(ca.MissDistance.Lunar))
				cad.MissDistance.Kilometers = fmt.Sprintf("%g", float64(ca.MissDistance.Kilometers))
				cad.MissDistance.Miles = fmt.Sprintf("%g", float64(ca.MissDistance.Miles))
				a.CloseApproachData = append(a.CloseApproachData, cad)
			}

			asteroids = append(asteroids, a)
		}
		nl.NearEarthObjects[date] = asteroids
	}

	return nl
}
