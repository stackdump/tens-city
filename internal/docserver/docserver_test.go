package docserver

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackdump/tens-city/internal/rss"
)

func TestHandleUserRSS_BasicFeed(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown files
	doc1 := `---
title: First Post by Alice
description: Alice's first post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Alice
  type: Person
  url: https://github.com/alice
lang: en
slug: alice-first
---

Content of Alice's first post.
`

	doc2 := `---
title: Second Post by Alice
description: Alice's second post
datePublished: 2025-11-02T10:00:00Z
author:
  name: Alice
  type: Person
  url: https://github.com/alice
lang: en
slug: alice-second
---

Content of Alice's second post.
`

	doc3 := `---
title: Post by Bob
description: Bob's post
datePublished: 2025-11-01T12:00:00Z
author:
  name: Bob
  type: Person
  url: https://github.com/bob
lang: en
slug: bob-post
---

Content of Bob's post.
`

	// Write test files
	if err := os.WriteFile(filepath.Join(tmpDir, "alice1.md"), []byte(doc1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "alice2.md"), []byte(doc2), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "bob1.md"), []byte(doc3), 0644); err != nil {
		t.Fatal(err)
	}

	// Create docserver
	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/u/alice/posts.rss", nil)
	rec := httptest.NewRecorder()

	// Handle request
	ds.HandleUserRSS(rec, req, "alice")

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/rss+xml") {
		t.Errorf("Expected Content-Type to contain 'application/rss+xml', got %q", contentType)
	}

	// Parse RSS feed
	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should only have Alice's posts (2 items)
	if len(feed.Channel.Items) != 2 {
		t.Errorf("Expected 2 items for alice, got %d", len(feed.Channel.Items))
	}

	// Verify items are Alice's posts
	for _, item := range feed.Channel.Items {
		if !strings.Contains(item.Title, "Alice") {
			t.Errorf("Expected item title to contain 'Alice', got %q", item.Title)
		}
	}

	// Verify sorted by date (newest first)
	if feed.Channel.Items[0].Title != "Second Post by Alice" {
		t.Errorf("Expected first item to be 'Second Post by Alice', got %q", feed.Channel.Items[0].Title)
	}
}

func TestHandleUserRSS_FiltersByGitHubUsername(t *testing.T) {
	tmpDir := t.TempDir()

	// Create posts by different users
	alicePost := `---
title: Alice's Post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Alice
  url: https://github.com/alice
lang: en
slug: alice-post
---
Content
`

	bobPost := `---
title: Bob's Post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Bob
  url: https://github.com/bob
lang: en
slug: bob-post
---
Content
`

	if err := os.WriteFile(filepath.Join(tmpDir, "alice.md"), []byte(alicePost), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "bob.md"), []byte(bobPost), 0644); err != nil {
		t.Fatal(err)
	}

	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Request Bob's feed
	req := httptest.NewRequest(http.MethodGet, "/u/bob/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "bob")

	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should only have Bob's post
	if len(feed.Channel.Items) != 1 {
		t.Errorf("Expected 1 item for bob, got %d", len(feed.Channel.Items))
	}

	// Note: XML marshaling may escape apostrophes as &#39;
	expectedTitle := "Bob's Post"
	actualTitle := feed.Channel.Items[0].Title
	if actualTitle != expectedTitle && actualTitle != "Bob&#39;s Post" {
		t.Errorf("Expected 'Bob's Post' or 'Bob&#39;s Post', got %q", actualTitle)
	}
}

func TestHandleUserRSS_ExcludesDrafts(t *testing.T) {
	tmpDir := t.TempDir()

	publishedPost := `---
title: Published Post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Alice
  url: https://github.com/alice
lang: en
slug: published
draft: false
---
Content
`

	draftPost := `---
title: Draft Post
datePublished: 2025-11-02T10:00:00Z
author:
  name: Alice
  url: https://github.com/alice
lang: en
slug: draft
draft: true
---
Content
`

	if err := os.WriteFile(filepath.Join(tmpDir, "published.md"), []byte(publishedPost), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "draft.md"), []byte(draftPost), 0644); err != nil {
		t.Fatal(err)
	}

	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	req := httptest.NewRequest(http.MethodGet, "/u/alice/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "alice")

	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should only have the published post
	if len(feed.Channel.Items) != 1 {
		t.Errorf("Expected 1 item (draft excluded), got %d", len(feed.Channel.Items))
	}

	if feed.Channel.Items[0].Title != "Published Post" {
		t.Errorf("Expected 'Published Post', got %q", feed.Channel.Items[0].Title)
	}
}

