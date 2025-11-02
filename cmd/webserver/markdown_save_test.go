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
}
