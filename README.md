# nasa - Go library and CLI for NASA API

- Library for accessing and using the NASA API (APOD, NEO)
- Command line interface (CLI) for accessing NASA API's services

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
```


## Apps built on this library

#### Random APOD Images server
Random APOD server: generates a random APOD image and serves at a HTTP url
```
go get -u github.com/peteretelej/nasa/cmd/apod-random

apod-random
# launches web server that serves random APOD images from the last two years

apod-random -listen localhost:8000
# launch on a custom port (default :8080)

apod-random -interval 10m
# update images every 10 minutes (default 5s)
```
__DEMO__
- [nasa.etelej.com/random-apod](https://nasa.etelej.com/random-apod): Random images (updated every 5secs)
- [nasa.etelej.com/random-apod?hd=1](https://nasa.etelej.com/random-apod?hd=1): Gets High Definition images.
- [nasa.etelej.com/random-apod?auto=1](https://nasa.etelej.com/random-apod?auto=1): Automatically reloads page (default interval: 15 seconds)
- [nasa.etelej.com/random-apod?auto=1&interval=60](https://nasa.etelej.com/random-apod?auto=1&interval=60): Automatically reloads every 300 seconds (5 mins)
- [nasa.etelej.com/random-apod?hd=1&auto=1](https://nasa.etelej.com/random-apod?hd=1&auto=1): Automatically reloads HD images

