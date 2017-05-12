package nasa

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NeoEndpoint defines the API Endpoint for NASA Neo Web service
const NeoEndpoint = "https://api.nasa.gov/neo/rest/v1/feed"

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
			objs = append(objs, fmt.Sprintf("%s", each.Name))
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
	u, err := url.Parse(NeoEndpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse neo endpoint")
	}
	startdate, enddate := start.Format("2006-01-02"), end.Format("2006-01-02")
	q := u.Query()
	q.Set("api_key", nasaKey)
	q.Add("start_date", startdate)
	q.Add("end_date", enddate)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	cl := &http.Client{Timeout: time.Second * 20}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()
	var nl NeoList

	err = json.Unmarshal(dat, &nl)
	if err != nil {
		return nil, err
	}
	nl.Start = startdate
	nl.End = enddate
	return &nl, nil
}
