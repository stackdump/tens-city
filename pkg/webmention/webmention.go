package webmention

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Webmention represents a received or sent webmention
type Webmention struct {
	Source      string    `json:"source"`
	Target      string    `json:"target"`
	Verified    bool      `json:"verified"`
	VerifiedAt  time.Time `json:"verified_at,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	AuthorName  string    `json:"author_name,omitempty"`
	AuthorURL   string    `json:"author_url,omitempty"`
	Content     string    `json:"content,omitempty"`
	MentionType string    `json:"mention_type,omitempty"` // "reply", "like", "repost", "mention"
}

// MentionStore manages webmention storage on disk
type MentionStore struct {
	dataDir string
	mu      sync.Mutex            // guards slugMu map
	slugMu  map[string]*sync.Mutex // per-slug locks
}

// NewMentionStore creates a new MentionStore
func NewMentionStore(dataDir string) *MentionStore {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Warning: failed to create webmention data dir: %v", err)
	}
	return &MentionStore{
		dataDir: dataDir,
		slugMu:  make(map[string]*sync.Mutex),
	}
}

// slugLock returns a per-slug mutex
func (ms *MentionStore) slugLock(slug string) *sync.Mutex {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if m, ok := ms.slugMu[slug]; ok {
		return m
	}
	m := &sync.Mutex{}
	ms.slugMu[slug] = m
	return m
}

func (ms *MentionStore) filePath(slug string) string {
	return filepath.Join(ms.dataDir, slug+".json")
}

// LoadMentions loads all mentions for a slug
func (ms *MentionStore) LoadMentions(slug string) []Webmention {
	data, err := os.ReadFile(ms.filePath(slug))
	if err != nil {
		return []Webmention{}
	}
	var mentions []Webmention
	if err := json.Unmarshal(data, &mentions); err != nil {
		log.Printf("webmention: failed to parse %s: %v", slug, err)
		return []Webmention{}
	}
	return mentions
}

// SaveMention adds or updates a mention for a slug (deduplicates by source+target)
func (ms *MentionStore) SaveMention(slug string, mention Webmention) error {
	mu := ms.slugLock(slug)
	mu.Lock()
	defer mu.Unlock()

	mentions := ms.LoadMentions(slug)

	// Update existing or append
	found := false
	for i, m := range mentions {
		if m.Source == mention.Source && m.Target == mention.Target {
			mentions[i] = mention
			found = true
			break
		}
	}
	if !found {
		mentions = append(mentions, mention)
	}

	data, err := json.MarshalIndent(mentions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ms.filePath(slug), data, 0644)
}

// GetVerifiedMentions returns only verified mentions for a slug
func (ms *MentionStore) GetVerifiedMentions(slug string) []Webmention {
	mentions := ms.LoadMentions(slug)
	var verified []Webmention
	for _, m := range mentions {
		if m.Verified {
			verified = append(verified, m)
		}
	}
	return verified
}
