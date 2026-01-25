package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stackdump/tens-city/internal/docserver"
	"github.com/stackdump/tens-city/internal/static"
	"github.com/stackdump/tens-city/internal/store"
)

func TestHandleGetObject(t *testing.T) {
	tmpDir := t.TempDir()
	fsStore := store.NewFSStore(tmpDir)

	// Create test object directly
	cid := "test-cid-456"
	testData := []byte(`{"@context":"test","name":"test"}`)
	canonical := []byte(`canonical`)
	if err := fsStore.SaveObject(cid, testData, canonical); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}

	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil, "http://localhost:8080", "", nil, "")

	// Test successful retrieval
	req := httptest.NewRequest("GET", "/o/"+cid, nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/ld+json" {
		t.Errorf("Expected Content-Type application/ld+json, got %s", contentType)
	}

	// Test not found
	req = httptest.NewRequest("GET", "/o/nonexistent", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestHandleGetLatest(t *testing.T) {
	tmpDir := t.TempDir()
	fsStore := store.NewFSStore(tmpDir)

	// Create user gist structure directly
	user := "testuser"
	slug := "testslug"
	cid := "test-cid-789"

	if err := fsStore.UpdateLatest(user, slug, cid); err != nil {
		t.Fatalf("Failed to update latest: %v", err)
	}

	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil, "http://localhost:8080", "", nil, "")

	req := httptest.NewRequest("GET", "/u/"+user+"/g/"+slug+"/latest", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != cid {
		t.Errorf("Expected CID %s, got %s", cid, string(body))
	}
}

func TestHandleGetHistory(t *testing.T) {
	tmpDir := t.TempDir()
	fsStore := store.NewFSStore(tmpDir)

	user := "testuser"
	slug := "testslug"
	cid1 := "cid-001"
	cid2 := "cid-002"

	// Add history entries directly
	if err := fsStore.AppendHistory(user, slug, cid1); err != nil {
		t.Fatalf("Failed to append history: %v", err)
	}
	if err := fsStore.AppendHistory(user, slug, cid2); err != nil {
		t.Fatalf("Failed to append history: %v", err)
	}

	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil, "http://localhost:8080", "", nil, "")

	req := httptest.NewRequest("GET", "/u/"+user+"/g/"+slug+"/_history", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var history []store.HistoryEntry
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		t.Fatalf("Failed to decode history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(history))
	}
}

func TestStaticFileServing(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Get the embedded public filesystem
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	server := NewServer(storage, publicFS, nil, "http://localhost:8080", "", nil, "")

	// Test serving index.html at root
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	// Just check that we got some content (the actual embedded index.html)
	if len(body) == 0 {
		t.Error("Expected index.html content")
	}
}

func TestIndexPageWithJSONLD(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Get the embedded public filesystem
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	// Create a temporary content directory with test posts
	contentDir := t.TempDir()
	testPost := `---
title: Test Post
description: A test blog post
datePublished: 2025-11-03T00:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/testauthor
tags:
  - test
lang: en
slug: test-post
draft: false
---

# Test Post

This is a test post.
`
	if err := os.WriteFile(contentDir+"/test-post.md", []byte(testPost), 0644); err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	// Create docserver with the test content
	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 0, "")
	server := NewServer(storage, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	// Test serving index.html at root with JSON-LD
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check that JSON-LD script tag is present
	if !strings.Contains(bodyStr, `<script type="application/ld+json">`) {
		t.Error("Expected JSON-LD script tag in index page")
	}

	// Check that it contains the collection metadata
	if !strings.Contains(bodyStr, `"@type": "CollectionPage"`) {
		t.Error("Expected CollectionPage type in JSON-LD")
	}

	// Check that it contains the test post
	if !strings.Contains(bodyStr, "Test Post") {
		t.Error("Expected test post headline in JSON-LD")
	}

	if !strings.Contains(bodyStr, "test-post") {
		t.Error("Expected test post slug in JSON-LD")
	}
}

func TestRSSListPage(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Get the embedded public filesystem
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	// Create a temporary content directory with test posts
	contentDir := t.TempDir()

	// Create posts from different authors
	testPost1 := `---
title: Post by Alice
description: A test blog post by Alice
datePublished: 2025-11-03T00:00:00Z
author:
  name: Alice Smith
  type: Person
  url: https://github.com/alicesmith
tags:
  - test
lang: en
slug: alice-post
draft: false
---

# Post by Alice

This is Alice's post.
`
	testPost2 := `---
title: Post by Bob
description: A test blog post by Bob
datePublished: 2025-11-02T00:00:00Z
author:
  name: Bob Jones
  type: Person
  url: https://github.com/bobjones
tags:
  - test
lang: en
slug: bob-post
draft: false
---

# Post by Bob

This is Bob's post.
`

	if err := os.WriteFile(contentDir+"/alice-post.md", []byte(testPost1), 0644); err != nil {
		t.Fatalf("Failed to create test post 1: %v", err)
	}
	if err := os.WriteFile(contentDir+"/bob-post.md", []byte(testPost2), 0644); err != nil {
		t.Fatalf("Failed to create test post 2: %v", err)
	}

	// Create docserver with the test content
	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 0, "")
	server := NewServer(storage, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	// Test serving /rss page
	req := httptest.NewRequest("GET", "/rss", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check that the page title is correct
	if !strings.Contains(bodyStr, "<title>RSS Feeds - Tens City</title>") {
		t.Error("Expected RSS Feeds page title")
	}

	// Check that both authors are listed
	if !strings.Contains(bodyStr, "Alice Smith") {
		t.Error("Expected Alice Smith in RSS feeds list")
	}
	if !strings.Contains(bodyStr, "Bob Jones") {
		t.Error("Expected Bob Jones in RSS feeds list")
	}

	// Check that RSS feed URLs are present
	if !strings.Contains(bodyStr, "/u/alicesmith/posts.rss") {
		t.Error("Expected Alice's RSS feed URL")
	}
	if !strings.Contains(bodyStr, "/u/bobjones/posts.rss") {
		t.Error("Expected Bob's RSS feed URL")
	}

	// Check that author GitHub links are present
	if !strings.Contains(bodyStr, "https://github.com/alicesmith") {
		t.Error("Expected Alice's GitHub link")
	}
	if !strings.Contains(bodyStr, "https://github.com/bobjones") {
		t.Error("Expected Bob's GitHub link")
	}
}

func TestRSSListPageWithProxyHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Get the embedded public filesystem
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	// Create a temporary content directory with test posts
	contentDir := t.TempDir()

	// Create posts from different authors
	testPost := `---
title: Test Post
description: A test blog post
datePublished: 2025-11-03T00:00:00Z
author:
  name: Alice Smith
  type: Person
  url: https://github.com/alicesmith
tags:
  - test
lang: en
slug: test-post
draft: false
---

# Test Post

This is a test post.
`

	if err := os.WriteFile(contentDir+"/test-post.md", []byte(testPost), 0644); err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	// Create docserver with the test content
	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 0, "")
	server := NewServer(storage, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	// Test with HTTPS proxy headers
	t.Run("With X-Forwarded-Proto HTTPS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rss", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", "tens.city")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check that RSS feed URLs use HTTPS protocol
		if !strings.Contains(bodyStr, `href="https://tens.city/posts.rss"`) {
			t.Error("Expected site-wide RSS feed URL to use https://tens.city")
		}
		if !strings.Contains(bodyStr, `href="https://tens.city/u/alicesmith/posts.rss"`) {
			t.Error("Expected user RSS feed URL to use https://tens.city")
		}

		// Verify URLs are displayed in link text as well
		if !strings.Contains(bodyStr, `>https://tens.city/posts.rss<`) {
			t.Error("Expected displayed URL to show https://tens.city/posts.rss")
		}
		if !strings.Contains(bodyStr, `>https://tens.city/u/alicesmith/posts.rss<`) {
			t.Error("Expected displayed URL to show https://tens.city/u/alicesmith/posts.rss")
		}
	})

	// Test with X-Forwarded-Scheme HTTPS (alternative header)
	t.Run("With X-Forwarded-Scheme HTTPS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rss", nil)
		req.Header.Set("X-Forwarded-Scheme", "https")
		req.Header.Set("X-Forwarded-Host", "example.com")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check that RSS feed URLs use HTTPS protocol
		if !strings.Contains(bodyStr, `href="https://example.com/posts.rss"`) {
			t.Error("Expected RSS feed URL to use https://example.com")
		}
	})

	// Test without proxy headers (should use Host header)
	t.Run("Without proxy headers (uses Host header)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rss", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check that RSS feed URLs use HTTP (not HTTPS) when no proxy headers present
		if !strings.Contains(bodyStr, `href="http://`) {
			t.Error("Expected RSS feed URLs to use http:// when no proxy headers present")
		}
		// httptest.NewRequest sets Host to "example.com" by default
		// Should use Host header (http://example.com) since no proxy headers or TLS
		if strings.Contains(bodyStr, `https://tens.city`) || strings.Contains(bodyStr, `https://example.com`) {
			// Print first 500 chars to debug
			t.Logf("Body preview: %s", bodyStr[:min(500, len(bodyStr))])
			t.Error("Expected to use http:// not https:// when no proxy headers or TLS present")
		}
	})

	// Test with X-Forwarded-Ssl on
	t.Run("With X-Forwarded-Ssl on", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rss", nil)
		req.Header.Set("X-Forwarded-Ssl", "on")
		req.Header.Set("X-Forwarded-Host", "secure.example.com")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check that RSS feed URLs use HTTPS protocol
		if !strings.Contains(bodyStr, `href="https://secure.example.com/posts.rss"`) {
			t.Error("Expected RSS feed URL to use https://secure.example.com")
		}
	})

	// Test with Forwarded header (RFC 7239)
	t.Run("With Forwarded header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rss", nil)
		req.Header.Set("Forwarded", "proto=https;host=forwarded.example.com")
		req.Header.Set("X-Forwarded-Host", "forwarded.example.com")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check that RSS feed URLs use HTTPS protocol
		if !strings.Contains(bodyStr, `href="https://forwarded.example.com/posts.rss"`) {
			t.Error("Expected RSS feed URL to use https://forwarded.example.com")
		}
	})

	// Test with typical nginx proxy headers (like the user's configuration)
	t.Run("With nginx proxy headers (production scenario)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rss", nil)
		// Simulate nginx proxy_set_header configuration:
		// proxy_set_header Host $host;
		// proxy_set_header X-Forwarded-Proto $scheme;
		// proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		req.Host = "tens.city" // Set request Host to match nginx's Host header
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-For", "192.0.2.1")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Verify all RSS URLs use HTTPS with correct domain
		if !strings.Contains(bodyStr, `href="https://tens.city/posts.rss"`) {
			t.Error("Expected site-wide RSS feed URL to use https://tens.city")
		}
		if !strings.Contains(bodyStr, `>https://tens.city/posts.rss<`) {
			t.Error("Expected displayed URL to show https://tens.city/posts.rss")
		}

		// Verify user RSS feeds also use HTTPS
		if !strings.Contains(bodyStr, `href="https://tens.city/u/alicesmith/posts.rss"`) {
			t.Error("Expected user RSS feed URL to use https://tens.city")
		}
		if !strings.Contains(bodyStr, `>https://tens.city/u/alicesmith/posts.rss<`) {
			t.Error("Expected displayed user RSS URL to show https://tens.city/u/alicesmith/posts.rss")
		}
	})
}