func TestHandleUserRSS_EmptyFeed(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a post by a different user
	alicePost := `---
title: Alice's Post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Alice
  url: https://github.com/alice
lang: en
slug: alice-post
---
Content
`

	if err := os.WriteFile(filepath.Join(tmpDir, "alice.md"), []byte(alicePost), 0644); err != nil {
		t.Fatal(err)
	}

	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Request feed for a user with no posts
	req := httptest.NewRequest(http.MethodGet, "/u/bob/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "bob")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 even for empty feed, got %d", rec.Code)
	}

	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should have an empty feed
	if len(feed.Channel.Items) != 0 {
		t.Errorf("Expected 0 items for user with no posts, got %d", len(feed.Channel.Items))
	}
}

func TestHandleUserRSS_OnlyGetMethod(t *testing.T) {
	tmpDir := t.TempDir()
	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Test GET is allowed
	req := httptest.NewRequest(http.MethodGet, "/u/alice/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "alice")
	if rec.Code != http.StatusOK {
		t.Errorf("Expected GET to be allowed, got status %d", rec.Code)
	}

	// Test HEAD is allowed
	req = httptest.NewRequest(http.MethodHead, "/u/alice/posts.rss", nil)
	rec = httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "alice")
	if rec.Code != http.StatusOK {
		t.Errorf("Expected HEAD to be allowed, got status %d", rec.Code)
	}

	// Test POST is not allowed
	req = httptest.NewRequest(http.MethodPost, "/u/alice/posts.rss", nil)
	rec = httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "alice")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected POST to be disallowed, got status %d", rec.Code)
	}
}

func TestHandleUserRSS_MultipleAuthors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a post with multiple authors
	multiAuthorPost := `---
title: Multi-Author Post
datePublished: 2025-11-01T10:00:00Z
author:
  - name: Alice
    url: https://github.com/alice
  - name: Bob
    url: https://github.com/bob
lang: en
slug: multi-author
---
Content
`

	aliceOnlyPost := `---
title: Alice Only Post
datePublished: 2025-11-02T10:00:00Z
author:
  name: Alice
  url: https://github.com/alice
lang: en
slug: alice-only
---
Content
`

	if err := os.WriteFile(filepath.Join(tmpDir, "multi.md"), []byte(multiAuthorPost), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "alice.md"), []byte(aliceOnlyPost), 0644); err != nil {
		t.Fatal(err)
	}

	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Request Alice's feed
	req := httptest.NewRequest(http.MethodGet, "/u/alice/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "alice")

	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should include both posts (multi-author is matched by first author)
	if len(feed.Channel.Items) != 2 {
		t.Errorf("Expected 2 items (including multi-author post), got %d", len(feed.Channel.Items))
	}
}

func TestHandleUserRSS_FeedMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	alicePost := `---
title: Alice's Post
description: A great post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Alice
  url: https://github.com/alice
lang: en
slug: alice-post
---
Content
`

	if err := os.WriteFile(filepath.Join(tmpDir, "alice.md"), []byte(alicePost), 0644); err != nil {
		t.Fatal(err)
	}

	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	req := httptest.NewRequest(http.MethodGet, "/u/alice/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleUserRSS(rec, req, "alice")

	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Verify channel metadata
	if !strings.Contains(feed.Channel.Title, "alice") {
		t.Errorf("Expected channel title to contain 'alice', got %q", feed.Channel.Title)
	}

	expectedLink := "https://tens.city/u/alice"
	if feed.Channel.Link != expectedLink {
		t.Errorf("Expected channel link %q, got %q", expectedLink, feed.Channel.Link)
	}

	if !strings.Contains(feed.Channel.Description, "alice") {
		t.Errorf("Expected channel description to contain 'alice', got %q", feed.Channel.Description)
	}

	// Verify item fields
	item := feed.Channel.Items[0]
	expectedItemLink := "https://tens.city/posts/alice-post"
	if item.Link != expectedItemLink {
		t.Errorf("Expected item link %q, got %q", expectedItemLink, item.Link)
	}

	if item.Description != "A great post" {
		t.Errorf("Expected description 'A great post', got %q", item.Description)
	}
}

