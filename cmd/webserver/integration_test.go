package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestEndToEndWorkflow tests the complete workflow:
// 1. Save a JSON-LD object via API
// 2. Retrieve it by CID
// 3. Verify the content
func TestEndToEndWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1*1024*1024)

	// Create test auth token
	authToken := createTestToken("test-user-123", "test@example.com", "testuser", "123456")

	// Step 1: Create and save a JSON-LD document
	doc := map[string]interface{}{
		"@context": map[string]string{
			"name":        "http://schema.org/name",
			"description": "http://schema.org/description",
		},
		"@type":       "TestDocument",
		"name":        "Integration Test Document",
		"description": "This is a test document for the end-to-end workflow",
	}

	body, _ := json.Marshal(doc)
	req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var saveResponse map[string]string
	if err := json.NewDecoder(w.Body).Decode(&saveResponse); err != nil {
		t.Fatalf("Failed to decode save response: %v", err)
	}

	cid := saveResponse["cid"]
	if cid == "" {
		t.Fatal("No CID in response")
	}

	t.Logf("Saved document with CID: %s", cid)

	// Step 2: Retrieve the document by CID
	req = httptest.NewRequest("GET", "/o/"+cid, nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Step 3: Verify the content
	var retrievedDoc map[string]interface{}
	body, _ = io.ReadAll(w.Body)
	if err := json.Unmarshal(body, &retrievedDoc); err != nil {
		t.Fatalf("Failed to unmarshal retrieved document: %v", err)
	}

	// Verify @id was injected
	if id, ok := retrievedDoc["@id"].(string); !ok || id != "ipfs://"+cid {
		t.Errorf("Expected @id to be ipfs://%s, got %v", cid, retrievedDoc["@id"])
	}

	// Verify original fields
	if name, ok := retrievedDoc["name"].(string); !ok || name != "Integration Test Document" {
		t.Errorf("Expected name 'Integration Test Document', got %v", retrievedDoc["name"])
	}

	if docType, ok := retrievedDoc["@type"].(string); !ok || docType != "TestDocument" {
		t.Errorf("Expected @type 'TestDocument', got %v", retrievedDoc["@type"])
	}

	t.Log("End-to-end workflow test passed successfully!")
}

// TestWorkflowWithMultipleObjects tests saving and retrieving multiple objects
func TestWorkflowWithMultipleObjects(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, false, 1*1024*1024)

	// Create test auth token
	authToken := createTestToken("test-user-123", "test@example.com", "testuser", "123456")

	documents := []map[string]interface{}{
		{
			"@context": map[string]string{"name": "http://schema.org/name"},
			"@type":    "Document1",
			"name":     "First Document",
		},
		{
			"@context": map[string]string{"name": "http://schema.org/name"},
			"@type":    "Document2",
			"name":     "Second Document",
		},
		{
			"@context": map[string]string{"name": "http://schema.org/name"},
			"@type":    "Document3",
			"name":     "Third Document",
		},
	}

	cids := make([]string, len(documents))

	// Save all documents
	for i, doc := range documents {
		body, _ := json.Marshal(doc)
		req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to save document %d: status %d", i, w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		cids[i] = response["cid"]
		t.Logf("Saved document %d with CID: %s", i+1, cids[i])
	}

	// Retrieve and verify all documents
	for i, cid := range cids {
		req := httptest.NewRequest("GET", "/o/"+cid, nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to retrieve document %d: status %d", i, w.Code)
		}

		var retrieved map[string]interface{}
		body, _ := io.ReadAll(w.Body)
		json.Unmarshal(body, &retrieved)

		expectedType := documents[i]["@type"]
		if retrieved["@type"] != expectedType {
			t.Errorf("Document %d: expected @type %v, got %v", i, expectedType, retrieved["@type"])
		}
	}

	t.Logf("Successfully saved and retrieved %d documents", len(documents))
}
