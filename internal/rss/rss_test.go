package rss

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stackdump/tens-city/internal/markdown"
)

func TestGenerateUserFeed_BasicFeed(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "First Post",
				Description:   "This is the first post",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "first-post",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Second Post",
				Description:   "This is the second post",
				DatePublished: "2025-11-02T10:00:00Z",
				Slug:          "second-post",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	// Parse the XML to verify structure
	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Verify basic structure
	if feed.Version != "2.0" {
		t.Errorf("Expected RSS version 2.0, got %s", feed.Version)
	}

	if feed.Channel == nil {
		t.Fatal("Expected channel to be non-nil")
	}

	// Verify channel metadata
	expectedTitle := "testuser's Blog Posts - Tens City"
	if feed.Channel.Title != expectedTitle {
		t.Errorf("Expected title %q, got %q", expectedTitle, feed.Channel.Title)
	}

	expectedLink := "https://tens.city/u/testuser"
	if feed.Channel.Link != expectedLink {
		t.Errorf("Expected link %q, got %q", expectedLink, feed.Channel.Link)
	}

	expectedDesc := "Latest blog posts from testuser"
	if feed.Channel.Description != expectedDesc {
		t.Errorf("Expected description %q, got %q", expectedDesc, feed.Channel.Description)
	}

	if feed.Channel.Language != "en" {
		t.Errorf("Expected language en, got %s", feed.Channel.Language)
	}

	// Verify items count
	if len(feed.Channel.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(feed.Channel.Items))
	}

	// Verify items are sorted by date (newest first)
	if feed.Channel.Items[0].Title != "Second Post" {
		t.Errorf("Expected first item to be 'Second Post', got %s", feed.Channel.Items[0].Title)
	}

	if feed.Channel.Items[1].Title != "First Post" {
		t.Errorf("Expected second item to be 'First Post', got %s", feed.Channel.Items[1].Title)
	}
}

func TestGenerateUserFeed_FiltersDrafts(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Published Post",
				Description:   "This is published",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "published-post",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Draft Post",
				Description:   "This is a draft",
				DatePublished: "2025-11-02T10:00:00Z",
				Slug:          "draft-post",
				Draft:         true, // This should be filtered out
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Should only have the published post
	if len(feed.Channel.Items) != 1 {
		t.Errorf("Expected 1 item (draft filtered), got %d", len(feed.Channel.Items))
	}

	if feed.Channel.Items[0].Title != "Published Post" {
		t.Errorf("Expected 'Published Post', got %s", feed.Channel.Items[0].Title)
	}
}

func TestGenerateUserFeed_SortsByDateNewestFirst(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Oldest Post",
				DatePublished: "2025-10-01T10:00:00Z",
				Slug:          "oldest",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Newest Post",
				DatePublished: "2025-11-03T10:00:00Z",
				Slug:          "newest",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Middle Post",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "middle",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Verify sort order
	expectedOrder := []string{"Newest Post", "Middle Post", "Oldest Post"}
	for i, expected := range expectedOrder {
		if feed.Channel.Items[i].Title != expected {
			t.Errorf("Item %d: expected %q, got %q", i, expected, feed.Channel.Items[i].Title)
		}
	}
}

func TestGenerateUserFeed_LimitsTo20Items(t *testing.T) {
	// Create 25 documents
	docs := make([]*markdown.Document, 25)
	for i := 0; i < 25; i++ {
		date := time.Date(2025, 11, i+1, 10, 0, 0, 0, time.UTC)
		docs[i] = &markdown.Document{
			Frontmatter: markdown.Frontmatter{
				Title:         fmt.Sprintf("Post %d", i),
				DatePublished: date.Format(time.RFC3339),
				Slug:          fmt.Sprintf("post-%d", i),
				Draft:         false,
			},
		}
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Should be limited to 20 items
	if len(feed.Channel.Items) != 20 {
		t.Errorf("Expected 20 items (max limit), got %d", len(feed.Channel.Items))
	}
}

func TestGenerateUserFeed_HandlesInvalidDates(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Valid Date Post",
				DatePublished: "2025-11-02T10:00:00Z",
				Slug:          "valid",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Invalid Date Post",
				DatePublished: "invalid-date", // Invalid date format
				Slug:          "invalid",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Another Valid Date Post",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "another-valid",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// All items should be present
	if len(feed.Channel.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(feed.Channel.Items))
	}

	// Valid dates should be sorted first
	if feed.Channel.Items[0].Title != "Valid Date Post" {
		t.Errorf("Expected 'Valid Date Post' first (newest valid date), got %s", feed.Channel.Items[0].Title)
	}

	// Invalid date should be last
	if feed.Channel.Items[2].Title != "Invalid Date Post" {
		t.Errorf("Expected 'Invalid Date Post' last (invalid date), got %s", feed.Channel.Items[2].Title)
	}
}

