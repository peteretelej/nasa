package nasa

// ImageService provides access to the NASA Image and Video Library API
// (images-api.nasa.gov). No API key required.
type ImageService struct {
	client  *Client
	baseURL string
}

// Compile-time reference to suppress unused linter until methods are added.
var _ = (*ImageService)(nil)
