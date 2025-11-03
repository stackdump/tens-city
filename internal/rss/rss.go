package rss

import (
	"encoding/xml"
	"fmt"
	"html"
	"sort"
	"time"

	"github.com/stackdump/tens-city/internal/markdown"
)

// RSS represents an RSS 2.0 feed
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel *Channel `xml:"channel"`
}

// Channel represents an RSS channel
type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	Language      string `xml:"language,omitempty"`
	LastBuildDate string `xml:"lastBuildDate,omitempty"`
	Items         []Item `xml:"item"`
}

// Item represents an RSS item
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
	GUID        string `xml:"guid"`
}

// GenerateUserFeed generates an RSS feed for a user's blog posts
func GenerateUserFeed(docs []*markdown.Document, userName, baseURL string) ([]byte, error) {
	// Filter and sort documents by publication date (newest first)
	var validDocs []*markdown.Document
	for _, doc := range docs {
		if !doc.Frontmatter.Draft {
			validDocs = append(validDocs, doc)
		}
	}

	// Sort by date published (newest first)
	sort.Slice(validDocs, func(i, j int) bool {
		dateI, errI := time.Parse(time.RFC3339, validDocs[i].Frontmatter.DatePublished)
		dateJ, errJ := time.Parse(time.RFC3339, validDocs[j].Frontmatter.DatePublished)

		// Handle parse errors: put invalid dates at the end
		if errI != nil && errJ != nil {
			return false // Both invalid, maintain original order
		}
		if errI != nil {
			return false // i is invalid, put it after j
		}
		if errJ != nil {
			return true // j is invalid, put i before it
		}

		return dateI.After(dateJ)
	})

	// Limit to most recent items
	const maxRSSItems = 20
	if len(validDocs) > maxRSSItems {
		validDocs = validDocs[:maxRSSItems]
	}

	// Build RSS feed
	channel := &Channel{
		Title:       fmt.Sprintf("%s's Blog Posts - Tens City", html.EscapeString(userName)),
		Link:        fmt.Sprintf("%s/u/%s", baseURL, userName),
		Description: fmt.Sprintf("Latest blog posts from %s", html.EscapeString(userName)),
		Language:    "en",
	}

	// Set last build date from the most recent document
	if len(validDocs) > 0 {
		if t, err := time.Parse(time.RFC3339, validDocs[0].Frontmatter.DatePublished); err == nil {
			channel.LastBuildDate = t.Format(time.RFC1123Z)
		}
	}

	// Convert documents to RSS items
	for _, doc := range validDocs {
		item := Item{
			Title:       html.EscapeString(doc.Frontmatter.Title),
			Link:        fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
			Description: html.EscapeString(doc.Frontmatter.Description),
			GUID:        fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
		}

		// Parse and format publication date
		if doc.Frontmatter.DatePublished != "" {
			if t, err := time.Parse(time.RFC3339, doc.Frontmatter.DatePublished); err == nil {
				item.PubDate = t.Format(time.RFC1123Z)
			}
		}

		channel.Items = append(channel.Items, item)
	}

	feed := &RSS{
		Version: "2.0",
		Channel: channel,
	}

	return xml.MarshalIndent(feed, "", "  ")
}

// GenerateSiteFeed generates an RSS feed for all blog posts across all authors
func GenerateSiteFeed(docs []*markdown.Document, baseURL string) ([]byte, error) {
	// Filter and sort documents by publication date (newest first)
	var validDocs []*markdown.Document
	for _, doc := range docs {
		if !doc.Frontmatter.Draft {
			validDocs = append(validDocs, doc)
		}
	}

	// Sort by date published (newest first)
	sort.Slice(validDocs, func(i, j int) bool {
		dateI, errI := time.Parse(time.RFC3339, validDocs[i].Frontmatter.DatePublished)
		dateJ, errJ := time.Parse(time.RFC3339, validDocs[j].Frontmatter.DatePublished)

		// Handle parse errors: put invalid dates at the end
		if errI != nil && errJ != nil {
			return false // Both invalid, maintain original order
		}
		if errI != nil {
			return false // i is invalid, put it after j
		}
		if errJ != nil {
			return true // j is invalid, put i before it
		}

		return dateI.After(dateJ)
	})

	// Limit to most recent items
	const maxRSSItems = 20
	if len(validDocs) > maxRSSItems {
		validDocs = validDocs[:maxRSSItems]
	}

	// Build RSS feed
	channel := &Channel{
		Title:       "Tens City - All Posts",
		Link:        fmt.Sprintf("%s/posts", baseURL),
		Description: "Latest blog posts from all authors on Tens City",
		Language:    "en",
	}

	// Set last build date from the most recent document
	if len(validDocs) > 0 {
		if t, err := time.Parse(time.RFC3339, validDocs[0].Frontmatter.DatePublished); err == nil {
			channel.LastBuildDate = t.Format(time.RFC1123Z)
		}
	}

	// Convert documents to RSS items
	for _, doc := range validDocs {
		item := Item{
			Title:       html.EscapeString(doc.Frontmatter.Title),
			Link:        fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
			Description: html.EscapeString(doc.Frontmatter.Description),
			GUID:        fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
		}

		// Parse and format publication date
		if doc.Frontmatter.DatePublished != "" {
			if t, err := time.Parse(time.RFC3339, doc.Frontmatter.DatePublished); err == nil {
				item.PubDate = t.Format(time.RFC1123Z)
			}
		}

		channel.Items = append(channel.Items, item)
	}

	feed := &RSS{
		Version: "2.0",
		Channel: channel,
	}

	return xml.MarshalIndent(feed, "", "  ")
}
