package nasa_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/peteretelej/nasa"
)

func ExampleNewClient() {
	client := nasa.NewClient(nasa.WithAPIKey("your-api-key"))

	// The client is ready to use with all six service endpoints:
	//   client.APOD, client.NEO, client.DONKI,
	//   client.Images, client.SSD, client.EONET
	_ = client
	fmt.Println("client created")
	// Output: client created
}

func ExampleNewClient_withOptions() {
	client := nasa.NewClient(
		nasa.WithAPIKeys([]string{"key1", "key2"}),
		nasa.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
		nasa.WithRateLimit(5),
	)
	_ = client
	fmt.Println("client with options created")
	// Output: client with options created
}

func ExampleAPODService_Get() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/apod_single.json")
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	client := nasa.NewClient(nasa.WithBaseURL(ts.URL), nasa.WithAPIKey("test"))
	apod, err := client.APOD.Get(context.Background(), time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(apod.Title)
	fmt.Println(apod.MediaType)
	// Output:
	// Star Cluster IC 348 from Webb
	// image
}

func ExampleAPODService_Today() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/apod_single.json")
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	client := nasa.NewClient(nasa.WithBaseURL(ts.URL), nasa.WithAPIKey("test"))
	apod, err := client.APOD.Today(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(apod.Title)
	fmt.Println("copyright:", apod.Copyright)
	// Output:
	// Star Cluster IC 348 from Webb
	// copyright:
}

func ExampleAPODService_Random() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/apod_range.json")
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	client := nasa.NewClient(nasa.WithBaseURL(ts.URL), nasa.WithAPIKey("test"))
	imgs, err := client.APOD.Random(context.Background(), 3)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(imgs), "images")
	fmt.Println(imgs[0].Title)
	fmt.Println(imgs[1].Title)
	fmt.Println(imgs[2].Title)
	// Output:
	// 3 images
	// The Light, the Dark, and the Dusty
	// Quadrantids of the North
	// Good Morning Moon
}

func ExampleNEOService_Feed() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/neo_feed.json")
		if _, err := w.Write(data); err != nil {
			fmt.Println("error:", err)
		}
	}))
	defer ts.Close()

	client := nasa.NewClient(nasa.WithBaseURL(ts.URL), nasa.WithAPIKey("test"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	feed, err := client.NEO.Feed(context.Background(), start, end)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("count:", feed.ElementCount)
	neo := feed.NearEarthObjects["2024-01-15"][0]
	fmt.Println(neo.Name, "hazardous:", neo.IsPotentiallyHazardous)
	// Output:
	// count: 19
	// (2006 AM8) hazardous: false
}

func ExampleDONKIService() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/donki_cme.json")
		if _, err := w.Write(data); err != nil {
			fmt.Println("error:", err)
		}
	}))
	defer ts.Close()

	client := nasa.NewClient(nasa.WithBaseURL(ts.URL), nasa.WithAPIKey("test"))
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	cmes, err := client.DONKI.CME(context.Background(), start, end)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(cmes), "CME events")
	fmt.Println(cmes[0].ActivityID)
	fmt.Println("source:", cmes[0].SourceLocation)
	// Output:
	// 3 CME events
	// 2024-01-10T00:12:00-CME-001
	// source: N15E80
}

func ExampleImageService_Search() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/images_search.json")
		if _, err := w.Write(data); err != nil {
			fmt.Println("error:", err)
		}
	}))
	defer ts.Close()

	// ImageService uses its own base URL, so we use a custom RoundTripper
	// to redirect requests to the test server.
	client := nasa.NewClient(
		nasa.WithAPIKey("test"),
		nasa.WithHTTPClient(&http.Client{
			Transport: rewriteTransport{ts.URL},
		}),
	)
	results, err := client.Images.Search(context.Background(), "apollo")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("total:", results.TotalHits)
	fmt.Println(results.Items[0].NasaID, "-", results.Items[0].Title)
	// Output:
	// total: 1596
	// KSC-19850101-MH-NAS01-0001-Apollo_11_Buzz_Aldrin_Experts-B_2141 - Apollo 11 Productions
}

func ExampleSSDService() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/ssd_cad.json")
		if _, err := w.Write(data); err != nil {
			fmt.Println("error:", err)
		}
	}))
	defer ts.Close()

	client := nasa.NewClient(
		nasa.WithAPIKey("test"),
		nasa.WithHTTPClient(&http.Client{
			Transport: rewriteTransport{ts.URL},
		}),
	)
	cads, err := client.SSD.CloseApproach(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(cads), "close approaches")
	fmt.Println(cads[2].Designation, "dist:", cads[2].Distance, "AU")
	// Output:
	// 3 close approaches
	// 2023 DZ2 dist: 0.00676762046379765 AU
}

func ExampleEONETService() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile("testdata/eonet_events.json")
		if _, err := w.Write(data); err != nil {
			fmt.Println("error:", err)
		}
	}))
	defer ts.Close()

	client := nasa.NewClient(
		nasa.WithAPIKey("test"),
		nasa.WithHTTPClient(&http.Client{
			Transport: rewriteTransport{ts.URL},
		}),
	)
	events, err := client.EONET.Events(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(len(events), "events")
	fmt.Println(events[0].Title)
	fmt.Println("categories:", events[0].Categories[0].Title)
	// Output:
	// 5 events
	// rx-12_ 13_ 14 Prescribed Fire, Burnett, Wisconsin
	// categories: Wildfires
}

// rewriteTransport is an http.RoundTripper that rewrites all request URLs to
// point at a test server. Used in examples for services that have their own
// base URLs (Images, SSD, EONET).
type rewriteTransport struct {
	targetURL string
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.targetURL[len("http://"):]
	return http.DefaultTransport.RoundTrip(req)
}
