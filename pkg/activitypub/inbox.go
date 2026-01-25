package activitypub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IncomingActivity represents an incoming ActivityPub activity
type IncomingActivity struct {
	Context interface{} `json:"@context,omitempty"`
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Actor   string      `json:"actor"`
	Object  interface{} `json:"object"` // Can be string (for Follow) or object (for Create, etc.)
}

// HandleInbox handles incoming ActivityPub activities.
// POST /users/{username}/inbox
func (a *Actor) HandleInbox(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if username != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Read the activity body
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 1MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log the incoming activity
	sig := r.Header.Get("Signature")
	log.Printf("ActivityPub inbox received: %d bytes from %s", len(body), truncateSignature(sig))

	// Parse the activity
	var activity IncomingActivity
	if err := json.Unmarshal(body, &activity); err != nil {
		log.Printf("Failed to parse activity: %v", err)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Log activity details
	log.Printf("Activity: type=%s actor=%s", activity.Type, activity.Actor)

	// Handle different activity types
	switch activity.Type {
	case "Follow":
		a.handleFollow(w, activity)
	case "Undo":
		a.handleUndo(w, activity)
	case "Accept":
		// Someone accepted our follow request (we don't follow others, so just acknowledge)
		log.Printf("Received Accept from %s", activity.Actor)
		w.WriteHeader(http.StatusAccepted)
	case "Reject":
		log.Printf("Received Reject from %s", activity.Actor)
		w.WriteHeader(http.StatusAccepted)
	case "Create", "Update", "Delete", "Announce", "Like":
		// Content activities - acknowledge but don't process
		log.Printf("Received %s from %s", activity.Type, activity.Actor)
		w.WriteHeader(http.StatusAccepted)
	default:
		log.Printf("Unknown activity type: %s", activity.Type)
		w.WriteHeader(http.StatusAccepted)
	}
}

// handleFollow processes a Follow activity and sends an Accept response
func (a *Actor) handleFollow(w http.ResponseWriter, activity IncomingActivity) {
	followerActor := activity.Actor

	// The object should be our actor ID (who they want to follow)
	var targetActor string
	switch obj := activity.Object.(type) {
	case string:
		targetActor = obj
	case map[string]interface{}:
		if id, ok := obj["id"].(string); ok {
			targetActor = id
		}
	}

	// Verify they're trying to follow us
	if targetActor != a.ID() {
		log.Printf("Follow target %s doesn't match our ID %s", targetActor, a.ID())
		w.WriteHeader(http.StatusAccepted)
		return
	}

	log.Printf("Processing Follow request from %s", followerActor)

	// Save the follower
	if err := a.saveFollower(followerActor); err != nil {
		log.Printf("Failed to save follower: %v", err)
	}

	// Send Accept asynchronously (don't block the response)
	go func() {
		if err := a.sendAccept(activity); err != nil {
			log.Printf("Failed to send Accept to %s: %v", followerActor, err)
		} else {
			log.Printf("Sent Accept to %s", followerActor)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

// handleUndo processes an Undo activity (typically Undo Follow = unfollow)
func (a *Actor) handleUndo(w http.ResponseWriter, activity IncomingActivity) {
	// Check what's being undone
	var undoneType string
	switch obj := activity.Object.(type) {
	case map[string]interface{}:
		if t, ok := obj["type"].(string); ok {
			undoneType = t
		}
	}

	if undoneType == "Follow" {
		log.Printf("Processing Unfollow from %s", activity.Actor)
		if err := a.removeFollower(activity.Actor); err != nil {
			log.Printf("Failed to remove follower: %v", err)
		}
	} else {
		log.Printf("Received Undo for %s from %s", undoneType, activity.Actor)
	}

	w.WriteHeader(http.StatusAccepted)
}

// sendAccept sends an Accept activity in response to a Follow
func (a *Actor) sendAccept(followActivity IncomingActivity) error {
	// Get the follower's inbox
	inbox, err := a.FetchActorInbox(followActivity.Actor)
	if err != nil {
		return fmt.Errorf("failed to fetch follower inbox: %w", err)
	}

	// Create Accept activity
	accept := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       fmt.Sprintf("%s/accepts/%d", a.ID(), time.Now().UnixNano()),
		"type":     "Accept",
		"actor":    a.ID(),
		"object":   followActivity, // Include the original Follow activity
	}

	body, err := json.Marshal(accept)
	if err != nil {
		return fmt.Errorf("failed to marshal Accept: %w", err)
	}

	log.Printf("Sending Accept to inbox: %s", inbox)

	resp, err := a.SignedPost(inbox, body)
	if err != nil {
		return fmt.Errorf("failed to POST Accept: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Accept delivered successfully (status %d)", resp.StatusCode)
		return nil
	}

	return fmt.Errorf("Accept delivery failed with status %d: %s", resp.StatusCode, string(respBody))
}

// saveFollower saves a follower to the followers list
func (a *Actor) saveFollower(followerID string) error {
	followersPath := a.followersFilePath()

	// Ensure directory exists
	dir := filepath.Dir(followersPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Load existing followers
	followers := a.loadFollowers()

	// Check if already following
	for _, f := range followers {
		if f == followerID {
			return nil // Already a follower
		}
	}

	// Add new follower
	followers = append(followers, followerID)

	// Save back
	data, err := json.MarshalIndent(followers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(followersPath, data, 0644)
}

// removeFollower removes a follower from the followers list
func (a *Actor) removeFollower(followerID string) error {
	followersPath := a.followersFilePath()

	followers := a.loadFollowers()

	// Filter out the unfollower
	var newFollowers []string
	for _, f := range followers {
		if f != followerID {
			newFollowers = append(newFollowers, f)
		}
	}

	if len(newFollowers) == len(followers) {
		return nil // Wasn't a follower
	}

	data, err := json.MarshalIndent(newFollowers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(followersPath, data, 0644)
}

// loadFollowers loads the list of followers from disk
func (a *Actor) loadFollowers() []string {
	data, err := os.ReadFile(a.followersFilePath())
	if err != nil {
		return []string{}
	}

	var followers []string
	if err := json.Unmarshal(data, &followers); err != nil {
		return []string{}
	}

	return followers
}

// followersFilePath returns the path to the followers file
func (a *Actor) followersFilePath() string {
	dir := filepath.Dir(a.config.KeyPath)
	return filepath.Join(dir, "followers.json")
}

// GetFollowersCount returns the number of followers
func (a *Actor) GetFollowersCount() int {
	return len(a.loadFollowers())
}

// HandleInboxLegacy handles requests to the legacy write.as/writefreely inbox path.
// POST /api/collections/{username}/inbox
func (a *Actor) HandleInboxLegacy(w http.ResponseWriter, r *http.Request, username string) {
	// Forward to the standard inbox handler
	a.HandleInbox(w, r, username)
}

// truncateSignature truncates a signature header for logging
func truncateSignature(sig string) string {
	if len(sig) > 100 {
		return sig[:100] + "..."
	}
	return sig
}

// extractKeyID extracts the keyId from a Signature header
func extractKeyID(sig string) string {
	// Look for keyId="..."
	start := strings.Index(sig, `keyId="`)
	if start == -1 {
		return ""
	}
	start += 7
	end := strings.Index(sig[start:], `"`)
	if end == -1 {
		return ""
	}
	return sig[start : start+end]
}