func TestSiteWideRSSFeed(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Get the embedded public filesystem
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	// Create a temporary content directory with test posts
	contentDir := t.TempDir()

	// Create posts from different authors
	testPost1 := `---
title: Post by Alice
description: A test blog post by Alice
datePublished: 2025-11-03T00:00:00Z
author:
  name: Alice Smith
  type: Person
  url: https://github.com/alicesmith
tags:
  - test
lang: en
slug: alice-post
draft: false
---

# Post by Alice

This is Alice's post.
`
	testPost2 := `---
title: Post by Bob
description: A test blog post by Bob
datePublished: 2025-11-02T00:00:00Z
author:
  name: Bob Jones
  type: Person
  url: https://github.com/bobjones
tags:
  - test
lang: en
slug: bob-post
draft: false
---

# Post by Bob

This is Bob's post.
`

	if err := os.WriteFile(contentDir+"/alice-post.md", []byte(testPost1), 0644); err != nil {
		t.Fatalf("Failed to create test post 1: %v", err)
	}
	if err := os.WriteFile(contentDir+"/bob-post.md", []byte(testPost2), 0644); err != nil {
		t.Fatalf("Failed to create test post 2: %v", err)
	}

	// Create docserver with the test content
	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 0, "")
	server := NewServer(storage, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	// Test serving /posts.rss site-wide feed
	req := httptest.NewRequest("GET", "/posts.rss", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/rss+xml") {
		t.Errorf("Expected Content-Type to contain 'application/rss+xml', got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check RSS feed structure
	if !strings.Contains(bodyStr, "<rss") {
		t.Error("Expected <rss> tag in feed")
	}

	if !strings.Contains(bodyStr, `version="2.0"`) {
		t.Error("Expected RSS version 2.0")
	}

	// Check channel metadata
	if !strings.Contains(bodyStr, "Tens City - All Posts") {
		t.Error("Expected site-wide feed title")
	}

	if !strings.Contains(bodyStr, "Latest blog posts from all authors on Tens City") {
		t.Error("Expected site-wide feed description")
	}

	// Check that both posts are included
	if !strings.Contains(bodyStr, "Post by Alice") {
		t.Error("Expected Alice's post in feed")
	}

	if !strings.Contains(bodyStr, "Post by Bob") {
		t.Error("Expected Bob's post in feed")
	}

	// Check that post URLs are correct
	if !strings.Contains(bodyStr, "http://localhost:8080/posts/alice-post") {
		t.Error("Expected Alice's post URL in feed")
	}

	if !strings.Contains(bodyStr, "http://localhost:8080/posts/bob-post") {
		t.Error("Expected Bob's post URL in feed")
	}
}

func TestRobotsTxt(t *testing.T) {
	server := NewServer(nil, nil, nil, "http://localhost:8080", "", nil, "")

	req := httptest.NewRequest("GET", "/robots.txt", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type text/plain; charset=utf-8, got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "User-agent: *") {
		t.Error("Expected User-agent in robots.txt")
	}

	if !strings.Contains(bodyStr, "Allow: /") {
		t.Error("Expected Allow directive in robots.txt")
	}
}

func TestWellKnownSecurityTxt(t *testing.T) {
	server := NewServer(nil, nil, nil, "http://localhost:8080", "", nil, "")

	req := httptest.NewRequest("GET", "/.well-known/security.txt", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type text/plain; charset=utf-8, got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "Contact:") {
		t.Error("Expected Contact field in security.txt")
	}

	if !strings.Contains(bodyStr, "Expires:") {
		t.Error("Expected Expires field in security.txt")
	}
}

func TestWellKnownNotFound(t *testing.T) {
	server := NewServer(nil, nil, nil, "http://localhost:8080", "", nil, "")

	req := httptest.NewRequest("GET", "/.well-known/nonexistent.txt", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestRSSFeedAliases(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := tmpDir + "/posts"
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content directory: %v", err)
	}

	// Create a test post
	testPost := `---
title: Test Post
description: A test post
datePublished: 2025-11-03T00:00:00Z
author:
  name: Alice
  type: Person
  url: https://github.com/alice
lang: en
slug: test-post
---

# Test Post

Test content.`

	if err := os.WriteFile(contentDir+"/test-post.md", []byte(testPost), 0644); err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public FS: %v", err)
	}

	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 20, "")
	server := NewServer(nil, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	// Test /feed.xml
	t.Run("/feed.xml", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed.xml", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/rss+xml; charset=utf-8" {
			t.Errorf("Expected Content-Type application/rss+xml; charset=utf-8, got %s", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		if !strings.Contains(bodyStr, "<rss") {
			t.Error("Expected RSS feed XML")
		}

		if !strings.Contains(bodyStr, "Test Post") {
			t.Error("Expected test post in feed")
		}
	})

	// Test /rss.xml
	t.Run("/rss.xml", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rss.xml", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/rss+xml; charset=utf-8" {
			t.Errorf("Expected Content-Type application/rss+xml; charset=utf-8, got %s", contentType)
		}
	})

	// Test /posts.rss (original)
	t.Run("/posts.rss", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts.rss", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestIndexPageWithSameAs(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := tmpDir + "/posts"
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content directory: %v", err)
	}

	// Create index.md with sameAs
	indexContent := `---
title: Test Blog
description: A test blog
icon: 🏕️
lang: en
sameAs:
  - https://github.com/test
  - https://twitter.com/test
---

Test blog content.`

	if err := os.WriteFile(tmpDir+"/index.md", []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to create index.md: %v", err)
	}

	// Create a test post
	testPost := `---
title: Test Post
description: A test post
datePublished: 2025-11-03T00:00:00Z
author:
  name: Alice
  type: Person
  url: https://github.com/alice
lang: en
slug: test-post
---

# Test Post

Test content.`

	if err := os.WriteFile(contentDir+"/test-post.md", []byte(testPost), 0644); err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public FS: %v", err)
	}

	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 20, "")
	server := NewServer(nil, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check for RSS autodiscovery link
	if !strings.Contains(bodyStr, `<link rel="alternate" type="application/rss+xml"`) {
		t.Error("Expected RSS autodiscovery link in homepage")
	}

	// Check for JSON-LD script tag
	if !strings.Contains(bodyStr, `<script type="application/ld+json">`) {
		t.Error("Expected JSON-LD script tag in homepage")
	}

	// Extract and check the JSON-LD
	startIdx := strings.Index(bodyStr, `<script type="application/ld+json">`)
	if startIdx == -1 {
		t.Fatal("Could not find JSON-LD script tag")
	}
	endIdx := strings.Index(bodyStr[startIdx:], `</script>`)
	if endIdx == -1 {
		t.Fatal("Could not find end of JSON-LD script tag")
	}

	// Extract JSON content
	jsonStart := strings.Index(bodyStr[startIdx:], "\n") + startIdx + 1
	jsonContent := bodyStr[jsonStart : startIdx+endIdx]
	jsonContent = strings.TrimSpace(jsonContent)

	// Parse the JSON-LD
	var jsonld map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &jsonld); err != nil {
		t.Fatalf("Failed to parse JSON-LD: %v", err)
	}

	// Check for sameAs field
	sameAs, ok := jsonld["sameAs"]
	if !ok {
		t.Error("Expected sameAs field in JSON-LD")
	}

	sameAsArray, ok := sameAs.([]interface{})
	if !ok {
		t.Error("Expected sameAs to be an array")
	}

	if len(sameAsArray) != 2 {
		t.Errorf("Expected 2 items in sameAs, got %d", len(sameAsArray))
	}

	// Check for expected URLs
	expectedURLs := map[string]bool{
		"https://github.com/test":  false,
		"https://twitter.com/test": false,
	}

	for _, url := range sameAsArray {
		urlStr, ok := url.(string)
		if !ok {
			continue
		}
		if _, exists := expectedURLs[urlStr]; exists {
			expectedURLs[urlStr] = true
		}
	}

	for url, found := range expectedURLs {
		if !found {
			t.Errorf("Expected URL %s in sameAs", url)
		}
	}
}

