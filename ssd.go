// SSD/CNEOS service types and methods for JPL Small-Body Database and
// close approach, fireball, NHATS, Scout, and Sentry APIs.

package nasa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

// SSDService provides access to the JPL Small-Body Database and CNEOS APIs
// (ssd-api.jpl.nasa.gov). No API key required.
type SSDService struct {
	client  *Client
	baseURL string
}

// --- Internal envelope types ---

// ssdSignature is the common signature block in SSD API responses.
type ssdSignature struct {
	Source  string `json:"source"`
	Version string `json:"version"`
}

// ssdColumnarResponse is the envelope for columnar-format SSD APIs (CAD, Fireball).
// Count and Total use json.RawMessage because the API returns them as either
// a JSON string or a JSON number depending on the sub-API.
type ssdColumnarResponse struct {
	Signature ssdSignature    `json:"signature"`
	Count     json.RawMessage `json:"count"`
	Total     json.RawMessage `json:"total,omitempty"`
	Fields    []string        `json:"fields"`
	Data      [][]string      `json:"data"`
}

// ssdListResponse is the envelope for JSON-object-array SSD APIs (NHATS, Scout, Sentry).
// Data is json.RawMessage to allow per-sub-API struct types.
type ssdListResponse struct {
	Signature ssdSignature    `json:"signature"`
	Count     json.RawMessage `json:"count"`
	Data      json.RawMessage `json:"data"`
}

// parseCount handles the count field which can be either a JSON number or a
// JSON string containing a number.
func parseCount(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, nil
	}

	// Try as number first.
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, nil
	}

	// Try as string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strconv.Atoi(s)
	}

	return 0, fmt.Errorf("nasa: ssd: cannot parse count from %s", string(raw))
}

// --- Generic columnar parser ---

// zipColumns maps columnar data (fields + rows of strings) into a slice of
// typed structs. It uses the "ssd" struct tag to match column names to fields.
// Supported field types: string, float64, int. Empty strings produce zero
// values. Missing columns and extra columns are silently skipped.
func zipColumns[T any](fields []string, data [][]string) ([]*T, error) {
	// Build column-name-to-index map.
	colIndex := make(map[string]int, len(fields))
	for i, f := range fields {
		colIndex[f] = i
	}

	// Reflect on T to build ssd-tag-to-field-index map.
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	type fieldMapping struct {
		colIdx   int
		fieldIdx int
	}

	var mappings []fieldMapping
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("ssd")
		if tag == "" {
			continue
		}
		ci, ok := colIndex[tag]
		if !ok {
			continue // column not present in response, leave zero value
		}
		mappings = append(mappings, fieldMapping{colIdx: ci, fieldIdx: i})
	}

	result := make([]*T, 0, len(data))
	for rowIdx, row := range data {
		v := reflect.New(t)
		elem := v.Elem()

		for _, m := range mappings {
			// Guard against short rows.
			if m.colIdx >= len(row) {
				continue
			}

			val := row[m.colIdx]
			if val == "" {
				continue // leave zero value
			}

			field := elem.Field(m.fieldIdx)
			ft := field.Type()

			// Dereference pointer types: allocate a new value if needed.
			if ft.Kind() == reflect.Ptr {
				ptr := reflect.New(ft.Elem())
				field.Set(ptr)
				field = ptr.Elem()
				ft = ft.Elem()
			}

			switch ft.Kind() {
			case reflect.String:
				field.SetString(val)
			case reflect.Float64, reflect.Float32:
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return nil, fmt.Errorf("nasa: ssd: row %d, column %q: cannot parse %q as float: %w", rowIdx, fields[m.colIdx], val, err)
				}
				field.SetFloat(f)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("nasa: ssd: row %d, column %q: cannot parse %q as int: %w", rowIdx, fields[m.colIdx], val, err)
				}
				field.SetInt(n)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				n, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("nasa: ssd: row %d, column %q: cannot parse %q as uint: %w", rowIdx, fields[m.colIdx], val, err)
				}
				field.SetUint(n)
			}
		}

		result = append(result, v.Interface().(*T))
	}

	return result, nil
}

// --- Data structs: Columnar (with ssd: tags) ---

