// Package webserver provides the HTTP server for tens-city
package webserver

import (
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stackdump/tens-city/pkg/activitypub"
	"github.com/stackdump/tens-city/pkg/docserver"
	"github.com/stackdump/tens-city/pkg/httputil"
	"github.com/stackdump/tens-city/pkg/markdown"
	"github.com/stackdump/tens-city/pkg/store"
)

// Storage interface abstracts filesystem backends
type Storage interface {
	GetObject(cid string) ([]byte, error)
	GetLatest(user, slug string) (string, error)
	GetHistory(user, slug string) ([]store.HistoryEntry, error)
	ReadMarkdownContent(cid string) ([]byte, error)
}

// FSStorage implements Storage using filesystem
type FSStorage struct {
	store *store.FSStore
}

// NewFSStorage creates a new filesystem-backed storage
func NewFSStorage(basePath string) *FSStorage {
	return &FSStorage{store: store.NewFSStore(basePath)}
}

func (fs *FSStorage) GetObject(cid string) ([]byte, error) {
	return fs.store.ReadObject(cid)
}

func (fs *FSStorage) GetLatest(user, slug string) (string, error) {
	return fs.store.ReadLatest(user, slug)
}

func (fs *FSStorage) GetHistory(user, slug string) ([]store.HistoryEntry, error) {
	return fs.store.ReadHistory(user, slug)
}

func (fs *FSStorage) ReadMarkdownContent(cid string) ([]byte, error) {
	return fs.store.ReadMarkdownContent(cid)
}

// Server is the main HTTP server for tens-city
type Server struct {
	storage           Storage
	publicFS          fs.FS
	docServer         *docserver.DocServer
	fallbackURL       string             // Fallback Base URL when headers are not available
	googleAnalyticsID string             // Google Analytics measurement ID (empty = disabled)
	actor             *activitypub.Actor // ActivityPub actor (nil if federation disabled)
	contentDir        string             // Content directory for blog posts
}

// NewServer creates a new Server instance
func NewServer(storage Storage, publicFS fs.FS, docServer *docserver.DocServer, fallbackURL string, googleAnalyticsID string, actor *activitypub.Actor, contentDir string) *Server {
	return &Server{
		storage:           storage,
		publicFS:          publicFS,
		docServer:         docServer,
		fallbackURL:       fallbackURL,
		googleAnalyticsID: googleAnalyticsID,
		actor:             actor,
		contentDir:        contentDir,
	}
}

// Handler for /o/{cid} - get object by CID
func (s *Server) handleGetObject(w http.ResponseWriter, r *http.Request) {
	// Extract CID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/o/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "CID required", http.StatusBadRequest)
		return
	}
	cid := parts[0]

	data, err := s.storage.GetObject(cid)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Object not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting object %s: %v", cid, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/ld+json")
	w.Write(data)
}

// Handler for /o/{cid}/markdown - get markdown content for CID
func (s *Server) handleGetMarkdown(w http.ResponseWriter, r *http.Request) {
	// Extract CID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/o/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] != "markdown" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	cid := parts[0]

	data, err := s.storage.ReadMarkdownContent(cid)
	if err != nil {
		log.Printf("Error getting markdown content for %s: %v", cid, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return empty content if no markdown exists (backward compatibility)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// Handler for /u/{user}/g/{slug}/latest - get latest CID
func (s *Server) handleGetLatest(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/u/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[2] == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	user := parts[0]
	slug := parts[2]

	cid, err := s.storage.GetLatest(user, slug)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting latest for %s/%s: %v", user, slug, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(cid))
}

// Handler for /u/{user}/g/{slug}/_history - get history
func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/u/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[2] == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	user := parts[0]
	slug := parts[2]

	history, err := s.storage.GetHistory(user, slug)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting history for %s/%s: %v", user, slug, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleRobotsTxt serves a default robots.txt file
func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	baseURL := httputil.GetBaseURL(r, s.fallbackURL)
	robotsTxt := fmt.Sprintf(`User-agent: *
Allow: /

Sitemap: %s/sitemap.xml`, baseURL)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(robotsTxt))
}

// handleWellKnown serves default files for .well-known directory
func (s *Server) handleWellKnown(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/.well-known/")

	// Handle common .well-known endpoints with defaults
	switch {
	case path == "security.txt":
		// Calculate expiration date (1 year from now)
		expirationDate := time.Now().AddDate(1, 0, 0).UTC().Format("2006-01-02T15:04:05.000Z")

		// Use a placeholder that makes it clear this should be customized
		securityTxt := fmt.Sprintf(`# Security contact information
# Please customize this file with your actual security contact
Contact: mailto:security@example.com
Expires: %s
Preferred-Languages: en

# To customize: Create your own .well-known/security.txt file
# and serve it from your web root or update this handler`, expirationDate)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(securityTxt))
		return

	case path == "webfinger" && s.actor != nil:
		// ActivityPub WebFinger discovery
		s.actor.HandleWebFinger(w, r)
		return

	case path == "nodeinfo" && s.actor != nil:
		// ActivityPub NodeInfo discovery
		s.actor.HandleNodeInfoWellKnown(w, r)
		return

	default:
		http.NotFound(w, r)
		return
	}
}

