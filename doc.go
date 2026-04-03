// Package nasa provides a Go client for NASA's public APIs.
//
// The recommended way to use this package is through the Client:
//
//	client := nasa.NewClient(nasa.WithAPIKey("your-key"))
//	apod, err := client.APOD.Today(context.Background())
//
// For quick usage, package-level convenience functions are also available:
//
//	apod, err := nasa.APODToday()
//	neo, err := nasa.NeoFeed(start, end)
//
// These convenience functions read the API key from the NASA_API_KEY
// or NASAKEY environment variable, falling back to DEMO_KEY.
//
// For best experience, get a NASA API key from https://api.nasa.gov
// and set it:
//
//	export NASA_API_KEY=your-key
package nasa
