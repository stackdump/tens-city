package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDocument(t *testing.T) {
	content := []byte(`---
title: Test Document
description: A test document
datePublished: 2025-11-02T00:00:00Z
author:
  name: Test Author
  type: Person
lang: en
slug: test-doc
---

# Test Content

This is a test paragraph with **bold** and *italic* text.

## Second Heading

- Item 1
- Item 2
`)

	doc, err := ParseDocumentFromBytes(content, "test.md")
	if err != nil {
		t.Fatalf("ParseDocumentFromBytes failed: %v", err)
	}

	if doc.Frontmatter.Title != "Test Document" {
		t.Errorf("Expected title 'Test Document', got '%s'", doc.Frontmatter.Title)
	}

	if doc.Frontmatter.Slug != "test-doc" {
		t.Errorf("Expected slug 'test-doc', got '%s'", doc.Frontmatter.Slug)
	}

	if doc.Content == "" {
		t.Error("Expected non-empty content")
	}

	if doc.HTML == "" {
		t.Error("Expected non-empty HTML")
	}

	// Check that HTML contains expected elements
	if !strings.Contains(doc.HTML, "<h1") {
		t.Error("Expected HTML to contain h1 tag")
	}

	if !strings.Contains(doc.HTML, "<strong>bold</strong>") {
		t.Error("Expected HTML to contain bold text")
	}
}

func TestParseDocument_AutoSlug(t *testing.T) {
	content := []byte(`---
title: Test Document
datePublished: 2025-11-02T00:00:00Z
author:
  name: Test Author
lang: en
---

Content here
`)

	doc, err := ParseDocumentFromBytes(content, "my-test-file.md")
	if err != nil {
		t.Fatalf("ParseDocumentFromBytes failed: %v", err)
	}

	if doc.Frontmatter.Slug != "my-test-file" {
		t.Errorf("Expected auto-generated slug 'my-test-file', got '%s'", doc.Frontmatter.Slug)
	}
}

func TestParseDocument_MissingFrontmatter(t *testing.T) {
	content := []byte(`# Just a heading

No frontmatter here.
`)

	_, err := ParseDocumentFromBytes(content, "test.md")
	if err == nil {
		t.Error("Expected error for missing frontmatter")
	}
}

func TestToJSONLD(t *testing.T) {
	doc := &Document{
		Frontmatter: Frontmatter{
			Title:         "Test Article",
			Description:   "Test description",
			DatePublished: "2025-11-02T00:00:00Z",
			DateModified:  "2025-11-02T12:00:00Z",
			Author: map[string]interface{}{
				"name": "Test Author",
				"type": "Person",
				"url":  "https://example.com",
			},
			Tags:       []string{"test", "example"},
			Collection: "guides",
			Lang:       "en",
			Slug:       "test-article",
		},
	}

	jsonld := doc.ToJSONLD("https://tens.city")

	if jsonld["@context"] != "https://schema.org" {
		t.Error("Expected @context to be schema.org")
	}

	if jsonld["@type"] != "Article" {
		t.Error("Expected @type to be Article")
	}

	if jsonld["headline"] != "Test Article" {
		t.Error("Expected headline to match title")
	}

	if jsonld["url"] != "https://tens.city/posts/test-article" {
		t.Errorf("Expected correct URL, got %v", jsonld["url"])
	}

	// Check author normalization
	author := jsonld["author"].(map[string]interface{})
	if author["@type"] != "Person" {
		t.Error("Expected author type to be Person")
	}
	if author["name"] != "Test Author" {
		t.Error("Expected author name to be preserved")
	}
}

func TestToJSONLD_MultipleAuthors(t *testing.T) {
	doc := &Document{
		Frontmatter: Frontmatter{
			Title:         "Test Article",
			DatePublished: "2025-11-02T00:00:00Z",
			Author: []interface{}{
				map[string]interface{}{
					"name": "Author One",
					"type": "Person",
				},
				map[string]interface{}{
					"name": "Author Two",
					"type": "Person",
				},
			},
			Lang: "en",
			Slug: "test",
		},
	}

	jsonld := doc.ToJSONLD("https://tens.city")
	authors := jsonld["author"].([]interface{})

	if len(authors) != 2 {
		t.Errorf("Expected 2 authors, got %d", len(authors))
	}
}

func TestValidateFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		fm      Frontmatter
		wantErr bool
	}{
		{
			name: "valid frontmatter",
			fm: Frontmatter{
				Title:         "Test",
				DatePublished: "2025-11-02T00:00:00Z",
				Author:        map[string]interface{}{"name": "Test"},
				Lang:          "en",
			},
			wantErr: false,
		},
		{
			name: "missing title",
			fm: Frontmatter{
				DatePublished: "2025-11-02T00:00:00Z",
				Author:        map[string]interface{}{"name": "Test"},
				Lang:          "en",
			},
			wantErr: true,
		},
		{
			name: "missing datePublished",
			fm: Frontmatter{
				Title:  "Test",
				Author: map[string]interface{}{"name": "Test"},
				Lang:   "en",
			},
			wantErr: true,
		},
		{
			name: "invalid date format",
			fm: Frontmatter{
				Title:         "Test",
				DatePublished: "2025-11-02",
				Author:        map[string]interface{}{"name": "Test"},
				Lang:          "en",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFrontmatter(tt.fm)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"test-file.md", "test-file"},
		{"My Test File.md", "my-test-file"},
		{"test_with_underscores.md", "test-with-underscores"},
		{"Test!@#$%File.md", "testfile"},
		{"UPPERCASE.md", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := generateSlug(tt.filename)
			if got != tt.want {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "safe HTML",
			input: "<p>Hello <strong>world</strong></p>",
			want:  "<p>Hello <strong>world</strong></p>",
		},
		{
			name:  "remove script tags",
			input: "<p>Hello</p><script>alert('xss')</script>",
			want:  "<p>Hello</p>",
		},
		{
			name:  "remove onclick",
			input: `<a href="#" onclick="alert('xss')">Link</a>`,
			want:  `Link`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeHTML(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListDocuments(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create test documents
	doc1 := `---
title: Document 1
datePublished: 2025-11-02T00:00:00Z
author:
  name: Test
lang: en
---
Content 1`

	doc2 := `---
title: Document 2
datePublished: 2025-11-02T00:00:00Z
author:
  name: Test
lang: en
---
Content 2`

	if err := os.WriteFile(filepath.Join(tmpDir, "doc1.md"), []byte(doc1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "doc2.md"), []byte(doc2), 0644); err != nil {
		t.Fatal(err)
	}

	docs, err := ListDocuments(tmpDir)
	if err != nil {
		t.Fatalf("ListDocuments failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

func TestBuildCollectionIndex(t *testing.T) {
	docs := []*Document{
		{
			Frontmatter: Frontmatter{
				Title:         "Doc 1",
				Description:   "First doc",
				DatePublished: "2025-11-02T00:00:00Z",
				Slug:          "doc1",
				Draft:         false,
			},
		},
		{
			Frontmatter: Frontmatter{
				Title:         "Doc 2",
				DatePublished: "2025-11-02T00:00:00Z",
				Slug:          "doc2",
				Draft:         true, // Should be excluded
			},
		},
		{
			Frontmatter: Frontmatter{
				Title:         "Doc 3",
				DatePublished: "2025-11-02T00:00:00Z",
				Slug:          "doc3",
				Draft:         false,
			},
		},
	}

	index := BuildCollectionIndex(docs, "https://tens.city")

	if index["@type"] != "CollectionPage" {
		t.Error("Expected @type to be CollectionPage")
	}

	items := index["itemListElement"].([]interface{})
	// Should only have 2 items (excluding draft)
	if len(items) != 2 {
		t.Errorf("Expected 2 items in index (drafts excluded), got %d", len(items))
	}

	if index["numberOfItems"] != 2 {
		t.Errorf("Expected numberOfItems to be 2, got %v", index["numberOfItems"])
	}
}
