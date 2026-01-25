package activitypub

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ActorResponse represents an ActivityPub Actor object
type ActorResponse struct {
	Context           interface{}       `json:"@context"`
	ID                string            `json:"id"`
	Type              string            `json:"type"`
	PreferredUsername string            `json:"preferredUsername"`
	Name              string            `json:"name"`
	Summary           string            `json:"summary,omitempty"`
	URL               string            `json:"url,omitempty"`
	Inbox             string            `json:"inbox"`
	Outbox            string            `json:"outbox"`
	Followers         string            `json:"followers"`
	Following         string            `json:"following"`
	PublicKey         ActorPublicKey    `json:"publicKey"`
	Icon              *ActorImage       `json:"icon,omitempty"`
	Endpoints         *ActorEndpoints   `json:"endpoints,omitempty"`
}

// ActorPublicKey represents the actor's public key for HTTP signatures
type ActorPublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

// ActorImage represents an image attachment (icon, image)
type ActorImage struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType,omitempty"`
	URL       string `json:"url"`
}

// ActorEndpoints contains additional endpoints
type ActorEndpoints struct {
	SharedInbox string `json:"sharedInbox,omitempty"`
}

// HandleActor serves the ActivityPub actor profile.
// GET /users/{username}
// Content negotiation: returns JSON-LD for ActivityPub clients, HTML redirect otherwise
func (a *Actor) HandleActor(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if this is our actor
	if username != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Check Accept header for content negotiation
	accept := r.Header.Get("Accept")
	wantsActivityPub := strings.Contains(accept, "application/activity+json") ||
		strings.Contains(accept, "application/ld+json")

	if !wantsActivityPub {
		// Redirect to HTML profile page
		if a.config.ProfileURL != "" {
			http.Redirect(w, r, a.config.ProfileURL, http.StatusSeeOther)
			return
		}
		// Fall through to serve JSON if no profile URL configured
	}

	// Build ActivityPub actor response
	response := ActorResponse{
		Context: []interface{}{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		ID:                a.ID(),
		Type:              "Person",
		PreferredUsername: a.config.Username,
		Name:              a.config.DisplayName,
		Summary:           a.config.Summary,
		URL:               a.config.ProfileURL,
		Inbox:             a.InboxURL(),
		Outbox:            a.OutboxURL(),
		Followers:         a.FollowersURL(),
		Following:         a.FollowingURL(),
		PublicKey: ActorPublicKey{
			ID:           a.KeyID(),
			Owner:        a.ID(),
			PublicKeyPem: a.PublicKeyPEM(),
		},
	}

	// Add icon if configured
	if a.config.IconURL != "" {
		mediaType := "image/png"
		if strings.HasSuffix(a.config.IconURL, ".svg") {
			mediaType = "image/svg+xml"
		} else if strings.HasSuffix(a.config.IconURL, ".jpg") || strings.HasSuffix(a.config.IconURL, ".jpeg") {
			mediaType = "image/jpeg"
		}
		response.Icon = &ActorImage{
			Type:      "Image",
			MediaType: mediaType,
			URL:       a.config.IconURL,
		}
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=1800")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleActorLegacy handles requests to the legacy write.as/writefreely path.
// GET /api/collections/{username}
// Redirects to the canonical /users/{username} path
func (a *Actor) HandleActorLegacy(w http.ResponseWriter, r *http.Request, username string) {
	if username != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Redirect to canonical path
	http.Redirect(w, r, a.ID(), http.StatusMovedPermanently)
}