// CloseApproachData represents a single close approach record from the
// SSD/CNEOS Close Approach Data API (/cad.api). Distances are in AU,
// velocities in km/s.
type CloseApproachData struct {
	Designation  string  `ssd:"des"`
	OrbitID      string  `ssd:"orbit_id"`
	JulianDate   string  `ssd:"jd"`
	CloseDate    string  `ssd:"cd"`
	Distance     float64 `ssd:"dist"`
	DistanceMin  float64 `ssd:"dist_min"`
	DistanceMax  float64 `ssd:"dist_max"`
	VRelative    float64 `ssd:"v_rel"`
	VInfinity    float64 `ssd:"v_inf"`
	TSigma       string  `ssd:"t_sigma_f"`
	AbsMagnitude float64 `ssd:"h"`
}

// FireballEvent represents a fireball/bolide event from the SSD/CNEOS
// Fireball API (/fireball.api).
type FireballEvent struct {
	Date     string  `ssd:"date"`
	Energy   float64 `ssd:"energy"`
	ImpactE  float64 `ssd:"impact-e"`
	Lat      float64 `ssd:"lat"`
	LatDir   string  `ssd:"lat-dir"`
	Lon      float64 `ssd:"lon"`
	LonDir   string  `ssd:"lon-dir"`
	Altitude float64 `ssd:"alt"`
	Velocity float64 `ssd:"vel"`
}

// --- Data structs: JSON object arrays (with json: tags) ---

// NHATSDeltaV represents a min delta-v or min duration trajectory summary
// from the NHATS API. The API returns these as nested objects.
type NHATSDeltaV struct {
	DV  string `json:"dv"`  // delta-v in km/s (string-encoded float)
	Dur int    `json:"dur"` // duration in days
}

// NHATSObject represents a near-Earth asteroid human exploration target
// from the SSD/CNEOS NHATS API (/nhats.api).
type NHATSObject struct {
	Designation string      `json:"des"`
	Fullname    string      `json:"fullname"`
	OrbitID     string      `json:"orbit_id"`
	H           string      `json:"h"`
	MinDV       NHATSDeltaV `json:"min_dv"`
	MinDuration NHATSDeltaV `json:"min_dur"`
	NViaTraj    int         `json:"n_via_traj"`
	ObsStart    *string     `json:"obs_start"`
	ObsEnd      *string     `json:"obs_end"`
	ObsMag      *string     `json:"obs_mag"`
	OCC         string      `json:"occ"`
	MinSize     *string     `json:"min_size"`
	MaxSize     *string     `json:"max_size"`
	ObsFlag     string      `json:"obs_flag"`
}

// ScoutObject represents a NEOCP (Near-Earth Object Confirmation Page) object
// from the SSD/CNEOS Scout API (/scout.api).
type ScoutObject struct {
	ObjectName      string   `json:"objectName"`
	Rating          *int     `json:"rating"`
	H               float64  `json:"H,string"`
	Vmag            float64  `json:"Vmag,string"`
	NeoScore        int      `json:"neoScore"`
	PhaScore        int      `json:"phaScore"`
	Neo1kmScore     int      `json:"neo1kmScore"`
	TisserandScore  int      `json:"tisserandScore"`
	GeocentricScore int      `json:"geocentricScore"`
	IeoScore        int      `json:"ieoScore"`
	NObs            int      `json:"nObs"`
	Arc             string   `json:"arc"`
	Moid            string   `json:"moid"`
	CaDist          *string  `json:"caDist"`
	VInf            *float64 `json:"vInf,string"`
	Elong           string   `json:"elong"`
	Rate            string   `json:"rate"`
	RA              string   `json:"ra"`
	Dec             string   `json:"dec"`
	Unc             string   `json:"unc"`
	UncP1           string   `json:"uncP1"`
	RmsN            string   `json:"rmsN"`
	LastRun         string   `json:"lastRun"`
	TEphem          string   `json:"tEphem"`
}

// SentryObject represents an object in the impact risk assessment from
// the SSD/CNEOS Sentry API (/sentry.api).
type SentryObject struct {
	Designation string `json:"des"`
	Fullname    string `json:"fullname"`
	H           string `json:"h"`
	Diameter    string `json:"diameter"`
	Range       string `json:"range"`
	NImpacts    int    `json:"n_imp"`
	IP          string `json:"ip"`
	PS          string `json:"ps_cum"`
	TS          string `json:"ts_max"`
	VInf        string `json:"v_inf"`
	LastObs     string `json:"last_obs"`
}

// --- Data structs: Single object (SBDB) ---

