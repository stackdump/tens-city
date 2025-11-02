package docserver

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stackdump/tens-city/internal/markdown"
)

// DocServer handles markdown document requests
type DocServer struct {
	contentDir string
	baseURL    string
	cache      *DocumentCache
}

// DocumentCache caches rendered documents
type DocumentCache struct {
	mu    sync.RWMutex
	docs  map[string]*CachedDoc
	index *CachedIndex
}

// CachedDoc represents a cached document
type CachedDoc struct {
	Doc      *markdown.Document
	ETag     string
	Modified time.Time
}

// CachedIndex represents the cached document index
type CachedIndex struct {
	Data     []byte
	ETag     string
	Modified time.Time
}

// NewDocServer creates a new document server
func NewDocServer(contentDir, baseURL string) *DocServer {
	return &DocServer{
		contentDir: contentDir,
		baseURL:    baseURL,
		cache: &DocumentCache{
			docs: make(map[string]*CachedDoc),
		},
	}
}

// loadDocument loads and caches a document
func (ds *DocServer) loadDocument(slug string) (*CachedDoc, error) {
	ds.cache.mu.RLock()
	cached, ok := ds.cache.docs[slug]
	ds.cache.mu.RUnlock()

	// Find the file
	var filePath string
	err := filepath.Walk(ds.contentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			// Parse to check slug
			doc, parseErr := markdown.ParseDocument(path)
			if parseErr == nil && doc.Frontmatter.Slug == slug {
				filePath = path
				return filepath.SkipAll
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if filePath == "" {
		return nil, fmt.Errorf("document not found: %s", slug)
	}

	// Check file modification time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Return cached version if still valid
	if ok && !fileInfo.ModTime().After(cached.Modified) {
		return cached, nil
	}

	// Parse the document
	doc, err := markdown.ParseDocument(filePath)
	if err != nil {
		return nil, err
	}

	// Validate frontmatter
	if err := markdown.ValidateFrontmatter(doc.Frontmatter); err != nil {
		return nil, fmt.Errorf("invalid frontmatter: %w", err)
	}

	// Generate ETag
	etag := generateETag(doc.HTML)

	cached = &CachedDoc{
		Doc:      doc,
		ETag:     etag,
		Modified: fileInfo.ModTime(),
	}

	// Update cache
	ds.cache.mu.Lock()
	ds.cache.docs[slug] = cached
	ds.cache.mu.Unlock()

	return cached, nil
}

// loadIndex loads and caches the document index
func (ds *DocServer) loadIndex() (*CachedIndex, error) {
	ds.cache.mu.RLock()
	cached := ds.cache.index
	ds.cache.mu.RUnlock()

	// Check if we need to rebuild
	needsRebuild := cached == nil

	if !needsRebuild {
		// Check if any files changed
		_ = filepath.Walk(ds.contentDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".md") {
				if info.ModTime().After(cached.Modified) {
					needsRebuild = true
					return filepath.SkipAll
				}
			}
			return nil
		})
	}

	if !needsRebuild {
		return cached, nil
	}

	// Rebuild index
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		return nil, err
	}

	index := markdown.BuildCollectionIndex(docs, ds.baseURL)
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, err
	}

	cached = &CachedIndex{
		Data:     data,
		ETag:     generateETag(string(data)),
		Modified: time.Now(),
	}

	ds.cache.mu.Lock()
	ds.cache.index = cached
	ds.cache.mu.Unlock()

	return cached, nil
}

