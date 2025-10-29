package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveObject_InjectsID(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFSStore(tmpDir)

	// Test data
	cid := "z4EBG9jTestCID123"
	raw := []byte(`{
		"@context": {
			"name": "http://schema.org/name"
		},
		"name": "Test"
	}`)
	canonical := []byte("_:c14n0 <http://schema.org/name> \"Test\" .\n")

	// Save the object
	err = store.SaveObject(cid, raw, canonical)
	if err != nil {
		t.Fatalf("SaveObject failed: %v", err)
	}

	// Read the saved object
	savedPath := filepath.Join(tmpDir, "o", cid)
	savedData, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved object: %v", err)
	}

	// Parse the saved JSON
	var savedDoc map[string]interface{}
	if err := json.Unmarshal(savedData, &savedDoc); err != nil {
		t.Fatalf("Failed to parse saved JSON: %v", err)
	}

	// Verify @id field was injected
	idValue, ok := savedDoc["@id"]
	if !ok {
		t.Errorf("@id field was not injected into saved object")
	}

	expectedID := "ipfs://" + cid
	if idValue != expectedID {
		t.Errorf("Expected @id to be %q, got %q", expectedID, idValue)
	}

	// Verify other fields are preserved
	if savedDoc["name"] != "Test" {
		t.Errorf("Original fields were not preserved")
	}
}

func TestSaveObject_SavesCanonical(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFSStore(tmpDir)

	cid := "z4EBG9jTestCID123"
	raw := []byte(`{"name": "Test"}`)
	canonical := []byte("_:c14n0 <http://schema.org/name> \"Test\" .\n")

	err = store.SaveObject(cid, raw, canonical)
	if err != nil {
		t.Fatalf("SaveObject failed: %v", err)
	}

	// Verify canonical file was saved
	canonicalPath := filepath.Join(tmpDir, "o", "canonical", cid+".nq")
	savedCanonical, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("Failed to read canonical file: %v", err)
	}

	if string(savedCanonical) != string(canonical) {
		t.Errorf("Canonical content mismatch.\nExpected: %s\nGot: %s", canonical, savedCanonical)
	}
}

func TestSaveObject_FormatsJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFSStore(tmpDir)

	cid := "z4EBG9jTestCID123"
	// Compact JSON input
	raw := []byte(`{"@context":{"name":"http://schema.org/name"},"name":"Test"}`)
	canonical := []byte("test")

	err = store.SaveObject(cid, raw, canonical)
	if err != nil {
		t.Fatalf("SaveObject failed: %v", err)
	}

	// Read the saved object
	savedPath := filepath.Join(tmpDir, "o", cid)
	savedData, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved object: %v", err)
	}

	// Verify it's formatted (has newlines)
	savedStr := string(savedData)
	if len(savedStr) == len(string(raw)) {
		t.Errorf("Expected JSON to be formatted with indentation")
	}

	// Verify it's valid JSON
	var doc map[string]interface{}
	if err := json.Unmarshal(savedData, &doc); err != nil {
		t.Errorf("Saved data is not valid JSON: %v", err)
	}
}

func TestReadObject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewFSStore(tmpDir)

	cid := "z4EBG9jTestCID123"
	raw := []byte(`{"name": "Test"}`)
	canonical := []byte("test")

	// Save first
	err = store.SaveObject(cid, raw, canonical)
	if err != nil {
		t.Fatalf("SaveObject failed: %v", err)
	}

	// Read back
	readData, err := store.ReadObject(cid)
	if err != nil {
		t.Fatalf("ReadObject failed: %v", err)
	}

	// Parse and verify @id was injected
	var doc map[string]interface{}
	if err := json.Unmarshal(readData, &doc); err != nil {
		t.Fatalf("Failed to parse read data: %v", err)
	}

	expectedID := "ipfs://" + cid
	if doc["@id"] != expectedID {
		t.Errorf("Expected @id to be %q, got %q", expectedID, doc["@id"])
	}
}
