package markdown

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/mermaid"
	"gopkg.in/yaml.v3"
)

// Frontmatter represents the YAML frontmatter of a document
type Frontmatter struct {
	Title         string      `yaml:"title" json:"title"`
	Description   string      `yaml:"description,omitempty" json:"description,omitempty"`
	DatePublished string      `yaml:"datePublished,omitempty" json:"datePublished,omitempty"`
	DateModified  string      `yaml:"dateModified,omitempty" json:"dateModified,omitempty"`
	Author        interface{} `yaml:"author,omitempty" json:"author,omitempty"`
	Tags          []string    `yaml:"tags,omitempty" json:"tags,omitempty"`
	Collection    string      `yaml:"collection,omitempty" json:"collection,omitempty"`
	Lang          string      `yaml:"lang,omitempty" json:"lang,omitempty"`
	Draft         bool        `yaml:"draft,omitempty" json:"draft,omitempty"`
	Slug          string      `yaml:"slug,omitempty" json:"slug,omitempty"`
	Image         string      `yaml:"image,omitempty" json:"image,omitempty"`
	Keywords      []string    `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	Icon          string      `yaml:"icon,omitempty" json:"icon,omitempty"`
}

// Document represents a parsed markdown document
type Document struct {
	Frontmatter Frontmatter
	Content     string // Raw markdown content
	HTML        string // Rendered HTML
	FilePath    string // Path to the source file
}

var frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)

// ParseDocument parses a markdown file with YAML frontmatter
func ParseDocument(filePath string) (*Document, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseDocumentFromBytes(content, filePath)
}

// ParseDocumentFromBytes parses markdown content with YAML frontmatter
func ParseDocumentFromBytes(content []byte, filePath string) (*Document, error) {
	matches := frontmatterRegex.FindSubmatch(content)
	if matches == nil || len(matches) < 3 {
		return nil, fmt.Errorf("no frontmatter found")
	}

	var fm Frontmatter
	if err := yaml.Unmarshal(matches[1], &fm); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Auto-generate slug from filename if not provided
	if fm.Slug == "" {
		fm.Slug = generateSlug(filepath.Base(filePath))
	}

	markdownContent := string(matches[2])

	// Render markdown to HTML
	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			&mermaid.Extender{},
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithUnsafe()), // We'll sanitize after
	)
	if err := md.Convert([]byte(markdownContent), &buf); err != nil {
		return nil, fmt.Errorf("failed to render markdown: %w", err)
	}

	// Sanitize HTML
	sanitized := sanitizeHTML(buf.String())

	return &Document{
		Frontmatter: fm,
		Content:     markdownContent,
		HTML:        sanitized,
		FilePath:    filePath,
	}, nil
}

// ParseIndexDocument parses an index.md file with minimal frontmatter requirements
// Unlike regular documents, index.md doesn't require datePublished, author, or lang
func ParseIndexDocument(filePath string) (*Document, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse using the standard function
	doc, err := ParseDocumentFromBytes(content, filePath)
	if err != nil {
		return nil, err
	}

	// Apply defaults for index.md specific fields
	if doc.Frontmatter.Title == "" {
		doc.Frontmatter.Title = "Tens City - A Minimal Blog Platform"
	}
	if doc.Frontmatter.Description == "" {
		doc.Frontmatter.Description = "Simple, elegant blog platform built on content-addressable storage"
	}
	if doc.Frontmatter.Icon == "" {
		doc.Frontmatter.Icon = "🏕️"
	}
	if doc.Frontmatter.Lang == "" {
		doc.Frontmatter.Lang = "en"
	}

	return doc, nil
}

// sanitizeHTML sanitizes HTML to prevent XSS attacks
func sanitizeHTML(html string) string {
	p := bluemonday.UGCPolicy()
	// Allow additional safe elements for documentation
	p.AllowAttrs("id").Matching(regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)).OnElements("h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^[a-zA-Z0-9\s\-_]+$`)).OnElements("code", "pre")

	// Allow Mermaid diagram elements - diagrams wrapped in <pre class="mermaid"> for client-side rendering
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^mermaid$`)).OnElements("pre")

	// Allow SVG elements for diagrams
	p.AllowElements("svg", "g", "path", "rect", "circle", "ellipse", "line", "polyline", "polygon", "text", "tspan", "defs", "use", "clipPath", "mask", "title", "desc")

	// Allow SVG-specific attributes only on SVG elements (not globally for security)
	p.AllowAttrs("xmlns", "xmlns:xlink", "version", "viewBox", "width", "height", "preserveAspectRatio").OnElements("svg")
	p.AllowAttrs("d").OnElements("path")
	p.AllowAttrs("fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin", "opacity").OnElements("path", "rect", "circle", "ellipse", "line", "polyline", "polygon")
	p.AllowAttrs("transform").OnElements("g", "path", "rect", "circle", "ellipse", "line", "polyline", "polygon", "text")
	p.AllowAttrs("x", "y", "width", "height").OnElements("rect", "text")
	p.AllowAttrs("x1", "y1", "x2", "y2").OnElements("line")
	p.AllowAttrs("cx", "cy", "r").OnElements("circle")
	p.AllowAttrs("cx", "cy", "rx", "ry").OnElements("ellipse")
	p.AllowAttrs("points").OnElements("polyline", "polygon")
	p.AllowAttrs("x", "y", "font-family", "font-size", "text-anchor", "dominant-baseline").OnElements("text", "tspan")
	p.AllowAttrs("class", "id").OnElements("svg", "g", "path", "rect", "circle", "ellipse", "line", "polyline", "polygon", "text")

	return p.Sanitize(html)
}

// generateSlug creates a URL-friendly slug from a filename
func generateSlug(filename string) string {
	// Remove .md extension
	name := strings.TrimSuffix(filename, ".md")
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	slug = regexp.MustCompile(`[\s_]+`).ReplaceAllString(slug, "-")
	// Remove non-alphanumeric characters except hyphens
	slug = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(slug, "")
	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	return slug
}

// ToJSONLD converts frontmatter to schema.org JSON-LD
func (d *Document) ToJSONLD(baseURL string) map[string]interface{} {
	fm := d.Frontmatter

	jsonld := map[string]interface{}{
		"@context":   "https://schema.org",
		"@type":      "Article",
		"headline":   fm.Title,
		"inLanguage": fm.Lang,
	}

	if fm.Description != "" {
		jsonld["description"] = fm.Description
	}

	if fm.DatePublished != "" {
		jsonld["datePublished"] = fm.DatePublished
	}

	if fm.DateModified != "" {
		jsonld["dateModified"] = fm.DateModified
	}

	// Handle author(s)
	if fm.Author != nil {
		jsonld["author"] = normalizeAuthor(fm.Author)
	}

	// Add URL if we have a slug
	if fm.Slug != "" && baseURL != "" {
		jsonld["url"] = fmt.Sprintf("%s/posts/%s", baseURL, fm.Slug)
		jsonld["@id"] = fmt.Sprintf("%s/posts/%s", baseURL, fm.Slug)
	}

	if fm.Image != "" {
		jsonld["image"] = fm.Image
	}

	if len(fm.Keywords) > 0 {
		jsonld["keywords"] = fm.Keywords
	} else if len(fm.Tags) > 0 {
		jsonld["keywords"] = fm.Tags
	}

	if fm.Collection != "" {
		jsonld["isPartOf"] = map[string]interface{}{
			"@type": "CollectionPage",
			"name":  fm.Collection,
		}
	}

	return jsonld
}

// normalizeAuthor converts author field to schema.org Person/Organization format
func normalizeAuthor(author interface{}) interface{} {
	switch a := author.(type) {
	case map[string]interface{}:
		return normalizePersonOrOrg(a)
	case []interface{}:
		authors := make([]interface{}, 0, len(a))
		for _, item := range a {
			if m, ok := item.(map[string]interface{}); ok {
				authors = append(authors, normalizePersonOrOrg(m))
			}
		}
		return authors
	default:
		return author
	}
}

// normalizePersonOrOrg converts a person/org map to schema.org format
func normalizePersonOrOrg(m map[string]interface{}) map[string]interface{} {
	person := make(map[string]interface{})

	// Determine type
	personType := "Person"
	if t, ok := m["type"].(string); ok && t == "Organization" {
		personType = "Organization"
	}
	person["@type"] = personType

	// Copy standard fields
	if name, ok := m["name"].(string); ok {
		person["name"] = name
	}
	if url, ok := m["url"].(string); ok {
		person["url"] = url
	}
	// Don't expose email to protect privacy
	if sameAs, ok := m["sameAs"].([]interface{}); ok {
		person["sameAs"] = sameAs
	}

	return person
}

// ListDocuments finds all markdown documents in a directory
func ListDocuments(contentDir string) ([]*Document, error) {
	var docs []*Document

	err := filepath.Walk(contentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			doc, err := ParseDocument(path)
			if err != nil {
				// Log error but continue processing other files
				fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
				return nil
			}
			docs = append(docs, doc)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return docs, nil
}

// BuildCollectionIndex creates a schema.org CollectionPage index
// limit controls how many items to include (0 means no limit)
func BuildCollectionIndex(docs []*Document, baseURL string, limit int) map[string]interface{} {
	now := time.Now().Format(time.RFC3339)

	// Filter out drafts and collect non-draft documents
	var publicDocs []*Document
	for _, doc := range docs {
		if !doc.Frontmatter.Draft {
			publicDocs = append(publicDocs, doc)
		}
	}

	// Sort by DatePublished descending (newest first), then by Title ascending
	SortDocumentsByDate(publicDocs)

	// Apply limit if specified
	if limit > 0 && len(publicDocs) > limit {
		publicDocs = publicDocs[:limit]
	}

	// Build the items list
	items := make([]interface{}, 0, len(publicDocs))
	for _, doc := range publicDocs {
		itemArticle := map[string]interface{}{
			"@type":    "Article",
			"headline": doc.Frontmatter.Title,
			"url":      fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
		}

		if doc.Frontmatter.Description != "" {
			itemArticle["description"] = doc.Frontmatter.Description
		}

		if doc.Frontmatter.DatePublished != "" {
			itemArticle["datePublished"] = doc.Frontmatter.DatePublished
		}

		if doc.Frontmatter.Author != nil {
			itemArticle["author"] = normalizeAuthor(doc.Frontmatter.Author)
		}

		// Include keywords (combination of tags and keywords)
		allKeywords := append([]string{}, doc.Frontmatter.Tags...)
		allKeywords = append(allKeywords, doc.Frontmatter.Keywords...)
		if len(allKeywords) > 0 {
			itemArticle["keywords"] = allKeywords
		}

		item := map[string]interface{}{
			"@type":    "ListItem",
			"position": len(items) + 1,
			"item":     itemArticle,
		}

		items = append(items, item)
	}

	return map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "CollectionPage",
		"name":            "Blog Posts Index",
		"description":     "Collection of blog posts",
		"dateModified":    now,
		"numberOfItems":   len(items),
		"itemListElement": items,
	}
}

// ValidateFrontmatter validates frontmatter against required fields
func ValidateFrontmatter(fm Frontmatter) error {
	if fm.Title == "" {
		return fmt.Errorf("title is required")
	}
	if fm.DatePublished == "" {
		return fmt.Errorf("datePublished is required")
	}
	if fm.Author == nil {
		return fmt.Errorf("author is required")
	}
	if fm.Lang == "" {
		return fmt.Errorf("lang is required")
	}

	// Validate date format
	if _, err := time.Parse(time.RFC3339, fm.DatePublished); err != nil {
		return fmt.Errorf("datePublished must be in RFC3339 format: %w", err)
	}

	if fm.DateModified != "" {
		if _, err := time.Parse(time.RFC3339, fm.DateModified); err != nil {
			return fmt.Errorf("dateModified must be in RFC3339 format: %w", err)
		}
	}

	return nil
}

// SerializeJSONLD serializes JSON-LD to formatted JSON
func SerializeJSONLD(jsonld map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(jsonld, "", "  ")
}

// SortDocumentsByDate sorts documents by DatePublished descending (newest first),
// then by Title ascending for items with the same date
func SortDocumentsByDate(docs []*Document) {
	sort.Slice(docs, func(i, j int) bool {
		// Parse dates for comparison
		dateI, errI := time.Parse(time.RFC3339, docs[i].Frontmatter.DatePublished)
		dateJ, errJ := time.Parse(time.RFC3339, docs[j].Frontmatter.DatePublished)

		// If both dates are valid, compare them
		if errI == nil && errJ == nil {
			if !dateI.Equal(dateJ) {
				// Descending order (newest first)
				return dateI.After(dateJ)
			}
			// If dates are equal, sort by title ascending
			return docs[i].Frontmatter.Title < docs[j].Frontmatter.Title
		}

		// If one date is invalid, put it after valid dates
		if errI != nil && errJ == nil {
			return false
		}
		if errI == nil && errJ != nil {
			return true
		}

		// If both dates are invalid, sort by title
		return docs[i].Frontmatter.Title < docs[j].Frontmatter.Title
	})
}
