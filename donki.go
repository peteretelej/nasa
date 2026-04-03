package nasa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// heliographicRe matches heliographic coordinate strings like "N10E80".
var heliographicRe = regexp.MustCompile(`^([NS])(\d+)([EW])(\d+)$`)

// DONKIService provides access to the Space Weather Database Of Notifications,
// Knowledge, Information (DONKI) API. DONKI offers 11 sub-APIs covering solar
// and geomagnetic events. All methods default to the last 30 days when zero-
// value times are passed.
type DONKIService struct {
	client *Client
}

// DONKITime wraps time.Time to handle DONKI's inconsistent ISO 8601 timestamps.
// It supports formats with or without seconds, with or without timezone info.
type DONKITime struct {
	time.Time
}

// donkiTimeLayouts are tried in order during unmarshaling.
var donkiTimeLayouts = []string{
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04Z",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

// UnmarshalJSON parses DONKI timestamps, trying multiple ISO 8601 layouts.
// Handles null values by setting the zero time.
func (t *DONKITime) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" || s == `""` {
		t.Time = time.Time{}
		return nil
	}

	// Unquote the JSON string.
	unquoted, err := strconv.Unquote(s)
	if err != nil {
		return fmt.Errorf("nasa: DONKITime: cannot unquote %s: %w", s, err)
	}
	if unquoted == "" {
		t.Time = time.Time{}
		return nil
	}

	for _, layout := range donkiTimeLayouts {
		if parsed, err := time.Parse(layout, unquoted); err == nil {
			t.Time = parsed
			return nil
		}
	}

	return fmt.Errorf("nasa: DONKITime: cannot parse %q with any known layout", unquoted)
}

// HeliographicCoord represents a heliographic coordinate with latitude and
// longitude in degrees. Latitude is positive for North, negative for South.
// Longitude is positive for East, negative for West.
type HeliographicCoord struct {
	Latitude  float64
	Longitude float64
}

// ParseHeliographicCoord parses a heliographic coordinate string like "N10E80"
// into structured coordinates. Returns an error for empty strings, "backside",
// or malformed input.
func ParseHeliographicCoord(s string) (HeliographicCoord, error) {
	if s == "" {
		return HeliographicCoord{}, fmt.Errorf("nasa: empty heliographic coordinate")
	}

	upper := strings.ToUpper(strings.TrimSpace(s))
	if upper == "BACKSIDE" {
		return HeliographicCoord{}, fmt.Errorf("nasa: heliographic coordinate is backside")
	}

	matches := heliographicRe.FindStringSubmatch(upper)
	if matches == nil {
		return HeliographicCoord{}, fmt.Errorf("nasa: malformed heliographic coordinate: %q", s)
	}

	lat, _ := strconv.ParseFloat(matches[2], 64)
	lon, _ := strconv.ParseFloat(matches[4], 64)

	if matches[1] == "S" {
		lat = -lat
	}
	if matches[3] == "W" {
		lon = -lon
	}

	return HeliographicCoord{Latitude: lat, Longitude: lon}, nil
}

// LinkedEvent represents a cross-reference to another DONKI event.
type LinkedEvent struct {
	ActivityID string `json:"activityID"`
}

// --- CME types ---

// CMEInstrument represents an instrument that observed a CME.
type CMEInstrument struct {
	DisplayName string `json:"displayName"`
}

// WSAEnlilData represents nested WSA-Enlil simulation data referenced from
// CME analysis entries.
type WSAEnlilData struct {
	ModelCompletionTime   DONKITime          `json:"modelCompletionTime"`
	Au                    float64            `json:"au"`
	SimulationID          string             `json:"simulationID"`
	EstimatedShockArrival DONKITime          `json:"estimatedShockArrivalTime"`
	EstimatedDuration     *float64           `json:"estimatedDuration"`
	RMin                  *float64           `json:"rmin_re"`
	KP18                  *int               `json:"kp_18"`
	KP90                  *int               `json:"kp_90"`
	KP135                 *int               `json:"kp_135"`
	KP180                 *int               `json:"kp_180"`
	IsEarthGB             bool               `json:"isEarthGB"`
	Link                  string             `json:"link"`
	ImpactList            []WSAEnlilImpact   `json:"impactList"`
	CMEInputs             []WSAEnlilCMEInput `json:"cmeInputs"`
}

