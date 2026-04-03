package nasa

// EONETService provides access to the Earth Observatory Natural Event Tracker
// API (eonet.gsfc.nasa.gov/api/v3). API key is optional.
type EONETService struct {
	client  *Client
	baseURL string
	apiKey  string
}

// Compile-time reference to suppress unused linter until methods are added.
var _ = (*EONETService)(nil)
