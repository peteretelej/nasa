package nasa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// StringFloat64 is a float64 that can be unmarshalled from both JSON strings
// ("34.02") and JSON numbers (34.02). Empty strings unmarshal to 0.
type StringFloat64 float64

// UnmarshalJSON handles both JSON strings ("34.02") and JSON numbers (34.02).
// Empty string unmarshals to 0.
func (sf *StringFloat64) UnmarshalJSON(data []byte) error {
	// Try as a JSON number first.
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*sf = StringFloat64(f)
		return nil
	}

	// Try as a JSON string.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("nasa: StringFloat64: cannot unmarshal %s", string(data))
	}
	if s == "" {
		*sf = 0
		return nil
	}
	parsed, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("nasa: StringFloat64: cannot parse %q: %w", s, err)
	}
	*sf = StringFloat64(parsed)
	return nil
}

// MillisecondEpoch is a time.Time that unmarshals from a millisecond epoch
// timestamp (e.g. 1712016000000).
type MillisecondEpoch time.Time

// UnmarshalJSON converts a millisecond epoch timestamp to time.Time.
func (me *MillisecondEpoch) UnmarshalJSON(data []byte) error {
	var ms int64
	if err := json.Unmarshal(data, &ms); err != nil {
		return fmt.Errorf("nasa: MillisecondEpoch: cannot unmarshal %s: %w", string(data), err)
	}
	sec := ms / 1000
	nsec := (ms % 1000) * 1e6
	*me = MillisecondEpoch(time.Unix(sec, nsec))
	return nil
}

// Time returns the underlying time.Time value.
func (me MillisecondEpoch) Time() time.Time {
	return time.Time(me)
}

// DiameterRange represents minimum and maximum estimated diameter values.
type DiameterRange struct {
	Min float64 `json:"estimated_diameter_min"`
	Max float64 `json:"estimated_diameter_max"`
}

// EstimatedDiameter contains estimated diameter in various units.
type EstimatedDiameter struct {
	Kilometers DiameterRange `json:"kilometers"`
	Meters     DiameterRange `json:"meters"`
	Miles      DiameterRange `json:"miles"`
	Feet       DiameterRange `json:"feet"`
}

// VelocityData contains relative velocity in various units.
// Values are strings in the NASA API, converted to float64 via StringFloat64.
type VelocityData struct {
	KilometersPerSecond StringFloat64 `json:"kilometers_per_second"`
	KilometersPerHour   StringFloat64 `json:"kilometers_per_hour"`
	MilesPerHour        StringFloat64 `json:"miles_per_hour"`
}

// DistanceData contains miss distance in various units.
// Values are strings in the NASA API, converted to float64 via StringFloat64.
type DistanceData struct {
	Astronomical StringFloat64 `json:"astronomical"`
	Lunar        StringFloat64 `json:"lunar"`
	Kilometers   StringFloat64 `json:"kilometers"`
	Miles        StringFloat64 `json:"miles"`
}

// CloseApproach contains close approach data for a near-earth object.
type CloseApproach struct {
	CloseApproachDate      string           `json:"close_approach_date"`
	CloseApproachDateFull  time.Time        `json:"-"`
	EpochDateCloseApproach MillisecondEpoch `json:"epoch_date_close_approach"`
	RelativeVelocity       VelocityData     `json:"relative_velocity"`
	MissDistance           DistanceData     `json:"miss_distance"`
	OrbitingBody           string           `json:"orbiting_body"`
}

