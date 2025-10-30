package main

import (
"encoding/json"
"flag"
"log"
"net/http"
"os"
"path/filepath"
"strings"

"github.com/stackdump/tens-city/internal/canonical"
"github.com/stackdump/tens-city/internal/seal"
"github.com/stackdump/tens-city/internal/store"
)

// Storage interface abstracts filesystem backends
type Storage interface {
GetObject(cid string) ([]byte, error)
SaveObject(cid string, raw []byte, canonical []byte) error
GetLatest(user, slug string) (string, error)
GetHistory(user, slug string) ([]store.HistoryEntry, error)
UpdateLatest(user, slug, cid string) error
AppendHistory(user, slug, cid string) error
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



type Server struct {
storage    Storage
publicDir  string
enableCORS bool
}

func NewServer(storage Storage, publicDir string, enableCORS bool) *Server {
return &Server{
storage:    storage,
publicDir:  publicDir,
enableCORS: enableCORS,
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

// Handler for POST /api/save - save JSON-LD and return CID
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
if s.handleCORS(w, r) {
return
}

if r.Method != http.MethodPost {
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
return
}

var doc map[string]interface{}
if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
http.Error(w, "Invalid JSON", http.StatusBadRequest)
return
}

// Validate JSON-LD: must have @context
if _, hasContext := doc["@context"]; !hasContext {
http.Error(w, "Invalid JSON-LD: missing @context", http.StatusBadRequest)
return
}

// Serialize back to JSON using canonical encoding (sorted keys)
raw, err := canonical.MarshalJSON(doc)
if err != nil {
http.Error(w, "Failed to serialize JSON", http.StatusInternalServerError)
return
}

// Compute CID
cid, canonical, err := seal.SealJSONLD(raw)
if err != nil {
log.Printf("Error sealing JSON-LD: %v", err)
http.Error(w, "Failed to compute CID", http.StatusInternalServerError)
return
}

// Save to storage
if err := s.storage.SaveObject(cid, raw, canonical); err != nil {
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
log.Printf("%s %s", r.Method, r.URL.Path)

// API routes
if r.URL.Path == "/api/save" {
s.handleSave(w, r)
return
}

// Object routes
if strings.HasPrefix(r.URL.Path, "/o/") {
s.handleGetObject(w, r)
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
flag.Parse()

log.Printf("Using filesystem storage: %s", *storeDir)
storage := NewFSStorage(*storeDir)

server := NewServer(storage, *publicDir, *enableCORS)

log.Printf("Starting server on %s", *addr)
log.Printf("Public directory: %s", *publicDir)
if err := http.ListenAndServe(*addr, server); err != nil {
log.Fatalf("Server failed: %v", err)
}
}