func TestDraftPosts(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := tmpDir + "/posts"
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a published post
	publishedPost := `---
title: Published Post
description: This is published
datePublished: 2025-11-01T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/testauthor
lang: en
slug: published-post
draft: false
---

# Published Post

This post is published.
`

	// Create a draft post
	draftPost := `---
title: Draft Post
description: This is a draft
datePublished: 2025-11-02T10:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/testauthor
lang: en
slug: draft-post
draft: true
---

# Draft Post

This post is a draft.
`

	if err := os.WriteFile(contentDir+"/published.md", []byte(publishedPost), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contentDir+"/draft.md", []byte(draftPost), 0644); err != nil {
		t.Fatal(err)
	}

	storage := NewFSStorage(tmpDir)
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}
	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 20, "")
	server := NewServer(storage, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	// Test 1: Draft post should be accessible when accessed directly by URL
	t.Run("draft post accessible by direct URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/draft-post", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for draft post, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Draft Post") {
			t.Error("Expected draft post title in response")
		}
		if !strings.Contains(body, "This post is a draft") {
			t.Error("Expected draft post content in response")
		}
	})

	// Test 2: Published post should be accessible
	t.Run("published post returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/published-post", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for published post, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Published Post") {
			t.Error("Expected published post title in response")
		}
	})

	// Test 3: Draft post should not appear in index
	t.Run("draft post not in index", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/index.jsonld", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for index, got %d", w.Code)
		}

		var index struct {
			ItemListElement []struct {
				Item struct {
					Headline string `json:"headline"`
				} `json:"item"`
			} `json:"itemListElement"`
		}

		if err := json.Unmarshal(w.Body.Bytes(), &index); err != nil {
			t.Fatalf("Failed to parse index JSON: %v", err)
		}

		// Check that only the published post appears
		if len(index.ItemListElement) != 1 {
			t.Errorf("Expected 1 post in index, got %d", len(index.ItemListElement))
		}

		if len(index.ItemListElement) > 0 && index.ItemListElement[0].Item.Headline != "Published Post" {
			t.Errorf("Expected 'Published Post' in index, got '%s'", index.ItemListElement[0].Item.Headline)
		}

		// Ensure draft post is not in the index
		for _, item := range index.ItemListElement {
			if item.Item.Headline == "Draft Post" {
				t.Error("Draft post should not appear in index")
			}
		}
	})

	// Test 4: Title should not have "- Tens City" suffix
	t.Run("title has no suffix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/published-post", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		body := w.Body.String()
		if strings.Contains(body, "<title>Published Post - Tens City</title>") {
			t.Error("Title should not have '- Tens City' suffix")
		}
		if !strings.Contains(body, "<title>Published Post</title>") {
			t.Error("Title should be 'Published Post' without suffix")
		}
	})

	// Test 5: Footer should have both Home and All Posts links
	t.Run("footer has home and all posts links", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/published-post", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		body := w.Body.String()
		if !strings.Contains(body, `<a href="/">← Home</a>`) {
			t.Error("Footer should contain Home link")
		}
		if !strings.Contains(body, `<a href="/posts">All Posts</a>`) {
			t.Error("Footer should contain All Posts link")
		}
		if !strings.Contains(body, `<span>•</span>`) {
			t.Error("Footer should contain separator between links")
		}
	})
}

