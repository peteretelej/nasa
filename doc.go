// Package nasa provides a Go client for NASA's public APIs.
//
// # Client API (recommended)
//
// The recommended way to use this package is through the Client:
//
//	client := nasa.NewClient(nasa.WithAPIKey("your-key"))
//	apod, err := client.APOD.Today(context.Background())
//	neo, err := client.NEO.Feed(context.Background(), start, end)
//
// The Client supports rate limiting, key rotation, and sub-services
// for each NASA API (APOD, NEO).
//
// # Convenience wrappers
//
// For quick, single-call usage with an explicit API key:
//
//	apod, err := nasa.APODTodayWithKey("your-key")
//	apod, err := nasa.APODGet("your-key", "2024-01-01")
//	neo, err := nasa.NEOFeedWithKey("your-key", "2024-01-01", "2024-01-07")
//
// # Legacy functions
//
// The following functions are retained for backward compatibility. They read
// the API key from the NASA_API_KEY or NASAKEY environment variable, falling
// back to DEMO_KEY:
//
//	apod, err := nasa.APODToday()      // returns *Image (legacy type)
//	apod, err := nasa.ApodImage(t)     // returns *Image (legacy type)
//	apod, err := nasa.RandomAPOD()     // returns *Image (legacy type)
//	neo, err := nasa.NeoFeed(start, end) // returns *NeoList (legacy type)
//
// New code should prefer the Client API or the convenience wrappers above,
// which return the richer APODImage and NEOFeed types.
//
// For best experience, get a NASA API key from https://api.nasa.gov
// and set it:
//
//	export NASA_API_KEY=your-key
package nasa
