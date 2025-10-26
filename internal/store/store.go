package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FSStore is a simple file-system-backed store for sealed objects and user containers.
// Structure:
//  {base}/o/{cid}              -> original JSON-LD bytes
//  {base}/o/canonical/{cid}.nq -> canonical n-quads text
//  {base}/u/{login}/g/{slug}/latest -> text file containing CID
//  {base}/u/{login}/g/{slug}/_history -> JSON array of history entries
type FSStore struct {
	base string
}

type HistoryEntry struct {
	CID       string    `json:"cid"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewFSStore(base string) *FSStore {
	return &FSStore{base: base}
}

// ObjectPath returns the path where the original JSON-LD for a CID is stored.
func (s *FSStore) ObjectPath(cid string) string {
	return filepath.Join(s.base, "o", cid)
}

// SaveObject writes the raw JSON-LD and canonical bytes to disk.
func (s *FSStore) SaveObject(cid string, raw []byte, canonical []byte) error {
	objDir := filepath.Join(s.base, "o")
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return err
	}
	// raw
	if err := os.WriteFile(filepath.Join(objDir, cid), raw, 0o644); err != nil {
		return err
	}
	// canonical
	canonDir := filepath.Join(objDir, "canonical")
	if err := os.MkdirAll(canonDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(canonDir, cid+".nq"), canonical, 0o644); err != nil {
		return err
	}
	return nil
}

// ensureGistDir ensures the directory structure for a user's gist exists.
func (s *FSStore) ensureGistDir(user, slug string) (string, error) {
	dir := filepath.Join(s.base, "u", user, "g", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// UpdateLatest writes the pointer file /u/{login}/g/{slug}/latest containing the CID.
func (s *FSStore) UpdateLatest(user, slug, cid string) error {
	dir, err := s.ensureGistDir(user, slug)
	if err != nil {
		return err
	}
	latestPath := filepath.Join(dir, "latest")
	return os.WriteFile(latestPath, []byte(cid), 0o644)
}

// AppendHistory appends a history entry (CID + timestamp) to _history (JSON array).
func (s *FSStore) AppendHistory(user, slug, cidStr string) error {
	dir, err := s.ensureGistDir(user, slug)
	if err != nil {
		return err
	}
	historyPath := filepath.Join(dir, "_history")

	var entries []HistoryEntry
	if data, err := os.ReadFile(historyPath); err == nil {
		_ = json.Unmarshal(data, &entries) // ignore error; we'll overwrite if corrupted
	}

	entries = append(entries, HistoryEntry{
		CID:       cidStr,
		CreatedAt: time.Now().UTC(),
	})

	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyPath, out, 0o644)
}

// ReadLatest returns the CID stored in latest (or empty string if not present).
func (s *FSStore) ReadLatest(user, slug string) (string, error) {
	dir := filepath.Join(s.base, "u", user, "g", slug)
	latestPath := filepath.Join(dir, "latest")
	b, err := os.ReadFile(latestPath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ReadHistory reads and returns the history entries for a gist.
func (s *FSStore) ReadHistory(user, slug string) ([]HistoryEntry, error) {
	dir := filepath.Join(s.base, "u", user, "g", slug)
	historyPath := filepath.Join(dir, "_history")
	b, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, err
	}
	var entries []HistoryEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Utility: Create container for user (e.g., when user logs in)
func (s *FSStore) EnsureUserContainer(user string) error {
	dir := filepath.Join(s.base, "u", user)
	return os.MkdirAll(dir, 0o755)
}

// Utility: Save identity doc at /id/github:{id}
func (s *FSStore) SaveIdentity(githubID string, doc []byte) error {
	dir := filepath.Join(s.base, "id")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "github:"+githubID+".jsonld")
	return os.WriteFile(path, doc, 0o644)
}

// Utility: Read object raw JSON-LD
func (s *FSStore) ReadObject(cid string) ([]byte, error) {
	path := filepath.Join(s.base, "o", cid)
	return os.ReadFile(path)
}

// Utility: Read canonical n-quads for a cid
func (s *FSStore) ReadCanonical(cid string) ([]byte, error) {
	path := filepath.Join(s.base, "o", "canonical", cid+".nq")
	return os.ReadFile(path)
}
