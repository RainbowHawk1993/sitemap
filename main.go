package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sitemap/link"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Define the XML structure for the sitemap
const xmlns = "http://www.sitemaps.org/schemas/sitemap/0.9"

type loc struct {
	Value string `xml:"loc"`
}

type urlset struct {
	Urls  []loc  `xml:"url"`
	Xmlns string `xml:"xmlns,attr"`
}

var ignoredExtensions = map[string]struct{}{
	".txt": {}, ".pdf": {}, ".doc": {}, ".docx": {}, ".xls": {}, ".xlsx": {},
	".ppt": {}, ".pptx": {}, ".zip": {}, ".rar": {}, ".tar": {}, ".gz": {},
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {}, ".svg": {},
	".ico": {}, ".webp": {}, ".mp3": {}, ".mp4": {}, ".avi": {}, ".mov": {},
	".wmv": {}, ".flv": {}, ".css": {}, ".js": {}, ".json": {}, ".xml": {},
	".rss": {}, ".atom": {}, ".webmanifest": {}, ".map": {},
}

func main() {
	urlFlag := flag.String("url", "", "The URL to build a sitemap for. (Required)")
	maxDepth := flag.Int("depth", 3, "The maximum depth to traverse.")
	workersFlag := flag.Int("workers", 10, "Number of concurrent workers.")
	statsFlag := flag.Bool("stats", false, "Show periodic crawling stats.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: go run main.go --url <your-starting-url> [options]\n")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --url flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	log.SetOutput(os.Stderr)

	log.Printf("Starting sitemap build for %s (depth: %d, workers: %d)\n", *urlFlag, *maxDepth, *workersFlag)

	pages, err := buildSitemap(*urlFlag, *maxDepth, *workersFlag, *statsFlag)
	if err != nil {
		log.Fatalf("Error building sitemap for %s: %v", *urlFlag, err)
	}

	log.Printf("Finished crawling. Found %d unique pages.\n", len(pages))

	xmlBytes, err := generateXMLSitemap(pages)
	if err != nil {
		log.Fatalf("Error generating XML sitemap: %v", err)
	}

	fmt.Print(string(xmlBytes))
}

// buildSitemap crawls the website starting from startURL up to maxDepth
// and returns a list of unique URLs found within the same domain.
func buildSitemap(startURL string, maxDepth int, numWorkers int, showStats bool) ([]string, error) {
	type job struct {
		url   string
		depth int
	}

	jobs := make(chan job, numWorkers)
	var tasks sync.WaitGroup
	var visited sync.Map
	var mu sync.Mutex
	finalURLs := []string{}

	visited.Store(startURL, true)
	mu.Lock()
	finalURLs = append(finalURLs, startURL)
	mu.Unlock()

	var scannedCount atomic.Int64
	var addedCount atomic.Int64
	var queuedCount atomic.Int64
	var skippedExtCount atomic.Int64

	stopStats := make(chan struct{})
	if showStats {
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			start := time.Now()
			log.Println("Starting crawl...")
			for {
				select {
				case <-ticker.C:
					log.Printf("\rElapsed: %s | Scanned: %d | Added: %d | Queued: %d | Skipped (Ext): %d     ",
						time.Since(start).Round(time.Second),
						scannedCount.Load(),
						addedCount.Load(),
						queuedCount.Load(),
						skippedExtCount.Load())
				case <-stopStats:
					log.Println("\rStats display finished.")
					return
				}
			}
		}()
	}

	var workers sync.WaitGroup
	for i := range numWorkers {
		workers.Add(1)
		go func(workerID int) {
			defer workers.Done()
			for j := range jobs {
				scannedCount.Add(1)
				queuedCount.Add(-1)

				foundLinks, err := getAndParseLinks(j.url)
				if err != nil {
					if !strings.Contains(err.Error(), "content type is not HTML") && !strings.Contains(err.Error(),
						"received non-2xx status code") {
						log.Printf("Warning (URL: %s): %v", j.url, err)
					}
					tasks.Done()
					continue
				}

				if j.depth+1 >= maxDepth {
					tasks.Done()
					continue
				}

				base := getBaseURL(j.url)
				if base == nil {
					tasks.Done()
					continue
				}

				for _, l := range foundLinks {
					abs := resolveURL(base, l)
					if abs == "" {
						continue
					}

					parsedAbs, err := url.Parse(abs)
					if err != nil {
						continue
					}
					ext := strings.ToLower(path.Ext(parsedAbs.Path))
					if _, ignore := ignoredExtensions[ext]; ignore && ext != "" {
						skippedExtCount.Add(1)
						continue
					}

					if isSameDomain(startURL, abs) {
						if _, loaded := visited.LoadOrStore(abs, true); !loaded {
							mu.Lock()
							finalURLs = append(finalURLs, abs)
							mu.Unlock()
							addedCount.Add(1)
							tasks.Add(1)
							queuedCount.Add(1)
							jobs <- job{url: abs, depth: j.depth + 1}
						}
					}
				}

				tasks.Done()
			}
		}(i)
	}

	tasks.Add(1)
	queuedCount.Add(1)
	go func() {
		jobs <- job{url: startURL, depth: 0}
	}()

	go func() {
		tasks.Wait()
		close(jobs)
	}()

	workers.Wait()
	if showStats {
		close(stopStats)
		time.Sleep(50 * time.Millisecond)
	}
	log.Println("\rAll workers finished.")

	return finalURLs, nil
}

