package docserver

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackdump/tens-city/internal/httputil"
	"github.com/stackdump/tens-city/internal/markdown"
	"github.com/stackdump/tens-city/internal/rss"
	"github.com/stackdump/tens-city/internal/sitemap"
)

// GoogleAnalyticsTag returns the Google Analytics tracking code for the given ID.
// Returns empty string if id is empty.
func GoogleAnalyticsTag(id string) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf(`<!-- Google tag (gtag.js) -->
    <script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
    <script>
      window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date());
      gtag('config', '%s');
    </script>`, id, id)
}

// DocServer handles markdown document requests
type DocServer struct {
	contentDir        string
	fallbackURL       string // Fallback base URL when headers are not available
	indexLimit        int    // Maximum number of items to show in index (0 = no limit)
	googleAnalyticsID string // Google Analytics measurement ID (empty = disabled)
	cache             *DocumentCache
}

// DocumentCache caches rendered documents
type DocumentCache struct {
	mu       sync.RWMutex
	docs     map[string]*CachedDoc
	index    *CachedIndex
	indexDoc *CachedDoc // Cached index.md document
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
func NewDocServer(contentDir, fallbackURL string, indexLimit int, googleAnalyticsID string) *DocServer {
	return &DocServer{
		contentDir:        contentDir,
		fallbackURL:       fallbackURL,
		indexLimit:        indexLimit,
		googleAnalyticsID: googleAnalyticsID,
		cache: &DocumentCache{
			docs: make(map[string]*CachedDoc),
		},
	}
}

// HandleFavicon serves favicon from content directory
// Looks for favicon.ico, favicon.png, or favicon.svg in content parent directory
func (ds *DocServer) HandleFavicon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Content dir is typically "content/posts", so parent is "content"
	contentParent := filepath.Dir(ds.contentDir)

	// Try different favicon formats in order of preference
	faviconFiles := []struct {
		name        string
		contentType string
	}{
		{"favicon.ico", "image/x-icon"},
		{"favicon.png", "image/png"},
		{"favicon.svg", "image/svg+xml"},
	}

	for _, f := range faviconFiles {
		faviconPath := filepath.Join(contentParent, f.name)
		data, err := os.ReadFile(faviconPath)
		if err == nil {
			w.Header().Set("Content-Type", f.contentType)
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.Write(data)
			return
		}
	}

	http.NotFound(w, r)
}

// GetFaviconPath returns the path to favicon if one exists, empty string otherwise
func (ds *DocServer) GetFaviconPath() string {
	contentParent := filepath.Dir(ds.contentDir)
	faviconFiles := []string{"favicon.ico", "favicon.png", "favicon.svg"}

	for _, f := range faviconFiles {
		faviconPath := filepath.Join(contentParent, f)
		if _, err := os.Stat(faviconPath); err == nil {
			return "/" + f
		}
	}
	return ""
}

