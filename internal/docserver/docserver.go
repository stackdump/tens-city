package docserver

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stackdump/tens-city/internal/canonical"
	"github.com/stackdump/tens-city/internal/markdown"
	"github.com/stackdump/tens-city/internal/rss"
	"github.com/stackdump/tens-city/internal/seal"
)

// Storage interface for saving JSON-LD documents
type Storage interface {
	SaveObject(cid string, raw []byte, canonical []byte) error
}

// DocServer handles markdown document requests
type DocServer struct {
	contentDir string
	baseURL    string
	cache      *DocumentCache
	storage    Storage
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
	CID      string // CID of the JSON-LD representation
}

// CachedIndex represents the cached document index
type CachedIndex struct {
	Data     []byte
	ETag     string
	Modified time.Time
}

// NewDocServer creates a new document server
func NewDocServer(contentDir, baseURL string) *DocServer {
	return NewDocServerWithStorage(contentDir, baseURL, nil)
}

// NewDocServerWithStorage creates a new document server with storage
func NewDocServerWithStorage(contentDir, baseURL string, storage Storage) *DocServer {
	return &DocServer{
		contentDir: contentDir,
		baseURL:    baseURL,
		storage:    storage,
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

	// Convert to JSON-LD and compute CID if storage is available
	var docCID string
	if ds.storage != nil {
		jsonld := doc.ToJSONLD(ds.baseURL)

		// Serialize to JSON using canonical encoding
		raw, err := canonical.MarshalJSON(jsonld)
		if err != nil {
			log.Printf("Warning: failed to marshal JSON-LD for %s: %v", slug, err)
		} else {
			// Compute CID
			cid, canonicalData, err := seal.SealJSONLD(raw)
			if err != nil {
				log.Printf("Warning: failed to seal JSON-LD for %s: %v", slug, err)
			} else {
				// Save to storage
				if err := ds.storage.SaveObject(cid, raw, canonicalData); err != nil {
					log.Printf("Warning: failed to save JSON-LD for %s: %v", slug, err)
				} else {
					docCID = cid
				}
			}
		}
	}

	cached = &CachedDoc{
		Doc:      doc,
		ETag:     etag,
		Modified: fileInfo.ModTime(),
		CID:      docCID,
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
		escapedSlug := html.EscapeString(doc.Frontmatter.Slug)
		escapedTitle := html.EscapeString(doc.Frontmatter.Title)
		escapedDescription := html.EscapeString(doc.Frontmatter.Description)
		escapedDate := html.EscapeString(doc.Frontmatter.DatePublished)

		fmt.Fprintf(w, `        <li class="doc-item">
            <h2><a href="/docs/%s">%s</a></h2>
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

	// HTML-escape user-provided values
	escapedSlug := html.EscapeString(slug)
	escapedTitle := html.EscapeString(doc.Frontmatter.Title)
	escapedLang := html.EscapeString(doc.Frontmatter.Lang)
	escapedDatePublished := html.EscapeString(doc.Frontmatter.DatePublished)
	escapedDateModified := html.EscapeString(doc.Frontmatter.DateModified)
	escapedJSONLD := html.EscapeString(string(jsonldBytes))

	// Extract author URL from frontmatter for edit link comparison
	authorURL := extractAuthorURL(doc.Frontmatter.Author)

	// Extract username for RSS feed link
	userName := ""
	if authorURL != "" && strings.Contains(authorURL, "github.com/") {
		parts := strings.Split(authorURL, "github.com/")
		if len(parts) > 1 {
			userName = strings.TrimSuffix(parts[1], "/")
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s - Tens City</title>`, escapedLang, escapedTitle)

	// Add RSS autodiscovery link if we have a username
	if userName != "" {
		escapedUserName := html.EscapeString(userName)
		fmt.Fprintf(w, `
    <link rel="alternate" type="application/rss+xml" title="%s's Documents" href="%s/u/%s/docs.rss">`,
			escapedUserName, ds.baseURL, escapedUserName)
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
        .nav { margin-bottom: 2rem; }
        .nav a { margin-right: 1rem; }
        .meta { color: #666; font-size: 0.9rem; margin-bottom: 2rem; }
        .footer { margin-top: 3rem; padding-top: 2rem; border-top: 1px solid #e0e0e0; color: #666; font-size: 0.85rem; }
        .footer-menu { display: flex; flex-wrap: wrap; gap: 1rem; align-items: center; margin-bottom: 1rem; }
        .footer-menu a { color: #0066cc; text-decoration: none; }
        .footer-menu a:hover { text-decoration: underline; }
        .footer-edit { display: none; }
        .footer-edit.visible { display: inline; }
        .cid-link { color: #0066cc; text-decoration: none; font-family: monospace; }
        .cid-link:hover { text-decoration: underline; }
        .modal { display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%%; height: 100%%; overflow: auto; background-color: rgba(0,0,0,0.4); }
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
    <div class="nav">
        <a href="/docs">← Back to Docs</a>
        <a href="/docs/%s.jsonld">JSON-LD</a>
    </div>
    <div class="meta">
        Published: %s`, string(jsonldBytes),
		escapedSlug, escapedDatePublished)

	if doc.Frontmatter.DateModified != "" {
		fmt.Fprintf(w, ` | Modified: %s`, escapedDateModified)
	}

	fmt.Fprintf(w, `
    </div>
    %s
    <div class="footer">
        <div class="footer-menu">
            <a href="/docs">← All Docs</a>
            <a href="/docs/%s.jsonld">JSON-LD</a>`, doc.HTML, escapedSlug)

	// Add CID link if available
	if cached.CID != "" {
		escapedCID := html.EscapeString(cached.CID)
		cidShort := ""
		if len(cached.CID) > 8 {
			cidShort = cached.CID[len(cached.CID)-8:]
		} else {
			cidShort = cached.CID
		}
		escapedCIDShort := html.EscapeString(cidShort)

		fmt.Fprintf(w, `
            <a href="#" class="cid-link" onclick="showCIDModal(); return false;">CID: ...%s</a>
            <a href="/o/%s">Full Object</a>`, escapedCIDShort, escapedCID)
	}

	// Add edit link (will be shown/hidden by JavaScript based on authorship)
	escapedAuthorURL := html.EscapeString(authorURL)
	fmt.Fprintf(w, `
            <a href="#" class="footer-edit" id="editLink" data-author-url="%s">✏️ Edit</a>
        </div>
    </div>`, escapedAuthorURL)

	// Add CID modal if available
	if cached.CID != "" {
		escapedCID := html.EscapeString(cached.CID)
		fmt.Fprintf(w, `
    <div id="cidModal" class="modal">
        <div class="modal-content">
            <span class="close" onclick="closeCIDModal()">&times;</span>
            <h2>JSON-LD Document</h2>
            <p><strong>CID:</strong> <code>%s</code></p>
            <div class="modal-actions">
                <a href="/o/%s" target="_blank">View JSON-LD</a>
                <a href="/docs/%s.jsonld" target="_blank">View Document JSON-LD</a>
            </div>
            <h3>Preview:</h3>
            <pre><code>%s</code></pre>
        </div>
    </div>`, escapedCID, escapedCID, escapedSlug, escapedJSONLD)
	}

	fmt.Fprintf(w, `
    <script>
        function showCIDModal() {
            document.getElementById('cidModal').style.display = 'block';
        }
        function closeCIDModal() {
            document.getElementById('cidModal').style.display = 'none';
        }
        window.onclick = function(event) {
            var modal = document.getElementById('cidModal');
            if (event.target == modal) {
                closeCIDModal();
            }
        }

        // Check if user is authenticated and is the author
        (function() {
            const editLink = document.getElementById('editLink');
            if (!editLink) return;
            
            const authorURL = editLink.getAttribute('data-author-url');
            if (!authorURL) return;

            // Get user info from localStorage (set by the main app)
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
</body>
</html>`)
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

// HandleUserRSS handles GET /u/{user}/docs.rss - return RSS feed for user's documents
func (ds *DocServer) HandleUserRSS(w http.ResponseWriter, r *http.Request, userName string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load all documents
	docs, err := markdown.ListDocuments(ds.contentDir)
	if err != nil {
		http.Error(w, "Failed to load documents", http.StatusInternalServerError)
		return
	}

	// Filter documents by author
	var userDocs []*markdown.Document
	for _, doc := range docs {
		// Skip drafts
		if doc.Frontmatter.Draft {
			continue
		}

		// Check if the author matches the requested user
		authorURL := extractAuthorURL(doc.Frontmatter.Author)
		if authorURL != "" && strings.Contains(authorURL, "github.com/"+userName) {
			userDocs = append(userDocs, doc)
		}
	}

	// Generate RSS feed
	feedData, err := rss.GenerateUserFeed(userDocs, userName, ds.baseURL)
	if err != nil {
		http.Error(w, "Failed to generate RSS feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write(feedData)
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
