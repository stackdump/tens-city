package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestAuthenticationRequired ensures that save endpoint requires authentication
func TestAuthenticationRequired(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1*1024*1024, nil)

	doc := map[string]interface{}{
		"@context": "https://pflow.xyz/schema",
		"@type":    "Test",
		"name":     "Test Object",
	}
	body, _ := json.Marshal(doc)

	// Test without Authorization header
	req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for missing auth, got %d", w.Code)
	}

	// Test with empty Authorization header
	req = httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for empty auth, got %d", w.Code)
	}
}

// TestInvalidTokenRejected ensures that invalid tokens are rejected
func TestInvalidTokenRejected(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1*1024*1024, nil)

	doc := map[string]interface{}{
		"@context": "https://pflow.xyz/schema",
		"@type":    "Test",
		"name":     "Test Object",
	}
	body, _ := json.Marshal(doc)

	tests := []struct {
		name  string
		token string
	}{
		{"Invalid format", "not.a.valid.token"},
		{"Only two parts", "header.payload"},
		{"Random string", "completely-invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tt.token)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != 401 {
				t.Errorf("Expected status 401 for invalid token, got %d", w.Code)
			}
		})
	}
}

// TestAuthorInfoInjection validates that author information is properly injected
func TestAuthorInfoInjection(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1*1024*1024, nil)

	tests := []struct {
		name         string
		userID       string
		email        string
		username     string
		githubID     string
		expectedName string
		expectedID   string
	}{
		{
			name:         "Full user info",
			userID:       "user-123",
			email:        "user@example.com",
			username:     "testuser",
			githubID:     "987654",
			expectedName: "testuser",
			expectedID:   "github:987654",
		},
		{
			name:         "Username only",
			userID:       "user-456",
			email:        "",
			username:     "anotheruser",
			githubID:     "",
			expectedName: "anotheruser",
			expectedID:   "",
		},
		{
			name:         "GitHub ID only (no username)",
			userID:       "user-789",
			email:        "emailonly@example.com",
			username:     "",
			githubID:     "111222",
			expectedName: "", // No username, so no name field
			expectedID:   "github:111222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authToken := createTestToken(tt.userID, tt.email, tt.username, tt.githubID)

			doc := map[string]interface{}{
				"@context": "https://pflow.xyz/schema",
				"@type":    "Test",
				"name":     "Test with " + tt.name,
			}
			body, _ := json.Marshal(doc)

			req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+authToken)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			var result map[string]string
			if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			cid := result["cid"]
			if cid == "" {
				t.Fatal("No CID in response")
			}

			// Retrieve the saved object
			savedData, err := storage.GetObject(cid)
			if err != nil {
				t.Fatalf("Failed to retrieve object: %v", err)
			}

			var savedDoc map[string]interface{}
			if err := json.Unmarshal(savedData, &savedDoc); err != nil {
				t.Fatalf("Failed to unmarshal saved object: %v", err)
			}

			// Check author information
			author, hasAuthor := savedDoc["author"].(map[string]interface{})
			if !hasAuthor && (tt.expectedName != "" || tt.expectedID != "") {
				t.Fatal("Author information not found in saved object")
			}

			if hasAuthor {
				if tt.expectedName != "" {
					if authorName, ok := author["name"].(string); !ok || authorName != tt.expectedName {
						t.Errorf("Expected author name '%s', got '%v'", tt.expectedName, author["name"])
					}
				}

				if tt.expectedID != "" {
					if authorID, ok := author["id"].(string); !ok || authorID != tt.expectedID {
						t.Errorf("Expected author id '%s', got '%v'", tt.expectedID, author["id"])
					}
				}

				// Check that @type is always set
				if authorType, ok := author["@type"].(string); !ok || authorType != "Person" {
					t.Errorf("Expected author @type 'Person', got '%v'", author["@type"])
				}

				// Check identifier if username is present
				if tt.username != "" {
					expectedIdentifier := "https://github.com/" + tt.username
					if identifier, ok := author["identifier"].(string); !ok || identifier != expectedIdentifier {
						t.Errorf("Expected identifier '%s', got '%v'", expectedIdentifier, author["identifier"])
					}
				}
			}
		})
	}
}
