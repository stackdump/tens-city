package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackdump/tens-city/internal/canonical"
)

// TestSaveWithDifferentKeyOrder tests that JSON with different key orderings
// produces the same CID when saved via the API
func TestSaveWithDifferentKeyOrder(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, "", false)

	// Two JSON strings with the same content but different key order
	json1 := map[string]interface{}{
		"@context":    "https://pflow.xyz/schema",
		"@type":       "PetriNet",
		"@version":    "1.1",
		"arcs":        []interface{}{},
		"places":      map[string]interface{}{},
		"token":       []interface{}{"https://pflow.xyz/tokens/black"},
		"transitions": map[string]interface{}{},
	}

	json2 := map[string]interface{}{
		"transitions": map[string]interface{}{},
		"token":       []interface{}{"https://pflow.xyz/tokens/black"},
		"places":      map[string]interface{}{},
		"arcs":        []interface{}{},
		"@version":    "1.1",
		"@type":       "PetriNet",
		"@context":    "https://pflow.xyz/schema",
	}

	// Save first JSON
	body1, _ := json.Marshal(json1)
	req1 := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	server.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("First request failed with status %d", w1.Code)
	}

	var result1 map[string]string
	if err := json.NewDecoder(w1.Body).Decode(&result1); err != nil {
		t.Fatalf("Failed to decode first response: %v", err)
	}

	// Save second JSON
	body2, _ := json.Marshal(json2)
	req2 := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Second request failed with status %d", w2.Code)
	}

	var result2 map[string]string
	if err := json.NewDecoder(w2.Body).Decode(&result2); err != nil {
		t.Fatalf("Failed to decode second response: %v", err)
	}

	// Both should produce the same CID
	if result1["cid"] != result2["cid"] {
		t.Errorf("Expected same CID for different key orders\nFirst: %s\nSecond: %s", result1["cid"], result2["cid"])
	}

	t.Logf("Successfully produced consistent CID: %s", result1["cid"])
}

// TestSaveMultipleTimes tests that saving the same JSON multiple times
// always produces the same CID
func TestSaveMultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, "", false)

	doc := map[string]interface{}{
		"@context":    "https://pflow.xyz/schema",
		"@type":       "PetriNet",
		"@version":    "1.1",
		"arcs":        []interface{}{},
		"places":      map[string]interface{}{},
		"token":       []interface{}{"https://pflow.xyz/tokens/black"},
		"transitions": map[string]interface{}{},
	}

	cids := make(map[string]bool)

	// Save the same document 10 times
	for i := 0; i < 10; i++ {
		body, _ := json.Marshal(doc)
		req := httptest.NewRequest("POST", "/api/save", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Request %d failed with status %d", i+1, w.Code)
		}

		var result map[string]string
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response %d: %v", i+1, err)
		}

		cids[result["cid"]] = true
	}

	// All saves should produce exactly one unique CID
	if len(cids) != 1 {
		t.Errorf("Expected exactly 1 unique CID, got %d unique CIDs", len(cids))
		for cid := range cids {
			t.Logf("  CID: %s", cid)
		}
	} else {
		for cid := range cids {
			t.Logf("Successfully produced consistent CID across 10 saves: %s", cid)
		}
	}
}

// TestCanonicalJSONMarshaling tests the canonical JSON marshaling directly
func TestCanonicalJSONMarshaling(t *testing.T) {
	obj := map[string]interface{}{
		"z":        "last",
		"a":        "first",
		"m":        "middle",
		"nested": map[string]interface{}{
			"b": 2,
			"a": 1,
		},
	}

	canonical1, err := canonical.MarshalJSON(obj)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Keys should be sorted
	expected := `{"a":"first","m":"middle","nested":{"a":1,"b":2},"z":"last"}`
	if string(canonical1) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(canonical1))
	}

	// Multiple marshals should produce identical output
	for i := 0; i < 5; i++ {
		canonical2, err := canonical.MarshalJSON(obj)
		if err != nil {
			t.Fatalf("Failed to marshal (run %d): %v", i, err)
		}
		if string(canonical1) != string(canonical2) {
			t.Errorf("Run %d produced different output", i)
		}
	}
}
