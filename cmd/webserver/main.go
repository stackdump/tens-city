package main

import (
"context"
"encoding/json"
"flag"
"fmt"
"log"
"net/http"
"os"
"path/filepath"
"strings"

"github.com/jackc/pgx/v5/pgxpool"
"github.com/stackdump/tens-city/internal/seal"
"github.com/stackdump/tens-city/internal/store"
)

// Storage interface abstracts filesystem and database backends
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

// DBStorage implements Storage using PostgreSQL
type DBStorage struct {
pool *pgxpool.Pool
}

func NewDBStorage(dbURL string) (*DBStorage, error) {
cfg, err := pgxpool.ParseConfig(dbURL)
if err != nil {
return nil, fmt.Errorf("parse db url: %w", err)
}
pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
if err != nil {
return nil, fmt.Errorf("connect db: %w", err)
}
return &DBStorage{pool: pool}, nil
}

func (db *DBStorage) Close() {
db.pool.Close()
}

func (db *DBStorage) GetObject(cid string) ([]byte, error) {
var raw json.RawMessage
err := db.pool.QueryRow(context.Background(),
`SELECT raw FROM public.objects WHERE cid = $1`, cid).Scan(&raw)
if err != nil {
return nil, err
}
return []byte(raw), nil
}

func (db *DBStorage) SaveObject(cid string, raw []byte, canonical []byte) error {
// Default owner UUID (zero UUID for anonymous)
ownerUUID := "00000000-0000-0000-0000-000000000000"

_, err := db.pool.Exec(context.Background(),
`INSERT INTO public.objects (cid, owner_uuid, raw, canonical, storage_path) 
 VALUES ($1, $2, $3, $4, $5)
 ON CONFLICT (cid) DO NOTHING`,
cid, ownerUUID, json.RawMessage(raw), string(canonical), nil)
return err
}

func (db *DBStorage) GetLatest(user, slug string) (string, error) {
// For DB storage, we could store this in a separate table
// For now, return empty as this is filesystem-specific
return "", fmt.Errorf("not implemented for database storage")
}

func (db *DBStorage) GetHistory(user, slug string) ([]store.HistoryEntry, error) {
return nil, fmt.Errorf("not implemented for database storage")
}

func (db *DBStorage) UpdateLatest(user, slug, cid string) error {
return fmt.Errorf("not implemented for database storage")
}

func (db *DBStorage) AppendHistory(user, slug, cid string) error {
return fmt.Errorf("not implemented for database storage")
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
if os.IsNotExist(err) || strings.Contains(err.Error(), "no rows") {
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

// Serialize back to JSON
raw, err := json.Marshal(doc)
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
dbURL := flag.String("db", "", "PostgreSQL database URL (if not set, uses filesystem)")
storeDir := flag.String("store", "data", "Filesystem store directory (used if -db not set)")
publicDir := flag.String("public", "public", "Public directory for static files")
enableCORS := flag.Bool("cors", true, "Enable CORS headers")
flag.Parse()

var storage Storage

if *dbURL != "" {
log.Printf("Using PostgreSQL storage: %s", *dbURL)
dbStorage, err := NewDBStorage(*dbURL)
if err != nil {
log.Fatalf("Failed to initialize database storage: %v", err)
}
defer dbStorage.Close()
storage = dbStorage
} else {
log.Printf("Using filesystem storage: %s", *storeDir)
storage = NewFSStorage(*storeDir)
}

server := NewServer(storage, *publicDir, *enableCORS)

log.Printf("Starting server on %s", *addr)
log.Printf("Public directory: %s", *publicDir)
if err := http.ListenAndServe(*addr, server); err != nil {
log.Fatalf("Server failed: %v", err)
}
}
