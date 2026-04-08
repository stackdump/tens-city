package webmention

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// HandleWebmention handles POST /webmention
func (ms *MentionStore) HandleWebmention(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	source := r.FormValue("source")
	target := r.FormValue("target")

	if source == "" || target == "" {
		http.Error(w, "source and target are required", http.StatusBadRequest)
		return
	}

	// Validate URLs
	sourceURL, err := url.Parse(source)
	if err != nil || (sourceURL.Scheme != "http" && sourceURL.Scheme != "https") {
		http.Error(w, "invalid source URL", http.StatusBadRequest)
		return
	}
	targetURL, err := url.Parse(target)
	if err != nil || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
		http.Error(w, "invalid target URL", http.StatusBadRequest)
		return
	}

	// Extract slug from target path (expect /posts/{slug})
	slug := extractSlug(targetURL.Path)
	if slug == "" {
		http.Error(w, "target must be a blog post URL", http.StatusBadRequest)
		return
	}

	// Store unverified mention
	mention := Webmention{
		Source:     source,
		Target:     target,
		Verified:   false,
		ReceivedAt: time.Now().UTC(),
	}

	if err := ms.SaveMention(slug, mention); err != nil {
		log.Printf("webmention: failed to save mention: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Verify asynchronously
	go ms.verify(slug, source, target)

	w.WriteHeader(http.StatusAccepted)
	fmt.Fprint(w, "Webmention accepted")
}

// verify fetches the source URL and confirms it links to the target
func (ms *MentionStore) verify(slug, source, target string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(source)
	if err != nil {
		log.Printf("webmention: failed to fetch source %s: %v", source, err)
		ms.removeMention(slug, source, target)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("webmention: source %s returned %d", source, resp.StatusCode)
		ms.removeMention(slug, source, target)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		log.Printf("webmention: failed to read source %s: %v", source, err)
		ms.removeMention(slug, source, target)
		return
	}

	html := string(body)

	// Confirm the source page actually links to the target
	if !strings.Contains(html, target) {
		log.Printf("webmention: source %s does not contain target %s", source, target)
		ms.removeMention(slug, source, target)
		return
	}

	// Extract author info from <title> and source domain
	authorName := extractTitle(html)
	srcURL, _ := url.Parse(source)
	authorURL := fmt.Sprintf("%s://%s", srcURL.Scheme, srcURL.Host)

	// Update mention as verified
	mention := Webmention{
		Source:      source,
		Target:      target,
		Verified:    true,
		VerifiedAt:  time.Now().UTC(),
		ReceivedAt:  time.Now().UTC(),
		AuthorName:  authorName,
		AuthorURL:   authorURL,
		MentionType: "mention",
	}

	if err := ms.SaveMention(slug, mention); err != nil {
		log.Printf("webmention: failed to update verified mention: %v", err)
	} else {
		log.Printf("webmention: verified %s → %s", source, target)
	}
}

// removeMention removes an unverified mention
func (ms *MentionStore) removeMention(slug, source, target string) {
	mu := ms.slugLock(slug)
	mu.Lock()
	defer mu.Unlock()

	mentions := ms.LoadMentions(slug)
	var filtered []Webmention
	for _, m := range mentions {
		if m.Source == source && m.Target == target && !m.Verified {
			continue
		}
		filtered = append(filtered, m)
	}

	if len(filtered) != len(mentions) {
		data, err := json.MarshalIndent(filtered, "", "  ")
		if err == nil {
			_ = os.WriteFile(ms.filePath(slug), data, 0644)
		}
	}
}

// extractSlug extracts the post slug from a URL path like /posts/{slug}
func extractSlug(path string) string {
	path = strings.TrimSuffix(path, "/")
	if strings.HasPrefix(path, "/posts/") {
		slug := strings.TrimPrefix(path, "/posts/")
		if slug != "" && !strings.Contains(slug, "/") {
			return slug
		}
	}
	return ""
}

var titleRegex = regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)

// extractTitle extracts the <title> content from HTML
func extractTitle(html string) string {
	matches := titleRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