// HandleContentAsset serves static assets (images) from the content directory
func (ds *DocServer) HandleContentAsset(w http.ResponseWriter, r *http.Request, filename string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Content dir is typically "content/posts", so parent is "content"
	contentParent := filepath.Dir(ds.contentDir)
	assetPath := filepath.Join(contentParent, filename)

	// Security: ensure we're not escaping the content directory
	absContent, _ := filepath.Abs(contentParent)
	absAsset, _ := filepath.Abs(assetPath)
	if !strings.HasPrefix(absAsset, absContent) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	data, err := os.ReadFile(assetPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type based on extension
	ext := strings.ToLower(filepath.Ext(filename))
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
	}

	if ct, ok := contentTypes[ext]; ok {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
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

// loadIndexDocument loads and caches the index.md document
// If the file doesn't exist, it creates a default one to help users discover customization
func (ds *DocServer) loadIndexDocument() (*CachedDoc, error) {
	ds.cache.mu.RLock()
	cached := ds.cache.indexDoc
	ds.cache.mu.RUnlock()

	// Determine the base content directory (parent of posts directory)
	baseContentDir := filepath.Dir(ds.contentDir)
	indexPath := filepath.Join(baseContentDir, "index.md")

	// Check if index.md exists
	fileInfo, err := os.Stat(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default index.md to help users discover customization
			defaultContent := `---
title: Tens City - A Minimal Blog Platform
description: Simple, elegant blog platform built on content-addressable storage
icon: 🏕️
lang: en
---

A minimal blog platform built on simplicity and content ownership.

<!-- Edit this file (content/index.md) to customize your blog's homepage -->
`
			if err := os.WriteFile(indexPath, []byte(defaultContent), 0644); err != nil {
				// If we can't create the file (e.g., read-only filesystem), just use defaults
				return nil, nil
			}
			// Re-stat the file we just created
			fileInfo, err = os.Stat(indexPath)
			if err != nil {
				return nil, nil
			}
		} else {
			return nil, err
		}
	}

	// Return cached version if still valid
	if cached != nil && !fileInfo.ModTime().After(cached.Modified) {
		return cached, nil
	}

	// Parse the index document
	doc, err := markdown.ParseIndexDocument(indexPath)
	if err != nil {
		return nil, err
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
	ds.cache.indexDoc = cached
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

	// Try to load the index.md document for sameAs metadata
	var indexDoc *markdown.Document
	cachedIndexDoc, err := ds.loadIndexDocument()
	if err == nil && cachedIndexDoc != nil {
		indexDoc = cachedIndexDoc.Doc
	}

	// Use fallback URL for the cached index (will be used when serving via GetIndexJSONLD)
	index := markdown.BuildCollectionIndexWithMeta(docs, ds.fallbackURL, ds.indexLimit, indexDoc)
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

// HandleDocList handles GET /posts - list all posts
func (ds *DocServer) HandleDocList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Filter out drafts
	var publicDocs []*markdown.Document
	for _, doc := range docs {
		if !doc.Frontmatter.Draft {
			publicDocs = append(publicDocs, doc)
		}
	}

	// Sort by DatePublished descending (newest first), then by Title ascending
	markdown.SortDocumentsByDate(publicDocs)

	// Apply limit if specified
	if ds.indexLimit > 0 && len(publicDocs) > ds.indexLimit {
		publicDocs = publicDocs[:ds.indexLimit]
	}

	// Load cached JSON-LD index
	cached, err := ds.loadIndex()
	if err != nil {
		http.Error(w, "Failed to load index", http.StatusInternalServerError)
		return
	}

	// Render HTML list
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	faviconLink := ""
	if faviconPath := ds.GetFaviconPath(); faviconPath != "" {
		faviconLink = fmt.Sprintf(`<link rel="icon" href="%s">`, faviconPath)
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Blog Posts - Tens City</title>
    %s
    <link rel="alternate" type="application/rss+xml" title="All Posts - Tens City" href="%s/posts.rss">
    %s
    <script type="application/ld+json">
%s
    </script>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; }
        h1 { color: #333; }
        .doc-list { list-style: none; padding: 0; }
        .doc-item { margin: 1rem 0; padding: 1rem; border: 1px solid #ddd; border-radius: 4px; }
        .doc-item h2 { margin: 0 0 0.5rem 0; }
        .doc-item a { color: #0066cc; text-decoration: none; }
        .doc-item a:hover { text-decoration: underline; }
        .doc-meta { color: #666; font-size: 0.9375rem; }
        .doc-description { margin-top: 0.5rem; }
    </style>
</head>
<body>
    <h1>Blog Posts</h1>
    <ul class="doc-list">
`, faviconLink, baseURL, GoogleAnalyticsTag(ds.googleAnalyticsID), string(cached.Data))

	for _, doc := range publicDocs {
		escapedSlug := html.EscapeString(doc.Frontmatter.Slug)
		escapedTitle := html.EscapeString(doc.Frontmatter.Title)
		escapedDescription := html.EscapeString(doc.Frontmatter.Description)
		escapedDate := html.EscapeString(doc.Frontmatter.DatePublished)

		fmt.Fprintf(w, `        <li class="doc-item">
            <h2><a href="/posts/%s">%s</a></h2>
`, escapedSlug, escapedTitle)

		if doc.Frontmatter.Description != "" {
			fmt.Fprintf(w, `            <p class="doc-description">%s</p>
`, escapedDescription)
		}

		fmt.Fprintf(w, `            <div class="doc-meta">Published: %s</div>
        </li>
`, escapedDate)
	}

	fmt.Fprintf(w, `    </ul>
</body>
</html>`)
}

// HandleDoc handles GET /posts/:slug - render a single document
func (ds *DocServer) HandleDoc(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

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

	// Set cache headers - use no-cache to always revalidate but allow ETag caching
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", cached.ETag)
	w.Header().Set("Last-Modified", cached.Modified.UTC().Format(http.TimeFormat))

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == cached.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if wantsJSON {
		// Return JSON-LD
		jsonld := doc.ToJSONLD(baseURL)
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
	jsonld := doc.ToJSONLD(baseURL)
	jsonldBytes, _ := json.MarshalIndent(jsonld, "    ", "  ")

	// HTML-escape user-provided values
	escapedTitle := html.EscapeString(doc.Frontmatter.Title)
	escapedLang := html.EscapeString(doc.Frontmatter.Lang)
	escapedDatePublished := html.EscapeString(doc.Frontmatter.DatePublished)
	escapedDateModified := html.EscapeString(doc.Frontmatter.DateModified)

	// Extract author URL from frontmatter for edit link comparison
	authorURL := extractAuthorURL(doc.Frontmatter.Author)

	// Extract username for RSS feed link
	userName := extractGitHubUsername(authorURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check for favicon
	faviconLink := ""
	if faviconPath := ds.GetFaviconPath(); faviconPath != "" {
		faviconLink = fmt.Sprintf(`<link rel="icon" href="%s">`, faviconPath)
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    %s
    %s`, escapedLang, escapedTitle, faviconLink, GoogleAnalyticsTag(ds.googleAnalyticsID))

	// Add RSS autodiscovery link if we have a username
	if userName != "" {
		escapedUserName := html.EscapeString(userName)
		fmt.Fprintf(w, `
    <link rel="alternate" type="application/rss+xml" title="%s's Posts" href="%s/u/%s/posts.rss">`,
			escapedUserName, baseURL, escapedUserName)
	}

	fmt.Fprintf(w, `
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
        img { max-width: 100%%; height: auto; display: block; margin: 1rem 0; }
        table { border-collapse: collapse; width: 100%%; margin: 1rem 0; }
        th, td { border: 1px solid #ddd; padding: 0.5rem 0.75rem; text-align: left; }
        th { background: #f4f4f4; font-weight: 600; }
        tr:nth-child(even) { background: #f9f9f9; }
        .post-header { margin-bottom: 2rem; }
        .post-tags { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-top: 1rem; }
        .tag { background: #e6f3ff; color: #0066cc; padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.9375rem; font-weight: 500; text-decoration: none; display: inline-block; transition: all 0.2s; }
        .tag:hover { background: #0066cc; color: white; }
        .footer { margin-top: 3rem; padding-top: 2rem; border-top: 1px solid #e0e0e0; color: #666; font-size: 0.9375rem; }
        .footer-menu { display: flex; flex-wrap: wrap; gap: 1rem; align-items: center; margin-bottom: 1rem; }
        .footer-menu a { color: #0066cc; text-decoration: none; }
        .footer-menu a:hover { text-decoration: underline; }
        .footer-dates { margin-top: 0.5rem; }
        .footer-edit { display: none; }
        .footer-edit.visible { display: inline; }
        .cid-link { color: #0066cc; text-decoration: none; font-family: monospace; }
        .cid-link:hover { text-decoration: underline; }
        .modal { display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%%; height: 100%%; background-color: rgba(0,0,0,0.4); }
        .modal-content { background-color: #fefefe; margin: 5%% auto; padding: 2rem; border: 1px solid #888; border-radius: 8px; width: 80%%; max-width: 900px; max-height: 80vh; overflow: auto; }
        .close { color: #aaa; float: right; font-size: 28px; font-weight: bold; cursor: pointer; }
        .close:hover, .close:focus { color: #000; }
        .modal h2 { margin-top: 0; }
        .modal pre { background: #f4f4f4; padding: 1rem; border-radius: 4px; overflow-x: auto; white-space: pre-wrap; word-wrap: break-word; }
        .modal-actions { margin-top: 1rem; }
        .modal-actions a { margin-right: 1rem; }
    </style>
</head>
<body>
    <div class="post-header">`,
		jsonldBytes)

	// Add tags at the top of the post
	allTags := append([]string{}, doc.Frontmatter.Tags...)
	allTags = append(allTags, doc.Frontmatter.Keywords...)

	if len(allTags) > 0 {
		fmt.Fprintf(w, `
        <div class="post-tags">`)
		for _, tag := range allTags {
			if tag != "" {
				escapedTag := html.EscapeString(tag)
				urlEncodedTag := url.PathEscape(tag)
				fmt.Fprintf(w, `
            <a href="/tags/%s" class="tag">%s</a>`, urlEncodedTag, escapedTag)
			}
		}
		fmt.Fprintf(w, `
        </div>`)
	}

	fmt.Fprintf(w, `
    </div>
    %s
    <div class="footer">
        <div class="footer-menu">
            <a href="/">← Home</a>
            <span>•</span>
            <a href="/posts">All Posts</a>`,
		doc.HTML)

	// Add edit link (will be shown/hidden by JavaScript based on authorship)
	escapedAuthorURL := html.EscapeString(authorURL)
	fmt.Fprintf(w, `
            <a href="#" class="footer-edit" id="editLink" data-author-url="%s">✏️ Edit</a>
        </div>
        <div class="footer-dates">
            Published: %s`, escapedAuthorURL, escapedDatePublished)

	if doc.Frontmatter.DateModified != "" {
		fmt.Fprintf(w, ` | Modified: %s`, escapedDateModified)
	}

	fmt.Fprintf(w, `
        </div>
    </div>

    <script>
        (function() {
            const editLink = document.getElementById('editLink');
            if (!editLink) return;
            
            const authorURL = editLink.getAttribute('data-author-url');
            if (!authorURL) return;

            // Get user info from localStorage (set by the Supabase web client in public/tens-city.js)
            // The key format is 'sb-{project-ref}-auth-token' from Supabase client configuration
            const userStr = localStorage.getItem('sb-gquccmagslcoytktmcfa-auth-token');
            if (!userStr) return;

            try {
                const authData = JSON.parse(userStr);
                if (!authData || !authData.user) return;

                const user = authData.user;
                const userMetadata = user.user_metadata || {};
                const userName = userMetadata.user_name || '';
                
                // Check if the user's GitHub URL matches the author URL
                if (userName && authorURL.includes('github.com/' + userName)) {
                    editLink.classList.add('visible');
                    // For now, edit link redirects to GitHub (can be customized later)
                    editLink.href = authorURL;
                    editLink.target = '_blank';
                    editLink.title = 'Edit on GitHub';
                }
            } catch (e) {
                console.error('Failed to parse user auth data:', e);
            }
        })();
    </script>
    <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
    <script>mermaid.initialize({startOnLoad: true});</script>
</body>
</html>`)
}

// HandleDocJSONLD handles GET /posts/:slug.jsonld - return JSON-LD only
func (ds *DocServer) HandleDocJSONLD(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

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

	// Set cache headers - use no-cache to always revalidate but allow ETag caching
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", cached.ETag)
	w.Header().Set("Last-Modified", cached.Modified.UTC().Format(http.TimeFormat))

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == cached.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	jsonld := doc.ToJSONLD(baseURL)
	data, err := json.MarshalIndent(jsonld, "", "  ")
	if err != nil {
		http.Error(w, "Failed to serialize JSON-LD", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Write(data)
}

// HandleIndexJSONLD handles GET /posts/index.jsonld - return collection index
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

	// Set cache headers - use no-cache to always revalidate but allow ETag caching
	w.Header().Set("Cache-Control", "no-cache")
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

// GetIndexJSONLD returns the JSON-LD index data without setting HTTP headers
// This is used for embedding JSON-LD in HTML pages
func (ds *DocServer) GetIndexJSONLD() ([]byte, error) {
	cached, err := ds.loadIndex()
	if err != nil {
		return nil, err
	}
	return cached.Data, nil
}

// GetIndexDocument returns the parsed index.md document, or nil if not found
func (ds *DocServer) GetIndexDocument() (*markdown.Document, error) {
	cached, err := ds.loadIndexDocument()
	if err != nil {
		return nil, err
	}
	if cached == nil {
		return nil, nil
	}
	return cached.Doc, nil
}

// HandleUserRSS handles GET /u/{user}/posts.rss - return RSS feed for user's blog posts
func (ds *DocServer) HandleUserRSS(w http.ResponseWriter, r *http.Request, userName string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	// Load all posts
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Filter posts by author
	var userDocs []*markdown.Document
	for _, doc := range docs {
		// Skip drafts
		if doc.Frontmatter.Draft {
			continue
		}

		// Check if the author matches the requested user
		authorURL := extractAuthorURL(doc.Frontmatter.Author)
		if extractGitHubUsername(authorURL) == userName {
			userDocs = append(userDocs, doc)
		}
	}

	// Generate RSS feed
	feedData, err := rss.GenerateUserFeed(userDocs, userName, baseURL)
	if err != nil {
		http.Error(w, "Failed to generate RSS feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write(feedData)
}

// HandleSiteRSS handles GET /posts.rss - return RSS feed for all blog posts
func (ds *DocServer) HandleSiteRSS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	// Load all posts
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Generate site-wide RSS feed
	feedData, err := rss.GenerateSiteFeed(docs, baseURL)
	if err != nil {
		http.Error(w, "Failed to generate RSS feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write(feedData)
}

// HandleRSSList handles GET /rss - return HTML page listing all available RSS feeds
func (ds *DocServer) HandleRSSList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	// Load all posts
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Collect unique authors
	authorMap := make(map[string]struct {
		name string
		url  string
	})

	for _, doc := range docs {
		// Skip drafts
		if doc.Frontmatter.Draft {
			continue
		}

		authorURL := extractAuthorURL(doc.Frontmatter.Author)
		userName := extractGitHubUsername(authorURL)
		if userName != "" {
			// Extract author name
			authorName := extractAuthorName(doc.Frontmatter.Author)
			if authorName != "" {
				authorMap[userName] = struct {
					name string
					url  string
				}{
					name: authorName,
					url:  authorURL,
				}
			}
		}
	}

	// Render HTML page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	allPostsFeedURL := fmt.Sprintf("%s/posts.rss", baseURL)

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>RSS Feeds - Tens City</title>
    %s
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; line-height: 1.6; }
        h1 { color: #333; margin-bottom: 1rem; }
        .intro { color: #666; margin-bottom: 2rem; }
        .feed-list { list-style: none; padding: 0; }
        .feed-item { margin: 1rem 0; padding: 1rem; border: 1px solid #ddd; border-radius: 4px; background: #f9f9f9; }
        .feed-item.featured { background: #e6f3ff; border-color: #0066cc; }
        .feed-item h2 { margin: 0 0 0.5rem 0; font-size: 1.25rem; }
        .feed-item a { color: #0066cc; text-decoration: none; font-family: monospace; }
        .feed-item a:hover { text-decoration: underline; }
        .feed-meta { color: #666; font-size: 0.9375rem; margin-top: 0.5rem; }
        .author-link { color: #0066cc; text-decoration: none; }
        .author-link:hover { text-decoration: underline; }
        .back-link { display: inline-block; margin-bottom: 1rem; color: #0066cc; text-decoration: none; }
        .back-link:hover { text-decoration: underline; }
        .section-title { color: #333; margin-top: 2rem; margin-bottom: 1rem; font-size: 1.5rem; }
    </style>
</head>
<body>
    <a href="/" class="back-link">← Back to Home</a>
    <h1>RSS Feeds</h1>
    <p class="intro">Subscribe to RSS feeds to stay updated with the latest posts.</p>
    <ul class="feed-list">
        <li class="feed-item featured">
            <h2>All Posts</h2>
            <a href="%s">%s</a>
            <div class="feed-meta">
                Latest blog posts from all authors on Tens City
            </div>
        </li>
    </ul>
    <h2 class="section-title">Author Feeds</h2>
    <ul class="feed-list">
`, GoogleAnalyticsTag(ds.googleAnalyticsID), allPostsFeedURL, allPostsFeedURL)

	// Sort authors alphabetically by username for consistent ordering
	var userNames []string
	for userName := range authorMap {
		userNames = append(userNames, userName)
	}
	sort.Strings(userNames)

	for _, userName := range userNames {
		author := authorMap[userName]
		escapedUserName := html.EscapeString(userName)
		escapedAuthorName := html.EscapeString(author.name)
		escapedAuthorURL := html.EscapeString(author.url)

		fmt.Fprintf(w, `        <li class="feed-item">
            <h2>%s</h2>
            <a href="%s/u/%s/posts.rss">%s/u/%s/posts.rss</a>
            <div class="feed-meta">
                Author: <a href="%s" class="author-link" target="_blank">%s</a>
            </div>
        </li>
`, escapedAuthorName, baseURL, escapedUserName, baseURL, escapedUserName, escapedAuthorURL, escapedAuthorName)
	}

	if len(userNames) == 0 {
		fmt.Fprintf(w, `        <li class="feed-item">No author feeds available yet.</li>
`)
	}

	fmt.Fprintf(w, `    </ul>
</body>
</html>`)
}

// generateETag generates an ETag from content
func generateETag(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf(`"%x"`, hash)
}

// extractAuthorURL extracts the author's URL from frontmatter
func extractAuthorURL(author interface{}) string {
	switch a := author.(type) {
	case map[string]interface{}:
		if url, ok := a["url"].(string); ok {
			return url
		}
	case []interface{}:
		// For multiple authors, return the first author's URL
		if len(a) > 0 {
			if m, ok := a[0].(map[string]interface{}); ok {
				if url, ok := m["url"].(string); ok {
					return url
				}
			}
		}
	}
	return ""
}

// extractGitHubUsername extracts the GitHub username from a GitHub profile URL
// Returns empty string if the URL is not a valid GitHub profile URL
func extractGitHubUsername(githubURL string) string {
	if githubURL == "" || !strings.Contains(githubURL, "github.com/") {
		return ""
	}

	// Extract just the username from GitHub URL
	// Expected format: https://github.com/username or https://github.com/username/
	parts := strings.Split(githubURL, "github.com/")
	if len(parts) < 2 {
		return ""
	}

	userPath := strings.TrimPrefix(parts[1], "/")
	userPath = strings.TrimSuffix(userPath, "/")

	// Extract just the username (first path segment)
	pathParts := strings.Split(userPath, "/")
	if len(pathParts) > 0 && pathParts[0] != "" {
		return pathParts[0]
	}

	return ""
}

// extractAuthorName extracts the author's name from frontmatter
func extractAuthorName(author interface{}) string {
	switch a := author.(type) {
	case map[string]interface{}:
		if name, ok := a["name"].(string); ok {
			return name
		}
	case []interface{}:
		// For multiple authors, return the first author's name
		if len(a) > 0 {
			if m, ok := a[0].(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					return name
				}
			}
		}
	}
	return ""
}

// TagInfo represents a tag with its frequency
type TagInfo struct {
	Tag   string
	Count int
}

// collectTags aggregates all tags from documents and returns them with frequencies
func (ds *DocServer) collectTags() ([]TagInfo, error) {
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		return nil, err
	}

	tagCounts := make(map[string]int)
	for _, doc := range docs {
		// Skip drafts
		if doc.Frontmatter.Draft {
			continue
		}

		// Collect from both tags and keywords
		allTags := append([]string{}, doc.Frontmatter.Tags...)
		allTags = append(allTags, doc.Frontmatter.Keywords...)

		for _, tag := range allTags {
			if tag != "" {
				tagCounts[tag]++
			}
		}
	}

	// Convert map to slice and sort by count descending, then alphabetically
	tags := make([]TagInfo, 0, len(tagCounts))
	for tag, count := range tagCounts {
		tags = append(tags, TagInfo{Tag: tag, Count: count})
	}

	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Tag < tags[j].Tag
	})

	return tags, nil
}

// buildTagsCollectionJSONLD creates a JSON-LD CollectionPage for all tags
func (ds *DocServer) buildTagsCollectionJSONLD(tags []TagInfo, baseURL string) map[string]interface{} {
	// Build items list for each tag
	items := make([]interface{}, 0, len(tags))
	for i, tagInfo := range tags {
		item := map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"item": map[string]interface{}{
				"@type": "DefinedTerm",
				"name":  tagInfo.Tag,
				"url":   fmt.Sprintf("%s/tags/%s", baseURL, url.PathEscape(tagInfo.Tag)),
			},
		}
		items = append(items, item)
	}

	return map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "CollectionPage",
		"name":            "Tags",
		"description":     "Collection of all tags used in blog posts",
		"url":             fmt.Sprintf("%s/tags", baseURL),
		"numberOfItems":   len(items),
		"itemListElement": items,
	}
}

// buildTagSearchResultsJSONLD creates a JSON-LD CollectionPage for tag search results
func (ds *DocServer) buildTagSearchResultsJSONLD(tag string, docs []*markdown.Document, baseURL string) map[string]interface{} {
	// Build items list for each document
	items := make([]interface{}, 0, len(docs))
	for i, doc := range docs {
		article := map[string]interface{}{
			"@type":    "Article",
			"headline": doc.Frontmatter.Title,
			"url":      fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
		}

		if doc.Frontmatter.Description != "" {
			article["description"] = doc.Frontmatter.Description
		}

		if doc.Frontmatter.DatePublished != "" {
			article["datePublished"] = doc.Frontmatter.DatePublished
		}

		item := map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"item":     article,
		}

		items = append(items, item)
	}

	return map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "CollectionPage",
		"name":            fmt.Sprintf("Posts tagged with \"%s\"", tag),
		"description":     fmt.Sprintf("Blog posts tagged with %s", tag),
		"url":             fmt.Sprintf("%s/tags/%s", baseURL, url.PathEscape(tag)),
		"numberOfItems":   len(items),
		"itemListElement": items,
	}
}

// HandleTagsPage handles GET /tags - display word cloud of all tags
func (ds *DocServer) HandleTagsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	tags, err := ds.collectTags()
	if err != nil {
		http.Error(w, "Failed to load tags", http.StatusInternalServerError)
		return
	}

	// Build JSON-LD for the tags collection
	jsonld := ds.buildTagsCollectionJSONLD(tags, baseURL)
	jsonldBytes, err := json.MarshalIndent(jsonld, "    ", "  ")
	if err != nil {
		http.Error(w, "Failed to generate JSON-LD", http.StatusInternalServerError)
		return
	}

	// Calculate min and max counts for sizing
	var minCount, maxCount int
	if len(tags) > 0 {
		minCount = tags[len(tags)-1].Count
		maxCount = tags[0].Count
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Tags - Tens City</title>
    %s
    <script type="application/ld+json">
    %s
    </script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        :root {
            --primary: #2563eb;
            --primary-dark: #1e40af;
            --text: #1f2937;
            --text-light: #6b7280;
            --bg: #ffffff;
            --bg-alt: #f9fafb;
            --border: #e5e7eb;
            --shadow: rgba(0, 0, 0, 0.1);
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.7;
            color: var(--text);
            background: var(--bg-alt);
        }
        
        .hero {
            background: linear-gradient(135deg, var(--primary) 0%%, var(--primary-dark) 100%%);
            color: white;
            padding: 3rem 2rem;
            text-align: center;
            box-shadow: 0 4px 6px var(--shadow);
        }
        
        .hero h1 {
            font-size: 2.5rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }
        
        .hero p {
            font-size: 1.125rem;
            opacity: 0.95;
        }
        
        main {
            max-width: 1200px;
            margin: 3rem auto;
            padding: 0 1.5rem;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 2rem;
            color: var(--primary);
            text-decoration: none;
            font-weight: 500;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .tag-cloud {
            background: var(--bg);
            border-radius: 12px;
            border: 1px solid var(--border);
            padding: 3rem;
            text-align: center;
            box-shadow: 0 1px 3px var(--shadow);
            line-height: 2.5;
        }
        
        .tag-cloud a {
            display: inline-block;
            margin: 0.5rem;
            padding: 0.5rem 1rem;
            color: var(--primary);
            text-decoration: none;
            border-radius: 8px;
            transition: all 0.2s;
            background: var(--bg-alt);
        }
        
        .tag-cloud a:hover {
            background: var(--primary);
            color: white;
            transform: scale(1.05);
        }
        
        .empty-state {
            text-align: center;
            padding: 4rem 2rem;
            color: var(--text-light);
        }
        
        footer {
            text-align: center;
            padding: 3rem 2rem;
            margin-top: 4rem;
            border-top: 1px solid var(--border);
            color: var(--text-light);
            font-size: 0.9375rem;
        }
        
        footer a {
            color: var(--primary);
            text-decoration: none;
            font-weight: 500;
        }
        
        footer a:hover {
            text-decoration: underline;
        }
        
        @media (max-width: 768px) {
            .hero h1 {
                font-size: 2rem;
            }
            
            .tag-cloud {
                padding: 2rem 1.5rem;
            }
        }
    </style>
</head>
<body>
    <div class="hero">
        <h1>🏷️ Tags</h1>
        <p>Explore posts by topic</p>
    </div>
    
    <main>
        <a href="/" class="back-link">← Back to Home</a>
        
        <div class="tag-cloud">
`, GoogleAnalyticsTag(ds.googleAnalyticsID), jsonldBytes)

	if len(tags) == 0 {
		fmt.Fprintf(w, `            <div class="empty-state">No tags found</div>
`)
	} else {
		for _, tagInfo := range tags {
			// Calculate font size based on count (range: 1.0rem to 3.0rem)
			var fontSize float64
			if maxCount > minCount {
				fontSize = 1.0 + (float64(tagInfo.Count-minCount)/float64(maxCount-minCount))*2.0
			} else {
				fontSize = 2.0
			}

			escapedTag := html.EscapeString(tagInfo.Tag)
			urlEncodedTag := url.PathEscape(tagInfo.Tag)
			fmt.Fprintf(w, `            <a href="/tags/%s" style="font-size: %.2frem;" title="%s (%d post%s)">%s</a>
`, urlEncodedTag, fontSize, escapedTag, tagInfo.Count, pluralize(tagInfo.Count), escapedTag)
		}
	}

	fmt.Fprintf(w, `        </div>
    </main>
    
    <footer>
        <p>
            <a href="/posts">📝 All Posts</a> • 
            <a href="/search">🔍 Search</a> • 
            <a href="/rss">📡 RSS Feeds</a> • 
            Built with <a href="https://github.com/stackdump/tens-city" target="_blank">Tens City</a>
        </p>
    </footer>
</body>
</html>`)
}

// HandleTagPage handles GET /tags/{tag} - show posts with a specific tag
func (ds *DocServer) HandleTagPage(w http.ResponseWriter, r *http.Request, tag string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	// Load all documents
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Filter documents by tag
	var filteredDocs []*markdown.Document
	for _, doc := range docs {
		// Skip drafts
		if doc.Frontmatter.Draft {
			continue
		}

		// Check if document has the tag
		hasTag := false
		for _, t := range doc.Frontmatter.Tags {
			if t == tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			for _, k := range doc.Frontmatter.Keywords {
				if k == tag {
					hasTag = true
					break
				}
			}
		}

		if hasTag {
			filteredDocs = append(filteredDocs, doc)
		}
	}

	// Sort by DatePublished descending (newest first), then by Title ascending
	markdown.SortDocumentsByDate(filteredDocs)

	// Build JSON-LD for the tag search results
	jsonld := ds.buildTagSearchResultsJSONLD(tag, filteredDocs, baseURL)
	jsonldBytes, err := json.MarshalIndent(jsonld, "    ", "  ")
	if err != nil {
		http.Error(w, "Failed to generate JSON-LD", http.StatusInternalServerError)
		return
	}

	escapedTag := html.EscapeString(tag)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Tag: %s - Tens City</title>
    %s
    <script type="application/ld+json">
    %s
    </script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        :root {
            --primary: #2563eb;
            --primary-dark: #1e40af;
            --text: #1f2937;
            --text-light: #6b7280;
            --bg: #ffffff;
            --bg-alt: #f9fafb;
            --border: #e5e7eb;
            --shadow: rgba(0, 0, 0, 0.1);
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.7;
            color: var(--text);
            background: var(--bg-alt);
        }
        
        .hero {
            background: linear-gradient(135deg, var(--primary) 0%%, var(--primary-dark) 100%%);
            color: white;
            padding: 3rem 2rem;
            text-align: center;
            box-shadow: 0 4px 6px var(--shadow);
        }
        
        .hero h1 {
            font-size: 2.5rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }
        
        .hero p {
            font-size: 1.125rem;
            opacity: 0.95;
        }
        
        main {
            max-width: 900px;
            margin: 3rem auto;
            padding: 0 1.5rem;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 2rem;
            color: var(--primary);
            text-decoration: none;
            font-weight: 500;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .post-list {
            list-style: none;
            padding: 0;
        }
        
        .post-item {
            margin: 1.5rem 0;
            padding: 1.5rem;
            background: var(--bg);
            border: 1px solid var(--border);
            border-radius: 12px;
            box-shadow: 0 1px 3px var(--shadow);
            transition: all 0.3s ease;
        }
        
        .post-item:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 16px var(--shadow);
            border-color: var(--primary);
        }
        
        .post-item h2 {
            margin: 0 0 0.75rem 0;
            font-size: 1.5rem;
        }
        
        .post-item a {
            color: var(--text);
            text-decoration: none;
            transition: color 0.2s;
        }
        
        .post-item a:hover {
            color: var(--primary);
        }
        
        .post-meta {
            color: var(--text-light);
            font-size: 0.9375rem;
            margin-bottom: 0.75rem;
        }
        
        .post-description {
            color: var(--text-light);
            line-height: 1.6;
        }
        
        .empty-state {
            text-align: center;
            padding: 4rem 2rem;
            color: var(--text-light);
        }
        
        footer {
            text-align: center;
            padding: 3rem 2rem;
            margin-top: 4rem;
            border-top: 1px solid var(--border);
            color: var(--text-light);
            font-size: 0.9375rem;
        }
        
        footer a {
            color: var(--primary);
            text-decoration: none;
            font-weight: 500;
        }
        
        footer a:hover {
            text-decoration: underline;
        }
        
        @media (max-width: 768px) {
            .hero h1 {
                font-size: 2rem;
            }
        }
    </style>
</head>
<body>
    <div class="hero">
        <h1>🏷️ %s</h1>
        <p>%d post%s tagged with "%s"</p>
    </div>
    
    <main>
        <a href="/tags" class="back-link">← All Tags</a>
        
        <ul class="post-list">
`, escapedTag, GoogleAnalyticsTag(ds.googleAnalyticsID), jsonldBytes, escapedTag, len(filteredDocs), pluralize(len(filteredDocs)), escapedTag)

	if len(filteredDocs) == 0 {
		fmt.Fprintf(w, `            <div class="empty-state">No posts found with this tag</div>
`)
	} else {
		for _, doc := range filteredDocs {
			escapedSlug := html.EscapeString(doc.Frontmatter.Slug)
			escapedTitle := html.EscapeString(doc.Frontmatter.Title)
			escapedDescription := html.EscapeString(doc.Frontmatter.Description)
			escapedDate := html.EscapeString(doc.Frontmatter.DatePublished)

			fmt.Fprintf(w, `            <li class="post-item">
                <h2><a href="/posts/%s">%s</a></h2>
`, escapedSlug, escapedTitle)

			if doc.Frontmatter.DatePublished != "" {
				fmt.Fprintf(w, `                <div class="post-meta">Published: %s</div>
`, escapedDate)
			}

			if doc.Frontmatter.Description != "" {
				fmt.Fprintf(w, `                <p class="post-description">%s</p>
`, escapedDescription)
			}

			fmt.Fprintf(w, `            </li>
`)
		}
	}

	fmt.Fprintf(w, `        </ul>
    </main>
    
    <footer>
        <p>
            <a href="/posts">📝 All Posts</a> • 
            <a href="/tags">🏷️ All Tags</a> • 
            <a href="/search">🔍 Search</a> • 
            <a href="/rss">📡 RSS Feeds</a> • 
            Built with <a href="https://github.com/stackdump/tens-city" target="_blank">Tens City</a>
        </p>
    </footer>
</body>
</html>`)
}

// pluralize returns "s" if count is not 1, otherwise empty string
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// HandleSitemap handles GET /sitemap.xml - return XML sitemap for all pages
func (ds *DocServer) HandleSitemap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	// Load all posts
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Generate sitemap
	sitemapData, err := sitemap.GenerateSitemap(docs, baseURL)
	if err != nil {
		http.Error(w, "Failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(sitemapData)
}

// HandleSearch handles GET /search - client-side search page
func (ds *DocServer) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	baseURL := httputil.GetBaseURL(r, ds.fallbackURL)

	// Load all posts for client-side search
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Filter out drafts and prepare search data
	type SearchResult struct {
		Title         string   `json:"title"`
		Description   string   `json:"description"`
		Slug          string   `json:"slug"`
		URL           string   `json:"url"`
		Tags          []string `json:"tags"`
		DatePublished string   `json:"datePublished"`
	}

	var searchData []SearchResult
	for _, doc := range docs {
		if doc.Frontmatter.Draft {
			continue
		}

		allTags := append([]string{}, doc.Frontmatter.Tags...)
		allTags = append(allTags, doc.Frontmatter.Keywords...)

		searchData = append(searchData, SearchResult{
			Title:         doc.Frontmatter.Title,
			Description:   doc.Frontmatter.Description,
			Slug:          doc.Frontmatter.Slug,
			URL:           fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug),
			Tags:          allTags,
			DatePublished: doc.Frontmatter.DatePublished,
		})
	}

	searchDataJSON, err := json.Marshal(searchData)
	if err != nil {
		http.Error(w, "Failed to prepare search data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Search - Tens City</title>
    %s
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        :root {
            --primary: #2563eb;
            --primary-dark: #1e40af;
            --text: #1f2937;
            --text-light: #6b7280;
            --bg: #ffffff;
            --bg-alt: #f9fafb;
            --border: #e5e7eb;
            --shadow: rgba(0, 0, 0, 0.1);
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            line-height: 1.7;
            color: var(--text);
            background: var(--bg-alt);
        }
        
        .hero {
            background: linear-gradient(135deg, var(--primary) 0%%, var(--primary-dark) 100%%);
            color: white;
            padding: 3rem 2rem;
            text-align: center;
            box-shadow: 0 4px 6px var(--shadow);
        }
        
        .hero h1 {
            font-size: 2.5rem;
            font-weight: 700;
            margin-bottom: 1rem;
        }
        
        .search-container {
            max-width: 600px;
            margin: 0 auto;
        }
        
        .search-box {
            width: 100%%;
            padding: 1rem 1.5rem;
            font-size: 1.125rem;
            border: 2px solid white;
            border-radius: 8px;
            outline: none;
        }
        
        .search-box:focus {
            border-color: var(--bg-alt);
        }
        
        main {
            max-width: 900px;
            margin: 3rem auto;
            padding: 0 1.5rem;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 2rem;
            color: var(--primary);
            text-decoration: none;
            font-weight: 500;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .results-info {
            margin-bottom: 1.5rem;
            color: var(--text-light);
        }
        
        .search-results {
            list-style: none;
            padding: 0;
        }
        
        .result-item {
            margin: 1.5rem 0;
            padding: 1.5rem;
            background: var(--bg);
            border: 1px solid var(--border);
            border-radius: 12px;
            box-shadow: 0 1px 3px var(--shadow);
            transition: all 0.3s ease;
        }
        
        .result-item:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 16px var(--shadow);
            border-color: var(--primary);
        }
        
        .result-item h2 {
            margin: 0 0 0.75rem 0;
            font-size: 1.5rem;
        }
        
        .result-item a {
            color: var(--text);
            text-decoration: none;
            transition: color 0.2s;
        }
        
        .result-item a:hover {
            color: var(--primary);
        }
        
        .result-meta {
            color: var(--text-light);
            font-size: 0.9375rem;
            margin-bottom: 0.75rem;
        }
        
        .result-description {
            color: var(--text-light);
            line-height: 1.6;
        }
        
        .result-tags {
            display: flex;
            flex-wrap: wrap;
            gap: 0.5rem;
            margin-top: 0.75rem;
        }
        
        .tag {
            background: #e6f3ff;
            color: var(--primary);
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.9375rem;
            font-weight: 500;
        }
        
        .empty-state {
            text-align: center;
            padding: 4rem 2rem;
            color: var(--text-light);
        }
        
        footer {
            text-align: center;
            padding: 3rem 2rem;
            margin-top: 4rem;
            border-top: 1px solid var(--border);
            color: var(--text-light);
            font-size: 0.9375rem;
        }
        
        footer a {
            color: var(--primary);
            text-decoration: none;
            font-weight: 500;
        }
        
        footer a:hover {
            text-decoration: underline;
        }
        
        @media (max-width: 768px) {
            .hero h1 {
                font-size: 2rem;
            }
            
            .search-box {
                font-size: 1rem;
            }
        }
    </style>
</head>
<body>
    <div class="hero">
        <h1>🔍 Search</h1>
        <div class="search-container">
            <input type="text" id="searchInput" class="search-box" placeholder="Search posts..." autofocus>
        </div>
    </div>
    
    <main>
        <a href="/" class="back-link">← Back to Home</a>
        
        <div id="resultsInfo" class="results-info"></div>
        
        <ul id="searchResults" class="search-results"></ul>
    </main>
    
    <footer>
        <p>
            <a href="/posts">📝 All Posts</a> • 
            <a href="/tags">🏷️ All Tags</a> • 
            <a href="/rss">📡 RSS Feeds</a> • 
            Built with <a href="https://github.com/stackdump/tens-city" target="_blank">Tens City</a>
        </p>
    </footer>
    
    <script>
        const searchData = %s;
        const searchInput = document.getElementById('searchInput');
        const searchResults = document.getElementById('searchResults');
        const resultsInfo = document.getElementById('resultsInfo');
        
        function performSearch() {
            const query = searchInput.value.toLowerCase().trim();
            
            if (!query) {
                searchResults.innerHTML = '<div class="empty-state">Enter a search term to find posts</div>';
                resultsInfo.textContent = '';
                return;
            }
            
            const results = searchData.filter(post => {
                return post.title.toLowerCase().includes(query) ||
                       post.description.toLowerCase().includes(query) ||
                       post.tags.some(tag => tag.toLowerCase().includes(query));
            });
            
            if (results.length === 0) {
                searchResults.innerHTML = '<div class="empty-state">No posts found matching "' + escapeHtml(query) + '"</div>';
                resultsInfo.textContent = '';
                return;
            }
            
            resultsInfo.textContent = 'Found ' + results.length + ' post' + (results.length === 1 ? '' : 's');
            
            searchResults.innerHTML = results.map(post => {
                let html = '<li class="result-item">';
                html += '<h2><a href="' + escapeHtml(post.url) + '">' + escapeHtml(post.title) + '</a></h2>';
                
                if (post.datePublished) {
                    html += '<div class="result-meta">Published: ' + escapeHtml(new Date(post.datePublished).toLocaleDateString()) + '</div>';
                }
                
                if (post.description) {
                    html += '<p class="result-description">' + escapeHtml(post.description) + '</p>';
                }
                
                if (post.tags && post.tags.length > 0) {
                    html += '<div class="result-tags">';
                    post.tags.forEach(tag => {
                        html += '<span class="tag">' + escapeHtml(tag) + '</span>';
                    });
                    html += '</div>';
                }
                
                html += '</li>';
                return html;
            }).join('');
        }
        
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        // Search on input
        searchInput.addEventListener('input', performSearch);
        
        // Initial search if there's a query parameter
        const urlParams = new URLSearchParams(window.location.search);
        const initialQuery = urlParams.get('q');
        if (initialQuery) {
            searchInput.value = initialQuery;
            performSearch();
        }
    </script>
</body>
</html>`, GoogleAnalyticsTag(ds.googleAnalyticsID), string(searchDataJSON))
}
