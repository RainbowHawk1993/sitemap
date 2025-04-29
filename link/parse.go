package link

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

type Link struct {
	Href string
	Text string
}

// Parse will take an HTML document (as an io.Reader) and will return
// a slice of links parsed from it, or an error if parsing fails.
func Parse(r io.Reader) ([]Link, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	nodes := linkNodes(doc)

	var links []Link
	for _, node := range nodes {
		links = append(links, buildLink(node))
	}

	return links, nil
}

// buildLink extracts the href and text from an <a> node
func buildLink(n *html.Node) Link {
	var link Link
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			link.Href = attr.Val
			break
		}
	}
	link.Text = extractText(n)
	return link
}

// extractText recursively extracts all text content from a node and its children,
// cleaning up whitespace.
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}

	if n.Type != html.ElementNode {
		return ""
	}

	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extracted := extractText(c)
		if extracted != "" {
			if sb.Len() > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(extracted)
		}
	}

	return strings.Join(strings.Fields(sb.String()), " ")
}

// linkNodes performs a depth-first search to find all <a> nodes
// within the given HTML node tree.
func linkNodes(n *html.Node) []*html.Node {
	if n.Type == html.ElementNode && n.Data == "a" {
		return []*html.Node{n}
	}

	var nodes []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		nodes = append(nodes, linkNodes(c)...)
	}
	return nodes
}
