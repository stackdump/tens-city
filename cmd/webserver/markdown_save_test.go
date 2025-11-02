package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHandleSaveMarkdown(t *testing.T) {
	// Create temporary storage
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1024*1024, nil)
	server.baseURL = "http://localhost:8080" // Set base URL for tests

	// Set JWT secret for testing
	os.Setenv("SUPABASE_JWT_SECRET", "test-secret-key-for-testing-only")
	defer os.Unsetenv("SUPABASE_JWT_SECRET")

	t.Run("Requires authentication", func(t *testing.T) {
		req := MarkdownSaveRequest{
			Frontmatter: map[string]interface{}{
				"title": "Test Doc",
			},
			Content: "# Test",
			Slug:    "test-doc",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/docs/save", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.handleSaveMarkdown(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Rejects invalid token", func(t *testing.T) {
		req := MarkdownSaveRequest{
			Frontmatter: map[string]interface{}{
				"title": "Test Doc",
			},
			Content: "# Test",
			Slug:    "test-doc",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/docs/save", bytes.NewReader(body))
		r.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		server.handleSaveMarkdown(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Successfully saves markdown with valid token", func(t *testing.T) {
		// Create a valid JWT token
		claims := &jwt.MapClaims{
			"sub":   "test-user-id",
			"email": "test@example.com",
			"user_metadata": map[string]interface{}{
				"user_name":   "testuser",
				"provider_id": "12345",
			},
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("test-secret-key-for-testing-only"))
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		req := MarkdownSaveRequest{
			Frontmatter: map[string]interface{}{
				"title":         "Test Document",
				"headline":      "Test Document Headline",
				"datePublished": "2025-11-02T00:00:00Z",
				"author": map[string]interface{}{
					"@type": "Person",
					"name":  "Test Author",
				},
				"lang": "en",
			},
			Content: "# Test\n\nThis is a test document.",
			Slug:    "test-doc",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/docs/save", bytes.NewReader(body))
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		r.Header.Set("Content-Type", "application/json")
		r.Host = "localhost:8080"
		w := httptest.NewRecorder()

		server.handleSaveMarkdown(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["cid"] == "" {
			t.Error("Expected CID in response")
		}

		if response["slug"] != "test-doc" {
			t.Errorf("Expected slug 'test-doc', got %s", response["slug"])
		}

		// Verify the object was saved
		_, err = storage.GetObject(response["cid"])
		if err != nil {
			t.Errorf("Failed to retrieve saved object: %v", err)
		}
	})

	t.Run("Requires slug", func(t *testing.T) {
		claims := &jwt.MapClaims{
			"sub":   "test-user-id",
			"email": "test@example.com",
			"user_metadata": map[string]interface{}{
				"user_name": "testuser",
			},
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret-key-for-testing-only"))

		req := MarkdownSaveRequest{
			Frontmatter: map[string]interface{}{
				"title": "Test Doc",
			},
			Content: "# Test",
			Slug:    "", // Missing slug
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/docs/save", bytes.NewReader(body))
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSaveMarkdown(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Rejects invalid JSON", func(t *testing.T) {
		claims := &jwt.MapClaims{
			"sub":   "test-user-id",
			"email": "test@example.com",
			"user_metadata": map[string]interface{}{
				"user_name": "testuser",
			},
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret-key-for-testing-only"))

		r := httptest.NewRequest(http.MethodPost, "/api/docs/save", bytes.NewReader([]byte("invalid json")))
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSaveMarkdown(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Ignores client-provided author and enforces server-side author", func(t *testing.T) {
		// Create a valid JWT token with known user info
		claims := &jwt.MapClaims{
			"sub":   "test-user-id",
			"email": "test@example.com",
			"user_metadata": map[string]interface{}{
				"user_name":   "authenticateduser",
				"provider_id": "67890",
			},
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("test-secret-key-for-testing-only"))
		if err != nil {
			t.Fatalf("Failed to create token: %v", err)
		}

		// Client attempts to set a different author
		req := MarkdownSaveRequest{
			Frontmatter: map[string]interface{}{
				"title":         "Test Document",
				"headline":      "Test Document Headline",
				"datePublished": "2025-11-02T00:00:00Z",
				"author": map[string]interface{}{
					"@type": "Person",
					"name":  "FakeAuthor", // Client trying to set arbitrary author
					"url":   "https://example.com/fake",
					"id":    "github:99999",
				},
				"lang": "en",
			},
			Content: "# Test\n\nThis is a test document.",
			Slug:    "test-author-enforcement",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/docs/save", bytes.NewReader(body))
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		r.Header.Set("Content-Type", "application/json")
		r.Host = "localhost:8080"
		w := httptest.NewRecorder()

		server.handleSaveMarkdown(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["cid"] == "" {
			t.Error("Expected CID in response")
		}

		// Verify the saved object has the authenticated user as author, not the client-provided one
		savedData, err := storage.GetObject(response["cid"])
		if err != nil {
			t.Fatalf("Failed to retrieve saved object: %v", err)
		}

		var savedDoc map[string]interface{}
		if err := json.Unmarshal(savedData, &savedDoc); err != nil {
			t.Fatalf("Failed to unmarshal saved document: %v", err)
		}

		// Verify author field was enforced server-side
		author, ok := savedDoc["author"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected author field in saved document, got: %v", savedDoc["author"])
		}

		// Check that the author name matches the authenticated user
		authorName, ok := author["name"].(string)
		if !ok || authorName != "authenticateduser" {
			t.Errorf("Expected author name to be 'authenticateduser', got %v", authorName)
		}

		// Check that the author identifier is GitHub URL of authenticated user
		// Note: JSON-LD canonicalization may convert 'url' to 'identifier'
		authorIdentifier, hasIdentifier := author["identifier"].(string)
		authorURL, hasURL := author["url"].(string)
		expectedURL := "https://github.com/authenticateduser"

		if hasIdentifier && authorIdentifier != expectedURL {
			t.Errorf("Expected author identifier to be '%s', got %v", expectedURL, authorIdentifier)
		} else if hasURL && authorURL != expectedURL {
			t.Errorf("Expected author URL to be '%s', got %v", expectedURL, authorURL)
		} else if !hasIdentifier && !hasURL {
			t.Error("Expected author to have either 'identifier' or 'url' field")
		}

		// Check that the author ID matches authenticated user's GitHub ID
		authorID, ok := author["id"].(string)
		if !ok || authorID != "github:67890" {
			t.Errorf("Expected author ID to be 'github:67890', got %v", authorID)
		}

		// Verify client-provided author data was NOT used
		if authorName == "FakeAuthor" {
			t.Error("Client-provided author name was used instead of authenticated user")
		}
		if authorURL == "https://example.com/fake" {
			t.Error("Client-provided author URL was used instead of authenticated user")
		}
		if authorID == "github:99999" {
			t.Error("Client-provided author ID was used instead of authenticated user")
		}
	})
}
