// Package main demonstrates the NASA Image Library service by searching for
// Apollo 11 imagery, fetching asset manifests, metadata, and captions.
//
// No API key is required (uses images-api.nasa.gov).
//
// Usage:
//
//	go run ./examples/image-search
package main

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/peteretelej/nasa"
)

func main() {
	client := nasa.NewClient()
	ctx := context.Background()

	// Search for Apollo 11 imagery.
	results, err := client.Images.Search(ctx, "apollo 11")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== NASA Image Library: Apollo 11 ===")
	fmt.Printf("Total hits: %d\n\n", results.TotalHits)

	// Show first 5 results (or fewer if less available).
	limit := 5
	if len(results.Items) < limit {
		limit = len(results.Items)
	}

	for i := 0; i < limit; i++ {
		item := results.Items[i]
		desc := item.Description
		if len(desc) > 100 {
			desc = desc[:100] + "..."
		}
		fmt.Printf("  [%d] %s\n", i+1, item.Title)
		fmt.Printf("      NASA ID:    %s\n", item.NasaID)
		fmt.Printf("      Media Type: %s\n", item.MediaType)
		fmt.Printf("      Description: %s\n\n", desc)
	}

	if limit == 0 {
		fmt.Println("No results found.")
		return
	}

	// Use the first result's NASA ID for subsequent calls.
	nasaID := results.Items[0].NasaID
	fmt.Printf("=== Asset Manifest for %s ===\n", nasaID)

	manifest, err := client.Images.Asset(ctx, nasaID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Asset count: %d\n", len(manifest.Items))

	hrefLimit := 3
	if len(manifest.Items) < hrefLimit {
		hrefLimit = len(manifest.Items)
	}
	for i := 0; i < hrefLimit; i++ {
		fmt.Printf("  %s\n", manifest.Items[i].Href)
	}

	// Fetch metadata for the asset.
	fmt.Printf("\n=== Metadata for %s ===\n", nasaID)

	metadata, err := client.Images.Metadata(ctx, nasaID)
	if err != nil {
		log.Fatal(err)
	}

	// Print up to 5 top-level keys sorted for deterministic output.
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	keyLimit := 5
	if len(keys) < keyLimit {
		keyLimit = len(keys)
	}
	for i := 0; i < keyLimit; i++ {
		v := fmt.Sprintf("%v", metadata[keys[i]])
		if len(v) > 80 {
			v = v[:80] + "..."
		}
		fmt.Printf("  %s: %s\n", keys[i], v)
	}

	// Attempt to fetch captions (only available for video assets).
	fmt.Printf("\n=== Captions for %s ===\n", nasaID)

	captions, err := client.Images.Captions(ctx, nasaID)
	if err != nil {
		fmt.Println("No captions available (expected for image assets)")
	} else {
		if len(captions) > 200 {
			captions = captions[:200] + "..."
		}
		fmt.Println(captions)
	}
}
