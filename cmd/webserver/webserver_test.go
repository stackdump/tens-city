package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stackdump/tens-city/internal/docserver"
	"github.com/stackdump/tens-city/internal/static"
	"github.com/stackdump/tens-city/internal/store"
)

func TestHandleGetObject(t *testing.T) {
	tmpDir := t.TempDir()
	fsStore := store.NewFSStore(tmpDir)

	// Create test object directly
	cid := "test-cid-456"
	testData := []byte(`{"@context":"test","name":"test"}`)
	canonical := []byte(`canonical`)
	if err := fsStore.SaveObject(cid, testData, canonical); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}

	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil)

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

func TestHandleGetLatest(t *testing.T) {
	tmpDir := t.TempDir()
	fsStore := store.NewFSStore(tmpDir)

	// Create user gist structure directly
	user := "testuser"
	slug := "testslug"
	cid := "test-cid-789"

	if err := fsStore.UpdateLatest(user, slug, cid); err != nil {
		t.Fatalf("Failed to update latest: %v", err)
	}

	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil)

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
	fsStore := store.NewFSStore(tmpDir)

	user := "testuser"
	slug := "testslug"
	cid1 := "cid-001"
	cid2 := "cid-002"

	// Add history entries directly
	if err := fsStore.AppendHistory(user, slug, cid1); err != nil {
		t.Fatalf("Failed to append history: %v", err)
	}
	if err := fsStore.AppendHistory(user, slug, cid2); err != nil {
		t.Fatalf("Failed to append history: %v", err)
	}

	storage := NewFSStorage(tmpDir)
	server := NewServer(storage, nil, nil)

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

	server := NewServer(storage, publicFS, nil)

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
}

func TestIndexPageWithJSONLD(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFSStorage(tmpDir)

	// Get the embedded public filesystem
	publicFS, err := static.Public()
	if err != nil {
		t.Fatalf("Failed to get public filesystem: %v", err)
	}

	// Create a temporary content directory with test posts
	contentDir := t.TempDir()
	testPost := `---
title: Test Post
description: A test blog post
datePublished: 2025-11-03T00:00:00Z
author:
  name: Test Author
  type: Person
  url: https://github.com/testauthor
tags:
  - test
lang: en
slug: test-post
draft: false
---

# Test Post

This is a test post.
`
	if err := os.WriteFile(contentDir+"/test-post.md", []byte(testPost), 0644); err != nil {
		t.Fatalf("Failed to create test post: %v", err)
	}

	// Create docserver with the test content
	docServer := docserver.NewDocServer(contentDir, "http://localhost:8080")
	server := NewServer(storage, publicFS, docServer)

	// Test serving index.html at root with JSON-LD
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check that JSON-LD script tag is present
	if !strings.Contains(bodyStr, `<script type="application/ld+json">`) {
		t.Error("Expected JSON-LD script tag in index page")
	}

	// Check that it contains the collection metadata
	if !strings.Contains(bodyStr, `"@type": "CollectionPage"`) {
		t.Error("Expected CollectionPage type in JSON-LD")
	}

	// Check that it contains the test post
	if !strings.Contains(bodyStr, "Test Post") {
		t.Error("Expected test post headline in JSON-LD")
	}

	if !strings.Contains(bodyStr, "test-post") {
		t.Error("Expected test post slug in JSON-LD")
	}
}
