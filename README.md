# nasa - Go library, CLI and apps based on the NASA API

- Library for accessing and using the NASA API (APOD, NEO)
- Command line interface (CLI) for accessing NASA API's services
- Applications using the library (e.g. web server)

## NASA API KEY

Kindly grab a NASA API key from [here](https://api.nasa.gov/index.html#apply-for-an-api-key), and set it to the environment variable __NASAKEY__.
```
export NASAKEY=YOUR-API_KEY
```
The API Key will increase the rate limits for your API to access the NASA API. This package & its apps default to the demo key `NASAKEY=DEMO_KEY` if you haven't set one. 
The DEMO_KEY has very low limits (30reqs/hr, 50req/day), only sufficient for demoing.


## nasa Library Usage
``` go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/peteretelej/nasa"
)

func main() {
	apod, err := nasa.ApodImage(time.Now())
	handle(err)
	fmt.Println(apod)

	// apod has structure of nasa.Image, hence get details with:
	// apod.Date, apod.Title, apod.Explanation, apod.URL, apod.HDURL etc
	fmt.Printf("Today's APOD is %s, available at %s", apod.Title, apod.HDURL)

	lastweek := time.Now().Add(-(7 * 24 * time.Hour))
	apod, err = nasa.ApodImage(lastweek)
	handle(err)
	fmt.Printf("APOD for 1 week ago:\n%s\n", apod)
}
func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
```

## nasa CLI
``` sh
# installation
go get -u github.com/peteretelej/nasa/cmd/nasa

nasa apod 
# returns the NASA Astronomy Picture of the day

nasa apod -date 2016-01-17 
# returns the NASA APOD for the date specified

nasa neo
# returns Near Earth Objects for today

nasa neo -start 2017-05-10 -end 2017-05-12
# returns Near Earth Objects for the range of dates specified
```

## Serve website for NASA APOD

Serve APOD on the web 
```
nasa web
# serves website at :8080

nasa web -listen localhost:9000
# serves website at localhost:9000
```

__Web server demo:__
- [nasa.etelej.com](https://nasa.etelej.com): NASA Astronomy Picture of the Day (for today)
- [nasa.etelej.com/random-apod](https://nasa.etelej.com/random-apod): Random images (HD images, updated every second, no autoreload)
- [nasa.etelej.com/random-apod?sd=1](https://nasa.etelej.com/random-apod?sd=1): Gets Standard Definition images (lower quality,faster load, saves bandwidth).
- [nasa.etelej.com/random-apod?auto=1](https://nasa.etelej.com/random-apod?auto=1): Automatically reload page (default reload interval: 5 minutes)
- [nasa.etelej.com/random-apod?auto=1&interval=60](https://nasa.etelej.com/random-apod?auto=1&interval=60): Automatically reloads every 1800 seconds (1 hr)
- [nasa.etelej.com/random-apod?sd=1&auto=1&interval=5](https://nasa.etelej.com/random-apod?sd=1&auto=1&interval=5): Automatically reloads SD images every 5 seconds


## NASA Wallpapers Desktop background
- Only support Ubuntu Desktop atm.

Automatically change your desktop wallpaper to randomly selected NASA Astronomy Pictures of the Day.

```
go get -u github.com/peteretelej/github.com/nasa/cmd/nasa-wallpapers

nasa-wallpapers 
# automatically changes wallpaper with a random NASA picture every 10 minutes

nasa-wallpapers -interval 30s
# automatically changes wallpaper every 30seconds
# rem to get and set NASA API KEY to env NASAKEY to avoid ratelimits
```

