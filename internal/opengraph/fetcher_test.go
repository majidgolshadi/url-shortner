package opengraph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOgTags_FullOgTags(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Test Title" />
<meta property="og:description" content="Test Description" />
<meta property="og:image" content="https://example.com/image.jpg" />
<meta property="og:type" content="article" />
<meta property="og:site_name" content="Example Site" />
<meta property="og:url" content="https://example.com/page" />
</head>
<body></body>
</html>`

	og := parseOgTags(htmlContent)
	assert.Equal(t, "Test Title", og.Title)
	assert.Equal(t, "Test Description", og.Description)
	assert.Equal(t, "https://example.com/image.jpg", og.Image)
	assert.Equal(t, "article", og.Type)
	assert.Equal(t, "Example Site", og.SiteName)
	assert.Equal(t, "https://example.com/page", og.URL)
}

func TestParseOgTags_FallbackToTitleTag(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
<title>Fallback Title</title>
<meta name="description" content="Fallback Description" />
</head>
<body></body>
</html>`

	og := parseOgTags(htmlContent)
	assert.Equal(t, "Fallback Title", og.Title)
	assert.Equal(t, "Fallback Description", og.Description)
}

func TestParseOgTags_OgOverridesFallback(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
<title>Fallback Title</title>
<meta property="og:title" content="OG Title" />
<meta name="description" content="Fallback Description" />
<meta property="og:description" content="OG Description" />
</head>
<body></body>
</html>`

	og := parseOgTags(htmlContent)
	assert.Equal(t, "OG Title", og.Title)
	assert.Equal(t, "OG Description", og.Description)
}

func TestParseOgTags_EmptyHTML(t *testing.T) {
	og := parseOgTags("")
	assert.Equal(t, "", og.Title)
	assert.Equal(t, "", og.Description)
	assert.Equal(t, "", og.Image)
}

func TestParseOgTags_NoMetaTags(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head></head>
<body><p>Hello</p></body>
</html>`

	og := parseOgTags(htmlContent)
	assert.Equal(t, "", og.Title)
	assert.Equal(t, "", og.Description)
}

func TestParseOgTags_SpecialCharactersInContent(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Title with &quot;quotes&quot; &amp; stuff" />
<meta property="og:description" content="Description &lt;with&gt; tags" />
</head>
<body></body>
</html>`

	og := parseOgTags(htmlContent)
	assert.Equal(t, `Title with "quotes" & stuff`, og.Title)
	assert.Equal(t, "Description <with> tags", og.Description)
}

func TestRenderOgHTML_FullData(t *testing.T) {
	og := ogData{
		Title:       "Test Title",
		Description: "Test Description",
		Image:       "https://example.com/image.jpg",
		Type:        "article",
		SiteName:    "Example",
		URL:         "https://example.com/page",
	}

	result := renderOgHTML(og)
	assert.Contains(t, result, `<meta property="og:title" content="Test Title" />`)
	assert.Contains(t, result, `<meta property="og:description" content="Test Description" />`)
	assert.Contains(t, result, `<meta property="og:image" content="https://example.com/image.jpg" />`)
	assert.Contains(t, result, `<meta property="og:type" content="article" />`)
	assert.Contains(t, result, `<meta property="og:site_name" content="Example" />`)
	assert.Contains(t, result, `<meta property="og:url" content="https://example.com/page" />`)
}

func TestRenderOgHTML_EmptyData(t *testing.T) {
	og := ogData{}
	result := renderOgHTML(og)
	assert.Equal(t, "", result)
}

func TestRenderOgHTML_OnlyTitle(t *testing.T) {
	og := ogData{Title: "Only Title"}
	result := renderOgHTML(og)
	assert.Equal(t, `<meta property="og:title" content="Only Title" />`, result)
}

func TestRenderOgHTML_EscapesSpecialCharacters(t *testing.T) {
	og := ogData{
		Title:       `Title with "quotes" & <tags>`,
		Description: "Normal description",
	}
	result := renderOgHTML(og)
	assert.Contains(t, result, `content="Title with &#34;quotes&#34; &amp; &lt;tags&gt;"`)
}

func TestFetchOgHTML_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Server Title" />
<meta property="og:description" content="Server Description" />
</head>
<body></body>
</html>`))
	}))
	defer server.Close()

	fetcher := NewFetcher(5)
	result := fetcher.FetchOgHTML(context.Background(), server.URL)

	assert.Contains(t, result, `<meta property="og:title" content="Server Title" />`)
	assert.Contains(t, result, `<meta property="og:description" content="Server Description" />`)
}

func TestFetchOgHTML_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := NewFetcher(5)
	result := fetcher.FetchOgHTML(context.Background(), server.URL)
	assert.Equal(t, "", result)
}

func TestFetchOgHTML_InvalidURL(t *testing.T) {
	fetcher := NewFetcher(5)
	result := fetcher.FetchOgHTML(context.Background(), "http://invalid-host-that-does-not-exist.example.com")
	assert.Equal(t, "", result)
}

func TestFetchOgHTML_NoOgTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head></head><body>No OG</body></html>`))
	}))
	defer server.Close()

	fetcher := NewFetcher(5)
	result := fetcher.FetchOgHTML(context.Background(), server.URL)
	assert.Equal(t, "", result)
}