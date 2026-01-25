package sitemap

import (
	"encoding/xml"
	"fmt"
	"sort"
	"time"

	"github.com/stackdump/tens-city/pkg/markdown"
)

// URLSet represents a sitemap URL set (sitemap protocol)
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// URL represents a single URL entry in a sitemap
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// GenerateSitemap generates a sitemap.xml for the entire site
func GenerateSitemap(docs []*markdown.Document, baseURL string) ([]byte, error) {
	urlset := &URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	// Add homepage
	urlset.URLs = append(urlset.URLs, URL{
		Loc:        baseURL + "/",
		ChangeFreq: "weekly",
		Priority:   1.0,
	})

	// Add posts list page
	urlset.URLs = append(urlset.URLs, URL{
		Loc:        baseURL + "/posts",
		ChangeFreq: "daily",
		Priority:   0.9,
	})

	// Add tags page
	urlset.URLs = append(urlset.URLs, URL{
		Loc:        baseURL + "/tags",
		ChangeFreq: "weekly",
		Priority:   0.7,
	})

	// Add search page
	urlset.URLs = append(urlset.URLs, URL{
		Loc:        baseURL + "/search",
		ChangeFreq: "monthly",
		Priority:   0.6,
	})

	// Add RSS feeds page
	urlset.URLs = append(urlset.URLs, URL{
		Loc:        baseURL + "/rss",
		ChangeFreq: "weekly",
		Priority:   0.6,
	})

	// Collect all unique tags
	tagSet := make(map[string]struct{})
	for _, doc := range docs {
		if doc.Frontmatter.Draft {
			continue
		}
		for _, tag := range doc.Frontmatter.Tags {
			if tag != "" {
				tagSet[tag] = struct{}{}
			}
		}
		for _, keyword := range doc.Frontmatter.Keywords {
			if keyword != "" {
				tagSet[keyword] = struct{}{}
			}
		}
	}

	// Add tag pages
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		urlset.URLs = append(urlset.URLs, URL{
			Loc:        fmt.Sprintf("%s/tags/%s", baseURL, tag),
			ChangeFreq: "weekly",
			Priority:   0.6,
		})
	}

	// Add individual blog posts
	// Sort by date for consistent ordering
	sort.Slice(docs, func(i, j int) bool {
		dateI, errI := time.Parse(time.RFC3339, docs[i].Frontmatter.DatePublished)
		dateJ, errJ := time.Parse(time.RFC3339, docs[j].Frontmatter.DatePublished)

		if errI != nil && errJ != nil {
			return false
		}
		if errI != nil {
			return false
		}
		if errJ != nil {
			return true
		}

		return dateI.After(dateJ)
	})

	for _, doc := range docs {
		// Skip drafts
		if doc.Frontmatter.Draft {
			continue
		}

		url := URL{
			Loc:        fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
			ChangeFreq: "monthly",
			Priority:   0.8,
		}

		// Use date modified if available, otherwise date published
		var lastMod string
		if doc.Frontmatter.DateModified != "" {
			lastMod = doc.Frontmatter.DateModified
		} else if doc.Frontmatter.DatePublished != "" {
			lastMod = doc.Frontmatter.DatePublished
		}

		// Format as W3C datetime (YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS+00:00)
		if lastMod != "" {
			if t, err := time.Parse(time.RFC3339, lastMod); err == nil {
				url.LastMod = t.Format("2006-01-02")
			}
		}

		urlset.URLs = append(urlset.URLs, url)
	}

	// Generate XML with header
	output, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return nil, err
	}

	// Add XML header
	result := []byte(xml.Header)
	result = append(result, output...)
	return result, nil
}
