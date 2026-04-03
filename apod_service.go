package nasa

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"time"
)

// APODService provides access to the Astronomy Picture of the Day API.
type APODService struct {
	client *Client
}

// Get retrieves the APOD for a specific date.
func (s *APODService) Get(ctx context.Context, date time.Time) (*Image, error) {
	params := url.Values{}
	if !date.IsZero() {
		params.Set("date", date.Format("2006-01-02"))
	}

	resp, err := s.client.get(ctx, "/planetary/apod", params, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var img Image
	if err := json.NewDecoder(resp.Body).Decode(&img); err != nil {
		return nil, err
	}

	if t, err := time.Parse("2006-01-02", img.Date); err == nil {
		img.ApodDate = t
	}

	// Preserve existing validation: require at least one image URL
	if img.URL == "" && img.HDURL == "" {
		return nil, fmt.Errorf("nasa: APOD response for %s has no image URL", date.Format("2006-01-02"))
	}

	return &img, nil
}

// Today retrieves today's APOD.
func (s *APODService) Today(ctx context.Context) (*Image, error) {
	return s.Get(ctx, time.Now())
}

// Random retrieves an APOD from a random date in the last 2 years.
func (s *APODService) Random(ctx context.Context) (*Image, error) {
	days := 2 * 365
	randDays := time.Duration(rand.Intn(days))
	t := time.Now().Add(-(time.Hour * 24 * randDays))
	return s.Get(ctx, t)
}