func TestExtractGitHubUsername(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "standard github url",
			url:      "https://github.com/alice",
			expected: "alice",
		},
		{
			name:     "github url with trailing slash",
			url:      "https://github.com/alice/",
			expected: "alice",
		},
		{
			name:     "github url with repository",
			url:      "https://github.com/alice/repo",
			expected: "alice",
		},
		{
			name:     "empty url",
			url:      "",
			expected: "",
		},
		{
			name:     "non-github url",
			url:      "https://example.com/alice",
			expected: "",
		},
		{
			name:     "github.com without username",
			url:      "https://github.com/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractGitHubUsername(tt.url)
			if result != tt.expected {
				t.Errorf("extractGitHubUsername(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestExtractAuthorURL(t *testing.T) {
	tests := []struct {
		name     string
		author   interface{}
		expected string
	}{
		{
			name: "single author with url",
			author: map[string]interface{}{
				"name": "Alice",
				"url":  "https://github.com/alice",
			},
			expected: "https://github.com/alice",
		},
		{
			name: "multiple authors",
			author: []interface{}{
				map[string]interface{}{
					"name": "Alice",
					"url":  "https://github.com/alice",
				},
				map[string]interface{}{
					"name": "Bob",
					"url":  "https://github.com/bob",
				},
			},
			expected: "https://github.com/alice", // Returns first author's URL
		},
		{
			name:     "nil author",
			author:   nil,
			expected: "",
		},
		{
			name: "author without url",
			author: map[string]interface{}{
				"name": "Alice",
			},
			expected: "",
		},
		{
			name:     "empty array",
			author:   []interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAuthorURL(tt.author)
			if result != tt.expected {
				t.Errorf("extractAuthorURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHandleSiteRSS_BasicFeed(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown files from different authors
	doc1 := `---
title: Post by Alice
description: Alice's post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Alice
  type: Person
  url: https://github.com/alice
lang: en
slug: alice-post
---

Content of Alice's post.
`

	doc2 := `---
title: Post by Bob
description: Bob's post
datePublished: 2025-11-02T10:00:00Z
author:
  name: Bob
  type: Person
  url: https://github.com/bob
lang: en
slug: bob-post
---

Content of Bob's post.
`

	doc3 := `---
title: Another Post by Alice
description: Alice's second post
datePublished: 2025-11-03T10:00:00Z
author:
  name: Alice
  type: Person
  url: https://github.com/alice
lang: en
slug: alice-second
---

Content of Alice's second post.
`

	// Write test files
	if err := os.WriteFile(filepath.Join(tmpDir, "alice1.md"), []byte(doc1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "bob1.md"), []byte(doc2), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "alice2.md"), []byte(doc3), 0644); err != nil {
		t.Fatal(err)
	}

	// Create docserver
	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/posts.rss", nil)
	rec := httptest.NewRecorder()

	// Handle request
	ds.HandleSiteRSS(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/rss+xml") {
		t.Errorf("Expected Content-Type to contain 'application/rss+xml', got %q", contentType)
	}

	// Parse RSS feed
	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should have all posts from all authors (3 items)
	if len(feed.Channel.Items) != 3 {
		t.Errorf("Expected 3 items from all authors, got %d", len(feed.Channel.Items))
	}

	// Verify sorted by date (newest first)
	if feed.Channel.Items[0].Title != "Another Post by Alice" {
		t.Errorf("Expected first item to be 'Another Post by Alice', got %q", feed.Channel.Items[0].Title)
	}

	if feed.Channel.Items[1].Title != "Post by Bob" {
		t.Errorf("Expected second item to be 'Post by Bob', got %q", feed.Channel.Items[1].Title)
	}

	if feed.Channel.Items[2].Title != "Post by Alice" {
		t.Errorf("Expected third item to be 'Post by Alice', got %q", feed.Channel.Items[2].Title)
	}

	// Verify channel metadata
	if feed.Channel.Title != "Tens City - All Posts" {
		t.Errorf("Expected title 'Tens City - All Posts', got %q", feed.Channel.Title)
	}

	expectedLink := "https://tens.city/posts"
	if feed.Channel.Link != expectedLink {
		t.Errorf("Expected link %q, got %q", expectedLink, feed.Channel.Link)
	}
}

func TestHandleSiteRSS_ExcludesDrafts(t *testing.T) {
	tmpDir := t.TempDir()

	publishedPost := `---
title: Published Post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Alice
  url: https://github.com/alice
lang: en
slug: published
draft: false
---
Content
`

	draftPost := `---
title: Draft Post
datePublished: 2025-11-02T10:00:00Z
author:
  name: Bob
  url: https://github.com/bob
lang: en
slug: draft
draft: true
---
Content
`

	if err := os.WriteFile(filepath.Join(tmpDir, "published.md"), []byte(publishedPost), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "draft.md"), []byte(draftPost), 0644); err != nil {
		t.Fatal(err)
	}

	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	req := httptest.NewRequest(http.MethodGet, "/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleSiteRSS(rec, req)

	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should only have the published post
	if len(feed.Channel.Items) != 1 {
		t.Errorf("Expected 1 item (draft excluded), got %d", len(feed.Channel.Items))
	}

	if feed.Channel.Items[0].Title != "Published Post" {
		t.Errorf("Expected 'Published Post', got %q", feed.Channel.Items[0].Title)
	}
}

func TestHandleSiteRSS_EmptyFeed(t *testing.T) {
	tmpDir := t.TempDir()

	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Request site feed with no posts
	req := httptest.NewRequest(http.MethodGet, "/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleSiteRSS(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 even for empty feed, got %d", rec.Code)
	}

	var feed rss.RSS
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("Failed to parse RSS feed: %v", err)
	}

	// Should have an empty feed
	if len(feed.Channel.Items) != 0 {
		t.Errorf("Expected 0 items for site with no posts, got %d", len(feed.Channel.Items))
	}
}

func TestHandleSiteRSS_OnlyGetMethod(t *testing.T) {
	tmpDir := t.TempDir()
	ds := NewDocServer(tmpDir, "https://tens.city", 0)

	// Test GET is allowed
	req := httptest.NewRequest(http.MethodGet, "/posts.rss", nil)
	rec := httptest.NewRecorder()
	ds.HandleSiteRSS(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected GET to be allowed, got status %d", rec.Code)
	}

	// Test HEAD is allowed
	req = httptest.NewRequest(http.MethodHead, "/posts.rss", nil)
	rec = httptest.NewRecorder()
	ds.HandleSiteRSS(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected HEAD to be allowed, got status %d", rec.Code)
	}

	// Test POST is not allowed
	req = httptest.NewRequest(http.MethodPost, "/posts.rss", nil)
	rec = httptest.NewRecorder()
	ds.HandleSiteRSS(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected POST to be disallowed, got status %d", rec.Code)
	}
}

func TestCollectTags(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown files with tags
	doc1 := `---
title: Test Post 1
description: First test post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: test-1
tags:
  - golang
  - tutorial
---

Content 1.
`

	doc2 := `---
title: Test Post 2
description: Second test post
datePublished: 2025-11-02T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: test-2
tags:
  - golang
  - web-development
keywords:
  - tutorial
---

Content 2.
`

	doc3 := `---
title: Draft Post
description: A draft post
datePublished: 2025-11-03T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: draft-post
draft: true
tags:
  - draft-tag
---

Draft content.
`

	os.WriteFile(filepath.Join(tmpDir, "post1.md"), []byte(doc1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "post2.md"), []byte(doc2), 0644)
	os.WriteFile(filepath.Join(tmpDir, "post3.md"), []byte(doc3), 0644)

	ds := NewDocServer(tmpDir, "http://localhost:8080", 0)

	tags, err := ds.collectTags()
	if err != nil {
		t.Fatalf("collectTags failed: %v", err)
	}

	// Should have 3 unique tags (draft-tag should be excluded)
	if len(tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(tags))
	}

	// Check tags are sorted by count descending, then alphabetically
	expectedOrder := []string{"golang", "tutorial", "web-development"}
	for i, tag := range tags {
		if tag.Tag != expectedOrder[i] {
			t.Errorf("Expected tag %d to be %s, got %s", i, expectedOrder[i], tag.Tag)
		}
	}

	// golang and tutorial should have count 2
	if tags[0].Count != 2 || tags[1].Count != 2 {
		t.Errorf("Expected golang and tutorial to have count 2, got %d and %d", tags[0].Count, tags[1].Count)
	}

	// web-development should have count 1
	if tags[2].Count != 1 {
		t.Errorf("Expected web-development to have count 1, got %d", tags[2].Count)
	}
}

func TestHandleTagsPage(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown file
	doc1 := `---
title: Test Post
description: A test post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: test-post
tags:
  - golang
  - tutorial
---

Content.
`

	os.WriteFile(filepath.Join(tmpDir, "post1.md"), []byte(doc1), 0644)

	ds := NewDocServer(tmpDir, "http://localhost:8080", 0)

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	rec := httptest.NewRecorder()

	ds.HandleTagsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Check for expected content
	if !strings.Contains(body, "Tags") {
		t.Error("Expected page title 'Tags'")
	}

	if !strings.Contains(body, "golang") {
		t.Error("Expected tag 'golang' to be present")
	}

	if !strings.Contains(body, "tutorial") {
		t.Error("Expected tag 'tutorial' to be present")
	}

	if !strings.Contains(body, "/tags/golang") {
		t.Error("Expected link to /tags/golang")
	}

	// Test method not allowed
	req = httptest.NewRequest(http.MethodPost, "/tags", nil)
	rec = httptest.NewRecorder()
	ds.HandleTagsPage(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected POST to be disallowed, got status %d", rec.Code)
	}
}

func TestHandleTagPage(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown files
	doc1 := `---
title: Golang Post
description: About Golang
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: golang-post
tags:
  - golang
---

Content about Golang.
`

	doc2 := `---
title: Tutorial Post
description: A tutorial
datePublished: 2025-11-02T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: tutorial-post
tags:
  - tutorial
---

Tutorial content.
`

	doc3 := `---
title: Golang Tutorial
description: Golang tutorial
datePublished: 2025-11-03T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: golang-tutorial
tags:
  - golang
  - tutorial
---

Golang tutorial content.
`

	os.WriteFile(filepath.Join(tmpDir, "post1.md"), []byte(doc1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "post2.md"), []byte(doc2), 0644)
	os.WriteFile(filepath.Join(tmpDir, "post3.md"), []byte(doc3), 0644)

	ds := NewDocServer(tmpDir, "http://localhost:8080", 0)

	// Test filtering by "golang" tag
	req := httptest.NewRequest(http.MethodGet, "/tags/golang", nil)
	rec := httptest.NewRecorder()

	ds.HandleTagPage(rec, req, "golang")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Check for expected content
	if !strings.Contains(body, "golang") {
		t.Error("Expected tag 'golang' in title")
	}

	if !strings.Contains(body, "2 posts") {
		t.Error("Expected '2 posts' in description")
	}

	if !strings.Contains(body, "Golang Post") {
		t.Error("Expected 'Golang Post' to be listed")
	}

	if !strings.Contains(body, "Golang Tutorial") {
		t.Error("Expected 'Golang Tutorial' to be listed")
	}

	if strings.Contains(body, "Tutorial Post") {
		t.Error("Did not expect 'Tutorial Post' to be listed (doesn't have golang tag)")
	}

	// Test method not allowed
	req = httptest.NewRequest(http.MethodPost, "/tags/golang", nil)
	rec = httptest.NewRecorder()
	ds.HandleTagPage(rec, req, "golang")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected POST to be disallowed, got status %d", rec.Code)
	}
}

func TestHandleTagPage_NoMatchingPosts(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown file without the searched tag
	doc1 := `---
title: Test Post
description: A test post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: test-post
tags:
  - golang
---

Content.
`

	os.WriteFile(filepath.Join(tmpDir, "post1.md"), []byte(doc1), 0644)

	ds := NewDocServer(tmpDir, "http://localhost:8080", 0)

	// Test filtering by a tag that doesn't exist in any post
	req := httptest.NewRequest(http.MethodGet, "/tags/nonexistent", nil)
	rec := httptest.NewRecorder()

	ds.HandleTagPage(rec, req, "nonexistent")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Check for expected content
	if !strings.Contains(body, "nonexistent") {
		t.Error("Expected tag 'nonexistent' in title")
	}

	if !strings.Contains(body, "0 posts") {
		t.Error("Expected '0 posts' in description")
	}

	if !strings.Contains(body, "No posts found") {
		t.Error("Expected 'No posts found' message")
	}
}

func TestHandleTagPage_URLEncoding(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown file with a tag containing special characters
	doc1 := `---
title: Test Post
description: A test post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: test-post
tags:
  - schema.org
  - web development
---

Content.
`

	os.WriteFile(filepath.Join(tmpDir, "post1.md"), []byte(doc1), 0644)

	ds := NewDocServer(tmpDir, "http://localhost:8080", 0)

	// Test filtering by a tag with special characters
	req := httptest.NewRequest(http.MethodGet, "/tags/schema.org", nil)
	rec := httptest.NewRecorder()

	ds.HandleTagPage(rec, req, "schema.org")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "schema.org") {
		t.Error("Expected tag 'schema.org' in content")
	}

	if !strings.Contains(body, "Test Post") {
		t.Error("Expected 'Test Post' to be listed")
	}

	// Test tag with space
	req = httptest.NewRequest(http.MethodGet, "/tags/web%20development", nil)
	rec = httptest.NewRecorder()

	ds.HandleTagPage(rec, req, "web development")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body = rec.Body.String()

	if !strings.Contains(body, "web development") {
		t.Error("Expected tag 'web development' in content")
	}
}

