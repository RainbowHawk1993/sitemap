# Go Sitemap Builder

A command-line tool written in Go to generate XML sitemaps for websites by crawling them. It starts at a given URL, parses HTML to find links, and follows them within the same domain up to a specified depth, producing a standard XML sitemap output.

## Features

*   Crawls a website starting from a given URL.
*   Parses `<a>` tags from HTML to find links (`href` attributes).
*   Generates an XML sitemap conforming to the [Sitemaps protocol version 0.9](https://www.sitemaps.org/protocol.html).
*   Filters links to stay within the original domain (based on the host of the starting URL).
*   Correctly handles both relative (`/contact`) and absolute (`https://domain.com/contact`) URLs.
*   Prevents infinite loops from cyclical links by tracking visited pages.
*   Allows configuration of maximum crawl depth.
*   Outputs the generated XML sitemap to standard output.

## Setup

You might need to run `go mod tidy` first to download dependencies like `golang.org/x/net/html`*

## Usage

Run the sitemap generator by specifying the starting URL. You can customize the crawl behavior with several optional flags:

*   `--url <STARTING_URL>`: **(Required)** The website URL to start crawling from (e.g., `https://example.com/`).
*   `--depth <MAX_DEPTH>`: (Optional) The maximum depth of links to follow relative to the starting URL. Defaults to `3`. A depth of 0 only includes the start URL, 1 includes the start URL and pages linked directly from it, etc.
*   `--workers <N>`: (Optional) The number of concurrent workers (goroutines) used for fetching and parsing pages. Defaults to `10`. Increasing this *may* speed up crawling on sites with many pages, especially if network latency is high. Be mindful of the load this places on the target server.
*   `--stats`: (Optional) If present, displays periodic progress statistics to standard error (`stderr`) during the crawl. This includes elapsed time, pages scanned, unique URLs added to the sitemap, approximate queue size, and URLs skipped due to non-HTML extensions.

### Running the Tool

**Using `go run` (without compiling):**

```bash
# Basic usage
go run main.go --url https://example.com

# Specify depth
go run main.go --url https://example.com --depth 5

# Increase concurrency
go run main.go --url https://example.com --workers 20

# Show progress statistics
go run main.go --url https://example.com --stats

# Combine flags and redirect sitemap output to a file
# (Stats will still appear in your terminal)
go run main.go --url https://example.com --depth 4 --workers 15 --stats > sitemap.xml
```

## Example Output (sitemap.xml)
```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
  </url>
  <url>
    <loc>https://example.com/about</loc>
  </url>
  <url>
    <loc>https://example.com/contact</loc>
  </url>
  <!-- ... more url elements ... -->
</urlset>
```

## Potential Improvements
- Respect robots.txt rules.
- Implement rate limiting or delays between requests to be polite to servers.
- Add support for `<lastmod>`, `<changefreq>`, and `<priority>` tags in the sitemap XML.
- More sophisticated error handling (e.g., retries, handling specific HTTP errors).
- Option to specify an output file path via a command-line flag
