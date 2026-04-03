package nasa

import (
	"context"
	"fmt"
	"time"
)

// APODGet retrieves the APOD for a specific date using the given API key.
// Convenience wrapper. For repeated use, prefer NewClient() to avoid creating
// a new client per call.
func APODGet(apiKey string, date string) (*APODImage, error) {
	t, err := time.Parse(dateFormat, date)
	if err != nil {
		return nil, fmt.Errorf("nasa: invalid date %q: %w", date, err)
	}
	c := NewClient(WithAPIKey(apiKey))
	return c.APOD.Get(context.Background(), t)
}

// APODTodayWithKey retrieves today's APOD using the given API key.
// Convenience wrapper. For repeated use, prefer NewClient() to avoid creating
// a new client per call.
func APODTodayWithKey(apiKey string) (*APODImage, error) {
	c := NewClient(WithAPIKey(apiKey))
	return c.APOD.Today(context.Background())
}

// NEOFeedWithKey retrieves near-earth objects for a date range using the
// given API key. Convenience wrapper. For repeated use, prefer NewClient()
// to avoid creating a new client per call.
func NEOFeedWithKey(apiKey string, startDate string, endDate string) (*NEOFeed, error) {
	start, err := time.Parse(dateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("nasa: invalid start date %q: %w", startDate, err)
	}
	end, err := time.Parse(dateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("nasa: invalid end date %q: %w", endDate, err)
	}
	c := NewClient(WithAPIKey(apiKey))
	return c.NEO.Feed(context.Background(), start, end)
}