// OrbitalElement represents a single orbital element from the SBDB API.
type OrbitalElement struct {
	Name  string  `json:"name"`
	Title string  `json:"title"`
	Value string  `json:"value"`
	Units *string `json:"units"`
	Sigma string  `json:"sigma"`
	Label string  `json:"label"`
}

// SSDOrbitClass represents the orbit classification of a small body from
// the SSD/CNEOS API. Named SSDOrbitClass to avoid collision with the NeoWs
// OrbitClass type which has different fields.
type SSDOrbitClass struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// SmallBody represents the response from the SSD/CNEOS Small-Body Database
// API (/sbdb.api). It contains object identification and orbital data.
type SmallBody struct {
	Object struct {
		Fullname   string        `json:"fullname"`
		ShortName  string        `json:"shortname"`
		Des        string        `json:"des"`
		Kind       string        `json:"kind"`
		OrbitID    string        `json:"orbit_id"`
		OrbitClass SSDOrbitClass `json:"orbit_class"` // nested {name, code}, NOT a string
		NEO        bool          `json:"neo"`
		PHA        bool          `json:"pha"`
		SPKID      string        `json:"spkid"`
		Prefix     *string       `json:"prefix"`
	} `json:"object"`
	Orbit struct {
		Elements      []OrbitalElement `json:"elements"`
		Epoch         string           `json:"epoch"`
		OrbitID       string           `json:"orbit_id"`
		Source        string           `json:"source"`
		ConditionCode string           `json:"condition_code"`
		MOID          string           `json:"moid"`
		MOIDJup       string           `json:"moid_jup"`
		TJup          string           `json:"t_jup"`
		DataArc       string           `json:"data_arc"`
		NObs          int              `json:"n_obs_used"`
		FirstObs      string           `json:"first_obs"`
		LastObs       string           `json:"last_obs"`
		RMS           string           `json:"rms"`
		Producer      string           `json:"producer"`
	} `json:"orbit"`
}

// Element looks up an orbital element by name (e.g., "e" for eccentricity,
// "a" for semi-major axis). Returns the element's value, or an empty string
// if not found.
func (sb *SmallBody) Element(name string) string {
	for _, el := range sb.Orbit.Elements {
		if el.Name == name {
			return el.Value
		}
	}
	return ""
}

// --- Functional option types ---

// CADOption configures optional query parameters for the CloseApproach method.
type CADOption func(url.Values)

// CADWithDateMin sets the minimum close-approach date.
func CADWithDateMin(t time.Time) CADOption {
	return func(v url.Values) { v.Set("date-min", t.Format("2006-01-02")) }
}

// CADWithDateMax sets the maximum close-approach date.
func CADWithDateMax(t time.Time) CADOption {
	return func(v url.Values) { v.Set("date-max", t.Format("2006-01-02")) }
}

// CADWithDistMax sets the maximum approach distance in AU.
func CADWithDistMax(dist float64) CADOption {
	return func(v url.Values) { v.Set("dist-max", strconv.FormatFloat(dist, 'f', -1, 64)) }
}

// CADWithBody filters close approaches to a specific body (e.g., "Earth", "Mars").
func CADWithBody(body string) CADOption {
	return func(v url.Values) { v.Set("body", body) }
}

// CADWithLimit sets the maximum number of results to return.
func CADWithLimit(n int) CADOption {
	return func(v url.Values) { v.Set("limit", strconv.Itoa(n)) }
}

// FireballOption configures optional query parameters for the Fireball method.
type FireballOption func(url.Values)

// FireballWithDateMin sets the minimum event date.
func FireballWithDateMin(t time.Time) FireballOption {
	return func(v url.Values) { v.Set("date-min", t.Format("2006-01-02")) }
}

// FireballWithDateMax sets the maximum event date.
func FireballWithDateMax(t time.Time) FireballOption {
	return func(v url.Values) { v.Set("date-max", t.Format("2006-01-02")) }
}

// NHATSOption configures optional query parameters for the NHATS method.
type NHATSOption func(url.Values)

// NHATSWithDV sets the maximum delta-v filter in km/s.
func NHATSWithDV(dv float64) NHATSOption {
	return func(v url.Values) { v.Set("dv", strconv.FormatFloat(dv, 'f', -1, 64)) }
}

// ScoutOption configures optional query parameters for the Scout method.
type ScoutOption func(url.Values)