// handleIndex serves the index page with embedded JSON-LD
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Read the index.html template
	data, err := fs.ReadFile(s.publicFS, "index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	htmlContent := string(data)

	// Default values
	pageTitle := "Tens City - A Minimal Blog Platform"
	pageDescription := "Simple, elegant blog platform built on content-addressable storage"
	pageIcon := "🏕️"
	pageMessage := ""

	// If docServer is available, get index.md and inject JSON-LD script tag and RSS link
	if s.docServer != nil {
		baseURL := html.EscapeString(httputil.GetBaseURL(r, s.fallbackURL))

		// Try to load index.md
		indexDoc, err := s.docServer.GetIndexDocument()
		if err == nil && indexDoc != nil {
			// Use values from index.md
			if indexDoc.Frontmatter.Title != "" {
				pageTitle = indexDoc.Frontmatter.Title
			}
			if indexDoc.Frontmatter.Description != "" {
				pageDescription = indexDoc.Frontmatter.Description
			}
			if indexDoc.Frontmatter.Icon != "" {
				pageIcon = indexDoc.Frontmatter.Icon
			}
			if indexDoc.HTML != "" {
				pageMessage = indexDoc.HTML
			}
		}

		// HTML-escape values before replacement to prevent XSS
		escapedTitle := html.EscapeString(pageTitle)
		escapedDescription := html.EscapeString(pageDescription)
		escapedIcon := html.EscapeString(pageIcon)

		// Replace placeholders in the HTML
		htmlContent = strings.Replace(htmlContent, "Tens City - A Minimal Blog Platform", escapedTitle, -1)
		htmlContent = strings.Replace(htmlContent, "Simple, elegant blog platform built on content-addressable storage", escapedDescription, -1)
		htmlContent = strings.Replace(htmlContent, "🏕️", escapedIcon, 1) // Only replace first occurrence (the emoji)
		htmlContent = strings.Replace(htmlContent, "Tens City</h1>", escapedTitle+"</h1>", 1)

		// Replace the message paragraph if we have custom HTML content
		// Note: pageMessage is already sanitized HTML from markdown rendering
		if pageMessage != "" {
			// Replace the default paragraph with the custom message
			htmlContent = strings.Replace(htmlContent,
				"<p>A minimal blog platform built on simplicity and content ownership</p>",
				pageMessage,
				1)
		}

		// Add RSS autodiscovery link
		rssLink := fmt.Sprintf(`    <link rel="alternate" type="application/rss+xml" title="All Posts - %s" href="%s/posts.rss">
`, escapedTitle, baseURL)
		htmlContent = strings.Replace(htmlContent, "</head>", rssLink+"</head>", 1)

		// Add Google Analytics tag if configured
		if s.googleAnalyticsID != "" {
			gaTag := docserver.GoogleAnalyticsTag(s.googleAnalyticsID) + "\n"
			htmlContent = strings.Replace(htmlContent, "</head>", gaTag+"</head>", 1)
		}

		// Add follow button if ActivityPub is enabled
		if s.actor != nil {
			fediHandle := s.actor.GetFediHandle()
			if fediHandle != "" {
				// Add CSS for follow button
				followCSS := `
        .section-header-title {
            display: flex;
            align-items: center;
            gap: 1rem;
            flex-wrap: wrap;
        }
        .fedi-follow-btn {
            background: linear-gradient(135deg, #6364ff 0%, #563acc 100%);
            color: white;
            border: none;
            padding: 0.4rem 0.8rem;
            border-radius: 6px;
            font-size: 0.8rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            display: inline-flex;
            align-items: center;
            gap: 0.3rem;
        }
        .fedi-follow-btn:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(99, 100, 255, 0.4);
        }
        .fedi-modal-overlay {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.5);
            z-index: 1000;
            justify-content: center;
            align-items: center;
        }
        .fedi-modal {
            background: white;
            border-radius: 12px;
            padding: 1.5rem;
            max-width: 400px;
            width: 90%;
            box-shadow: 0 20px 40px rgba(0,0,0,0.3);
        }
        .fedi-modal h3 {
            margin: 0 0 0.5rem 0;
            font-size: 1.1rem;
        }
        .fedi-modal p {
            color: #666;
            font-size: 0.9rem;
            margin: 0 0 1rem 0;
        }
        .fedi-modal input {
            width: 100%;
            padding: 0.6rem;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 0.9rem;
            margin-bottom: 1rem;
        }
        .fedi-modal-buttons {
            display: flex;
            gap: 0.5rem;
            justify-content: flex-end;
        }
        .fedi-modal-buttons button {
            padding: 0.5rem 1rem;
            border-radius: 6px;
            font-size: 0.9rem;
            cursor: pointer;
        }
        .fedi-modal-cancel {
            background: #f0f0f0;
            border: none;
            color: #666;
        }
        .fedi-modal-go {
            background: linear-gradient(135deg, #6364ff 0%, #563acc 100%);
            border: none;
            color: white;
        }
    </style>`
				htmlContent = strings.Replace(htmlContent, "</style>", followCSS, 1)

				// Replace section-header with version including follow button
				oldHeader := `<div class="section-header">
            <h2>Recent Posts</h2>`
				newHeader := fmt.Sprintf(`<div class="section-header">
            <div class="section-header-title">
                <h2>Recent Posts</h2>
                <button class="fedi-follow-btn" onclick="openFollowModal()">🐘 Follow</button>
            </div>`)
				htmlContent = strings.Replace(htmlContent, oldHeader, newHeader, 1)

				// Add modal HTML before closing body tag
				modalHTML := fmt.Sprintf(`
    <div id="fedi-modal-overlay" class="fedi-modal-overlay" onclick="closeFollowModal(event)">
        <div class="fedi-modal" onclick="event.stopPropagation()">
            <h3>🐘 Follow on Mastodon</h3>
            <p>Enter your Mastodon instance to follow <strong>%s</strong></p>
            <input type="text" id="fedi-instance" placeholder="mastodon.social" />
            <div class="fedi-modal-buttons">
                <button class="fedi-modal-cancel" onclick="closeFollowModal()">Cancel</button>
                <button class="fedi-modal-go" onclick="goToFollow()">Follow</button>
            </div>
        </div>
    </div>
    <script>
        const fediHandle = '%s';
        function openFollowModal() {
            const saved = localStorage.getItem('fedi-instance');
            if (saved) document.getElementById('fedi-instance').value = saved;
            document.getElementById('fedi-modal-overlay').style.display = 'flex';
        }
        function closeFollowModal(e) {
            if (!e || e.target.id === 'fedi-modal-overlay') {
                document.getElementById('fedi-modal-overlay').style.display = 'none';
            }
        }
        function goToFollow() {
            const instance = document.getElementById('fedi-instance').value.trim();
            if (!instance) return;
            localStorage.setItem('fedi-instance', instance);
            const url = 'https://' + instance.replace(/^https?:\/\//, '') + '/authorize_interaction?uri=' + encodeURIComponent(fediHandle);
            window.open(url, '_blank');
            closeFollowModal();
        }
        document.getElementById('fedi-instance').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') goToFollow();
        });
    </script>
</body>`, html.EscapeString(fediHandle), html.EscapeString(fediHandle))
				htmlContent = strings.Replace(htmlContent, "</body>", modalHTML, 1)
			}
		}

		// Get the collection index JSON-LD
		indexData, err := s.docServer.GetIndexJSONLD()
		if err == nil && len(indexData) > 0 {
			// Create the script tag with proper indentation
			jsonldScript := fmt.Sprintf(`    <script type="application/ld+json">
%s
    </script>
`, string(indexData))

			// Insert the script tag in the <head> section, before </head>
			htmlContent = strings.Replace(htmlContent, "</head>", jsonldScript+"</head>", 1)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(htmlContent))
}

// handleActivityPub handles ActivityPub requests to /users/{username}/*
func (s *Server) handleActivityPub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}

	username := parts[0]
	endpoint := ""
	if len(parts) > 1 {
		endpoint = parts[1]
	}

	switch endpoint {
	case "":
		// Actor profile
		s.actor.HandleActor(w, r, username)
	case "inbox":
		// Inbox
		s.actor.HandleInbox(w, r, username)
	case "outbox":
		// Outbox - get blog posts
		posts := s.GetBlogPosts()
		s.actor.HandleOutbox(w, r, username, posts)
	case "followers":
		// Followers collection
		s.actor.HandleFollowers(w, r, username)
	case "following":
		// Following collection
		s.actor.HandleFollowing(w, r, username)
	default:
		http.NotFound(w, r)
	}
}

