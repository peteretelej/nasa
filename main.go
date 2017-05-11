package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

// NASA API Endpoints
const (
	APODEndpoint = "https://api.nasa.gov/planetary/apod"
	NeoEndpoint  = "https://api.nasa.gov/neo/rest/v1/feed"
)

var (
	nasaKey = os.Getenv("NASAKEY")
)

// subcommands and flags
var (
	apodCommand = flag.NewFlagSet("apod", flag.ExitOnError)
	apodDate    = apodCommand.String("date", "", "APOD on a particular date YYYY-MM-DD")

	neoCommand = flag.NewFlagSet("neo", flag.ExitOnError)
)

func main() {
	flag.Parse()
	if nasaKey == "" {
		fmt.Println("missing NASAKEY api key, required. Use NASAKEY=DEMO_KEY to demo.")
		os.Exit(1)
	}

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "apod")
	}

	switch os.Args[1] {
	case "apod":
		t, err := time.Parse("2006-01-02", *apodDate)
		if err != nil {
			t = time.Now()
		}
		apod, err := ApodImage(t, false)
		if err != nil {
			fmt.Printf("unable to get apod: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s\n", apod)
	case "neo":
	}

}

// Image defines the structure of NASA images
type Image struct {
	Date        string `json:"date"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	HDURL       string `json:"hdurl"`
	Explanation string `json:"explanation"`

	ApodDate time.Time `json:",omitempty"`
}

func (ni Image) String() string {
	return fmt.Sprintf(`Title: %s
URL: %s
HDURL: %s
About:
%s
`, ni.Title, ni.URL, ni.HDURL, ni.Explanation)
}

//ApodImage returns the
func ApodImage(t time.Time, hd bool) (*Image, error) {
	if t.After(time.Now()) {
		t = time.Now()
	}
	date := t.Format("2006-01-02")
	u, err := url.Parse(APODEndpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("date", date)
	q.Add("api_key", nasaKey)
	q.Add("hd", fmt.Sprintf("%t", hd))
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: time.Second * 15}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()
	var ni Image
	err = json.Unmarshal(dat, &ni)
	if err != nil {
		return nil, err
	}
	if t, err := time.Parse("2006-01-02", ni.Date); err == nil {
		ni.ApodDate = t
	}
	return &ni, nil

}

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