// ScoutWithTDesMin sets the minimum T_des filter.
func ScoutWithTDesMin(t time.Time) ScoutOption {
	return func(v url.Values) { v.Set("tdes-min", t.Format("2006-01-02")) }
}

// SentryOption configures optional query parameters for the Sentry method.
type SentryOption func(url.Values)

// SentryWithIP filters Sentry objects by minimum impact probability.
func SentryWithIP(ip float64) SentryOption {
	return func(v url.Values) { v.Set("ip-min", strconv.FormatFloat(ip, 'f', -1, 64)) }
}

// --- Service methods ---

// CloseApproach retrieves close approach data from the SSD/CNEOS CAD API.
// Returns asteroid/comet close approaches to Earth (or other bodies).
// The response uses columnar format, parsed via zipColumns.
func (s *SSDService) CloseApproach(ctx context.Context, opts ...CADOption) ([]*CloseApproachData, error) {
	params := url.Values{}
	for _, o := range opts {
		o(params)
	}

	resp, err := s.client.getExternal(ctx, s.baseURL, "/cad.api", params)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var env ssdColumnarResponse
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	if len(env.Data) == 0 {
		return []*CloseApproachData{}, nil
	}

	return zipColumns[CloseApproachData](env.Fields, env.Data)
}

// Fireball retrieves fireball/bolide event data from the SSD/CNEOS Fireball API.
// Always requests location data (req-loc=true). The response uses columnar
// format with count as a string.
func (s *SSDService) Fireball(ctx context.Context, opts ...FireballOption) ([]*FireballEvent, error) {
	params := url.Values{}
	params.Set("req-loc", "true")
	for _, o := range opts {
		o(params)
	}

	resp, err := s.client.getExternal(ctx, s.baseURL, "/fireball.api", params)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var env ssdColumnarResponse
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	if len(env.Data) == 0 {
		return []*FireballEvent{}, nil
	}

	return zipColumns[FireballEvent](env.Fields, env.Data)
}

// NHATS retrieves near-Earth asteroid human exploration targets from the
// SSD/CNEOS NHATS API. Returns objects with trajectory summaries including
// delta-v and duration data.
func (s *SSDService) NHATS(ctx context.Context, opts ...NHATSOption) ([]*NHATSObject, error) {
	params := url.Values{}
	for _, o := range opts {
		o(params)
	}

	resp, err := s.client.getExternal(ctx, s.baseURL, "/nhats.api", params)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var env ssdListResponse
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	var result []*NHATSObject
	if err := json.Unmarshal(env.Data, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = []*NHATSObject{}
	}
	return result, nil
}

// Scout retrieves NEOCP (Near-Earth Object Confirmation Page) objects from
// the SSD/CNEOS Scout API. Some fields (rating, vInf, caDist) may be null.
func (s *SSDService) Scout(ctx context.Context, opts ...ScoutOption) ([]*ScoutObject, error) {
	params := url.Values{}
	for _, o := range opts {
		o(params)
	}

	resp, err := s.client.getExternal(ctx, s.baseURL, "/scout.api", params)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var env ssdListResponse
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	var result []*ScoutObject
	if err := json.Unmarshal(env.Data, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = []*ScoutObject{}
	}
	return result, nil
}

// Sentry retrieves impact risk assessment data from the SSD/CNEOS Sentry API.
func (s *SSDService) Sentry(ctx context.Context, opts ...SentryOption) ([]*SentryObject, error) {
	params := url.Values{}
	for _, o := range opts {
		o(params)
	}

	resp, err := s.client.getExternal(ctx, s.baseURL, "/sentry.api", params)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var env ssdListResponse
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	var result []*SentryObject
	if err := json.Unmarshal(env.Data, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = []*SentryObject{}
	}
	return result, nil
}

// SBDB retrieves small body information from the SSD/CNEOS Small-Body
// Database API. Pass the object designation (e.g., "433" for Eros,
// "2099942" for Apophis).
func (s *SSDService) SBDB(ctx context.Context, designation string) (*SmallBody, error) {
	params := url.Values{}
	params.Set("des", designation)

	resp, err := s.client.getExternal(ctx, s.baseURL, "/sbdb.api", params)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var sb SmallBody
	if err := json.Unmarshal(body, &sb); err != nil {
		return nil, err
	}
	return &sb, nil
}