func TestHandleTagsPage_JSONLD(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown files with tags
	doc1 := `---
title: First Post
description: A test post
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: first-post
tags:
  - golang
  - testing
---

Content.
`

	doc2 := `---
title: Second Post
description: Another test post
datePublished: 2025-11-02T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: second-post
tags:
  - golang
  - web
---

More content.
`

	os.WriteFile(filepath.Join(tmpDir, "post1.md"), []byte(doc1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "post2.md"), []byte(doc2), 0644)

	ds := NewDocServer(tmpDir, "http://localhost:8080", 0)

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	rec := httptest.NewRecorder()

	ds.HandleTagsPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Check for JSON-LD script tag
	if !strings.Contains(body, `<script type="application/ld+json">`) {
		t.Error("Expected JSON-LD script tag to be present")
	}

	// Check for schema.org context
	if !strings.Contains(body, `"@context": "https://schema.org"`) {
		t.Error("Expected schema.org context in JSON-LD")
	}

	// Check for CollectionPage type
	if !strings.Contains(body, `"@type": "CollectionPage"`) {
		t.Error("Expected CollectionPage type in JSON-LD")
	}

	// Check for tags collection name
	if !strings.Contains(body, `"name": "Tags"`) {
		t.Error("Expected 'Tags' name in JSON-LD")
	}

	// Check for itemListElement
	if !strings.Contains(body, `"itemListElement"`) {
		t.Error("Expected itemListElement in JSON-LD")
	}

	// Check for DefinedTerm type for tags
	if !strings.Contains(body, `"@type": "DefinedTerm"`) {
		t.Error("Expected DefinedTerm type for tag items")
	}

	// Check that tag names are included
	if !strings.Contains(body, `"name": "golang"`) {
		t.Error("Expected tag 'golang' in JSON-LD")
	}

	// Check that tag URLs are included
	if !strings.Contains(body, `"url": "http://localhost:8080/tags/golang"`) {
		t.Error("Expected tag URL in JSON-LD")
	}

	// Check for numberOfItems
	if !strings.Contains(body, `"numberOfItems"`) {
		t.Error("Expected numberOfItems in JSON-LD")
	}
}