// CMEAnalysisData represents analysis data for a CME event.
type CMEAnalysisData struct {
	Time21_5       DONKITime      `json:"time21_5"`
	Latitude       float64        `json:"latitude"`
	Longitude      float64        `json:"longitude"`
	HalfAngle      float64        `json:"halfAngle"`
	Speed          float64        `json:"speed"`
	Type           string         `json:"type"`
	IsMostAccurate bool           `json:"isMostAccurate"`
	Note           string         `json:"note"`
	LevelOfData    int            `json:"levelOfData"`
	Link           string         `json:"link"`
	EnlilList      []WSAEnlilData `json:"enlilList"`
}

// CMEEvent represents a Coronal Mass Ejection event from the DONKI API.
type CMEEvent struct {
	ActivityID      string            `json:"activityID"`
	Catalog         string            `json:"catalog"`
	StartTime       DONKITime         `json:"startTime"`
	SourceLocation  string            `json:"sourceLocation"`
	ActiveRegionNum int               `json:"activeRegionNum"`
	Link            string            `json:"link"`
	Note            string            `json:"note"`
	Instruments     []CMEInstrument   `json:"instruments"`
	CMEAnalyses     []CMEAnalysisData `json:"cmeAnalyses"`
	LinkedEvents    []LinkedEvent     `json:"linkedEvents"`
}

// CMEAnalysis represents a top-level CME analysis entry from the /CMEAnalysis
// endpoint. It includes the associated CME ID alongside analysis fields.
type CMEAnalysis struct {
	Time21_5        DONKITime      `json:"time21_5"`
	Latitude        float64        `json:"latitude"`
	Longitude       float64        `json:"longitude"`
	HalfAngle       float64        `json:"halfAngle"`
	Speed           float64        `json:"speed"`
	Type            string         `json:"type"`
	IsMostAccurate  bool           `json:"isMostAccurate"`
	Note            string         `json:"note"`
	Link            string         `json:"link"`
	EnlilList       []WSAEnlilData `json:"enlilList"`
	AssociatedCMEID string         `json:"associatedCMEID"`
}

// --- GST types ---

// KpIndex represents a geomagnetic storm Kp index observation.
type KpIndex struct {
	ObservedTime DONKITime `json:"observedTime"`
	KpIndex      float64   `json:"kpIndex"`
	Source       string    `json:"source"`
}

// GSTEvent represents a Geomagnetic Storm event from the DONKI API.
type GSTEvent struct {
	GstID        string        `json:"gstID"`
	StartTime    DONKITime     `json:"startTime"`
	Link         string        `json:"link"`
	LinkedEvents []LinkedEvent `json:"linkedEvents"`
	AllKpIndex   []KpIndex     `json:"allKpIndex"`
}

// --- IPS types ---

// IPSInstrument represents an instrument that observed an interplanetary shock.
type IPSInstrument struct {
	DisplayName string `json:"displayName"`
}

// IPSEvent represents an Interplanetary Shock event from the DONKI API.
type IPSEvent struct {
	Catalog      string          `json:"catalog"`
	ActivityID   string          `json:"activityID"`
	Location     string          `json:"location"`
	EventTime    DONKITime       `json:"eventTime"`
	Link         string          `json:"link"`
	Instruments  []IPSInstrument `json:"instruments"`
	LinkedEvents []LinkedEvent   `json:"linkedEvents"`
}

// --- FLR types ---

// FLRInstrument represents an instrument that observed a solar flare.
type FLRInstrument struct {
	DisplayName string `json:"displayName"`
}

// FLREvent represents a Solar Flare event from the DONKI API.
type FLREvent struct {
	FlrID           string          `json:"flrID"`
	Instruments     []FLRInstrument `json:"instruments"`
	BeginTime       DONKITime       `json:"beginTime"`
	PeakTime        DONKITime       `json:"peakTime"`
	EndTime         DONKITime       `json:"endTime"`
	ClassType       string          `json:"classType"`
	SourceLocation  string          `json:"sourceLocation"`
	ActiveRegionNum int             `json:"activeRegionNum"`
	Link            string          `json:"link"`
	LinkedEvents    []LinkedEvent   `json:"linkedEvents"`
}

