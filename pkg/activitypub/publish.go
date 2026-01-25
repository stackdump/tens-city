package activitypub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// PublishResult tracks the result of publishing to a follower
type PublishResult struct {
	Follower string
	Success  bool
	Error    string
}

// PublishPost sends a Create activity for a blog post to all followers.
// Returns the number of successful deliveries and any errors.
func (a *Actor) PublishPost(post BlogPost) ([]PublishResult, error) {
	followers := a.loadFollowers()
	if len(followers) == 0 {
		log.Printf("No followers to notify for post: %s", post.Title)
		return nil, nil
	}

	// Check if already published
	if a.isPostPublished(post.ID) {
		log.Printf("Post already published: %s", post.Title)
		return nil, nil
	}

	activity := a.postToActivity(post)
	activity.Context = "https://www.w3.org/ns/activitystreams"

	body, err := json.Marshal(activity)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity: %w", err)
	}

	log.Printf("Publishing '%s' to %d followers", post.Title, len(followers))

	var results []PublishResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, follower := range followers {
		wg.Add(1)
		go func(followerURL string) {
			defer wg.Done()

			result := PublishResult{Follower: followerURL}

			inbox, err := a.FetchActorInbox(followerURL)
			if err != nil {
				result.Error = fmt.Sprintf("failed to fetch inbox: %v", err)
				log.Printf("Failed to fetch inbox for %s: %v", followerURL, err)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			resp, err := a.SignedPost(inbox, body)
			if err != nil {
				result.Error = fmt.Sprintf("failed to POST: %v", err)
				log.Printf("Failed to POST to %s: %v", inbox, err)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				result.Success = true
				log.Printf("Published to %s (status %d)", followerURL, resp.StatusCode)
			} else {
				result.Error = fmt.Sprintf("status %d: %s", resp.StatusCode, string(respBody))
				log.Printf("Publish to %s failed with status %d: %s", followerURL, resp.StatusCode, string(respBody))
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(follower)
	}

	wg.Wait()

	// Mark as published if at least one delivery succeeded
	for _, r := range results {
		if r.Success {
			if err := a.markPostPublished(post.ID); err != nil {
				log.Printf("Warning: failed to mark post as published: %v", err)
			}
			break
		}
	}

	return results, nil
}

// PublishNewPosts checks for posts that haven't been published yet and publishes them.
// Returns the number of posts published.
func (a *Actor) PublishNewPosts(posts []BlogPost) (int, error) {
	published := 0
	for _, post := range posts {
		if !a.isPostPublished(post.ID) {
			results, err := a.PublishPost(post)
			if err != nil {
				log.Printf("Error publishing post %s: %v", post.Title, err)
				continue
			}
			// Count as published if we attempted delivery (even if no followers)
			if results != nil || len(a.loadFollowers()) == 0 {
				published++
			}
		}
	}
	return published, nil
}

// publishedFilePath returns the path to the published posts tracking file
func (a *Actor) publishedFilePath() string {
	dir := filepath.Dir(a.config.KeyPath)
	return filepath.Join(dir, "published.json")
}

// loadPublished loads the set of published post IDs
func (a *Actor) loadPublished() map[string]bool {
	data, err := os.ReadFile(a.publishedFilePath())
	if err != nil {
		return make(map[string]bool)
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return make(map[string]bool)
	}

	published := make(map[string]bool)
	for _, id := range ids {
		published[id] = true
	}
	return published
}

// isPostPublished checks if a post has already been published
func (a *Actor) isPostPublished(postID string) bool {
	published := a.loadPublished()
	return published[postID]
}

// markPostPublished marks a post as published
func (a *Actor) markPostPublished(postID string) error {
	publishedPath := a.publishedFilePath()

	// Ensure directory exists
	dir := filepath.Dir(publishedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	published := a.loadPublished()
	published[postID] = true

	// Convert back to slice
	var ids []string
	for id := range published {
		ids = append(ids, id)
	}

	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(publishedPath, data, 0644)
}

// GetPublishedCount returns the number of published posts
func (a *Actor) GetPublishedCount() int {
	return len(a.loadPublished())
}

// ClearPublished clears the published tracking (for testing/reset)
func (a *Actor) ClearPublished() error {
	return os.Remove(a.publishedFilePath())
}