func TestSitemapXML(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	// Create a temporary content directory with test posts
	contentDir := t.TempDir()

	testPost1 := `---
title: First Post
description: Test post one
datePublished: 2025-01-01T00:00:00Z
dateModified: 2025-01-02T00:00:00Z
slug: first-post
tags:
  - test
  - blog
author:
  name: Test User
  type: Person
  url: https://github.com/testuser
---

# First Post
Test content.
`
	testPost2 := `---
title: Second Post
description: Test post two
datePublished: 2025-01-03T00:00:00Z
slug: second-post
tags:
  - example
author:
  name: Test User
  type: Person
  url: https://github.com/testuser
---

# Second Post
More test content.
`
	if err := os.WriteFile(contentDir+"/first-post.md", []byte(testPost1), 0644); err != nil {
		t.Fatalf("Failed to write test post: %v", err)
	}
	if err := os.WriteFile(contentDir+"/second-post.md", []byte(testPost2), 0644); err != nil {
		t.Fatalf("Failed to write test post: %v", err)
	}

	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 20, "")
	server := NewServer(storage, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	req := httptest.NewRequest("GET", "/sitemap.xml", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/xml; charset=utf-8" {
		t.Errorf("Expected Content-Type application/xml; charset=utf-8, got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Debug: print the actual response
	if testing.Verbose() {
		t.Logf("Response body:\n%s", bodyStr)
	}

	// Check for XML header
	if !strings.Contains(bodyStr, "<?xml") {
		t.Error("Sitemap should contain XML header")
	}

	// Check for sitemap namespace
	if !strings.Contains(bodyStr, `xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`) {
		t.Error("Sitemap should contain proper namespace")
	}

	// Check for required pages
	requiredPages := []string{
		"<loc>http://localhost:8080/</loc>",
		"<loc>http://localhost:8080/posts</loc>",
		"<loc>http://localhost:8080/tags</loc>",
		"<loc>http://localhost:8080/rss</loc>",
		"<loc>http://localhost:8080/posts/first-post</loc>",
		"<loc>http://localhost:8080/posts/second-post</loc>",
		"<loc>http://localhost:8080/tags/test</loc>",
		"<loc>http://localhost:8080/tags/blog</loc>",
		"<loc>http://localhost:8080/tags/example</loc>",
	}

	for _, page := range requiredPages {
		if !strings.Contains(bodyStr, page) {
			t.Errorf("Sitemap should contain %s", page)
		}
	}

	// Check for lastmod dates
	if !strings.Contains(bodyStr, "<lastmod>2025-01-02</lastmod>") {
		t.Error("Sitemap should contain lastmod date for first post")
	}
	if !strings.Contains(bodyStr, "<lastmod>2025-01-03</lastmod>") {
		t.Error("Sitemap should contain lastmod date for second post")
	}

	// Check for priorities
	if !strings.Contains(bodyStr, "<priority>1") {
		t.Error("Sitemap should contain priority values")
	}
}

func TestRobotsTxtWithSitemap(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil, "http://localhost:8080", "", nil, "")

	req := httptest.NewRequest("GET", "/robots.txt", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type text/plain; charset=utf-8, got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check for basic robots.txt content
	if !strings.Contains(bodyStr, "User-agent: *") {
		t.Error("robots.txt should contain User-agent directive")
	}
	if !strings.Contains(bodyStr, "Allow: /") {
		t.Error("robots.txt should contain Allow directive")
	}

	// Check for sitemap reference
	if !strings.Contains(bodyStr, "Sitemap: http://localhost:8080/sitemap.xml") {
		t.Error("robots.txt should reference sitemap.xml")
	}
}

func TestRobotsTxtWithProxyHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil, "http://localhost:8080", "", nil, "")

	req := httptest.NewRequest("GET", "/robots.txt", nil)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check that sitemap URL uses https and correct host
	if !strings.Contains(bodyStr, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("robots.txt should reference sitemap.xml with https and correct host from proxy headers")
	}
}