func TestGenerateUserFeed_HTMLEscaping(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Test & Special <Characters>",
				Description:   "Description with <script>alert('xss')</script> & quotes \"",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "test-escaping",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "test&user", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Verify HTML entities are properly escaped in title
	if !strings.Contains(feed.Channel.Items[0].Title, "&amp;") {
		t.Error("Expected & to be escaped as &amp; in item title")
	}

	if !strings.Contains(feed.Channel.Items[0].Title, "&lt;") {
		t.Error("Expected < to be escaped as &lt; in item title")
	}

	if !strings.Contains(feed.Channel.Items[0].Title, "&gt;") {
		t.Error("Expected > to be escaped as &gt; in item title")
	}

	// Verify description is escaped
	if !strings.Contains(feed.Channel.Items[0].Description, "&amp;") {
		t.Error("Expected & to be escaped in description")
	}

	// Verify channel title escaping
	if !strings.Contains(feed.Channel.Title, "test&amp;user") {
		t.Error("Expected username to be escaped in channel title")
	}
}

func TestGenerateUserFeed_ItemFields(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Test Post",
				Description:   "Test description",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "test-post",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	item := feed.Channel.Items[0]

	// Verify link format
	expectedLink := "https://tens.city/posts/test-post"
	if item.Link != expectedLink {
		t.Errorf("Expected link %q, got %q", expectedLink, item.Link)
	}

	// Verify GUID matches link
	if item.GUID != expectedLink {
		t.Errorf("Expected GUID to match link, got %q", item.GUID)
	}

	// Verify pubDate format (should be RFC1123Z)
	if item.PubDate == "" {
		t.Error("Expected pubDate to be set")
	}

	// Parse the date to verify format
	_, err = time.Parse(time.RFC1123Z, item.PubDate)
	if err != nil {
		t.Errorf("pubDate not in RFC1123Z format: %v", err)
	}
}

func TestGenerateUserFeed_LastBuildDate(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Older Post",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "older",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Newer Post",
				DatePublished: "2025-11-02T15:30:00Z",
				Slug:          "newer",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// LastBuildDate should be set to the most recent post's date
	if feed.Channel.LastBuildDate == "" {
		t.Error("Expected LastBuildDate to be set")
	}

	// Parse and verify it matches the newer post's date
	buildDate, err := time.Parse(time.RFC1123Z, feed.Channel.LastBuildDate)
	if err != nil {
		t.Fatalf("Failed to parse LastBuildDate: %v", err)
	}

	expectedDate := time.Date(2025, 11, 2, 15, 30, 0, 0, time.UTC)
	if !buildDate.Equal(expectedDate) {
		t.Errorf("Expected LastBuildDate to be %v, got %v", expectedDate, buildDate)
	}
}

func TestGenerateUserFeed_EmptyDocuments(t *testing.T) {
	docs := []*markdown.Document{}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Should have a valid feed structure even with no items
	if feed.Channel == nil {
		t.Fatal("Expected channel to be non-nil")
	}

	if len(feed.Channel.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(feed.Channel.Items))
	}

	// LastBuildDate should be empty
	if feed.Channel.LastBuildDate != "" {
		t.Errorf("Expected empty LastBuildDate, got %s", feed.Channel.LastBuildDate)
	}
}

func TestGenerateUserFeed_XMLFormat(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Test Post",
				Description:   "Test description",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "test-post",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateUserFeed(docs, "testuser", "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateUserFeed failed: %v", err)
	}

	feedStr := string(feedData)

	// Verify XML declaration and structure
	if !strings.Contains(feedStr, "<rss") {
		t.Error("Expected <rss> tag in output")
	}

	if !strings.Contains(feedStr, `version="2.0"`) {
		t.Error("Expected version=\"2.0\" attribute")
	}

	if !strings.Contains(feedStr, "<channel>") {
		t.Error("Expected <channel> tag")
	}

	if !strings.Contains(feedStr, "<item>") {
		t.Error("Expected <item> tag")
	}

	// Verify it's indented (from MarshalIndent)
	if !strings.Contains(feedStr, "\n  ") {
		t.Error("Expected indented XML output")
	}
}