// UnmarshalJSON implements custom JSON unmarshalling for CloseApproach,
// parsing the close_approach_date_full string into a time.Time value.
func (ca *CloseApproach) UnmarshalJSON(data []byte) error {
	type Alias CloseApproach
	aux := &struct {
		CloseApproachDateFull string `json:"close_approach_date_full"`
		*Alias
	}{
		Alias: (*Alias)(ca),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.CloseApproachDateFull != "" {
		if t, err := time.Parse("2006-Jan-02 15:04", aux.CloseApproachDateFull); err == nil {
			ca.CloseApproachDateFull = t
		}
	}
	return nil
}

// PaginationLinks contains HATEOAS pagination links from the NeoWs API.
type PaginationLinks struct {
	Self string `json:"self"`
	Next string `json:"next"`
	Prev string `json:"prev"`
}

// PageInfo contains pagination metadata from the NeoWs browse endpoint.
type PageInfo struct {
	Size          int `json:"size"`
	TotalElements int `json:"total_elements"`
	TotalPages    int `json:"total_pages"`
	Number        int `json:"number"`
}

// OrbitClass describes the orbital classification of a near-earth object.
type OrbitClass struct {
	Type        string `json:"orbit_class_type"`
	Description string `json:"orbit_class_description"`
	Range       string `json:"orbit_class_range"`
}

// OrbitalData contains detailed orbital parameters for a near-earth object.
// Only present in single-object lookup responses.
type OrbitalData struct {
	OrbitID                   string      `json:"orbit_id"`
	OrbitDeterminationDate    string      `json:"orbit_determination_date"`
	FirstObservationDate      string      `json:"first_observation_date"`
	LastObservationDate       string      `json:"last_observation_date"`
	DataArcInDays             int         `json:"data_arc_in_days"`
	ObservationsUsed          int         `json:"observations_used"`
	OrbitUncertainty          string      `json:"orbit_uncertainty"`
	MinimumOrbitIntersection  string      `json:"minimum_orbit_intersection"`
	JupiterTisserandInvariant string      `json:"jupiter_tisserand_invariant"`
	EpochOsculation           string      `json:"epoch_osculation"`
	Eccentricity              string      `json:"eccentricity"`
	SemiMajorAxis             string      `json:"semi_major_axis"`
	Inclination               string      `json:"inclination"`
	AscendingNodeLongitude    string      `json:"ascending_node_longitude"`
	OrbitalPeriod             string      `json:"orbital_period"`
	PerihelionDistance        string      `json:"perihelion_distance"`
	PerihelionArgument        string      `json:"perihelion_argument"`
	AphelionDistance          string      `json:"aphelion_distance"`
	PerihelionTime            string      `json:"perihelion_time"`
	MeanAnomaly               string      `json:"mean_anomaly"`
	MeanMotion                string      `json:"mean_motion"`
	Equinox                   string      `json:"equinox"`
	OrbitClass                *OrbitClass `json:"orbit_class"`
}

// NearEarthObject represents a single near-earth object from the NeoWs API.
type NearEarthObject struct {
	ID                     string            `json:"id"`
	NeoReferenceID         string            `json:"neo_reference_id"`
	Name                   string            `json:"name"`
	JPLURL                 string            `json:"nasa_jpl_url"`
	AbsoluteMagnitude      float64           `json:"absolute_magnitude_h"`
	EstimatedDiameter      EstimatedDiameter `json:"estimated_diameter"`
	IsPotentiallyHazardous bool              `json:"is_potentially_hazardous_asteroid"`
	CloseApproachData      []CloseApproach   `json:"close_approach_data"`
	OrbitalData            *OrbitalData      `json:"orbital_data"`
	IsSentryObject         bool              `json:"is_sentry_object"`
}

// NEOFeed is the response from the NeoWs Feed endpoint.
type NEOFeed struct {
	Links            PaginationLinks               `json:"links"`
	ElementCount     int                           `json:"element_count"`
	NearEarthObjects map[string][]*NearEarthObject `json:"near_earth_objects"`
}

// NEOBrowse is the response from the NeoWs Browse endpoint.
type NEOBrowse struct {
	Links            PaginationLinks    `json:"links"`
	Page             PageInfo           `json:"page"`
	NearEarthObjects []*NearEarthObject `json:"near_earth_objects"`
}

// fixHTTPLinks replaces http://api.nasa.gov with https://api.nasa.gov
// in the given string. The NeoWs API returns HTTP links due to a backend bug.
func fixHTTPLinks(s string) string {
	return strings.Replace(s, "http://api.nasa.gov", "https://api.nasa.gov", -1)
}

// NEOService provides access to the Near Earth Object Web Service.
type NEOService struct {
	client *Client
}

// Feed retrieves near-earth objects for a date range (max 7 days).
func (s *NEOService) Feed(ctx context.Context, start, end time.Time) (*NEOFeed, error) {
	startDate := start.Format(dateFormat)
	endDate := end.Format(dateFormat)

	// Validate start <= end using truncated civil dates.
	startTrunc := start.Truncate(24 * time.Hour)
	endTrunc := end.Truncate(24 * time.Hour)
	if startTrunc.After(endTrunc) {
		return nil, fmt.Errorf("nasa: start date %s is after end date %s", startDate, endDate)
	}

	// Validate range <= 7 days.
	if endTrunc.Sub(startTrunc) > 7*24*time.Hour {
		return nil, fmt.Errorf("nasa: date range exceeds 7 days (%s to %s)", startDate, endDate)
	}

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	resp, err := s.client.get(ctx, "/neo/rest/v1/feed", params, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed NEOFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	feed.Links.Self = fixHTTPLinks(feed.Links.Self)
	feed.Links.Next = fixHTTPLinks(feed.Links.Next)
	feed.Links.Prev = fixHTTPLinks(feed.Links.Prev)

	return &feed, nil
}

// Get retrieves a single near-earth object by its asteroid ID.
func (s *NEOService) Get(ctx context.Context, asteroidID string) (*NearEarthObject, error) {
	if asteroidID == "" {
		return nil, fmt.Errorf("nasa: asteroid ID must not be empty")
	}

	path := "/neo/rest/v1/neo/" + asteroidID

	resp, err := s.client.get(ctx, path, nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var neo NearEarthObject
	if err := json.Unmarshal(body, &neo); err != nil {
		return nil, err
	}

	return &neo, nil
}

// Browse retrieves a paginated list of all near-earth objects.
func (s *NEOService) Browse(ctx context.Context, page, size int) (*NEOBrowse, error) {
	if page < 0 {
		return nil, fmt.Errorf("nasa: page must be >= 0, got %d", page)
	}
	if size < 1 {
		return nil, fmt.Errorf("nasa: size must be >= 1, got %d", size)
	}

	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("size", strconv.Itoa(size))

	resp, err := s.client.get(ctx, "/neo/rest/v1/neo/browse", params, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var browse NEOBrowse
	if err := json.Unmarshal(body, &browse); err != nil {
		return nil, err
	}

	browse.Links.Self = fixHTTPLinks(browse.Links.Self)
	browse.Links.Next = fixHTTPLinks(browse.Links.Next)
	browse.Links.Prev = fixHTTPLinks(browse.Links.Prev)

	return &browse, nil
}