// handleActivityPubLegacy handles legacy write.as/writefreely ActivityPub paths
// /api/collections/{username}/inbox -> /users/{username}/inbox
func (s *Server) handleActivityPubLegacy(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}

	username := parts[0]
	endpoint := ""
	if len(parts) > 1 {
		endpoint = parts[1]
	}

	switch endpoint {
	case "":
		// Actor profile - redirect to canonical
		s.actor.HandleActorLegacy(w, r, username)
	case "inbox":
		// Inbox - handle directly (don't redirect POSTs)
		s.actor.HandleInboxLegacy(w, r, username)
	default:
		http.NotFound(w, r)
	}
}

// GetBlogPosts loads blog posts and converts them to ActivityPub format
func (s *Server) GetBlogPosts() []activitypub.BlogPost {
	if s.contentDir == "" {
		return nil
	}

	docs, err := markdown.ListDocuments(s.contentDir)
	if err != nil {
		return nil
	}

	// Sort by date (newest first)
	markdown.SortDocumentsByDate(docs)

	var posts []activitypub.BlogPost
	baseURL := s.fallbackURL

	for _, doc := range docs {
		if doc.Frontmatter.Draft {
			continue
		}

		published, _ := time.Parse(time.RFC3339, doc.Frontmatter.DatePublished)
		var updated time.Time
		if doc.Frontmatter.DateModified != "" {
			updated, _ = time.Parse(time.RFC3339, doc.Frontmatter.DateModified)
		}

		postURL := fmt.Sprintf("%s/posts/%s", baseURL, doc.Frontmatter.Slug)

		// Resolve image URL (convert relative to absolute)
		imageURL := doc.Frontmatter.Image
		if imageURL != "" && !strings.HasPrefix(imageURL, "http") {
			imageURL = strings.TrimPrefix(imageURL, "/")
			imageURL = fmt.Sprintf("%s/%s", baseURL, imageURL)
		}

		posts = append(posts, activitypub.BlogPost{
			ID:          postURL,
			Slug:        doc.Frontmatter.Slug,
			Title:       doc.Frontmatter.Title,
			Description: doc.Frontmatter.Description,
			Content:     doc.HTML,
			Published:   published,
			Updated:     updated,
			Tags:        doc.Frontmatter.Tags,
			Image:       imageURL,
		})
	}

	return posts
}

