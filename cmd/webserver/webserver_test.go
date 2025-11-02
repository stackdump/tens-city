package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stackdump/tens-city/internal/auth"
	"github.com/stackdump/tens-city/internal/static"
	"github.com/stackdump/tens-city/internal/store"
)

// createTestToken creates a properly signed JWT token for testing
func createTestToken(userID, email, username, githubID string) string {
	// Use a test secret and set it in environment
	testSecret := "test-secret-key-for-testing"
	os.Setenv("SUPABASE_JWT_SECRET", testSecret)
	
	// Create claims
	claims := &auth.SupabaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email: email,
		UserMetadata: map[string]interface{}{
			"user_name":   username,
			"provider_id": githubID,
		},
	}
	
	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))
	
	return tokenString
}

// decodeJSON is a helper to decode JSON from an io.Reader
func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}


func TestFSStorageGetObject(t *testing.T) {
tmpDir := t.TempDir()
storage := NewFSStorage(tmpDir)

// Create test data
cid := "test-cid-123"
testData := []byte(`{"@context":"test","name":"test object"}`)
canonical := []byte(`canonical data`)

// Save object
if err := storage.SaveObject(cid, testData, canonical); err != nil {
t.Fatalf("Failed to save object: %v", err)
}

// Retrieve object
data, err := storage.GetObject(cid)
if err != nil {
t.Fatalf("Failed to get object: %v", err)
}

// Verify it contains the injected @id
var obj map[string]interface{}
if err := json.Unmarshal(data, &obj); err != nil {
t.Fatalf("Failed to unmarshal object: %v", err)
}

if _, ok := obj["@id"]; !ok {
t.Error("Object missing @id field")
}
}

func TestHandleGetObject(t *testing.T) {
tmpDir := t.TempDir()
storage := NewFSStorage(tmpDir)

// Create test object
cid := "test-cid-456"
testData := []byte(`{"@context":"test","name":"test"}`)
canonical := []byte(`canonical`)
if err := storage.SaveObject(cid, testData, canonical); err != nil {
t.Fatalf("Failed to save object: %v", err)
}

server := NewServer(storage, nil, false, 1*1024*1024, nil)

// Test successful retrieval
req := httptest.NewRequest("GET", "/o/"+cid, nil)
w := httptest.NewRecorder()
server.ServeHTTP(w, req)

resp := w.Result()
if resp.StatusCode != http.StatusOK {
t.Errorf("Expected status 200, got %d", resp.StatusCode)
}

contentType := resp.Header.Get("Content-Type")
if contentType != "application/ld+json" {
t.Errorf("Expected Content-Type application/ld+json, got %s", contentType)
}

// Test not found
req = httptest.NewRequest("GET", "/o/nonexistent", nil)
w = httptest.NewRecorder()
server.ServeHTTP(w, req)

resp = w.Result()
if resp.StatusCode != http.StatusNotFound {
t.Errorf("Expected status 404, got %d", resp.StatusCode)
}
}

func TestHandleSave(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1*1024*1024, nil)

	// Create test auth token
	authToken := createTestToken("test-user-123", "test@example.com", "testuser", "123456")

	// Test valid JSON-LD
	doc := map[string]interface{}{
		"@context": map[string]string{
			"name": "http://schema.org/name",
		},
		"@type": "Test",
		"name":  "Test Object",
	}
	body, _ := json.Marshal(doc)

	req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	cid, ok := result["cid"]
	if !ok || cid == "" {
		t.Error("Response missing CID")
	}

	// Verify object was saved
	savedData, err := storage.GetObject(cid)
	if err != nil {
		t.Errorf("Object not saved: %v", err)
	}

	// Verify author information was injected
	var savedDoc map[string]interface{}
	if err := json.Unmarshal(savedData, &savedDoc); err != nil {
		t.Fatalf("Failed to unmarshal saved object: %v", err)
	}

	author, ok := savedDoc["author"].(map[string]interface{})
	if !ok {
		t.Error("Author information not found in saved object")
	} else {
		if author["name"] != "testuser" {
			t.Errorf("Expected author name 'testuser', got '%v'", author["name"])
		}
		if author["id"] != "github:123456" {
			t.Errorf("Expected author id 'github:123456', got '%v'", author["id"])
		}
	}

	// Test invalid JSON-LD (missing @context)
	invalidDoc := map[string]interface{}{
		"name": "Invalid",
	}
	body, _ = json.Marshal(invalidDoc)

	req = httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	// Test missing authentication
	validDoc := map[string]interface{}{
		"@context": map[string]string{
			"name": "http://schema.org/name",
		},
		"name": "Test",
	}
	body, _ = json.Marshal(validDoc)

	req = httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No auth header
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing auth, got %d", resp.StatusCode)
	}
}