// --- SEP types ---

// SEPInstrument represents an instrument that observed a solar energetic particle event.
type SEPInstrument struct {
	DisplayName string `json:"displayName"`
}

// SEPEvent represents a Solar Energetic Particle event from the DONKI API.
type SEPEvent struct {
	SepID        string          `json:"sepID"`
	EventTime    DONKITime       `json:"eventTime"`
	Instruments  []SEPInstrument `json:"instruments"`
	Link         string          `json:"link"`
	LinkedEvents []LinkedEvent   `json:"linkedEvents"`
}

// --- MPC types ---

// MPCInstrument represents an instrument that observed a magnetopause crossing.
type MPCInstrument struct {
	DisplayName string `json:"displayName"`
}

// MPCEvent represents a Magnetopause Crossing event from the DONKI API.
type MPCEvent struct {
	MpcID        string          `json:"mpcID"`
	EventTime    DONKITime       `json:"eventTime"`
	Instruments  []MPCInstrument `json:"instruments"`
	Link         string          `json:"link"`
	LinkedEvents []LinkedEvent   `json:"linkedEvents"`
}

// --- RBE types ---

// RBEInstrument represents an instrument that observed a radiation belt enhancement.
type RBEInstrument struct {
	DisplayName string `json:"displayName"`
}

// RBEEvent represents a Radiation Belt Enhancement event from the DONKI API.
type RBEEvent struct {
	RbeID        string          `json:"rbeID"`
	EventTime    DONKITime       `json:"eventTime"`
	Instruments  []RBEInstrument `json:"instruments"`
	Link         string          `json:"link"`
	LinkedEvents []LinkedEvent   `json:"linkedEvents"`
}

// --- HSS types ---

// HSSInstrument represents an instrument that observed a high speed stream.
type HSSInstrument struct {
	DisplayName string `json:"displayName"`
}

// HSSEvent represents a High Speed Stream event from the DONKI API.
type HSSEvent struct {
	HssID        string          `json:"hssID"`
	EventTime    DONKITime       `json:"eventTime"`
	Instruments  []HSSInstrument `json:"instruments"`
	Link         string          `json:"link"`
	LinkedEvents []LinkedEvent   `json:"linkedEvents"`
}

// --- WSA-Enlil types ---

