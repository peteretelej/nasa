// Package nasa provides a Go client for six NASA public APIs:
//
//   - APOD (Astronomy Picture of the Day)
//   - NeoWs (Near Earth Object Web Service)
//   - DONKI (Space Weather Database Of Notifications, Knowledge, Information)
//   - NASA Image and Video Library
//   - SSD/CNEOS (JPL Small-Body Database, close approaches, fireballs, Sentry, Scout, NHATS)
//   - EONET (Earth Observatory Natural Event Tracker)
//
// # Quick Start
//
// Create a client and fetch today's Astronomy Picture of the Day:
//
//	client := nasa.NewClient(nasa.WithAPIKey("your-key"))
//	apod, err := client.APOD.Today(context.Background())
//	fmt.Println(apod.Title, apod.URL)
//
// # Rate Limiting and Multi-Key Rotation
//
// The client includes built-in token-bucket rate limiting that respects NASA's
// hourly quotas. When multiple API keys are supplied via [WithAPIKeys], the
// client rotates through them automatically and places rate-limited keys into
// cooldown, maximizing throughput:
//
//	client := nasa.NewClient(nasa.WithAPIKeys([]string{"key1", "key2", "key3"}))
//
// # DEMO_KEY Behavior
//
// If no API key is provided, the client reads from the NASA_API_KEY or NASAKEY
// environment variable, falling back to DEMO_KEY. DEMO_KEY is heavily
// rate-limited (30 requests per hour, 50 per day). For production use, register
// for a free API key at https://api.nasa.gov.
//
// # Convenience Wrappers
//
// For quick, one-off calls the package provides convenience functions that
// accept an explicit API key:
//
//	apod, err := nasa.APODGet("your-key", "2024-01-15")
//	apod, err := nasa.APODTodayWithKey("your-key")
//	neo, err := nasa.NEOFeedWithKey("your-key", "2024-01-15", "2024-01-22")
//
// # Legacy Functions
//
// The following functions are retained for backward compatibility. They use a
// shared internal client that reads the API key from the environment:
//
//	apod, err := nasa.APODToday()
//	apod, err := nasa.ApodImage(t)
//	apod, err := nasa.RandomAPOD()
//	neo, err := nasa.NeoFeed(start, end)
//
// New code should prefer the [Client] API or the convenience wrappers above,
// which return the richer [APODImage] and [NEOFeed] types.
//
// Full documentation: https://peteretelej.github.io/nasa
package nasa
