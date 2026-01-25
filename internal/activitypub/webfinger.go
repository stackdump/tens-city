package activitypub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WebFingerResponse represents a WebFinger JRD response
type WebFingerResponse struct {
	Subject string          `json:"subject"`
	Aliases []string        `json:"aliases,omitempty"`
	Links   []WebFingerLink `json:"links"`
}

// WebFingerLink represents a link in the WebFinger response
type WebFingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

// HandleWebFinger handles WebFinger requests for actor discovery.
// GET /.well-known/webfinger?resource=acct:username@domain
func (a *Actor) HandleWebFinger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resource := r.URL.Query().Get("resource")
	if resource == "" {
		http.Error(w, "Missing resource parameter", http.StatusBadRequest)
		return
	}

	// Parse the resource - expect acct:user@domain or https://domain/users/user
	var requestedUser, requestedDomain string

	if strings.HasPrefix(resource, "acct:") {
		// Format: acct:user@domain
		parts := strings.SplitN(strings.TrimPrefix(resource, "acct:"), "@", 2)
		if len(parts) != 2 {
			http.Error(w, "Invalid resource format", http.StatusBadRequest)
			return
		}
		requestedUser = parts[0]
		requestedDomain = parts[1]
	} else if strings.HasPrefix(resource, "https://") {
		// Format: https://domain/users/user
		resource = strings.TrimPrefix(resource, "https://")
		parts := strings.SplitN(resource, "/", 2)
		if len(parts) < 2 {
			http.Error(w, "Invalid resource format", http.StatusBadRequest)
			return
		}
		requestedDomain = parts[0]
		// Extract user from path like "users/username"
		pathParts := strings.Split(parts[1], "/")
		if len(pathParts) >= 2 && pathParts[0] == "users" {
			requestedUser = pathParts[1]
		}
	} else {
		http.Error(w, "Invalid resource format", http.StatusBadRequest)
		return
	}

	// Check if this is our actor
	if requestedDomain != a.config.Domain {
		http.NotFound(w, r)
		return
	}
	if requestedUser != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Build response
	response := WebFingerResponse{
		Subject: fmt.Sprintf("acct:%s@%s", a.config.Username, a.config.Domain),
		Aliases: []string{
			a.ID(),
		},
		Links: []WebFingerLink{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: a.ID(),
			},
			{
				Rel:  "http://webfinger.net/rel/profile-page",
				Type: "text/html",
				Href: a.config.ProfileURL,
			},
		},
	}

	// Add profile URL alias if different from actor ID
	if a.config.ProfileURL != "" && a.config.ProfileURL != a.ID() {
		response.Aliases = append(response.Aliases, a.config.ProfileURL)
	}

	w.Header().Set("Content-Type", "application/jrd+json; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
