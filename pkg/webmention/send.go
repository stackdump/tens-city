package webmention

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SendResult represents the outcome of sending a webmention to one target
type SendResult struct {
	Target   string `json:"target"`
	Endpoint string `json:"endpoint"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

var hrefRegex = regexp.MustCompile(`<a[^>]+href="(https?://[^"]+)"`)
var linkRelRegex = regexp.MustCompile(`<link[^>]+rel="webmention"[^>]+href="([^"]+)"`)
var linkRelRegex2 = regexp.MustCompile(`<link[^>]+href="([^"]+)"[^>]+rel="webmention"`)

// SendMentions extracts external links from HTML and sends webmentions to any that support them
func (ms *MentionStore) SendMentions(sourceURL, htmlContent string) []SendResult {
	// Extract unique external URLs
	matches := hrefRegex.FindAllStringSubmatch(htmlContent, -1)
	seen := make(map[string]bool)
	var targets []string
	for _, match := range matches {
		href := match[1]
		if seen[href] {
			continue
		}
		// Skip self-links
		if strings.Contains(href, sourceURL) {
			continue
		}
		seen[href] = true
		targets = append(targets, href)
	}

	if len(targets) == 0 {
		return nil
	}

	log.Printf("webmention: sending mentions for %s to %d targets", sourceURL, len(targets))

	var results []SendResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, target := range targets {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			result := SendResult{Target: targetURL}

			endpoint, err := discoverEndpoint(targetURL)
			if err != nil || endpoint == "" {
				// No webmention endpoint — not an error, most sites don't support it
				return
			}
			result.Endpoint = endpoint

			err = sendMention(endpoint, sourceURL, targetURL)
			if err != nil {
				result.Error = err.Error()
				log.Printf("webmention: failed to send to %s: %v", endpoint, err)
			} else {
				result.Success = true
				log.Printf("webmention: sent %s → %s via %s", sourceURL, targetURL, endpoint)
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(target)
	}

	wg.Wait()
	return results
}

// discoverEndpoint finds the webmention endpoint for a URL
func discoverEndpoint(targetURL string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check Link header first
	for _, link := range resp.Header.Values("Link") {
		if strings.Contains(link, `rel="webmention"`) {
			// Extract URL from < >
			start := strings.Index(link, "<")
			end := strings.Index(link, ">")
			if start >= 0 && end > start {
				return resolveURL(targetURL, link[start+1:end])
			}
		}
	}

	// Parse HTML for <link rel="webmention">
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	html := string(body)
	if matches := linkRelRegex.FindStringSubmatch(html); len(matches) > 1 {
		return resolveURL(targetURL, matches[1])
	}
	if matches := linkRelRegex2.FindStringSubmatch(html); len(matches) > 1 {
		return resolveURL(targetURL, matches[1])
	}

	return "", nil
}

// sendMention sends a webmention to the given endpoint
func sendMention(endpoint, source, target string) error {
	data := url.Values{
		"source": {source},
		"target": {target},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(endpoint, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
}

// resolveURL resolves a possibly relative URL against a base
func resolveURL(base, ref string) (string, error) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, nil
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(refURL).String(), nil
}
