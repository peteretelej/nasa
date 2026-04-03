package nasa

import (
	"context"
	"encoding/json"
	"net/url"
	"time"
)

// NEOService provides access to the Near Earth Object Web Service.
type NEOService struct {
	client *Client
}

// Feed retrieves near-earth objects for a date range (max 7 days).
func (s *NEOService) Feed(ctx context.Context, start, end time.Time) (*NeoList, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")

	params := url.Values{}
	params.Set("start_date", startDate)
	params.Set("end_date", endDate)

	resp, err := s.client.get(ctx, "/neo/rest/v1/feed", params, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var nl NeoList
	if err := json.NewDecoder(resp.Body).Decode(&nl); err != nil {
		return nil, err
	}
	nl.Start = startDate
	nl.End = endDate
	return &nl, nil
}
