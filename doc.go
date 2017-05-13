// Package nasa provides Go structs and functions for accessing the NASA API
//
// Queries are made with Go types as arguments and responses returned as valid Go types.
// For best experience, grap NASA API key from api.nasa.gov and set it to the environment variable NASAKEY
//   export NASAKEY=Your-NASA-API_KEY
//
// Example Usage
//
//  package main
//
//  import (
//  	"fmt"
//  	"log"
//  	"time"
//
//  	"github.com/peteretelej/nasa"
//  )
//
//  func main() {
//  	apod, err := nasa.ApodImage(time.Now())
//  	handle(err)
//  	fmt.Println(apod)
//
//  	// apod has structure of nasa.Image, hence get details with:
//  	// apod.Date, apod.Title, apod.Explanation, apod.URL, apod.HDURL etc
//  	fmt.Printf("Today's APOD is %s, available at %s", apod.Title, apod.HDURL)
//
//  	lastweek := time.Now().Add(-(7 * 24 * time.Hour))
//  	apod, err = nasa.ApodImage(lastweek)
//  	handle(err)
//  	fmt.Printf("APOD for 1 week ago:\n%s\n", apod)
//  }
//  func handle(err error) {
//  	if err != nil {
//  		log.Fatal(err)
//  	}
//  }
//

package nasa
