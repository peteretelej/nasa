# Images - NASA Image and Video Library

The `ImageService` provides access to NASA's Image and Video Library. No API key is required.

## Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Search` | `Search(ctx context.Context, query string, opts ...ImageSearchOption) (*ImageCollection, error)` | Search images and videos |
| `Asset` | `Asset(ctx context.Context, nasaID string) (*AssetManifest, error)` | Get asset manifest for a NASA ID |
| `Metadata` | `Metadata(ctx context.Context, nasaID string) (map[string]any, error)` | Get metadata for a NASA ID |
| `Captions` | `Captions(ctx context.Context, nasaID string) (string, error)` | Get captions for a video |

## Search Options

```go
nasa.ImageWithMediaType("image")   // filter by type: image, video, audio
nasa.ImageWithYearStart(2020)      // results from this year or later
nasa.ImageWithYearEnd(2024)        // results up to this year
nasa.ImageWithCenter("JPL")        // filter by NASA center
nasa.ImageWithKeywords("mars")     // filter by keywords
nasa.ImageWithPage(2)              // results page number
nasa.ImageWithPageSize(50)         // results per page
```

## Response Types

### ImageCollection

```go
type ImageCollection struct {
    Items     []*ImageItem
    TotalHits int
    NextPage  string
    PrevPage  string
}
```

### ImageItem

```go
type ImageItem struct {
    NasaID       string
    Title        string
    Description  string
    MediaType    string
    Center       string
    DateCreated  time.Time
    Keywords     []string
    Photographer string
    Location     string
    PreviewURL   string
    AssetURL     string
}
```

### AssetManifest

```go
type AssetManifest struct {
    Items []AssetItem
}

type AssetItem struct {
    Href string
}
```

## Example: Search for Apollo 11

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()

	results, err := client.Images.Search(
		context.Background(),
		"apollo 11",
		nasa.ImageWithMediaType("image"),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total hits: %d\n\n", results.TotalHits)
	for _, item := range results.Items {
		fmt.Printf("%s (ID: %s)\n", item.Title, item.NasaID)
		if item.PreviewURL != "" {
			fmt.Printf("  Preview: %s\n", item.PreviewURL)
		}
	}
}
```

## Example: Get Asset URLs

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()

	manifest, err := client.Images.Asset(context.Background(), "as11-40-5874")
	if err != nil {
		log.Fatal(err)
	}

	for _, asset := range manifest.Items {
		fmt.Println(asset.Href)
	}
}
```
