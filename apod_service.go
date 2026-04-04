// APOD service types and methods for the Astronomy Picture of the Day API.

package nasa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// MediaType represents the type of media returned by the APOD API.
type MediaType string

const (
	// MediaImage indicates the APOD entry is a still image.
	MediaImage MediaType = "image"
	// MediaVideo indicates the APOD entry is a video (typically YouTube/Vimeo embed).
	MediaVideo MediaType = "video"
)

// dateFormat is the date layout used by the NASA APOD and NeoWs APIs.
const dateFormat = "2006-01-02"

// apodEarliestDate is the earliest date for which an APOD entry exists.
var apodEarliestDate = time.Date(1995, 6, 16, 0, 0, 0, 0, time.UTC)

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// APODImage represents a single Astronomy Picture of the Day entry.
type APODImage struct {
	Date           time.Time `json:"-"`
	Title          string    `json:"title"`
	Explanation    string    `json:"explanation"`
	URL            string    `json:"url"`
	HDURL          string    `json:"hdurl"`
	MediaType      MediaType `json:"media_type"`
	Copyright      string    `json:"copyright"`
	ThumbnailURL   string    `json:"thumbnail_url"`
	ServiceVersion string    `json:"service_version"`
}

// UnmarshalJSON implements custom JSON unmarshalling for APODImage,
// parsing the "date" string field into a time.Time value.
func (a *APODImage) UnmarshalJSON(data []byte) error {
	type Alias APODImage
	aux := &struct {
		Date string `json:"date"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.Date != "" {
		t, err := time.Parse(dateFormat, aux.Date)
		if err != nil {
			return fmt.Errorf("nasa: invalid APOD date %q: %w", aux.Date, err)
		}
		a.Date = t
	}
	return nil
}

// decodeAPODResponse unmarshals a response body into APODImage values.
// When single is true, the body is expected to be a single JSON object;
// when false, a JSON array.
func decodeAPODResponse(body []byte, single bool) ([]*APODImage, error) {
	if single {
		var img APODImage
		if err := json.Unmarshal(body, &img); err != nil {
			return nil, err
		}
		return []*APODImage{&img}, nil
	}
	var imgs []*APODImage
	if err := json.Unmarshal(body, &imgs); err != nil {
		return nil, err
	}
	return imgs, nil
}

// APODService provides access to the Astronomy Picture of the Day API.
type APODService struct {
	client *Client
}

// Today retrieves today's Astronomy Picture of the Day. The NASA API
// determines "today" based on its own timezone, avoiding client-side
// timezone mismatches.
func (s *APODService) Today(ctx context.Context) (*APODImage, error) {
	params := url.Values{}
	params.Set("thumbs", "true")

	resp, err := s.client.get(ctx, "/planetary/apod", params, true)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	imgs, err := decodeAPODResponse(body, true)
	if err != nil {
		return nil, err
	}
	return imgs[0], nil
}

// Get retrieves the APOD for a specific date. The date must be between
// June 16, 1995 (the first APOD) and today.
func (s *APODService) Get(ctx context.Context, date time.Time) (*APODImage, error) {
	normalized := startOfDay(date)
	normalizedDate := normalized.Format(dateFormat)
	earliestDate := apodEarliestDate.Format(dateFormat)

	if normalizedDate < earliestDate {
		return nil, fmt.Errorf("nasa: date %s is before the earliest APOD (%s)",
			normalizedDate, earliestDate)
	}
	todayDate := startOfDay(time.Now().In(normalized.Location())).Format(dateFormat)
	if normalizedDate > todayDate {
		return nil, fmt.Errorf("nasa: date %s is in the future", normalizedDate)
	}

	params := url.Values{}
	params.Set("date", normalizedDate)
	params.Set("thumbs", "true")

	resp, err := s.client.get(ctx, "/planetary/apod", params, true)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	imgs, err := decodeAPODResponse(body, true)
	if err != nil {
		return nil, err
	}
	return imgs[0], nil
}

// GetRange retrieves APOD entries for a date range (inclusive). Both dates
// must be within the valid APOD range and start must not be after end.
func (s *APODService) GetRange(ctx context.Context, start, end time.Time) ([]*APODImage, error) {
	startDay := startOfDay(start)
	endDay := startOfDay(end)
	startDate := startDay.Format(dateFormat)
	endDate := endDay.Format(dateFormat)
	earliestDate := apodEarliestDate.Format(dateFormat)

	if startDate > endDate {
		return nil, fmt.Errorf("nasa: start date %s is after end date %s",
			startDate, endDate)
	}
	if startDate < earliestDate {
		return nil, fmt.Errorf("nasa: start date %s is before the earliest APOD (%s)",
			startDate, earliestDate)
	}
	todayDate := startOfDay(time.Now().In(endDay.Location())).Format(dateFormat)
	if endDate > todayDate {
		return nil, fmt.Errorf("nasa: end date %s is in the future", endDate)
	}

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)
	params.Set("thumbs", "true")

	resp, err := s.client.get(ctx, "/planetary/apod", params, true)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	return decodeAPODResponse(body, false)
}

// Random retrieves the specified number of random APOD entries. Count must
// be at least 1. Note that large counts can be slow because the NASA API
// shuffles all ~11,000 dates and fetches them sequentially.
func (s *APODService) Random(ctx context.Context, count int) ([]*APODImage, error) {
	if count < 1 {
		return nil, fmt.Errorf("nasa: count must be at least 1, got %d", count)
	}

	params := url.Values{}
	params.Set("count", fmt.Sprintf("%d", count))
	params.Set("thumbs", "true")

	resp, err := s.client.get(ctx, "/planetary/apod", params, true)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp.Body)

	body, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	return decodeAPODResponse(body, false)
}