func TestGenerateSiteFeed_BasicFeed(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "First Post",
				Description:   "This is the first post",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "first-post",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Second Post",
				Description:   "This is the second post",
				DatePublished: "2025-11-02T10:00:00Z",
				Slug:          "second-post",
				Draft:         false,
			},
		},
	}

	feedData, err := GenerateSiteFeed(docs, "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateSiteFeed failed: %v", err)
	}

	// Parse the XML to verify structure
	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Verify basic structure
	if feed.Version != "2.0" {
		t.Errorf("Expected RSS version 2.0, got %s", feed.Version)
	}

	if feed.Channel == nil {
		t.Fatal("Expected channel to be non-nil")
	}

	// Verify channel metadata
	expectedTitle := "Tens City - All Posts"
	if feed.Channel.Title != expectedTitle {
		t.Errorf("Expected title %q, got %q", expectedTitle, feed.Channel.Title)
	}

	expectedLink := "https://tens.city/posts"
	if feed.Channel.Link != expectedLink {
		t.Errorf("Expected link %q, got %q", expectedLink, feed.Channel.Link)
	}

	expectedDesc := "Latest blog posts from all authors on Tens City"
	if feed.Channel.Description != expectedDesc {
		t.Errorf("Expected description %q, got %q", expectedDesc, feed.Channel.Description)
	}

	if feed.Channel.Language != "en" {
		t.Errorf("Expected language en, got %s", feed.Channel.Language)
	}

	// Verify items count
	if len(feed.Channel.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(feed.Channel.Items))
	}

	// Verify items are sorted by date (newest first)
	if feed.Channel.Items[0].Title != "Second Post" {
		t.Errorf("Expected first item to be 'Second Post', got %s", feed.Channel.Items[0].Title)
	}

	if feed.Channel.Items[1].Title != "First Post" {
		t.Errorf("Expected second item to be 'First Post', got %s", feed.Channel.Items[1].Title)
	}
}

func TestGenerateSiteFeed_FiltersDrafts(t *testing.T) {
	docs := []*markdown.Document{
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Published Post",
				Description:   "This is published",
				DatePublished: "2025-11-01T10:00:00Z",
				Slug:          "published-post",
				Draft:         false,
			},
		},
		{
			Frontmatter: markdown.Frontmatter{
				Title:         "Draft Post",
				Description:   "This is a draft",
				DatePublished: "2025-11-02T10:00:00Z",
				Slug:          "draft-post",
				Draft:         true, // This should be filtered out
			},
		},
	}

	feedData, err := GenerateSiteFeed(docs, "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateSiteFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Should only have the published post
	if len(feed.Channel.Items) != 1 {
		t.Errorf("Expected 1 item (draft filtered), got %d", len(feed.Channel.Items))
	}

	if feed.Channel.Items[0].Title != "Published Post" {
		t.Errorf("Expected 'Published Post', got %s", feed.Channel.Items[0].Title)
	}
}

func TestGenerateSiteFeed_LimitsTo20Items(t *testing.T) {
	// Create 25 documents
	docs := make([]*markdown.Document, 25)
	for i := 0; i < 25; i++ {
		date := time.Date(2025, 11, i+1, 10, 0, 0, 0, time.UTC)
		docs[i] = &markdown.Document{
			Frontmatter: markdown.Frontmatter{
				Title:         fmt.Sprintf("Post %d", i),
				DatePublished: date.Format(time.RFC3339),
				Slug:          fmt.Sprintf("post-%d", i),
				Draft:         false,
			},
		}
	}

	feedData, err := GenerateSiteFeed(docs, "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateSiteFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Should be limited to 20 items
	if len(feed.Channel.Items) != 20 {
		t.Errorf("Expected 20 items (max limit), got %d", len(feed.Channel.Items))
	}
}

func TestGenerateSiteFeed_EmptyDocuments(t *testing.T) {
	docs := []*markdown.Document{}

	feedData, err := GenerateSiteFeed(docs, "https://tens.city")
	if err != nil {
		t.Fatalf("GenerateSiteFeed failed: %v", err)
	}

	var feed RSS
	if err := xml.Unmarshal(feedData, &feed); err != nil {
		t.Fatalf("Failed to parse generated RSS: %v", err)
	}

	// Should have a valid feed structure even with no items
	if feed.Channel == nil {
		t.Fatal("Expected channel to be non-nil")
	}

	if len(feed.Channel.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(feed.Channel.Items))
	}

	// LastBuildDate should be empty
	if feed.Channel.LastBuildDate != "" {
		t.Errorf("Expected empty LastBuildDate, got %s", feed.Channel.LastBuildDate)
	}
}