// WSAEnlilCMEInput represents a CME input to a WSA-Enlil simulation.
type WSAEnlilCMEInput struct {
	CMEStartTime   DONKITime `json:"cmeStartTime"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Speed          float64   `json:"speed"`
	HalfAngle      float64   `json:"halfAngle"`
	Time21_5       DONKITime `json:"time21_5"`
	IsMostAccurate bool      `json:"isMostAccurate"`
	LevelOfData    int       `json:"levelOfData"`
	Link           string    `json:"link"`
	Note           string    `json:"note"`
}

// WSAEnlilImpact represents an estimated impact from a WSA-Enlil simulation.
type WSAEnlilImpact struct {
	IsGlancingBlow bool      `json:"isGlancingBlow"`
	Location       string    `json:"location"`
	ArrivalTime    DONKITime `json:"arrivalTime"`
}

// WSAEnlilEvent represents a WSA-Enlil simulation result from the DONKI API.
type WSAEnlilEvent struct {
	SimulationID              string             `json:"simulationID"`
	ModelCompletionTime       DONKITime          `json:"modelCompletionTime"`
	Au                        float64            `json:"au"`
	CMEInputs                 []WSAEnlilCMEInput `json:"cmeInputs"`
	EstimatedShockArrivalTime DONKITime          `json:"estimatedShockArrivalTime"`
	EstimatedDuration         *float64           `json:"estimatedDuration"`
	Link                      string             `json:"link"`
	ImpactList                []WSAEnlilImpact   `json:"impactList"`
}

// --- Notification type ---

// Notification represents a DONKI notification message.
type Notification struct {
	MessageType      string        `json:"messageType"`
	MessageID        string        `json:"messageID"`
	MessageURL       string        `json:"messageURL"`
	MessageIssueTime DONKITime     `json:"messageIssueTime"`
	MessageBody      string        `json:"messageBody"`
	LinkedEvents     []LinkedEvent `json:"linkedEvents"`
}

// --- Option types ---

// CMEAnalysisOption configures optional query parameters for the CMEAnalysis
// method. Use the With* constructors to create options.
type CMEAnalysisOption func(url.Values)

// WithMostAccurateOnly filters CME analysis results to only the most accurate
// entry per CME event when true. The DONKI API defaults to true.
func WithMostAccurateOnly(v bool) CMEAnalysisOption {
	return func(params url.Values) {
		params.Set("mostAccurateOnly", strconv.FormatBool(v))
	}
}

// WithCompleteEntryOnly filters CME analysis results to only entries with
// complete data when true. The DONKI API defaults to true.
func WithCompleteEntryOnly(v bool) CMEAnalysisOption {
	return func(params url.Values) {
		params.Set("completeEntryOnly", strconv.FormatBool(v))
	}
}

// WithSpeed sets the minimum speed (km/s) lower-limit filter for CME analysis
// results. The min parameter sets the lower bound; max is accepted for
// forward-compatibility but the DONKI API currently only supports a lower
// limit (defaults to 0).
func WithSpeed(min, max float64) CMEAnalysisOption {
	return func(params url.Values) {
		params.Set("speed", strconv.FormatFloat(min, 'f', -1, 64))
	}
}

// WithHalfAngle sets the minimum half-angle (degrees) lower-limit filter for
// CME analysis results. The min parameter sets the lower bound; max is accepted
// for forward-compatibility but the DONKI API currently only supports a lower
// limit (defaults to 0).
func WithHalfAngle(min, max float64) CMEAnalysisOption {
	return func(params url.Values) {
		params.Set("halfAngle", strconv.FormatFloat(min, 'f', -1, 64))
	}
}

// WithCatalog filters CME analysis results by catalog name. Common values are
// "ALL", "M2M_CATALOG", "SWPC_ANNEX_CME_CATALOG", and "SEPVAL_CME_CATALOG".
// The DONKI API defaults to "ALL".
func WithCatalog(catalog string) CMEAnalysisOption {
	return func(params url.Values) {
		params.Set("catalog", catalog)
	}
}

// IPSOption configures optional query parameters for the IPS method.
// Use the With* constructors to create options.
type IPSOption func(url.Values)

// WithLocation filters interplanetary shock results by observer location.
// Common values are "Earth", "Mars", "MESSENGER", "STEREO A", "STEREO B".
// The DONKI API defaults to "ALL".
func WithLocation(loc string) IPSOption {
	return func(params url.Values) {
		params.Set("location", loc)
	}
}

// WithIPSCatalog filters interplanetary shock results by catalog name.
// Common values are "ALL", "M2M_CATALOG", "WINSLOW_MESSENGER_ICME_CATALOG".
// The DONKI API defaults to "ALL".
func WithIPSCatalog(catalog string) IPSOption {
	return func(params url.Values) {
		params.Set("catalog", catalog)
	}
}

// --- API methods ---

// CME retrieves Coronal Mass Ejection events from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) CME(ctx context.Context, start, end time.Time) ([]*CMEEvent, error) {
	return donkiGet[CMEEvent](ctx, s, "/DONKI/CME", donkiParams(start, end))
}

// CMEAnalysis retrieves CME analysis entries from the DONKI API, with optional
// filtering by accuracy, completeness, speed, half-angle, and catalog.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) CMEAnalysis(ctx context.Context, start, end time.Time, opts ...CMEAnalysisOption) ([]*CMEAnalysis, error) {
	params := donkiParams(start, end)
	for _, o := range opts {
		o(params)
	}
	return donkiGet[CMEAnalysis](ctx, s, "/DONKI/CMEAnalysis", params)
}

// GST retrieves Geomagnetic Storm events from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) GST(ctx context.Context, start, end time.Time) ([]*GSTEvent, error) {
	return donkiGet[GSTEvent](ctx, s, "/DONKI/GST", donkiParams(start, end))
}

// IPS retrieves Interplanetary Shock events from the DONKI API, with optional
// filtering by location and catalog.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) IPS(ctx context.Context, start, end time.Time, opts ...IPSOption) ([]*IPSEvent, error) {
	params := donkiParams(start, end)
	for _, o := range opts {
		o(params)
	}
	return donkiGet[IPSEvent](ctx, s, "/DONKI/IPS", params)
}

// FLR retrieves Solar Flare events from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) FLR(ctx context.Context, start, end time.Time) ([]*FLREvent, error) {
	return donkiGet[FLREvent](ctx, s, "/DONKI/FLR", donkiParams(start, end))
}

// SEP retrieves Solar Energetic Particle events from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) SEP(ctx context.Context, start, end time.Time) ([]*SEPEvent, error) {
	return donkiGet[SEPEvent](ctx, s, "/DONKI/SEP", donkiParams(start, end))
}

// MPC retrieves Magnetopause Crossing events from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) MPC(ctx context.Context, start, end time.Time) ([]*MPCEvent, error) {
	return donkiGet[MPCEvent](ctx, s, "/DONKI/MPC", donkiParams(start, end))
}

// RBE retrieves Radiation Belt Enhancement events from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) RBE(ctx context.Context, start, end time.Time) ([]*RBEEvent, error) {
	return donkiGet[RBEEvent](ctx, s, "/DONKI/RBE", donkiParams(start, end))
}

// HSS retrieves High Speed Stream events from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) HSS(ctx context.Context, start, end time.Time) ([]*HSSEvent, error) {
	return donkiGet[HSSEvent](ctx, s, "/DONKI/HSS", donkiParams(start, end))
}

// WSAEnlil retrieves WSA-Enlil simulation results from the DONKI API.
// When start is the zero value, defaults to 30 days ago. When end is the zero
// value, defaults to today. The DONKI API returns all results in a single
// response.
func (s *DONKIService) WSAEnlil(ctx context.Context, start, end time.Time) ([]*WSAEnlilEvent, error) {
	return donkiGet[WSAEnlilEvent](ctx, s, "/DONKI/WSAEnlilSimulations", donkiParams(start, end))
}

// Notifications retrieves DONKI notification messages, optionally filtered by
// type. When notificationType is non-empty, only notifications of that type are
// returned. Common values are "all", "FLR", "SEP", "CME", "IPS", "MPC", "GST",
// "RBE", and "report". When start is the zero value, defaults to 30 days ago.
// When end is the zero value, defaults to today. The DONKI API returns all
// results in a single response.
func (s *DONKIService) Notifications(ctx context.Context, start, end time.Time, notificationType string) ([]*Notification, error) {
	params := donkiParams(start, end)
	if notificationType != "" {
		params.Set("type", notificationType)
	}
	return donkiGet[Notification](ctx, s, "/DONKI/notifications", params)
}

// --- Helpers ---

// donkiParams builds the standard startDate/endDate query parameters for DONKI
// API requests. When start is zero, it defaults to 30 days ago. When end is
// zero, it is omitted (the API defaults to today).
func donkiParams(start, end time.Time) url.Values {
	params := url.Values{}
	if start.IsZero() {
		params.Set("startDate", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	} else {
		params.Set("startDate", start.Format("2006-01-02"))
	}
	if !end.IsZero() {
		params.Set("endDate", end.Format("2006-01-02"))
	}
	return params
}

// donkiGet is a generic helper that fetches a DONKI endpoint and decodes the
// bare JSON array response into a slice of the given type. DONKI endpoints
// return bare arrays (no wrapper object), so this decodes directly into []*T.
// A nil or empty JSON array is normalized to a non-nil empty slice.
func donkiGet[T any](ctx context.Context, s *DONKIService, path string, params url.Values) ([]*T, error) {
	resp, err := s.client.get(ctx, path, params, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result []*T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = []*T{}
	}
	return result, nil
}