func TestHandleTagPage_JSONLD(t *testing.T) {
	// Create a temporary directory with test content
	tmpDir := t.TempDir()

	// Create test markdown files
	doc1 := `---
title: Golang Tutorial
description: Learn Golang basics
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: golang-tutorial
tags:
  - golang
  - tutorial
---

Golang content.
`

	doc2 := `---
title: Advanced Golang
description: Advanced Golang concepts
datePublished: 2025-11-02T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: advanced-golang
tags:
  - golang
  - advanced
---

More Golang content.
`

	doc3 := `---
title: Python Tutorial
description: Learn Python
datePublished: 2025-11-03T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/test
lang: en
slug: python-tutorial
tags:
  - python
---

Python content.
`

	os.WriteFile(filepath.Join(tmpDir, "post1.md"), []byte(doc1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "post2.md"), []byte(doc2), 0644)
	os.WriteFile(filepath.Join(tmpDir, "post3.md"), []byte(doc3), 0644)

	ds := NewDocServer(tmpDir, "http://localhost:8080", 0)

	// Test filtering by golang tag
	req := httptest.NewRequest(http.MethodGet, "/tags/golang", nil)
	rec := httptest.NewRecorder()

	ds.HandleTagPage(rec, req, "golang")

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// Check for JSON-LD script tag
	if !strings.Contains(body, `<script type="application/ld+json">`) {
		t.Error("Expected JSON-LD script tag to be present")
	}

	// Check for schema.org context
	if !strings.Contains(body, `"@context": "https://schema.org"`) {
		t.Error("Expected schema.org context in JSON-LD")
	}

	// Check for CollectionPage type
	if !strings.Contains(body, `"@type": "CollectionPage"`) {
		t.Error("Expected CollectionPage type in JSON-LD")
	}

	// Check for search results name with tag
	if !strings.Contains(body, `"name": "Posts tagged with \"golang\""`) {
		t.Error("Expected 'Posts tagged with \"golang\"' name in JSON-LD")
	}

	// Check for itemListElement
	if !strings.Contains(body, `"itemListElement"`) {
		t.Error("Expected itemListElement in JSON-LD")
	}

	// Check for Article type in items
	if !strings.Contains(body, `"@type": "Article"`) {
		t.Error("Expected Article type for post items")
	}

	// Check that golang posts are included
	if !strings.Contains(body, `"headline": "Golang Tutorial"`) {
		t.Error("Expected 'Golang Tutorial' headline in JSON-LD")
	}

	if !strings.Contains(body, `"headline": "Advanced Golang"`) {
		t.Error("Expected 'Advanced Golang' headline in JSON-LD")
	}

	// Check that non-golang posts are NOT included
	if strings.Contains(body, `"headline": "Python Tutorial"`) {
		t.Error("Did not expect 'Python Tutorial' in golang tag results")
	}

	// Check for post URLs
	if !strings.Contains(body, `"url": "http://localhost:8080/posts/golang-tutorial"`) {
		t.Error("Expected post URL in JSON-LD")
	}

	// Check for numberOfItems (should be 2 golang posts)
	if !strings.Contains(body, `"numberOfItems": 2`) {
		t.Error("Expected numberOfItems to be 2 for golang tag")
	}

	// Check for descriptions
	if !strings.Contains(body, `"description": "Learn Golang basics"`) {
		t.Error("Expected post description in JSON-LD")
	}

	// Check for dates
	if !strings.Contains(body, `"datePublished": "2025-11-01T10:00:00Z"`) {
		t.Error("Expected datePublished in JSON-LD")
	}
}
