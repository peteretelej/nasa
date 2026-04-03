package nasa

// SSDService provides access to the JPL Small-Body Database and CNEOS APIs
// (ssd-api.jpl.nasa.gov). No API key required.
type SSDService struct {
	client  *Client
	baseURL string
}

// Compile-time reference to suppress unused linter until methods are added.
var _ = (*SSDService)(nil)
