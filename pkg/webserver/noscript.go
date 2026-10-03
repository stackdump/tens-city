package webserver

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// noscriptPostLimit is how many posts the no-JS fallback lists on the home page.
const noscriptPostLimit = 10

// postsPlaceholder is the element the home page script fills from /posts/index.jsonld.
const postsPlaceholder = `<div id="posts" class="loading"></div>`

// renderNoscriptPosts renders the latest posts from the collection index JSON-LD
// (the same document the home page script fetches from /posts/index.jsonld) as a
// <noscript> block, so the home page lists posts without JavaScript. It uses the
// script's card markup so a browser with CSS but no JS renders the same grid.
func renderNoscriptPosts(indexData []byte, limit int) string {
	var index struct {
		ItemListElement []struct {
			Item struct {
				Headline      string `json:"headline"`
				URL           string `json:"url"`
				Description   string `json:"description"`
				DatePublished string `json:"datePublished"`
			} `json:"item"`
		} `json:"itemListElement"`
	}
	if err := json.Unmarshal(indexData, &index); err != nil || len(index.ItemListElement) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n        <noscript>\n            <div class=\"post-grid\">\n")
	for i, el := range index.ItemListElement {
		if i >= limit {
			break
		}
		post := el.Item
		href := post.URL
		if parts := strings.SplitN(post.URL, "/posts/", 2); len(parts) == 2 {
			href = "/posts/" + parts[1]
		}
		title := post.Headline
		if title == "" {
			title = "Untitled"
		}
		b.WriteString("                <article class=\"post-card\"><div class=\"post-card-content\">\n")
		fmt.Fprintf(&b, "                    <h3><a href=\"%s\">%s</a></h3>\n", html.EscapeString(href), html.EscapeString(title))
		if t, err := time.Parse(time.RFC3339, post.DatePublished); err == nil {
			fmt.Fprintf(&b, "                    <div class=\"post-meta\"><div class=\"post-meta-item\">📅 <time datetime=\"%s\">%s</time></div></div>\n",
				html.EscapeString(post.DatePublished), t.Format("January 2, 2006"))
		}
		if post.Description != "" {
			fmt.Fprintf(&b, "                    <p class=\"post-description\">%s</p>\n", html.EscapeString(post.Description))
		}
		b.WriteString("                </div></article>\n")
	}
	b.WriteString("            </div>\n        </noscript>")
	return b.String()
}
