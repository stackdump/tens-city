package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleCheckOwnership(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Create a test object with author information
	cid := "test-ownership-cid"
	testData := []byte(`{"@context":"test","name":"test object"}`)
	canonical := []byte(`canonical`)
	githubUser := "testuser"
	githubID := "12345"

	if err := storage.SaveObjectWithAuthor(cid, testData, canonical, githubUser, githubID); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}

	server := NewServer(storage, nil, false, 1*1024*1024, nil)

	tests := []struct {
		name           string
		cid            string
		userID         string
		userName       string
		githubID       string
		includeAuth    bool
		expectedStatus int
		expectedOwned  bool
	}{
		{
			name:           "Owner by GitHub ID",
			cid:            cid,
			userID:         "user123",
			userName:       "otheruser",
			githubID:       "12345", // Matches author GitHub ID
			includeAuth:    true,
			expectedStatus: http.StatusOK,
			expectedOwned:  true,
		},
		{
			name:           "Owner by username",
			cid:            cid,
			userID:         "user456",
			userName:       "testuser", // Matches author username
			githubID:       "99999",
			includeAuth:    true,
			expectedStatus: http.StatusOK,
			expectedOwned:  true,
		},
		{
			name:           "Non-owner",
			cid:            cid,
			userID:         "user789",
			userName:       "otheruser",
			githubID:       "67890",
			includeAuth:    true,
			expectedStatus: http.StatusOK,
			expectedOwned:  false,
		},
		{
			name:           "No authentication",
			cid:            cid,
			includeAuth:    false,
			expectedStatus: http.StatusOK,
			expectedOwned:  false,
		},
		{
			name:           "Nonexistent object",
			cid:            "nonexistent-cid",
			userID:         "user123",
			userName:       githubUser,
			githubID:       githubID,
			includeAuth:    true,
			expectedStatus: http.StatusNotFound,
			expectedOwned:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/ownership/"+tt.cid, nil)
			
			// Add authentication header if required
			if tt.includeAuth {
				token := createTestToken(tt.userID, "test@example.com", tt.userName, tt.githubID)
				req.Header.Set("Authorization", "Bearer "+token)
			}

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Check response body for ownership status if status is OK
			if resp.StatusCode == http.StatusOK {
				var result map[string]bool
				if err := decodeJSON(w.Body, &result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if result["owned"] != tt.expectedOwned {
					t.Errorf("Expected owned=%v, got owned=%v", tt.expectedOwned, result["owned"])
				}
			}
		})
	}
}

func TestCheckOwnershipWithUsernameOnly(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Create a test object with only username (no GitHub ID)
	cid := "test-username-ownership"
	testData := []byte(`{"@context":"test","name":"test object"}`)
	canonical := []byte(`canonical`)
	githubUser := "testuser"
	githubID := "" // No GitHub ID

	if err := storage.SaveObjectWithAuthor(cid, testData, canonical, githubUser, githubID); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}

	server := NewServer(storage, nil, false, 1*1024*1024, nil)

	// Test that owner can be identified by username when GitHub ID is not available
	token := createTestToken("user123", "test@example.com", githubUser, "")
	req := httptest.NewRequest("GET", "/api/ownership/"+cid, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var result map[string]bool
	if err := decodeJSON(w.Body, &result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !result["owned"] {
		t.Error("Expected owned=true for matching username")
	}
}
