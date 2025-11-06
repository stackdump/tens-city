package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/stackdump/tens-city/internal/docserver"
	"github.com/stackdump/tens-city/internal/httputil"
	"github.com/stackdump/tens-city/internal/logger"
	"github.com/stackdump/tens-city/internal/static"
	"github.com/stackdump/tens-city/internal/store"
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

type Server struct {
	storage     Storage
	publicFS    fs.FS
	docServer   *docserver.DocServer
	fallbackURL string // Fallback Base URL when headers are not available
}

func NewServer(storage Storage, publicFS fs.FS, docServer *docserver.DocServer, fallbackURL string) *Server {
	return &Server{
		storage:     storage,
		publicFS:    publicFS,
		docServer:   docServer,
		fallbackURL: fallbackURL,
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

		// Replace placeholders in the HTML
		htmlContent = strings.Replace(htmlContent, "Tens City - A Minimal Blog Platform", pageTitle, -1)
		htmlContent = strings.Replace(htmlContent, "Simple, elegant blog platform built on content-addressable storage", pageDescription, -1)
		htmlContent = strings.Replace(htmlContent, "🏕️", pageIcon, 1) // Only replace first occurrence (the emoji)
		htmlContent = strings.Replace(htmlContent, "Tens City</h1>", html.EscapeString(pageTitle)+"</h1>", 1)

		// Replace the message paragraph if we have custom HTML content
		if pageMessage != "" {
			// Replace the default paragraph with the custom message
			htmlContent = strings.Replace(htmlContent,
				"<p>A minimal blog platform built on simplicity and content ownership</p>",
				pageMessage,
				1)
		}

		// Add RSS autodiscovery link
		rssLink := fmt.Sprintf(`    <link rel="alternate" type="application/rss+xml" title="All Posts - %s" href="%s/posts.rss">
`, html.EscapeString(pageTitle), baseURL)
		htmlContent = strings.Replace(htmlContent, "</head>", rssLink+"</head>", 1)

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
	w.Write([]byte(htmlContent))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Blog post routes (only if docServer is configured)
	if s.docServer != nil {
		if r.URL.Path == "/posts" {
			s.docServer.HandleDocList(w, r)
			return
		}
		if r.URL.Path == "/posts.rss" {
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

func main() {
	addr := flag.String("addr", ":8080", "Server address")
	storeDir := flag.String("store", "data", "Filesystem store directory")
	contentDir := flag.String("content", "content/posts", "Content directory for markdown blog posts")
	baseURL := flag.String("base-url", "http://localhost:8080", "Base URL for the server")
	indexLimit := flag.Int("index-limit", 20, "Maximum number of posts to show in index (0 = no limit)")
	jsonlLog := flag.Bool("jsonl", false, "Use JSONL format for logging")
	logHeaders := flag.Bool("log-headers", false, "Log incoming request headers (useful for debugging RSS http/https behavior)")
	flag.Parse()

	// Check for INDEX_LIMIT environment variable (overrides flag default)
	if envLimit := os.Getenv("INDEX_LIMIT"); envLimit != "" {
		if limit, err := strconv.Atoi(envLimit); err == nil {
			indexLimit = &limit
		} else {
			log.Printf("Warning: Invalid INDEX_LIMIT environment variable '%s': %v. Using default or flag value.", envLimit, err)
		}
	}

	// Create logger based on format
	var appLogger logger.Logger
	if *jsonlLog {
		appLogger = logger.NewJSONLLogger(os.Stdout)
		appLogger.LogInfo("Using JSONL logging format")
	} else {
		appLogger = logger.NewTextLogger()
	}

	appLogger.LogInfo(fmt.Sprintf("Using filesystem storage: %s", *storeDir))
	appLogger.LogInfo(fmt.Sprintf("Content directory: %s", *contentDir))
	appLogger.LogInfo(fmt.Sprintf("Fallback Base URL: %s", *baseURL))
	appLogger.LogInfo(fmt.Sprintf("Index limit: %d", *indexLimit))
	appLogger.LogInfo(fmt.Sprintf("Header logging: %v", *logHeaders))
	storage := NewFSStorage(*storeDir)

	// Get the embedded public filesystem
	publicSubFS, err := static.Public()
	if err != nil {
		log.Fatalf("Failed to access embedded public files: %v", err)
	}

	// Create document server with fallback URL
	docServer := docserver.NewDocServer(*contentDir, *baseURL, *indexLimit)

	server := NewServer(storage, publicSubFS, docServer, *baseURL)

	// Wrap server with logging middleware
	handler := logger.LoggingMiddleware(appLogger, *logHeaders)(server)

	appLogger.LogInfo(fmt.Sprintf("Starting server on %s", *addr))
	appLogger.LogInfo("Using embedded public files")
	appLogger.LogInfo("Server will detect protocol from proxy headers (X-Forwarded-Proto, X-Forwarded-Scheme, X-Forwarded-Ssl, Forwarded)")
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
