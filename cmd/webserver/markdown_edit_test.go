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

// TestMarkdownEditWorkflow tests the full workflow of saving and editing markdown documents
func TestMarkdownEditWorkflow(t *testing.T) {
	// Create temporary storage
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1024*1024, nil)
	server.baseURL = "http://localhost:8080"

	// Set JWT secret for testing
	os.Setenv("SUPABASE_JWT_SECRET", "test-secret-key-for-testing-only")
	defer os.Unsetenv("SUPABASE_JWT_SECRET")

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

	// Step 1: Save a markdown document
	t.Run("Save markdown document", func(t *testing.T) {
		req := MarkdownSaveRequest{
			Frontmatter: map[string]interface{}{
				"title":         "Test Article",
				"headline":      "Test Article Headline",
				"description":   "This is a test article",
				"datePublished": "2025-11-02T00:00:00Z",
				"lang":          "en",
			},
			Content: "# Test Article\n\nThis is the markdown content of the test article.\n\n## Section 1\n\nSome content here.",
			Slug:    "test-article",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/posts/save", bytes.NewReader(body))
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		r.Header.Set("Content-Type", "application/json")
		r.Host = "localhost:8080"
		w := httptest.NewRecorder()

		server.handleSaveMarkdown(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]string
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["cid"] == "" {
			t.Fatal("Expected CID in response")
		}

		savedCID := response["cid"]
		t.Logf("Saved CID: %s", savedCID)

		// Step 2: Retrieve the JSON-LD document
		t.Run("Retrieve JSON-LD", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/o/"+savedCID, nil)
			w := httptest.NewRecorder()

			server.handleGetObject(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var doc map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&doc); err != nil {
				t.Fatalf("Failed to decode JSON-LD: %v", err)
			}

			// Verify JSON-LD contains expected fields
			if doc["@type"] != "Article" {
				t.Errorf("Expected @type to be 'Article', got %v", doc["@type"])
			}

			if doc["headline"] != "Test Article Headline" {
				t.Errorf("Expected headline 'Test Article Headline', got %v", doc["headline"])
			}

			// Verify author was set server-side
			author, ok := doc["author"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected author field")
			}
			if author["name"] != "testuser" {
				t.Errorf("Expected author name 'testuser', got %v", author["name"])
			}
		})

		// Step 3: Retrieve the markdown content
		t.Run("Retrieve markdown content", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/o/"+savedCID+"/markdown", nil)
			w := httptest.NewRecorder()

			server.handleGetMarkdown(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			markdownContent := w.Body.String()

			// Verify markdown content matches what was saved
			expectedContent := "# Test Article\n\nThis is the markdown content of the test article.\n\n## Section 1\n\nSome content here."
			if markdownContent != expectedContent {
				t.Errorf("Markdown content mismatch.\nExpected: %s\nGot: %s", expectedContent, markdownContent)
			}

			t.Logf("Successfully retrieved markdown content (%d bytes)", len(markdownContent))
		})

		// Step 4: Update the markdown document
		t.Run("Update markdown document", func(t *testing.T) {
			updateReq := MarkdownSaveRequest{
				Frontmatter: map[string]interface{}{
					"title":         "Test Article",
					"headline":      "Updated Test Article Headline",
					"description":   "This is an updated test article",
					"datePublished": "2025-11-02T00:00:00Z",
					"dateModified":  "2025-11-03T00:00:00Z",
					"lang":          "en",
				},
				Content: "# Test Article\n\nThis is the UPDATED markdown content.\n\n## Section 1\n\nUpdated content here.\n\n## Section 2\n\nNew section!",
				Slug:    "test-article",
			}
			body, _ := json.Marshal(updateReq)

			r := httptest.NewRequest(http.MethodPost, "/api/posts/save", bytes.NewReader(body))
			r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
			r.Header.Set("Content-Type", "application/json")
			r.Host = "localhost:8080"
			w := httptest.NewRecorder()

			server.handleSaveMarkdown(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			updatedCID := response["cid"]
			t.Logf("Updated CID: %s", updatedCID)

			// CID should be different since content changed
			if updatedCID == savedCID {
				t.Error("Expected different CID after update")
			}

			// Retrieve updated markdown content
			r = httptest.NewRequest(http.MethodGet, "/o/"+updatedCID+"/markdown", nil)
			w = httptest.NewRecorder()

			server.handleGetMarkdown(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			markdownContent := w.Body.String()
			expectedContent := "# Test Article\n\nThis is the UPDATED markdown content.\n\n## Section 1\n\nUpdated content here.\n\n## Section 2\n\nNew section!"
			if markdownContent != expectedContent {
				t.Errorf("Updated markdown content mismatch.\nExpected: %s\nGot: %s", expectedContent, markdownContent)
			}

			t.Logf("Successfully retrieved updated markdown content")
		})
	})
}

// TestMarkdownContentBackwardCompatibility tests that objects saved without markdown still work
func TestMarkdownContentBackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1024*1024, nil)
	server.baseURL = "http://localhost:8080"

	os.Setenv("SUPABASE_JWT_SECRET", "test-secret-key-for-testing-only")
	defer os.Unsetenv("SUPABASE_JWT_SECRET")

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

	// Save a JSON-LD object (not markdown) using the /api/save endpoint
	jsonldData := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Thing",
		"name":     "Test Object",
	}
	body, _ := json.Marshal(jsonldData)

	r := httptest.NewRequest(http.MethodPost, "/api/save", bytes.NewReader(body))
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleSave(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	cid := response["cid"]
	t.Logf("Saved object CID: %s", cid)

	// Try to retrieve markdown content - should return empty, not error
	r = httptest.NewRequest(http.MethodGet, "/o/"+cid+"/markdown", nil)
	w = httptest.NewRecorder()

	server.handleGetMarkdown(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	markdownContent := w.Body.String()
	if markdownContent != "" {
		t.Errorf("Expected empty markdown content for non-markdown object, got: %s", markdownContent)
	}

	t.Log("✓ Backward compatibility verified: non-markdown objects work correctly")
}