// handlePublish handles the publish endpoint to push new posts to followers
// POST /publish?token=<secret>
func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for publish token
	publishToken := os.Getenv("ACTIVITYPUB_PUBLISH_TOKEN")
	if publishToken == "" {
		http.Error(w, "Publishing not configured", http.StatusServiceUnavailable)
		return
	}

	token := r.URL.Query().Get("token")
	if token != publishToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if s.actor == nil {
		http.Error(w, "ActivityPub not enabled", http.StatusServiceUnavailable)
		return
	}

	// Get all blog posts
	posts := s.GetBlogPosts()
	if len(posts) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "No posts to publish",
			"published": 0,
		})
		return
	}

	// Check for specific post slug
	slug := r.URL.Query().Get("slug")
	if slug != "" {
		// Publish specific post
		for _, post := range posts {
			if post.Slug == slug {
				results, err := s.actor.PublishPost(post)
				if err != nil {
					http.Error(w, fmt.Sprintf("Failed to publish: %v", err), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": fmt.Sprintf("Published: %s", post.Title),
					"post":    post.Slug,
					"results": results,
				})
				return
			}
		}
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Publish all new posts
	count, err := s.actor.PublishNewPosts(posts)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to publish: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Publish complete",
		"published": count,
		"total":     len(posts),
	})
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// robots.txt
	if r.URL.Path == "/robots.txt" {
		s.handleRobotsTxt(w, r)
		return
	}

	// favicon
	if r.URL.Path == "/favicon.ico" || r.URL.Path == "/favicon.png" || r.URL.Path == "/favicon.svg" {
		if s.docServer != nil {
			s.docServer.HandleFavicon(w, r)
			return
		}
	}

	// content assets (images)
	if s.docServer != nil {
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".svg" || ext == ".webp" {
			filename := strings.TrimPrefix(r.URL.Path, "/")
			s.docServer.HandleContentAsset(w, r, filename)
			return
		}
	}

	// sitemap.xml
	if r.URL.Path == "/sitemap.xml" {
		if s.docServer != nil {
			s.docServer.HandleSitemap(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}

	// .well-known directory
	if strings.HasPrefix(r.URL.Path, "/.well-known/") {
		s.handleWellKnown(w, r)
		return
	}

	// NodeInfo 2.0 endpoint
	if r.URL.Path == "/nodeinfo/2.0" && s.actor != nil {
		postCount := 0
		if s.docServer != nil {
			if docs, err := markdown.ListDocuments(s.contentDir); err == nil {
				for _, doc := range docs {
					if !doc.Frontmatter.Draft {
						postCount++
					}
				}
			}
		}
		s.actor.HandleNodeInfo(w, r, postCount)
		return
	}

	// ActivityPub publish endpoint
	if r.URL.Path == "/publish" && s.actor != nil {
		s.handlePublish(w, r)
		return
	}

	// ActivityPub actor routes
	if strings.HasPrefix(r.URL.Path, "/users/") && s.actor != nil {
		s.handleActivityPub(w, r)
		return
	}

	// Legacy write.as/writefreely ActivityPub routes
	if strings.HasPrefix(r.URL.Path, "/api/collections/") && s.actor != nil {
		s.handleActivityPubLegacy(w, r)
		return
	}

	// Blog post routes (only if docServer is configured)
	if s.docServer != nil {
		if r.URL.Path == "/posts" {
			s.docServer.HandleDocList(w, r)
			return
		}
		// Search page
		if r.URL.Path == "/search" {
			s.docServer.HandleSearch(w, r)
			return
		}
		// RSS feed - support multiple standard URLs
		if r.URL.Path == "/posts.rss" || r.URL.Path == "/feed.xml" || r.URL.Path == "/rss.xml" {
			s.docServer.HandleSiteRSS(w, r)
			return
		}
		if r.URL.Path == "/posts/index.jsonld" {
			s.docServer.HandleIndexJSONLD(w, r)
			return
		}
		if r.URL.Path == "/rss" {
			s.docServer.HandleRSSList(w, r)
			return
		}
		if r.URL.Path == "/tags" {
			s.docServer.HandleTagsPage(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/tags/") {
			tag := strings.TrimPrefix(r.URL.Path, "/tags/")
			if tag != "" {
				// Decode the URL-encoded tag
				decodedTag, err := url.PathUnescape(tag)
				if err != nil {
					// If decoding fails, use the original tag
					decodedTag = tag
				}
				s.docServer.HandleTagPage(w, r, decodedTag)
			} else {
				s.docServer.HandleTagsPage(w, r)
			}
			return
		}
		if strings.HasPrefix(r.URL.Path, "/posts/") {
			slug := strings.TrimPrefix(r.URL.Path, "/posts/")
			// Check for .jsonld extension
			if strings.HasSuffix(slug, ".jsonld") {
				slug = strings.TrimSuffix(slug, ".jsonld")
				s.docServer.HandleDocJSONLD(w, r, slug)
			} else {
				s.docServer.HandleDoc(w, r, slug)
			}
			return
		}
	}

	// Object routes (read-only)
	if strings.HasPrefix(r.URL.Path, "/o/") {
		// Check for markdown content request
		if strings.HasSuffix(r.URL.Path, "/markdown") {
			s.handleGetMarkdown(w, r)
			return
		}
		s.handleGetObject(w, r)
		return
	}

	// User routes
	if strings.HasPrefix(r.URL.Path, "/u/") {
		// Check for RSS feed request
		if strings.HasSuffix(r.URL.Path, "/posts.rss") && s.docServer != nil {
			// Extract username from /u/{user}/posts.rss
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/u/"), "/")
			if len(parts) >= 1 && parts[0] != "" {
				userName := parts[0]
				s.docServer.HandleUserRSS(w, r, userName)
				return
			}
		}
		if strings.Contains(r.URL.Path, "/latest") {
			s.handleGetLatest(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "/_history") {
			s.handleGetHistory(w, r)
			return
		}
	}

	// Serve static files from embedded filesystem
	if s.publicFS != nil {
		// For root path, serve index.html with embedded JSON-LD
		if r.URL.Path == "/" {
			s.handleIndex(w, r)
			return
		}

		// Serve other static files
		http.FileServer(http.FS(s.publicFS)).ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}
