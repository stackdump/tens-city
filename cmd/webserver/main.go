package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/stackdump/tens-city/internal/docserver"
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
	storage   Storage
	publicFS  fs.FS
	docServer *docserver.DocServer
	baseURL   string // Base URL for the server
}

func NewServer(storage Storage, publicFS fs.FS, docServer *docserver.DocServer) *Server {
	return &Server{
		storage:   storage,
		publicFS:  publicFS,
		docServer: docServer,
		baseURL:   "", // Will be set from command line flag
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

	// If docServer is available, inject JSON-LD script tag and RSS link
	if s.docServer != nil {
		// Add RSS autodiscovery link
		rssLink := fmt.Sprintf(`    <link rel="alternate" type="application/rss+xml" title="All Posts - Tens City" href="%s/posts.rss">
`, s.baseURL)
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
	log.Printf("%s %s", r.Method, r.URL.Path)

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
	flag.Parse()

	// Check for INDEX_LIMIT environment variable (overrides flag default)
	if envLimit := os.Getenv("INDEX_LIMIT"); envLimit != "" {
		if limit, err := strconv.Atoi(envLimit); err == nil {
			indexLimit = &limit
		} else {
			log.Printf("Warning: Invalid INDEX_LIMIT environment variable '%s': %v. Using default or flag value.", envLimit, err)
		}
	}

	log.Printf("Using filesystem storage: %s", *storeDir)
	log.Printf("Content directory: %s", *contentDir)
	log.Printf("Base URL: %s", *baseURL)
	log.Printf("Index limit: %d", *indexLimit)
	storage := NewFSStorage(*storeDir)

	// Get the embedded public filesystem
	publicSubFS, err := static.Public()
	if err != nil {
		log.Fatalf("Failed to access embedded public files: %v", err)
	}

	// Create document server
	docServer := docserver.NewDocServer(*contentDir, *baseURL, *indexLimit)

	server := NewServer(storage, publicSubFS, docServer)
	server.baseURL = *baseURL // Set the base URL from command line flag

	log.Printf("Starting server on %s", *addr)
	log.Println("Using embedded public files")
	if err := http.ListenAndServe(*addr, server); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
