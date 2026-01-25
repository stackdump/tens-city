package activitypub

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// OrderedCollection represents an ActivityPub OrderedCollection
type OrderedCollection struct {
	Context      string        `json:"@context"`
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	TotalItems   int           `json:"totalItems"`
	OrderedItems []interface{} `json:"orderedItems,omitempty"`
	First        string        `json:"first,omitempty"`
}

// Activity represents a generic ActivityPub activity
type Activity struct {
	Context   string      `json:"@context,omitempty"`
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Actor     string      `json:"actor"`
	Published string      `json:"published,omitempty"`
	To        []string    `json:"to,omitempty"`
	Cc        []string    `json:"cc,omitempty"`
	Object    interface{} `json:"object"`
}

// Article represents an ActivityPub Article object (for blog posts)
type Article struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	AttributedTo string        `json:"attributedTo"`
	Name         string        `json:"name"`
	Content      string        `json:"content"`
	Summary      string        `json:"summary,omitempty"`
	Published    string        `json:"published"`
	Updated      string        `json:"updated,omitempty"`
	URL          string        `json:"url"`
	To           []string      `json:"to,omitempty"`
	Cc           []string      `json:"cc,omitempty"`
	Tag          []Tag         `json:"tag,omitempty"`
	Image        *ArticleImage `json:"image,omitempty"`
}

// ArticleImage represents an image attachment for an Article
type ArticleImage struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType,omitempty"`
	URL       string `json:"url"`
	Name      string `json:"name,omitempty"`
}

// Tag represents a hashtag or mention
type Tag struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Href string `json:"href,omitempty"`
}

// BlogPost represents a blog post for conversion to ActivityPub
type BlogPost struct {
	ID          string    // Unique identifier (URL)
	Slug        string    // URL slug
	Title       string    // Post title
	Description string    // Short description/summary
	Content     string    // HTML content
	Published   time.Time // Publication date
	Updated     time.Time // Last modified date (optional)
	Tags        []string  // Post tags
	AuthorName  string    // Author display name
	Image       string    // Featured image URL (optional)
}

// HandleOutbox serves the actor's outbox (published activities).
// GET /users/{username}/outbox
func (a *Actor) HandleOutbox(w http.ResponseWriter, r *http.Request, username string, posts []BlogPost) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if username != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Build activities from blog posts
	activities := make([]interface{}, 0, len(posts))
	for _, post := range posts {
		activity := a.postToActivity(post)
		activities = append(activities, activity)
	}

	response := OrderedCollection{
		Context:      "https://www.w3.org/ns/activitystreams",
		ID:           a.OutboxURL(),
		Type:         "OrderedCollection",
		TotalItems:   len(activities),
		OrderedItems: activities,
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=300")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleFollowers serves the actor's followers collection.
// GET /users/{username}/followers
func (a *Actor) HandleFollowers(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if username != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Load followers from storage
	followers := a.loadFollowers()

	// Convert to interface slice for JSON
	items := make([]interface{}, len(followers))
	for i, f := range followers {
		items[i] = f
	}

	response := OrderedCollection{
		Context:      "https://www.w3.org/ns/activitystreams",
		ID:           a.FollowersURL(),
		Type:         "OrderedCollection",
		TotalItems:   len(followers),
		OrderedItems: items,
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=300")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleFollowing serves the actor's following collection.
// GET /users/{username}/following
func (a *Actor) HandleFollowing(w http.ResponseWriter, r *http.Request, username string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if username != a.config.Username {
		http.NotFound(w, r)
		return
	}

	// Return empty collection (this blog doesn't follow anyone)
	response := OrderedCollection{
		Context:      "https://www.w3.org/ns/activitystreams",
		ID:           a.FollowingURL(),
		Type:         "OrderedCollection",
		TotalItems:   0,
		OrderedItems: []interface{}{},
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	w.Header().Set("Cache-Control", "max-age=300")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// postToActivity converts a blog post to an ActivityPub Create activity
func (a *Actor) postToActivity(post BlogPost) Activity {
	// Build tags
	var tags []Tag
	for _, tag := range post.Tags {
		// Normalize tag to hashtag format
		tagName := tag
		if !strings.HasPrefix(tagName, "#") {
			tagName = "#" + tagName
		}
		tags = append(tags, Tag{
			Type: "Hashtag",
			Name: tagName,
		})
	}

	// Build article
	article := Article{
		ID:           post.ID,
		Type:         "Article",
		AttributedTo: a.ID(),
		Name:         post.Title,
		Content:      post.Content,
		Summary:      post.Description,
		Published:    post.Published.Format(time.RFC3339),
		URL:          post.ID,
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:           []string{a.FollowersURL()},
		Tag:          tags,
	}

	if !post.Updated.IsZero() && post.Updated.After(post.Published) {
		article.Updated = post.Updated.Format(time.RFC3339)
	}

	// Add featured image if available
	if post.Image != "" {
		mediaType := "image/jpeg" // default
		if strings.HasSuffix(post.Image, ".png") {
			mediaType = "image/png"
		} else if strings.HasSuffix(post.Image, ".svg") {
			mediaType = "image/svg+xml"
		} else if strings.HasSuffix(post.Image, ".gif") {
			mediaType = "image/gif"
		} else if strings.HasSuffix(post.Image, ".webp") {
			mediaType = "image/webp"
		}
		article.Image = &ArticleImage{
			Type:      "Image",
			MediaType: mediaType,
			URL:       post.Image,
		}
	}

	// Build Create activity
	return Activity{
		ID:        post.ID + "#create",
		Type:      "Create",
		Actor:     a.ID(),
		Published: post.Published.Format(time.RFC3339),
		To:        []string{"https://www.w3.org/ns/activitystreams#Public"},
		Cc:        []string{a.FollowersURL()},
		Object:    article,
	}
}