// HandleDocList handles GET /docs - list all documents
func (ds *DocServer) HandleDocList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load documents", http.StatusInternalServerError)
		return
	}

	// Filter out drafts
	var publicDocs []*markdown.Document
	for _, doc := range docs {
		if !doc.Frontmatter.Draft {
			publicDocs = append(publicDocs, doc)
		}
	}

	// Render HTML list
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Documentation - Tens City</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; }
        h1 { color: #333; }
        .doc-list { list-style: none; padding: 0; }
        .doc-item { margin: 1rem 0; padding: 1rem; border: 1px solid #ddd; border-radius: 4px; }
        .doc-item h2 { margin: 0 0 0.5rem 0; }
        .doc-item a { color: #0066cc; text-decoration: none; }
        .doc-item a:hover { text-decoration: underline; }
        .doc-meta { color: #666; font-size: 0.9rem; }
        .doc-description { margin-top: 0.5rem; }
    </style>
</head>
<body>
    <h1>Documentation</h1>
    <ul class="doc-list">
`)

	for _, doc := range publicDocs {
		fmt.Fprintf(w, `        <li class="doc-item">
            <h2><a href="/docs/%s">%s</a></h2>
`, doc.Frontmatter.Slug, doc.Frontmatter.Title)
		
		if doc.Frontmatter.Description != "" {
			fmt.Fprintf(w, `            <p class="doc-description">%s</p>
`, doc.Frontmatter.Description)
		}
		
		fmt.Fprintf(w, `            <div class="doc-meta">Published: %s</div>
        </li>
`, doc.Frontmatter.DatePublished)
	}

	fmt.Fprintf(w, `    </ul>
    <p><a href="/docs/index.jsonld">View as JSON-LD</a></p>
</body>
</html>`)
}

// HandleDoc handles GET /docs/:slug - render a single document
func (ds *DocServer) HandleDoc(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check Accept header for content negotiation
	accept := r.Header.Get("Accept")
	wantsJSON := strings.Contains(accept, "application/ld+json") || strings.Contains(accept, "application/json")

	cached, err := ds.loadDocument(slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Document not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to load document", http.StatusInternalServerError)
		}
		return
	}

	doc := cached.Doc

	// Check draft status
	if doc.Frontmatter.Draft {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	// Set cache headers
	w.Header().Set("ETag", cached.ETag)
	w.Header().Set("Last-Modified", cached.Modified.UTC().Format(http.TimeFormat))

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == cached.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if wantsJSON {
		// Return JSON-LD
		jsonld := doc.ToJSONLD(ds.baseURL)
		data, err := json.MarshalIndent(jsonld, "", "  ")
		if err != nil {
			http.Error(w, "Failed to serialize JSON-LD", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/ld+json")
		w.Write(data)
		return
	}

	// Render HTML with embedded JSON-LD
	jsonld := doc.ToJSONLD(ds.baseURL)
	jsonldBytes, _ := json.MarshalIndent(jsonld, "    ", "  ")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s - Tens City</title>
    <script type="application/ld+json">
    %s
    </script>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; line-height: 1.6; }
        h1, h2, h3, h4, h5, h6 { color: #333; margin-top: 1.5rem; }
        code { background: #f4f4f4; padding: 0.2rem 0.4rem; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 1rem; border-radius: 4px; overflow-x: auto; }
        pre code { background: none; padding: 0; }
        a { color: #0066cc; }
        .nav { margin-bottom: 2rem; }
        .nav a { margin-right: 1rem; }
        .meta { color: #666; font-size: 0.9rem; margin-bottom: 2rem; }
        .jsonld-toggle { margin-top: 2rem; padding: 1rem; background: #f9f9f9; border-radius: 4px; }
        .jsonld-content { display: none; margin-top: 1rem; }
        .jsonld-content.show { display: block; }
    </style>
</head>
<body>
    <div class="nav">
        <a href="/docs">← Back to Docs</a>
        <a href="/docs/%s.jsonld">JSON-LD</a>
    </div>
    <div class="meta">
        Published: %s
`, doc.Frontmatter.Lang, doc.Frontmatter.Title, string(jsonldBytes), 
   slug, doc.Frontmatter.DatePublished)

	if doc.Frontmatter.DateModified != "" {
		fmt.Fprintf(w, ` | Modified: %s`, doc.Frontmatter.DateModified)
	}

	fmt.Fprintf(w, `
    </div>
    %s
    <div class="jsonld-toggle">
        <button onclick="toggleJSONLD()">Toggle JSON-LD Preview</button>
        <pre class="jsonld-content" id="jsonld"><code>%s</code></pre>
    </div>
    <script>
        function toggleJSONLD() {
            document.getElementById('jsonld').classList.toggle('show');
        }
    </script>
</body>
</html>`, doc.HTML, string(jsonldBytes))
}

// HandleDocJSONLD handles GET /docs/:slug.jsonld - return JSON-LD only
func (ds *DocServer) HandleDocJSONLD(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cached, err := ds.loadDocument(slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Document not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to load document", http.StatusInternalServerError)
		}
		return
	}

	doc := cached.Doc

	// Check draft status
	if doc.Frontmatter.Draft {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	// Set cache headers
	w.Header().Set("ETag", cached.ETag)
	w.Header().Set("Last-Modified", cached.Modified.UTC().Format(http.TimeFormat))

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == cached.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	jsonld := doc.ToJSONLD(ds.baseURL)
	data, err := json.MarshalIndent(jsonld, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize JSON-LD", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Write(data)
}

// HandleIndexJSONLD handles GET /docs/index.jsonld - return collection index
func (ds *DocServer) HandleIndexJSONLD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cached, err := ds.loadIndex()
	if err != nil {
		http.Error(w, "Failed to load index", http.StatusInternalServerError)
		return
	}

	// Set cache headers
	w.Header().Set("ETag", cached.ETag)
	w.Header().Set("Last-Modified", cached.Modified.UTC().Format(http.TimeFormat))

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == cached.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Write(cached.Data)
}

// generateETag generates an ETag from content
func generateETag(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf(`"%x"`, hash)
}
