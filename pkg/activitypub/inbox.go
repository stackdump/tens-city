package activitypub

import (
	"io"
	"log"
	"net/http"
)

// HandleInbox handles incoming ActivityPub activities.
// POST /users/{username}/inbox
// Phase 1: Log and acknowledge. Phase 2 will add signature verification and processing.
func (a *Actor) HandleInbox(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if username != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Read the activity body (for logging in Phase 1)
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 1MB limit
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log the incoming activity (Phase 1: just acknowledge)
	// Phase 2 will add HTTP signature verification and activity processing
	log.Printf("ActivityPub inbox received: %d bytes from %s", len(body), r.Header.Get("Signature"))

	// Return 202 Accepted to indicate we received the activity
	// This tells the sender we got it, even if we don't process it yet
	w.WriteHeader(http.StatusAccepted)
}

// HandleInboxLegacy handles requests to the legacy write.as/writefreely inbox path.
// POST /api/collections/{username}/inbox
func (a *Actor) HandleInboxLegacy(w http.ResponseWriter, r *http.Request, username string) {
	// Forward to the standard inbox handler
	a.HandleInbox(w, r, username)
}
