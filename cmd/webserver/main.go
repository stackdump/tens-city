package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackdump/tens-city/internal/auth"
	"github.com/stackdump/tens-city/internal/canonical"
	"github.com/stackdump/tens-city/internal/seal"
	"github.com/stackdump/tens-city/internal/store"
)

// Storage interface abstracts filesystem backends
type Storage interface {
	GetObject(cid string) ([]byte, error)
	SaveObject(cid string, raw []byte, canonical []byte) error
	SaveObjectWithAuthor(cid string, raw []byte, canonical []byte, githubUser, githubID string) error
	GetLatest(user, slug string) (string, error)
	GetHistory(user, slug string) ([]store.HistoryEntry, error)
	UpdateLatest(user, slug, cid string) error
	AppendHistory(user, slug, cid string) error
	DeleteObject(cid string) error
	GetObjectAuthor(cid string) (githubUser, githubID string, err error)
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

func (fs *FSStorage) SaveObject(cid string, raw []byte, canonical []byte) error {
	return fs.store.SaveObject(cid, raw, canonical)
}

func (fs *FSStorage) SaveObjectWithAuthor(cid string, raw []byte, canonical []byte, githubUser, githubID string) error {
	return fs.store.SaveObjectWithAuthor(cid, raw, canonical, githubUser, githubID)
}

func (fs *FSStorage) GetLatest(user, slug string) (string, error) {
return fs.store.ReadLatest(user, slug)
}

func (fs *FSStorage) GetHistory(user, slug string) ([]store.HistoryEntry, error) {
return fs.store.ReadHistory(user, slug)
}

func (fs *FSStorage) UpdateLatest(user, slug, cid string) error {
return fs.store.UpdateLatest(user, slug, cid)
}

func (fs *FSStorage) AppendHistory(user, slug, cid string) error {
return fs.store.AppendHistory(user, slug, cid)
}

func (fs *FSStorage) DeleteObject(cid string) error {
	return fs.store.DeleteObject(cid)
}

func (fs *FSStorage) GetObjectAuthor(cid string) (string, string, error) {
	return fs.store.GetObjectAuthor(cid)
}

// validateJSONLD validates the structure and content of a JSON-LD document
func validateJSONLD(doc map[string]interface{}) error {
	// Check for required @context field
	context, hasContext := doc["@context"]
	if !hasContext {
		return fmt.Errorf("missing @context field")
	}

	// Validate @context is a valid type (string, object, or array)
	switch context.(type) {
	case string, map[string]interface{}, []interface{}:
		// Valid types
	default:
		return fmt.Errorf("@context must be a string, object, or array")
	}

	// Validate there are no excessively deep nested structures (prevent DoS)
	if err := validateDepth(doc, 0, 50); err != nil {
		return err
	}

	// Validate keys don't contain control characters or other dangerous content
	if err := validateKeys(doc); err != nil {
		return err
	}

	return nil
}

// validateDepth ensures the JSON structure doesn't exceed a maximum depth
func validateDepth(v interface{}, currentDepth, maxDepth int) error {
	if currentDepth > maxDepth {
		return fmt.Errorf("document exceeds maximum nesting depth of %d", maxDepth)
	}

	switch val := v.(type) {
	case map[string]interface{}:
		for _, value := range val {
			if err := validateDepth(value, currentDepth+1, maxDepth); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range val {
			if err := validateDepth(item, currentDepth+1, maxDepth); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateKeys ensures all keys in the JSON structure are safe
func validateKeys(doc map[string]interface{}) error {
	for key, value := range doc {
		// Check for control characters in keys
		for _, r := range key {
			if r < 32 && r != '\t' && r != '\n' && r != '\r' {
				return fmt.Errorf("key contains control characters: %q", key)
			}
		}

		// Recursively validate nested objects
		if nestedDoc, ok := value.(map[string]interface{}); ok {
			if err := validateKeys(nestedDoc); err != nil {
				return err
			}
		}

		// Check arrays of objects
		if arr, ok := value.([]interface{}); ok {
			for _, item := range arr {
				if nestedDoc, ok := item.(map[string]interface{}); ok {
					if err := validateKeys(nestedDoc); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}



type Server struct {
storage        Storage
publicDir      string
enableCORS     bool
maxContentSize int64 // Maximum content size in bytes
}

func NewServer(storage Storage, publicDir string, enableCORS bool, maxContentSize int64) *Server {
return &Server{
storage:        storage,
publicDir:      publicDir,
enableCORS:     enableCORS,
maxContentSize: maxContentSize,
}
}

func (s *Server) handleCORS(w http.ResponseWriter, r *http.Request) bool {
if s.enableCORS {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

if r.Method == "OPTIONS" {
w.WriteHeader(http.StatusOK)
return true
}
}
return false
}

// Handler for /o/{cid} - get object by CID
func (s *Server) handleGetObject(w http.ResponseWriter, r *http.Request) {
if s.handleCORS(w, r) {
return
}

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

// Handler for /u/{user}/g/{slug}/latest - get latest CID
func (s *Server) handleGetLatest(w http.ResponseWriter, r *http.Request) {
if s.handleCORS(w, r) {
return
}

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
if s.handleCORS(w, r) {
return
}

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

// Handler for GET /api/ownership/{cid} - check if current user owns the object
func (s *Server) handleCheckOwnership(w http.ResponseWriter, r *http.Request) {
	if s.handleCORS(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract CID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/ownership/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "CID required", http.StatusBadRequest)
		return
	}
	cid := parts[0]

	// Extract and validate authentication token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Return not owned if not authenticated
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"owned": false})
		return
	}

	userInfo, err := auth.ExtractUserFromToken(authHeader)
	if err != nil {
		// Return not owned if authentication fails
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"owned": false})
		return
	}

	// Get the object author
	authorUser, authorID, err := s.storage.GetObjectAuthor(cid)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Object not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting object author for %s: %v", cid, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if the user is the author
	// Priority: GitHub ID (most secure) > username (for backward compatibility)
	isOwned := false
	if authorID != "" && userInfo.GitHubID != "" && authorID == userInfo.GitHubID {
		isOwned = true
	} else if authorUser != "" && userInfo.UserName != "" && authorUser == userInfo.UserName {
		isOwned = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"owned": isOwned})
}

// Handler for POST /api/save - save JSON-LD and return CID
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if s.handleCORS(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract and validate authentication token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userInfo, err := auth.ExtractUserFromToken(authHeader)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
		return
	}

	// Limit request body size to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, s.maxContentSize)
	
	var doc map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate JSON-LD structure
	if err := validateJSONLD(doc); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON-LD: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Serialize back to JSON using canonical encoding (sorted keys)
	raw, err := canonical.MarshalJSON(doc)
	if err != nil {
		http.Error(w, "Failed to serialize JSON", http.StatusInternalServerError)
		return
	}

	// Compute CID
	cid, canonicalData, err := seal.SealJSONLD(raw)
	if err != nil {
		log.Printf("Error sealing JSON-LD: %v", err)
		http.Error(w, "Failed to compute CID", http.StatusInternalServerError)
		return
	}

	// Save to storage with author information
	githubUser := userInfo.UserName
	githubID := userInfo.GitHubID
	
	// Don't use email as username fallback since it would create invalid GitHub URLs
	// The storage layer will handle empty username appropriately

	if err := s.storage.SaveObjectWithAuthor(cid, raw, canonicalData, githubUser, githubID); err != nil {
		log.Printf("Error saving object: %v", err)
		http.Error(w, "Failed to save object", http.StatusInternalServerError)
		return
	}

	// Return CID
	response := map[string]string{
		"cid": cid,
	}
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

// Handler for DELETE /o/{cid} - delete object by CID (author only)
func (s *Server) handleDeleteObject(w http.ResponseWriter, r *http.Request) {
	if s.handleCORS(w, r) {
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract and validate authentication token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userInfo, err := auth.ExtractUserFromToken(authHeader)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
		return
	}

	// Extract CID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/o/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "CID required", http.StatusBadRequest)
		return
	}
	cid := parts[0]

	// Verify the object exists and get its author
	authorUser, authorID, err := s.storage.GetObjectAuthor(cid)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Object not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting object author for %s: %v", cid, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify the requesting user is the author
	// Priority: GitHub ID (most secure) > username (for backward compatibility)
	// Note: Username fallback is less secure as usernames can be changed or reused,
	// but is needed for objects saved before GitHub IDs were tracked.
	// GitHub ID verification is strongly preferred when available.
	isAuthor := false
	if authorID != "" && userInfo.GitHubID != "" && authorID == userInfo.GitHubID {
		isAuthor = true
	} else if authorUser != "" && userInfo.UserName != "" && authorUser == userInfo.UserName {
		// Username fallback for backward compatibility
		isAuthor = true
	}

	if !isAuthor {
		// Note: Logging usernames and GitHub IDs here is safe as they are public information
		// from GitHub profiles, not sensitive authentication data
		log.Printf("Delete denied: user %s (ID: %s) tried to delete object authored by %s (ID: %s)",
			userInfo.UserName, userInfo.GitHubID, authorUser, authorID)
		http.Error(w, "Forbidden: only the author can delete this object", http.StatusForbidden)
		return
	}

	// Delete the object
	if err := s.storage.DeleteObject(cid); err != nil {
		log.Printf("Error deleting object %s: %v", cid, err)
		http.Error(w, "Failed to delete object", http.StatusInternalServerError)
		return
	}

	// Note: Logging usernames and GitHub IDs here is safe as they are public information
	// from GitHub profiles, not sensitive authentication data
	log.Printf("Object %s deleted by author %s (ID: %s)", cid, userInfo.UserName, userInfo.GitHubID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
log.Printf("%s %s", r.Method, r.URL.Path)

// API routes
if r.URL.Path == "/api/save" {
s.handleSave(w, r)
return
}
if strings.HasPrefix(r.URL.Path, "/api/ownership/") {
s.handleCheckOwnership(w, r)
return
}

// Object routes
if strings.HasPrefix(r.URL.Path, "/o/") {
	if r.Method == http.MethodDelete {
		s.handleDeleteObject(w, r)
	} else {
		s.handleGetObject(w, r)
	}
	return
}

// User routes
if strings.HasPrefix(r.URL.Path, "/u/") {
if strings.Contains(r.URL.Path, "/latest") {
s.handleGetLatest(w, r)
return
}
if strings.Contains(r.URL.Path, "/_history") {
s.handleGetHistory(w, r)
return
}
}

// Serve static files from public directory
if s.publicDir != "" {
// For root path, serve index.html
if r.URL.Path == "/" {
http.ServeFile(w, r, filepath.Join(s.publicDir, "index.html"))
return
}

// Serve other static files
http.FileServer(http.Dir(s.publicDir)).ServeHTTP(w, r)
return
}

http.NotFound(w, r)
}

func main() {
addr := flag.String("addr", ":8080", "Server address")
storeDir := flag.String("store", "data", "Filesystem store directory")
publicDir := flag.String("public", "public", "Public directory for static files")
enableCORS := flag.Bool("cors", true, "Enable CORS headers")
maxContentMB := flag.Int("max-content-mb", 1, "Maximum content size in megabytes (default: 1MB)")
flag.Parse()

// Convert MB to bytes
maxContentSize := int64(*maxContentMB) * 1024 * 1024

log.Printf("Using filesystem storage: %s", *storeDir)
log.Printf("Maximum content size: %d MB (%d bytes)", *maxContentMB, maxContentSize)
storage := NewFSStorage(*storeDir)

server := NewServer(storage, *publicDir, *enableCORS, maxContentSize)

log.Printf("Starting server on %s", *addr)
log.Printf("Public directory: %s", *publicDir)
if err := http.ListenAndServe(*addr, server); err != nil {
log.Fatalf("Server failed: %v", err)
}
}