func TestHandleGetLatest(t *testing.T) {
tmpDir := t.TempDir()
storage := NewFSStorage(tmpDir)

// Create user gist structure
user := "testuser"
slug := "testslug"
cid := "test-cid-789"

if err := storage.UpdateLatest(user, slug, cid); err != nil {
t.Fatalf("Failed to update latest: %v", err)
}

server := NewServer(storage, nil, false, 1*1024*1024, nil)

req := httptest.NewRequest("GET", "/u/"+user+"/g/"+slug+"/latest", nil)
w := httptest.NewRecorder()
server.ServeHTTP(w, req)

resp := w.Result()
if resp.StatusCode != http.StatusOK {
t.Errorf("Expected status 200, got %d", resp.StatusCode)
}

body, _ := io.ReadAll(resp.Body)
if string(body) != cid {
t.Errorf("Expected CID %s, got %s", cid, string(body))
}
}

func TestHandleGetHistory(t *testing.T) {
tmpDir := t.TempDir()
storage := NewFSStorage(tmpDir)

user := "testuser"
slug := "testslug"
cid1 := "cid-001"
cid2 := "cid-002"

// Add history entries
if err := storage.AppendHistory(user, slug, cid1); err != nil {
t.Fatalf("Failed to append history: %v", err)
}
if err := storage.AppendHistory(user, slug, cid2); err != nil {
t.Fatalf("Failed to append history: %v", err)
}

server := NewServer(storage, nil, false, 1*1024*1024, nil)

req := httptest.NewRequest("GET", "/u/"+user+"/g/"+slug+"/_history", nil)
w := httptest.NewRecorder()
server.ServeHTTP(w, req)

resp := w.Result()
if resp.StatusCode != http.StatusOK {
t.Errorf("Expected status 200, got %d", resp.StatusCode)
}

var history []store.HistoryEntry
if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
t.Fatalf("Failed to decode history: %v", err)
}

if len(history) != 2 {
t.Errorf("Expected 2 history entries, got %d", len(history))
}
}

func TestStaticFileServing(t *testing.T) {
tmpDir := t.TempDir()

storage := NewFSStorage(tmpDir)

// Get the embedded public filesystem
publicFS, err := static.Public()
if err != nil {
t.Fatalf("Failed to get public filesystem: %v", err)
}

server := NewServer(storage, publicFS, false, 1*1024*1024, nil)

// Test serving index.html at root
req := httptest.NewRequest("GET", "/", nil)
w := httptest.NewRecorder()
server.ServeHTTP(w, req)

resp := w.Result()
if resp.StatusCode != http.StatusOK {
t.Errorf("Expected status 200, got %d", resp.StatusCode)
}

body, _ := io.ReadAll(resp.Body)
// Just check that we got some content (the actual embedded index.html)
if len(body) == 0 {
t.Error("Expected index.html content")
}

// Test serving other static file (tens-city.js exists in embedded files)
req = httptest.NewRequest("GET", "/tens-city.js", nil)
w = httptest.NewRecorder()
server.ServeHTTP(w, req)

resp = w.Result()
if resp.StatusCode != http.StatusOK {
t.Errorf("Expected status 200, got %d", resp.StatusCode)
}

body, _ = io.ReadAll(resp.Body)
if len(body) == 0 {
t.Error("Expected tens-city.js content")
}
}

func TestCORSHeaders(t *testing.T) {
tmpDir := t.TempDir()
storage := NewFSStorage(tmpDir)
server := NewServer(storage, nil, true, 1*1024*1024, nil) // CORS enabled

// Test OPTIONS request
req := httptest.NewRequest("OPTIONS", "/api/save", nil)
w := httptest.NewRecorder()
server.ServeHTTP(w, req)

resp := w.Result()
if resp.StatusCode != http.StatusOK {
t.Errorf("Expected status 200, got %d", resp.StatusCode)
}

origin := resp.Header.Get("Access-Control-Allow-Origin")
if origin != "*" {
t.Errorf("Expected CORS origin *, got %s", origin)
}

methods := resp.Header.Get("Access-Control-Allow-Methods")
if methods == "" {
t.Error("Expected CORS methods header")
}
}
