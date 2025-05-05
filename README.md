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
**Build the executable:**

Compiles the code and creates an executable file in the current directory by running:
```bash
go build
```
*(If you haven't already, you might need to run `go mod tidy` first to download dependencies like `golang.org/x/net/html`)*


## Usage

Run the compiled executable from your terminal, providing the starting URL via the `--url` flag. You can optionally specify the maximum crawl depth using the `--depth` flag (defaults to 3).

```bash
./sitemap --url <STARTING_URL> [--depth <MAX_DEPTH>]
```

Or you can build sitemap without creating an executable by running:
```bash
go run main.go --url <STARTING_URL> [--depth <MAX_DEPTH>]
```

## Arguments:
`--url <STARTING_URL>`: (Required) The full URL of the website you want to build a sitemap for (e.g., https://example.com).

`--depth <MAX_DEPTH>`: (Optional) The maximum number of links deep to crawl from the starting URL. Defaults to 3. A depth of 0 means only the starting URL will be included.


## Examples:
Crawl gobyexample.com up to 2 levels deep:
```bash
./sitemap-builder --url https://gobyexample.com --depth 2
```

Crawl a local development server (default depth 3):
```bash
./sitemap-builder --url http://localhost:8080
```

Save the output to a file:
Since the output is printed to standard output, you can redirect it to a file:
```bash
./sitemap-builder --url https://example.com > sitemap.xml
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
