package nasa

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"time"
)

// EONETService provides access to the Earth Observatory Natural Event Tracker
// API (eonet.gsfc.nasa.gov/api/v3). API key is optional; EONET has its own
// rate pool separate from api.nasa.gov.
type EONETService struct {
	client  *Client
	baseURL string
	apiKey  string
}

// --- Response envelope types (internal) ---

// eonetEventsResponse is the top-level wrapper for the /events endpoint.
type eonetEventsResponse struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Link        string        `json:"link"`
	Events      []*EONETEvent `json:"events"`
}

// eonetCategoriesResponse is the top-level wrapper for the /categories endpoint.
type eonetCategoriesResponse struct {
	Categories []*EONETCategory `json:"categories"`
}

// eonetSourcesResponse is the top-level wrapper for the /sources endpoint.
type eonetSourcesResponse struct {
	Sources []*EONETSource `json:"sources"`
}

// --- Public response types ---

// EONETEvent represents a natural event from the EONET API.
// Closed is nil for events that are still open.
type EONETEvent struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Link        string           `json:"link"`
	Closed      *time.Time       `json:"closed"`
	Categories  []*EONETCategory `json:"categories"`
	Sources     []*EONETSource   `json:"sources"`
	Geometry    []*EONETGeometry `json:"geometry"`
}

// EONETGeometry represents a GeoJSON-style geometry entry for an event.
// MagnitudeValue is nil when no magnitude data is available.
type EONETGeometry struct {
	Date           time.Time `json:"date"`
	Type           string    `json:"type"`
	Coordinates    []float64 `json:"coordinates"`
	MagnitudeValue *float64  `json:"magnitudeValue"`
	MagnitudeUnit  string    `json:"magnitudeUnit"`
}

// EONETCategory represents an EONET event category.
type EONETCategory struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Link        string `json:"link"`
	Layers      string `json:"layers,omitempty"`
}

// EONETSource represents a data source for EONET events.
type EONETSource struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// --- Functional options ---

// EONETOption configures optional query parameters for EONET event queries.
type EONETOption func(url.Values)

// EONETWithStatus filters events by status: "open" or "closed".
func EONETWithStatus(status string) EONETOption {
	return func(v url.Values) { v.Set("status", status) }
}

// EONETWithLimit sets the maximum number of events to return.
func EONETWithLimit(n int) EONETOption {
	return func(v url.Values) { v.Set("limit", strconv.Itoa(n)) }
}

// EONETWithDays filters events to those within the last n days.
func EONETWithDays(n int) EONETOption {
	return func(v url.Values) { v.Set("days", strconv.Itoa(n)) }
}

// EONETWithSource filters events by data source identifier.
func EONETWithSource(src string) EONETOption {
	return func(v url.Values) { v.Set("source", src) }
}

// EONETWithStart filters events starting on or after the given time.
func EONETWithStart(t time.Time) EONETOption {
	return func(v url.Values) { v.Set("start", t.Format("2006-01-02")) }
}

// EONETWithEnd filters events ending on or before the given time.
func EONETWithEnd(t time.Time) EONETOption {
	return func(v url.Values) { v.Set("end", t.Format("2006-01-02")) }
}

// EONETWithCategory filters events by category identifier.
func EONETWithCategory(cat string) EONETOption {
	return func(v url.Values) { v.Set("category", cat) }
}

// --- Auth helper ---

// addAuth adds the api_key parameter if a real (non-DEMO_KEY) key is configured.
// EONET works without auth but has a separate rate pool; sending DEMO_KEY
// provides no benefit.
func (s *EONETService) addAuth(params url.Values) {
	if s.apiKey != "" && s.apiKey != "DEMO_KEY" {
		params.Set("api_key", s.apiKey)
	}
}

// --- API methods ---

// Events retrieves natural events from the EONET API, with optional filtering.
func (s *EONETService) Events(ctx context.Context, opts ...EONETOption) ([]*EONETEvent, error) {
	params := url.Values{}
	for _, o := range opts {
		o(params)
	}
	s.addAuth(params)

	resp, err := s.client.getExternal(ctx, s.baseURL, "events", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope eonetEventsResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	if envelope.Events == nil {
		envelope.Events = []*EONETEvent{}
	}
	return envelope.Events, nil
}

// Event retrieves a single natural event by ID from the EONET API.
func (s *EONETService) Event(ctx context.Context, eventID string) (*EONETEvent, error) {
	params := url.Values{}
	s.addAuth(params)

	resp, err := s.client.getExternal(ctx, s.baseURL, "events/"+eventID, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var event EONETEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// Categories retrieves all EONET event categories.
func (s *EONETService) Categories(ctx context.Context) ([]*EONETCategory, error) {
	resp, err := s.client.getExternal(ctx, s.baseURL, "categories", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope eonetCategoriesResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	if envelope.Categories == nil {
		envelope.Categories = []*EONETCategory{}
	}
	return envelope.Categories, nil
}

// Sources retrieves all EONET data sources.
func (s *EONETService) Sources(ctx context.Context) ([]*EONETSource, error) {
	resp, err := s.client.getExternal(ctx, s.baseURL, "sources", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope eonetSourcesResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	if envelope.Sources == nil {
		envelope.Sources = []*EONETSource{}
	}
	return envelope.Sources, nil
}
