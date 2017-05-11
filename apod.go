package nasa

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

var nasaKey = os.Getenv("NASAKEY")

func init() {
	if nasaKey == "" {
		nasaKey = "DEMO_KEY"
	}
}

const APODEndpoint = "https://api.nasa.gov/planetary/apod"

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
Date: %s
Image: %s
HD Image: %s
About:
%s
`, ni.Title, ni.Date, ni.URL, ni.HDURL, ni.Explanation)
}

// ApodImage returns the NASA Astronomy Picture of the Day
func ApodImage(t time.Time) (*Image, error) {
	fmt.Println(t)
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
	u.RawQuery = q.Encode()
	fmt.Println(u.String())
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
