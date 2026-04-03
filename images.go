// Image and Video Library service types and methods
// (images-api.nasa.gov).

package nasa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ImageService provides access to the NASA Image and Video Library API
// (images-api.nasa.gov). No API key required.
type ImageService struct {
	client  *Client
	baseURL string
}

// --- Collection+JSON internal parsing types ---

type collectionJSON struct {
	Collection struct {
		Items    []collectionItem   `json:"items"`
		Metadata collectionMetadata `json:"metadata"`
		Links    []collectionLink   `json:"links"`
	} `json:"collection"`
}

type collectionItem struct {
	Data  []json.RawMessage `json:"data"`
	Links []collectionLink  `json:"links"`
	Href  string            `json:"href"`
}

type collectionLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Render string `json:"render,omitempty"`
}

type collectionMetadata struct {
	TotalHits int `json:"total_hits"`
}

// --- Public response types ---

// ImageCollection holds the result of an Image Library search, including
// HATEOAS pagination links.
type ImageCollection struct {
	Items     []*ImageItem
	TotalHits int
	NextPage  string // HATEOAS next link URL, empty if last page
	PrevPage  string // HATEOAS prev link URL, empty if first page
}

// ImageItem represents a single result from the NASA Image Library.
type ImageItem struct {
	NasaID       string    `json:"nasa_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	MediaType    string    `json:"media_type"`
	Center       string    `json:"center,omitempty"`
	DateCreated  time.Time `json:"date_created"`
	Keywords     []string  `json:"keywords,omitempty"`
	Photographer string    `json:"photographer,omitempty"`
	Location     string    `json:"location,omitempty"`
	PreviewURL   string    // from item links with rel="preview"
	AssetURL     string    // item's href (collection.json URL)
}

// AssetManifest holds the list of asset variants for a given NASA ID.
type AssetManifest struct {
	Items []AssetItem
}

// AssetItem represents a single asset variant (different sizes/formats).
type AssetItem struct {
	Href string `json:"href"`
}

// --- Functional options ---

// ImageSearchOption configures optional query parameters for image search.
type ImageSearchOption func(url.Values)

// ImageWithMediaType filters results by media type ("image", "video", "audio").
func ImageWithMediaType(t string) ImageSearchOption {
	return func(v url.Values) { v.Set("media_type", t) }
}

// ImageWithYearStart filters results to items created on or after the given year.
func ImageWithYearStart(year int) ImageSearchOption {
	return func(v url.Values) { v.Set("year_start", strconv.Itoa(year)) }
}

// ImageWithYearEnd filters results to items created on or before the given year.
func ImageWithYearEnd(year int) ImageSearchOption {
	return func(v url.Values) { v.Set("year_end", strconv.Itoa(year)) }
}

// ImageWithCenter filters results by NASA center (e.g. "JPL", "KSC").
func ImageWithCenter(center string) ImageSearchOption {
	return func(v url.Values) { v.Set("center", center) }
}

// ImageWithKeywords filters results by one or more keywords.
func ImageWithKeywords(kw ...string) ImageSearchOption {
	return func(v url.Values) { v.Set("keywords", strings.Join(kw, ",")) }
}

// ImageWithPage sets the page number for paginated results.
func ImageWithPage(page int) ImageSearchOption {
	return func(v url.Values) { v.Set("page", strconv.Itoa(page)) }
}

// ImageWithPageSize sets the number of results per page.
func ImageWithPageSize(size int) ImageSearchOption {
	return func(v url.Values) { v.Set("page_size", strconv.Itoa(size)) }
}

// --- API methods ---

// Search queries the NASA Image and Video Library for items matching the given
// query string. Results are returned as a paginated collection with HATEOAS
// next/prev links.
func (s *ImageService) Search(ctx context.Context, query string, opts ...ImageSearchOption) (*ImageCollection, error) {
	params := url.Values{}
	params.Set("q", query)
	for _, o := range opts {
		o(params)
	}

	resp, err := s.client.getExternal(ctx, s.baseURL, "search", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var coll collectionJSON
	if err := json.Unmarshal(body, &coll); err != nil {
		return nil, fmt.Errorf("nasa: images: failed to parse response: %w", err)
	}

	result := &ImageCollection{
		TotalHits: coll.Collection.Metadata.TotalHits,
		Items:     make([]*ImageItem, 0, len(coll.Collection.Items)),
	}

	// Extract HATEOAS pagination links, fixing http:// to https://.
	for _, link := range coll.Collection.Links {
		href := strings.Replace(link.Href, "http://", "https://", 1)
		switch link.Rel {
		case "next":
			result.NextPage = href
		case "prev":
			result.PrevPage = href
		}
	}

	// Parse each collection item into an ImageItem.
	for _, item := range coll.Collection.Items {
		if len(item.Data) == 0 {
			continue
		}

		var img ImageItem
		if err := json.Unmarshal(item.Data[0], &img); err != nil {
			continue // skip malformed items
		}

		// Extract preview URL from item links.
		for _, link := range item.Links {
			if link.Rel == "preview" {
				img.PreviewURL = link.Href
				break
			}
		}

		// Asset URL is the item's href (collection.json for this asset).
		img.AssetURL = item.Href

		result.Items = append(result.Items, &img)
	}

	return result, nil
}

// Asset retrieves the asset manifest for the given NASA ID, listing all
// available size/format variants.
func (s *ImageService) Asset(ctx context.Context, nasaID string) (*AssetManifest, error) {
	path := "asset/" + url.PathEscape(nasaID)

	resp, err := s.client.getExternal(ctx, s.baseURL, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var coll collectionJSON
	if err := json.Unmarshal(body, &coll); err != nil {
		return nil, fmt.Errorf("nasa: images: failed to parse asset response: %w", err)
	}

	manifest := &AssetManifest{
		Items: make([]AssetItem, 0, len(coll.Collection.Items)),
	}
	for _, item := range coll.Collection.Items {
		if item.Href != "" {
			manifest.Items = append(manifest.Items, AssetItem{Href: item.Href})
		}
	}

	return manifest, nil
}

// Metadata retrieves EXIF/XMP metadata for the given NASA ID. The returned
// map has variable structure depending on the asset.
func (s *ImageService) Metadata(ctx context.Context, nasaID string) (map[string]any, error) {
	path := "metadata/" + url.PathEscape(nasaID)

	resp, err := s.client.getExternal(ctx, s.baseURL, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("nasa: images: failed to parse metadata response: %w", err)
	}

	return result, nil
}

// Captions retrieves the SRT subtitle text for the given NASA ID (video assets).
// The response is plain text, not JSON.
func (s *ImageService) Captions(ctx context.Context, nasaID string) (string, error) {
	path := "captions/" + url.PathEscape(nasaID)

	resp, err := s.client.getExternal(ctx, s.baseURL, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
