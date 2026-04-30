package opengraph

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	htmlParser "golang.org/x/net/html"
)

// Fetcher retrieves Open Graph metadata from a URL.
type Fetcher struct {
	client *http.Client
}

// NewFetcher creates a new OG metadata fetcher with the given timeout.
func NewFetcher(timeoutSec int) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// ogData holds parsed Open Graph values.
type ogData struct {
	Title       string
	Description string
	Image       string
	Type        string
	SiteName    string
	URL         string
}

// FetchOgHTML fetches the given URL, parses its OG meta tags, and returns
// pre-rendered HTML meta tag string ready to be embedded in an HTML response.
// Returns empty string if fetching or parsing fails.
func (f *Fetcher) FetchOgHTML(ctx context.Context, targetURL string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("User-Agent", "url-shortener-og-fetcher/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// OG tags live in <head>, so 1MB is sufficient; reading the full body would waste memory.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}

	og := parseOgTags(string(body))

	return renderOgHTML(og)
}

// parseOgTags extracts OG meta tags and fallback title/description from HTML.
func parseOgTags(htmlContent string) ogData {
	var og ogData
	var fallbackTitle string
	var fallbackDescription string

	tokenizer := htmlParser.NewTokenizer(strings.NewReader(htmlContent))

	for {
		tt := tokenizer.Next()
		switch tt {
		case htmlParser.ErrorToken:
			// End of document or error
			if og.Title == "" && fallbackTitle != "" {
				og.Title = fallbackTitle
			}
			if og.Description == "" && fallbackDescription != "" {
				og.Description = fallbackDescription
			}
			return og

		case htmlParser.StartTagToken, htmlParser.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()
			tagName := string(tn)

			if tagName == "meta" && hasAttr {
				var property, name, content string
				for {
					key, val, more := tokenizer.TagAttr()
					k := string(key)
					v := string(val)
					switch k {
					case "property":
						property = v
					case "name":
						name = v
					case "content":
						content = v
					}
					if !more {
						break
					}
				}

				switch property {
				case "og:title":
					og.Title = content
				case "og:description":
					og.Description = content
				case "og:image":
					og.Image = content
				case "og:type":
					og.Type = content
				case "og:site_name":
					og.SiteName = content
				case "og:url":
					og.URL = content
				}

				if name == "description" && content != "" {
					fallbackDescription = content
				}
			}

			if tagName == "title" {
				tt = tokenizer.Next()
				if tt == htmlParser.TextToken {
					fallbackTitle = strings.TrimSpace(tokenizer.Token().Data)
				}
			}

			// OG meta tags must appear in <head>; stop at <body> to avoid scanning the full document.
			if tagName == "body" {
				if og.Title == "" && fallbackTitle != "" {
					og.Title = fallbackTitle
				}
				if og.Description == "" && fallbackDescription != "" {
					og.Description = fallbackDescription
				}
				return og
			}
		}
	}
}

// renderOgHTML produces pre-rendered HTML meta tags from the parsed OG data.
// Returns empty string if no meaningful data was found.
func renderOgHTML(og ogData) string {
	var sb strings.Builder

	if og.Title == "" && og.Description == "" && og.Image == "" {
		return ""
	}

	if og.Title != "" {
		fmt.Fprintf(&sb, `<meta property="og:title" content="%s" />`, html.EscapeString(og.Title))
		sb.WriteByte('\n')
	}
	if og.Description != "" {
		fmt.Fprintf(&sb, `<meta property="og:description" content="%s" />`, html.EscapeString(og.Description))
		sb.WriteByte('\n')
	}
	if og.Image != "" {
		fmt.Fprintf(&sb, `<meta property="og:image" content="%s" />`, html.EscapeString(og.Image))
		sb.WriteByte('\n')
	}
	if og.Type != "" {
		fmt.Fprintf(&sb, `<meta property="og:type" content="%s" />`, html.EscapeString(og.Type))
		sb.WriteByte('\n')
	}
	if og.SiteName != "" {
		fmt.Fprintf(&sb, `<meta property="og:site_name" content="%s" />`, html.EscapeString(og.SiteName))
		sb.WriteByte('\n')
	}
	if og.URL != "" {
		fmt.Fprintf(&sb, `<meta property="og:url" content="%s" />`, html.EscapeString(og.URL))
		sb.WriteByte('\n')
	}

	return strings.TrimRight(sb.String(), "\n")
}