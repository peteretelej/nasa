// Legacy APOD types and convenience functions for backward compatibility.

package nasa

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"
)

// APIKEYMissing is the message displayed when no custom API key is configured.
const APIKEYMissing = `Currently using the demo API Key DEMO_KEY. Apply for an API key at https://api.nasa.gov/index.html#apply-for-an-api-key
Check out https://github.com/peteretelej/nasa/blob/master/README.md for more info.
`

// APODEndpoint is the NASA APOD API endpoint URL.
var APODEndpoint = "https://api.nasa.gov/planetary/apod"

// defaultAPODEndpoint is the default value of APODEndpoint, used to detect
// user overrides.
const defaultAPODEndpoint = "https://api.nasa.gov/planetary/apod"

// Image is the legacy APOD response type. New code should use [APODImage]
// via the [Client].APOD service instead.
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

// caches todays APOD
type todaysAPOD struct {
	mu   sync.RWMutex // protects the following
	date string       // YYYY-MM-DD
	apod *Image
}

var tAPOD = &todaysAPOD{}

func (v *todaysAPOD) update(apod Image) {
	v.mu.Lock()
	v.apod = &apod
	v.date = apod.Date
	v.mu.Unlock()
}

var (
	defaultClient     *Client
	defaultClientOnce = &sync.Once{}
)

// getDefaultClient returns a lazily-initialized shared Client.
// All convenience wrappers share this client (and its rate limiter),
// so rate limiting actually works across calls.
//
// Endpoint variable values (APODEndpoint, NeoEndpoint) are captured at
// first-call time due to sync.Once; later changes have no effect.
func getDefaultClient() *Client {
	defaultClientOnce.Do(func() {
		var opts []Option

		// If the user changed either endpoint variable from its default,
		// extract the scheme+host as a custom base URL.
		for _, ep := range []struct{ current, dflt string }{
			{APODEndpoint, defaultAPODEndpoint},
			{NeoEndpoint, defaultNeoEndpoint},
		} {
			if ep.current != ep.dflt {
				if u, err := url.Parse(ep.current); err == nil && u.Scheme != "" && u.Host != "" {
					opts = append(opts, WithBaseURL(u.Scheme+"://"+u.Host))
					break // first override wins
				}
			}
		}

		defaultClient = NewClient(opts...)
	})
	return defaultClient
}

// apodImageToImage converts an APODImage to the legacy Image type.
// The conversion is lossy: MediaType, Copyright, ThumbnailURL are dropped.
func apodImageToImage(a *APODImage) *Image {
	return &Image{
		Date:        a.Date.Format(dateFormat),
		Title:       a.Title,
		URL:         a.URL,
		HDURL:       a.HDURL,
		Explanation: a.Explanation,
		ApodDate:    a.Date,
	}
}

// RandomAPOD returns a random Astronomy Picture of the Day. It uses a shared
// internal client that reads the API key from the environment. For full
// control, use [Client].APOD.Random.
func RandomAPOD() (*Image, error) {
	c := getDefaultClient()
	imgs, err := c.APOD.Random(context.Background(), 1)
	if err != nil {
		return nil, err
	}
	if len(imgs) == 0 {
		return nil, fmt.Errorf("nasa: APOD random returned no results")
	}
	return apodImageToImage(imgs[0]), nil
}

// APODToday returns today's Astronomy Picture of the Day, serving from cache
// when available. It uses a shared internal client that reads the API key from
// the environment. For full control, use [Client].APOD.Today.
func APODToday() (*Image, error) {
	d := time.Now().Format(dateFormat)

	tAPOD.mu.RLock()
	cacheddate, apod := tAPOD.date, tAPOD.apod
	tAPOD.mu.RUnlock()

	if cacheddate == d && apod != nil {
		cached := *apod
		return &cached, nil
	}

	c := getDefaultClient()
	apodImg, err := c.APOD.Today(context.Background())
	if err != nil {
		return nil, err
	}

	img := apodImageToImage(apodImg)
	if time.Now().Format(dateFormat) == img.Date {
		tAPOD.update(*img)
	}
	return img, nil
}

// ApodImage returns the Astronomy Picture of the Day for the given date. It
// uses a shared internal client that reads the API key from the environment.
// For full control, use [Client].APOD.Get.
func ApodImage(t time.Time) (*Image, error) {
	if t.After(time.Now()) {
		t = time.Now()
	}

	c := getDefaultClient()
	apodImg, err := c.APOD.Get(context.Background(), t)
	if err != nil {
		return nil, err
	}

	img := apodImageToImage(apodImg)
	if time.Now().Format(dateFormat) == img.Date {
		tAPOD.update(*img)
	}
	return img, nil
}
