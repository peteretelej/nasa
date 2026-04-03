package nasa_test

import (
	"fmt"
	"time"

	"github.com/peteretelej/nasa"
)

// ExampleAPODGet demonstrates fetching the APOD for a specific date using an
// explicit API key. This creates a new client internally; for repeated calls,
// prefer [nasa.NewClient] with [nasa.Client].APOD.Get.
func ExampleAPODGet() {
	apod, err := nasa.APODGet("your-api-key", "2024-01-15")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(apod.Title)
}

// ExampleAPODToday demonstrates fetching today's APOD using the environment
// API key. This uses a shared internal client. For full control, prefer
// [nasa.NewClient] with [nasa.Client].APOD.Today.
func ExampleAPODToday() {
	apod, err := nasa.APODToday()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(apod.Title, apod.Date)
}

// ExampleApodImage demonstrates fetching the APOD for a specific date using
// the environment API key. For full control, prefer [nasa.NewClient] with
// [nasa.Client].APOD.Get.
func ExampleApodImage() {
	apod, err := nasa.ApodImage(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(apod.Title, apod.URL)
}

// ExampleNeoFeed demonstrates fetching near-earth objects for a date range
// using the environment API key. For full control, prefer [nasa.NewClient]
// with [nasa.Client].NEO.Feed.
func ExampleNeoFeed() {
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC)
	neos, err := nasa.NeoFeed(start, end)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("element count:", neos.ElementCount)
}
