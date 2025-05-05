package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sitemap/link"
	"strings"
	"sync"
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

func main() {
	urlFlag := flag.String("url", "https://gobyexample.com/", "The URL to build a sitemap for.")
	maxDepth := flag.Int("depth", 3, "The maximum depth to traverse.")
	flag.Parse()

	if *urlFlag == "" {
		fmt.Println("Usage: go run main.go --url <your-starting-url>")
		log.Fatal("Error: --url flag is required")
	}

	pages, err := buildSitemap(*urlFlag, *maxDepth)
	if err != nil {
		log.Fatalf("Error building sitemap for %s: %v", *urlFlag, err)
	}

	xmlBytes, err := generateXMLSitemap(pages)
	if err != nil {
		log.Fatalf("Error generating XML sitemap: %v", err)
	}

	fmt.Print(string(xmlBytes))
}

// buildSitemap crawls the website starting from startURL up to maxDepth
// and returns a list of unique URLs found within the same domain.
func buildSitemap(startURL string, maxDepth int) ([]string, error) {
	type job struct {
		url   string
		depth int
	}

	const numWorkers = 10

	jobs := make(chan job)
	var tasks sync.WaitGroup
	var visited sync.Map
	var mu sync.Mutex
	finalURLs := []string{startURL}
	visited.Store(startURL, true)

	var workers sync.WaitGroup
	for range numWorkers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := range jobs {
				if j.depth >= maxDepth {
					tasks.Done()
					continue
				}

				foundLinks, err := getAndParseLinks(j.url)
				if err != nil {
					log.Printf("Warning: %v", err)
					tasks.Done()
					continue
				}

				base := getBaseURL(j.url)
				for _, l := range foundLinks {
					abs := resolveURL(base, l)
					if abs == "" {
						continue
					}
					if isSameDomain(startURL, abs) {
						if _, loaded := visited.LoadOrStore(abs, true); !loaded {
							mu.Lock()
							finalURLs = append(finalURLs, abs)
							mu.Unlock()
							tasks.Add(1)
							jobs <- job{url: abs, depth: j.depth + 1}
						}
					}
				}

				tasks.Done()
			}
		}()
	}

	// Kick off the first job
	tasks.Add(1)
	go func() {
		jobs <- job{url: startURL, depth: 0}
	}()

	// Close jobs channel when all tasks are done
	go func() {
		tasks.Wait()
		close(jobs)
	}()

	workers.Wait()
	return finalURLs, nil
}

// getAndParseLinks fetches a URL, reads its body, and parses links.
func getAndParseLinks(urlStr string) ([]link.Link, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL %s: %w", urlStr, err)
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
		log.Printf("Warning: could not parse url %s: %v", urlStr, err)
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
	if strings.HasPrefix(link.Href, "#") || strings.HasPrefix(strings.ToLower(link.Href), "mailto:") || strings.HasPrefix(strings.ToLower(link.Href), "javascript:") || strings.HasPrefix(strings.ToLower(link.Href), "tel:") {
		return ""
	}

	relativeURL, err := url.Parse(link.Href)
	if err != nil {
		log.Printf("Warning: could not parse relative link %s: %v", link.Href, err)
		return ""
	}

	absoluteURL := base.ResolveReference(relativeURL)

	absoluteURL.Fragment = ""
	// absoluteURL.Path = strings.TrimSuffix(absoluteURL.Path, "/")

	return absoluteURL.String()
}

// isSameDomain checks if a target URL belongs to the same host as the original start URL.
func isSameDomain(startURLStr, targetURLStr string) bool {
	start, err := url.Parse(startURLStr)
	if err != nil {
		log.Printf("Warning: could not parse start url %s: %v", startURLStr, err)
		return false
	}
	target, err := url.Parse(targetURLStr)
	if err != nil {
		log.Printf("Warning: could not parse target url %s: %v", targetURLStr, err)
		return false
	}
	return start.Host == target.Host
}

// generateXMLSitemap creates the sitemap XML structure.
func generateXMLSitemap(pages []string) ([]byte, error) {
	toXML := urlset{
		Xmlns: xmlns,
	}
	for _, page := range pages {
		toXML.Urls = append(toXML.Urls, loc{page})
	}

	xmlBytes, err := xml.MarshalIndent(toXML, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	finalXML := append([]byte(xml.Header), xmlBytes...)

	return finalXML, nil
}