// getAndParseLinks fetches a URL, reads its body, and parses links.
func getAndParseLinks(urlStr string) ([]link.Link, error) {
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to GET URL %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-2xx status code %d for %s", resp.StatusCode, urlStr)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "text/html") {
		return nil, fmt.Errorf("content type is not HTML (%s) for %s", contentType, urlStr)
	}

	links, err := link.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse links for %s: %w", urlStr, err)
	}
	return links, nil
}

// getBaseURL parses a URL string and returns its base.
func getBaseURL(urlStr string) *url.URL {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil
	}
	return parsedURL
}

// resolveURL takes a base URL and a potentially relative link,
// returns the absolute URL string, cleaned of fragments.
func resolveURL(base *url.URL, link link.Link) string {
	if base == nil || link.Href == "" {
		return ""
	}
	lowerHref := strings.ToLower(link.Href)
	if strings.HasPrefix(link.Href, "//") {
		newUrl := *base
		newUrl.Host = ""
		newUrl.Path = ""
		newUrl.RawQuery = ""
		newUrl.User = nil
		newUrl.Opaque = link.Href
		relUrl, err := url.Parse(newUrl.String())
		if err != nil {
			return ""
		}
		link.Href = relUrl.String()
	} else if strings.HasPrefix(lowerHref, "#") ||
		strings.HasPrefix(lowerHref, "mailto:") ||
		strings.HasPrefix(lowerHref, "javascript:") ||
		strings.HasPrefix(lowerHref, "tel:") ||
		strings.HasPrefix(lowerHref, "data:") ||
		(strings.Contains(lowerHref, ":") && !strings.HasPrefix(lowerHref, "http:") && !strings.HasPrefix(lowerHref, "https:")) {
		return ""
	}

	relativeURL, err := url.Parse(link.Href)
	if err != nil {
		return ""
	}

	absoluteURL := base.ResolveReference(relativeURL)

	if absoluteURL.Scheme != "http" && absoluteURL.Scheme != "https" {
		return ""
	}

	absoluteURL.Fragment = ""

	return absoluteURL.String()
}

// isSameDomain checks if a target URL belongs to the same host as the original start URL.
func isSameDomain(startURLStr, targetURLStr string) bool {
	start, err := url.Parse(startURLStr)
	if err != nil {
		return false
	}
	target, err := url.Parse(targetURLStr)
	if err != nil {
		return false
	}

	return strings.EqualFold(start.Host, target.Host)
}

// generateXMLSitemap creates the sitemap XML structure.
func generateXMLSitemap(pages []string) ([]byte, error) {
	toXML := urlset{
		Xmlns: xmlns,
	}
	addedUrls := make(map[string]struct{})

	for _, page := range pages {
		if _, err := url.ParseRequestURI(page); err != nil {
			log.Printf("Warning: Skipping invalid URL for XML sitemap: %s (%v)", page, err)
			continue
		}

		if _, exists := addedUrls[page]; !exists {
			toXML.Urls = append(toXML.Urls, loc{page})
			addedUrls[page] = struct{}{}
		}
	}

	xmlBytes, err := xml.MarshalIndent(toXML, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	finalXML := append([]byte(xml.Header), xmlBytes...)

	return finalXML, nil
}
