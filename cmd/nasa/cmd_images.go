package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/peteretelej/nasa"
)

func imagesCmd(args []string) {
	if len(args) == 0 {
		imagesUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "search":
		imagesSearchCmd(args[1:])
	case "asset":
		imagesAssetCmd(args[1:])
	case "metadata":
		imagesMetadataCmd(args[1:])
	case "captions":
		imagesCaptionsCmd(args[1:])
	case "--help", "-help", "-h":
		imagesUsage()
	default:
		fmt.Fprintf(os.Stderr, "nasa images: unknown command %q\n\n", args[0])
		imagesUsage()
		os.Exit(1)
	}
}

func imagesUsage() {
	fmt.Fprintf(os.Stderr, `Usage: nasa images <command> [flags]

Commands:
  search      Search the NASA Image and Video Library
  asset       Get asset manifest for a NASA ID
  metadata    Get metadata for a NASA ID
  captions    Get captions for a NASA ID

Use "nasa images <command> --help" for more information about a command.
`)
}

// imagesSearchCmd handles "nasa images search <query>".
func imagesSearchCmd(args []string) {
	fs := flag.NewFlagSet("images search", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa images search <query> [flags]

Search the NASA Image and Video Library.

Flags:
`)
		fs.PrintDefaults()
	}

	mediaType := fs.String("media-type", "", "media type filter (image, video, audio)")
	yearStart := fs.Int("year-start", 0, "results starting from this year")
	yearEnd := fs.Int("year-end", 0, "results ending at this year")
	center := fs.String("center", "", "NASA center filter")
	page := fs.Int("page", 0, "results page number")
	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	query := strings.Join(fs.Args(), " ")
	if query == "" {
		fatal("images search: search query is required\n\nUsage: nasa images search <query> [flags]")
	}

	var opts []nasa.ImageSearchOption
	if *mediaType != "" {
		opts = append(opts, nasa.ImageWithMediaType(*mediaType))
	}
	if *yearStart > 0 {
		opts = append(opts, nasa.ImageWithYearStart(*yearStart))
	}
	if *yearEnd > 0 {
		opts = append(opts, nasa.ImageWithYearEnd(*yearEnd))
	}
	if *center != "" {
		opts = append(opts, nasa.ImageWithCenter(*center))
	}
	if *page > 0 {
		opts = append(opts, nasa.ImageWithPage(*page))
	}

	client := nasa.NewClient()
	ctx := context.Background()

	collection, err := client.Images.Search(ctx, query, opts...)
	if err != nil {
		fatal("images search: %v", err)
	}

	if *jsonFlag {
		printJSON(collection)
		return
	}

	if len(collection.Items) == 0 {
		fmt.Println("No results found.")
		return
	}

	pageNum := 1
	if *page > 0 {
		pageNum = *page
	}
	fmt.Printf("NASA Image Library: %q (%d results, page %d)\n\n", query, collection.TotalHits, pageNum)

	tw := newTable("Title", "NASA ID", "Type", "Date", "Description")
	for _, item := range collection.Items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			truncate(item.Title, 30),
			item.NasaID,
			item.MediaType,
			formatDate(item.DateCreated),
			truncate(item.Description, 50),
		)
	}
	flushTable(tw)
}

// imagesAssetCmd handles "nasa images asset <nasa-id>".
func imagesAssetCmd(args []string) {
	fs := flag.NewFlagSet("images asset", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa images asset <nasa-id> [flags]

Get asset manifest for a NASA ID.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fatal("images asset: NASA ID is required\n\nUsage: nasa images asset <nasa-id> [flags]")
	}
	nasaID := fs.Arg(0)

	client := nasa.NewClient()
	ctx := context.Background()

	manifest, err := client.Images.Asset(ctx, nasaID)
	if err != nil {
		fatal("images asset: %v", err)
	}

	if *jsonFlag {
		printJSON(manifest)
		return
	}

	fmt.Printf("Asset Manifest: %s\n\n", nasaID)

	tw := newTable("URL")
	for _, item := range manifest.Items {
		fmt.Fprintf(tw, "%s\n", item.Href)
	}
	flushTable(tw)
}

// imagesMetadataCmd handles "nasa images metadata <nasa-id>".
func imagesMetadataCmd(args []string) {
	fs := flag.NewFlagSet("images metadata", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa images metadata <nasa-id> [flags]

Get metadata for a NASA ID.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fatal("images metadata: NASA ID is required\n\nUsage: nasa images metadata <nasa-id> [flags]")
	}
	nasaID := fs.Arg(0)

	client := nasa.NewClient()
	ctx := context.Background()

	metadata, err := client.Images.Metadata(ctx, nasaID)
	if err != nil {
		fatal("images metadata: %v", err)
	}

	if *jsonFlag {
		printJSON(metadata)
		return
	}

	fmt.Printf("Metadata: %s\n\n", nasaID)

	// Sort keys for consistent output.
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%-30s  %v\n", k, metadata[k])
	}
}

// imagesCaptionsCmd handles "nasa images captions <nasa-id>".
func imagesCaptionsCmd(args []string) {
	fs := flag.NewFlagSet("images captions", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nasa images captions <nasa-id> [flags]

Get captions for a NASA ID.

Flags:
`)
		fs.PrintDefaults()
	}

	jsonFlag := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fatal("images captions: NASA ID is required\n\nUsage: nasa images captions <nasa-id> [flags]")
	}
	nasaID := fs.Arg(0)

	client := nasa.NewClient()
	ctx := context.Background()

	captions, err := client.Images.Captions(ctx, nasaID)
	if err != nil {
		fatal("images captions: %v", err)
	}

	if *jsonFlag {
		printJSON(map[string]string{"captions": captions})
		return
	}

	fmt.Println(captions)
}
