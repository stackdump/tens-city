package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDeleteObject(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Create a test object with author information
	cid := "test-delete-cid"
	testData := []byte(`{"@context":"test","name":"test object"}`)
	canonical := []byte(`canonical`)
	githubUser := "testuser"
	githubID := "12345"

	if err := storage.SaveObjectWithAuthor(cid, testData, canonical, githubUser, githubID); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}

	server := NewServer(storage, "", false, 1*1024*1024)

	tests := []struct {
		name           string
		cid            string
		userID         string
		userName       string
		githubID       string
		expectedStatus int
	}{
		{
			name:           "Author can delete by GitHub ID",
			cid:            cid,
			userID:         "user123",
			userName:       "otheruser",
			githubID:       "12345", // Matches author GitHub ID
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Non-author cannot delete",
			cid:            cid,
			userID:         "user456",
			userName:       "otheruser",
			githubID:       "67890",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Missing authentication",
			cid:            cid,
			userID:         "",
			userName:       "",
			githubID:       "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Nonexistent object",
			cid:            "nonexistent-cid",
			userID:         "user123",
			userName:       githubUser,
			githubID:       githubID,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-create the object for each test (except the first which deletes it)
			if tt.name != "Author can delete by GitHub ID" && tt.name != "Nonexistent object" {
				storage.SaveObjectWithAuthor(cid, testData, canonical, githubUser, githubID)
			}

			req := httptest.NewRequest("DELETE", "/o/"+tt.cid, nil)
			
			// Add authentication header if user info is provided
			if tt.userID != "" {
				token := createTestToken(tt.userID, "test@example.com", tt.userName, tt.githubID)
				req.Header.Set("Authorization", "Bearer "+token)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Verify object is deleted for successful delete
			if tt.expectedStatus == http.StatusNoContent {
				_, err := storage.GetObject(tt.cid)
				if err == nil {
					t.Error("Expected object to be deleted, but it still exists")
				}
			}
		})
	}
}

func TestHandleDeleteObjectByUsername(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Create a test object with only username (no GitHub ID)
	cid := "test-username-delete"
	testData := []byte(`{"@context":"test","name":"test object"}`)
	canonical := []byte(`canonical`)
	githubUser := "testuser"
	githubID := "" // No GitHub ID

	if err := storage.SaveObjectWithAuthor(cid, testData, canonical, githubUser, githubID); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}

	server := NewServer(storage, "", false, 1*1024*1024)

	// Test that author can delete by username when GitHub ID is not available
	token := createTestToken("user123", "test@example.com", githubUser, "")
	req := httptest.NewRequest("DELETE", "/o/"+cid, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Verify object is deleted
	_, err := storage.GetObject(cid)
	if err == nil {
		t.Error("Expected object to be deleted, but it still exists")
	}
}

func TestGetObjectAuthor(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	tests := []struct {
		name           string
		cid            string
		saveUser       string
		saveID         string
		expectedUser   string
		expectedID     string
		expectError    bool
	}{
		{
			name:         "Object with full author info",
			cid:          "test-author-1",
			saveUser:     "testuser",
			saveID:       "12345",
			expectedUser: "testuser",
			expectedID:   "12345",
			expectError:  false,
		},
		{
			name:         "Object with only username",
			cid:          "test-author-2",
			saveUser:     "testuser",
			saveID:       "",
			expectedUser: "testuser",
			expectedID:   "",
			expectError:  false,
		},
		{
			name:         "Object with only GitHub ID",
			cid:          "test-author-3",
			saveUser:     "",
			saveID:       "12345",
			expectedUser: "",
			expectedID:   "12345",
			expectError:  false,
		},
		{
			name:        "Nonexistent object",
			cid:         "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save object if not testing nonexistent
			if !tt.expectError {
				testData := []byte(`{"@context":"test","name":"test"}`)
				canonical := []byte(`canonical`)
				if err := storage.SaveObjectWithAuthor(tt.cid, testData, canonical, tt.saveUser, tt.saveID); err != nil {
					t.Fatalf("Failed to save object: %v", err)
				}
			}

			user, id, err := storage.GetObjectAuthor(tt.cid)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if user != tt.expectedUser {
				t.Errorf("Expected user %q, got %q", tt.expectedUser, user)
			}

			if id != tt.expectedID {
				t.Errorf("Expected ID %q, got %q", tt.expectedID, id)
			}
		})
	}
}

func TestDeleteObject(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Create a test object
	cid := "test-delete-store"
	testData := []byte(`{"@context":"test","name":"test"}`)
	canonical := []byte(`canonical data`)
	
	if err := storage.SaveObject(cid, testData, canonical); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}

	// Verify object exists
	_, err := storage.GetObject(cid)
	if err != nil {
		t.Fatalf("Object should exist before deletion: %v", err)
	}

	// Delete the object
	if err := storage.DeleteObject(cid); err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}

	// Verify object is deleted
	_, err = storage.GetObject(cid)
	if err == nil {
		t.Error("Object should not exist after deletion")
	}

	// Test deleting nonexistent object (should not error)
	if err := storage.DeleteObject("nonexistent"); err != nil {
		t.Errorf("Deleting nonexistent object should not error: %v", err)
	}
}

func TestSaveWithValidation(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, "", false, 1*1024*1024)

	tests := []struct {
		name           string
		doc            map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid JSON-LD",
			doc: map[string]interface{}{
				"@context": map[string]interface{}{
					"name": "http://schema.org/name",
				},
				"@type": "Person",
				"name":  "John Doe",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Missing @context",
			doc: map[string]interface{}{
				"name": "John Doe",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid @context type",
			doc: map[string]interface{}{
				"@context": 123,
				"name":     "John Doe",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Key with control character",
			doc: map[string]interface{}{
				"@context": map[string]interface{}{
					"name": "http://schema.org/name",
				},
				"name\x00": "John Doe",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.doc)
			req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Add authentication
			token := createTestToken("user123", "test@example.com", "testuser", "12345")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				bodyBytes := new(bytes.Buffer)
				bodyBytes.ReadFrom(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, bodyBytes.String())
			}
		})
	}
}

func TestContentSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	// Set a very small content size limit for testing (1KB)
	server := NewServer(storage, "", false, 1024)

	// Create a document that exceeds the size limit
	largeDoc := map[string]interface{}{
		"@context": "http://schema.org",
		"@type":    "Person",
	}
	
	// Add a large string to exceed 1KB
	largeString := make([]byte, 2000)
	for i := range largeString {
		largeString[i] = 'A'
	}
	largeDoc["data"] = string(largeString)

	body, _ := json.Marshal(largeDoc)
	req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	// Add authentication
	token := createTestToken("user123", "test@example.com", "testuser", "12345")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	// Should fail with 413 Request Entity Too Large or 400 Bad Request
	if resp.StatusCode != http.StatusRequestEntityTooLarge && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 413 or 400 for oversized content, got %d", resp.StatusCode)
	}
}
