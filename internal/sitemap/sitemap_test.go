package sitemap

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stackdump/tens-city/internal/markdown"
)

func TestGenerateSitemap(t *testing.T) {
	// Create test documents
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "First Post",
				Slug:          "first-post",
				DatePublished: "2025-01-01T00:00:00Z",
				DateModified:  "2025-01-02T00:00:00Z",
				Tags:          []string{"test", "blog"},
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Second Post",
				Slug:          "second-post",
				DatePublished: "2025-01-03T00:00:00Z",
				Tags:          []string{"test"},
				Keywords:      []string{"example"},
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Draft Post",
				Slug:          "draft-post",
				DatePublished: "2025-01-04T00:00:00Z",
				Draft:         true,
			},
		},
	}

	baseURL := "https://example.com"
	data, err := GenerateSitemap(docs, baseURL)
	if err != nil {
		t.Fatalf("GenerateSitemap failed: %v", err)
	}

	// Parse the XML to verify structure
	var urlset URLSet
	if err := xml.Unmarshal(data, &urlset); err != nil {
		t.Fatalf("Failed to parse sitemap XML: %v", err)
	}

	// Check namespace
	if urlset.Xmlns != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Errorf("Expected xmlns to be sitemap namespace, got %s", urlset.Xmlns)
	}

	// Convert to string for easier checking
	xmlStr := string(data)

	// Check required static pages
	if !strings.Contains(xmlStr, "<loc>https://example.com/</loc>") {
		t.Error("Sitemap should include homepage")
	}
	if !strings.Contains(xmlStr, "<loc>https://example.com/posts</loc>") {
		t.Error("Sitemap should include posts page")
	}
	if !strings.Contains(xmlStr, "<loc>https://example.com/tags</loc>") {
		t.Error("Sitemap should include tags page")
	}
	if !strings.Contains(xmlStr, "<loc>https://example.com/rss</loc>") {
		t.Error("Sitemap should include RSS page")
	}

	// Check blog posts (only non-drafts)
	if !strings.Contains(xmlStr, "<loc>https://example.com/posts/first-post</loc>") {
		t.Error("Sitemap should include first post")
	}
	if !strings.Contains(xmlStr, "<loc>https://example.com/posts/second-post</loc>") {
		t.Error("Sitemap should include second post")
	}
	if strings.Contains(xmlStr, "<loc>https://example.com/posts/draft-post</loc>") {
		t.Error("Sitemap should not include draft post")
	}

	// Check tags
	if !strings.Contains(xmlStr, "<loc>https://example.com/tags/test</loc>") {
		t.Error("Sitemap should include test tag")
	}
	if !strings.Contains(xmlStr, "<loc>https://example.com/tags/blog</loc>") {
		t.Error("Sitemap should include blog tag")
	}
	if !strings.Contains(xmlStr, "<loc>https://example.com/tags/example</loc>") {
		t.Error("Sitemap should include example tag from keywords")
	}

	// Check lastmod dates
	if !strings.Contains(xmlStr, "<lastmod>2025-01-02</lastmod>") {
		t.Error("Sitemap should include lastmod date for first post (using DateModified)")
	}
	if !strings.Contains(xmlStr, "<lastmod>2025-01-03</lastmod>") {
		t.Error("Sitemap should include lastmod date for second post (using DatePublished)")
	}

	// Check XML header
	if !strings.HasPrefix(xmlStr, "<?xml") {
		t.Error("Sitemap should include XML header")
	}
}

func TestGenerateSitemapEmpty(t *testing.T) {
	docs := []*markdown.Document{}
	baseURL := "https://example.com"

	data, err := GenerateSitemap(docs, baseURL)
	if err != nil {
		t.Fatalf("GenerateSitemap failed with empty docs: %v", err)
	}

	// Should still include static pages
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "<loc>https://example.com/</loc>") {
		t.Error("Empty sitemap should still include homepage")
	}
	if !strings.Contains(xmlStr, "<loc>https://example.com/posts</loc>") {
		t.Error("Empty sitemap should still include posts page")
	}
}

func TestGenerateSitemapPriority(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Test Post",
				Slug:          "test-post",
				DatePublished: "2025-01-01T00:00:00Z",
				Draft:         false,
			},
		},
	}

	baseURL := "https://example.com"
	data, err := GenerateSitemap(docs, baseURL)
	if err != nil {
		t.Fatalf("GenerateSitemap failed: %v", err)
	}

	var urlset URLSet
	if err := xml.Unmarshal(data, &urlset); err != nil {
		t.Fatalf("Failed to parse sitemap XML: %v", err)
	}

	// Find specific URLs and check their priorities
	for _, url := range urlset.URLs {
		switch url.Loc {
		case "https://example.com/":
			if url.Priority != 1.0 {
				t.Errorf("Homepage priority should be 1.0, got %.1f", url.Priority)
			}
		case "https://example.com/posts":
			if url.Priority != 0.9 {
				t.Errorf("Posts page priority should be 0.9, got %.1f", url.Priority)
			}
		case "https://example.com/posts/test-post":
			if url.Priority != 0.8 {
				t.Errorf("Blog post priority should be 0.8, got %.1f", url.Priority)
			}
		}
	}
}
