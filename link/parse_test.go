package link_test

import (
	"reflect"
	"strings"
	"testing"

	"sitemap/link"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name          string
		html          string
		expectedLinks []link.Link
		expectError   bool
	}{
		{
			name: "Single Simple Link",
			html: `<html><body><a href="/page1">Link 1</a></body></html>`,
			expectedLinks: []link.Link{
				{Href: "/page1", Text: "Link 1"},
			},
			expectError: false,
		},
		{
			name: "Multiple Simple Links",
			html: `
				<html>
				  <body>
				    <a href="/page1">Link 1</a>
				    <div><a href="https://example.com/page2">Link 2</a></div>
				  </body>
				</html>`,
			expectedLinks: []link.Link{
				{Href: "/page1", Text: "Link 1"},
				{Href: "https://example.com/page2", Text: "Link 2"},
			},
			expectError: false,
		},
		{
			name: "Link with Nested Tags and Whitespace",
			html: `
				<a href="/nested">
				  Click <b>here</b>
				  <span> for 	more
				  </span> info!
				</a>`,
			expectedLinks: []link.Link{
				{Href: "/nested", Text: "Click here for more info!"},
			},
			expectError: false,
		},
		{
			name:          "No Links",
			html:          `<html><body><p>Just text.</p><div></div></body></html>`,
			expectedLinks: []link.Link{},
			expectError:   false,
		},
		{
			name: "Link with No Href",
			html: `<a>No Href Here</a>`,
			expectedLinks: []link.Link{
				{Href: "", Text: "No Href Here"},
			},
			expectError: false,
		},
		{
			name: "Link with Empty Href",
			html: `<a href="">Empty Href</a>`,
			expectedLinks: []link.Link{
				{Href: "", Text: "Empty Href"},
			},
			expectError: false,
		},
		{
			name: "Ignore Comments",
			html: `
				<a href="/real">Real Link</a>
				<!-- <a href="/commented">Commented Link</a> -->`,
			expectedLinks: []link.Link{
				{Href: "/real", Text: "Real Link"},
			},
			expectError: false,
		},
		{
			name:          "Link inside Comment",
			html:          `<!-- <a href="/commented">Commented Link</a> -->`,
			expectedLinks: []link.Link{},
			expectError:   false,
		},
		{
			name: "Text Normalization Edge Case",
			html: `<a href="/space">  leading and trailing   <span> internal	tab </span> multiple   spaces </a>`,
			expectedLinks: []link.Link{
				{Href: "/space", Text: "leading and trailing internal tab multiple spaces"},
			},
			expectError: false,
		},
		{
			name:          "Empty HTML",
			html:          ``,
			expectedLinks: []link.Link{},
			expectError:   false,
		},
		{
			name: "Fragment Link Only",
			html: `<a href="#section">Section</a>`,
			expectedLinks: []link.Link{
				{Href: "#section", Text: "Section"},
			},
			expectError: false,
		},
		{
			name: "Link with HTML entities in text",
			html: `<a href="/entity">Ben & Jerry</a>`,
			expectedLinks: []link.Link{
				{Href: "/entity", Text: "Ben & Jerry"},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.html)

			links, err := link.Parse(reader)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Did not expect an error, but got: %v", err)
			}

			if !reflect.DeepEqual(links, tc.expectedLinks) {
				t.Errorf("Parsed links mismatch:\nExpected: %+v\nActual:   %+v", tc.expectedLinks, links)
			}

			if len(links) != len(tc.expectedLinks) {
				t.Errorf("Length mismatch: Expected %d links, got %d", len(tc.expectedLinks), len(links))
			}
		})
	}
}