func TestDraftArticleDirectAccess(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Get the embedded public filesystem
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	// Create a temporary content directory with test posts
	contentDir := t.TempDir()

	// Create a draft post
	draftPost := `---
title: Draft Article
description: This is a draft article
datePublished: 2025-11-13T00:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/testauthor
tags:
  - draft
  - test
lang: en
slug: draft-article
draft: true
---

# Draft Article

This is a draft article that should be viewable directly.
`

	// Create a published post for comparison
	publishedPost := `---
title: Published Article
description: This is a published article
datePublished: 2025-11-13T00:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/testauthor
tags:
  - published
  - test
lang: en
slug: published-article
draft: false
---

# Published Article

This is a published article.
`

	if err := os.WriteFile(contentDir+"/draft.md", []byte(draftPost), 0644); err != nil {
		t.Fatalf("Failed to create draft post: %v", err)
	}
	if err := os.WriteFile(contentDir+"/published.md", []byte(publishedPost), 0644); err != nil {
		t.Fatalf("Failed to create published post: %v", err)
	}

	// Create docserver with the test content
	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080", 0, "")
	server := NewServer(storage, publicFS, docServer, "http://localhost:8080", "", nil, contentDir)

	// Test 1: Draft article should be accessible via direct URL
	t.Run("Draft accessible by direct URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/draft-article", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for draft article, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		if !strings.Contains(bodyStr, "Draft Article") {
			t.Error("Expected draft article title in response")
		}
		if !strings.Contains(bodyStr, "This is a draft article that should be viewable directly") {
			t.Error("Expected draft article content in response")
		}
	})

	// Test 2: Draft article JSON-LD should be accessible
	t.Run("Draft JSON-LD accessible by direct URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts/draft-article.jsonld", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for draft article JSON-LD, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/ld+json" {
			t.Errorf("Expected Content-Type application/ld+json, got %s", contentType)
		}

		body, _ := io.ReadAll(resp.Body)
		var jsonld map[string]interface{}
		if err := json.Unmarshal(body, &jsonld); err != nil {
			t.Fatalf("Failed to parse JSON-LD: %v", err)
		}

		if jsonld["headline"] != "Draft Article" {
			t.Error("Expected draft article headline in JSON-LD")
		}
	})

	// Test 3: Draft should NOT appear in /posts list
	t.Run("Draft not in posts list", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		if strings.Contains(bodyStr, "Draft Article") {
			t.Error("Draft article should not appear in /posts list")
		}
		if !strings.Contains(bodyStr, "Published Article") {
			t.Error("Published article should appear in /posts list")
		}
	})

	// Test 4: Draft should NOT appear in RSS feed
	t.Run("Draft not in RSS feed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/posts.rss", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		if strings.Contains(bodyStr, "Draft Article") {
			t.Error("Draft article should not appear in RSS feed")
		}
		if !strings.Contains(bodyStr, "Published Article") {
			t.Error("Published article should appear in RSS feed")
		}
	})

	// Test 5: Draft should NOT appear in search results
	t.Run("Draft not in search", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// The search page embeds all searchable posts as JSON
		if strings.Contains(bodyStr, `"title":"Draft Article"`) {
			t.Error("Draft article should not appear in search data")
		}
		if !strings.Contains(bodyStr, `"title":"Published Article"`) {
			t.Error("Published article should appear in search data")
		}
	})

	// Test 6: Draft should NOT appear in tag pages
	t.Run("Draft not in tag page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tags/test", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		if strings.Contains(bodyStr, "Draft Article") {
			t.Error("Draft article should not appear in tag page")
		}
		if !strings.Contains(bodyStr, "Published Article") {
			t.Error("Published article should appear in tag page")
		}
	})

	// Test 7: Draft should NOT appear in sitemap
	t.Run("Draft not in sitemap", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sitemap.xml", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		if strings.Contains(bodyStr, "draft-article") {
			t.Error("Draft article should not appear in sitemap")
		}
		if !strings.Contains(bodyStr, "published-article") {
			t.Error("Published article should appear in sitemap")
		}
	})
}
