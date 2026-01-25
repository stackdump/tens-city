# tens.city

## A Minimal Blog Platform 🏕️

Tens City https://tens.city is a simple, elegant blog platform built on markdown files and content-addressable storage. Write in markdown, publish instantly—no database, no complexity.

### Features

- **Markdown with frontmatter** - Write posts as `.md` files with YAML metadata
- **ActivityPub federation** - Follow your blog from Mastodon, Misskey, and other fediverse platforms
- **Beautiful, responsive design** - Modern card-based layout
- **Schema.org JSON-LD** - Automatic structured data generation for SEO
- **Server-side rendering** - Fast HTML generation from markdown
- **RSS feeds** - Automatic feed generation per author and site-wide
- **Static file serving** - Simple, secure blog hosting
- **No database** - All state stored as JSON files

### Quick Start

```bash
# Build the webserver
make build

# Start server
./webserver -addr :8080 -store data -content content/posts -base-url http://localhost:8080

# Access blog
# Home: http://localhost:8080
# Posts list: http://localhost:8080/posts
# Specific post: http://localhost:8080/posts/your-slug
# JSON-LD: http://localhost:8080/posts/your-slug.jsonld
# Site-wide RSS: http://localhost:8080/posts.rss
# Author RSS: http://localhost:8080/u/{author}/posts.rss
```

## Configuration

### Command-Line Flags

The webserver supports the following configuration options:

- `-addr` - Server address (default: `:8080`)
- `-store` - Filesystem store directory (default: `data`)
- `-content` - Content directory for markdown blog posts (default: `content/posts`)
- `-base-url` - Base URL for the server (default: `http://localhost:8080`)
- `-index-limit` - Maximum number of posts to show in index (default: `20`, use `0` for no limit)
- `-jsonl` - Use JSONL (JSON Lines) format for logging (default: `false`)
- `-log-headers` - Log incoming request headers, useful for debugging RSS http/https behavior (default: `false`)

Example:
```bash
./webserver -addr :8080 -store data -content content/posts -base-url http://localhost:8080 -index-limit 10
```

Example with JSONL logging and header logging enabled:
```bash
./webserver -addr :8080 -store data -content content/posts -jsonl -log-headers
```

### Environment Variables

- `INDEX_LIMIT` - Maximum number of posts to show in index (overrides the `-index-limit` flag default)

Example:
```bash
INDEX_LIMIT=10 ./webserver -addr :8080 -store data -content content/posts
```

### Logging

The webserver supports two logging formats:

#### Text Logging (Default)

Traditional text-based logging for easy reading:

```bash
./webserver -addr :8080 -store data -content content/posts
```

Output:
```
2025/11/05 14:17:42 INFO: Using filesystem storage: data
2025/11/05 14:17:42 INFO: Starting server on :8080
2025/11/05 14:17:45 GET / - 200 - 8.909212ms
```

#### JSONL Logging

Structured JSON Lines logging for parsing and analysis:

```bash
./webserver -addr :8080 -store data -content content/posts -jsonl
```

Output:
```json
{"timestamp":"2025-11-05T14:17:56Z","level":"info","message":"Using filesystem storage: data"}
{"timestamp":"2025-11-05T14:18:16Z","level":"info","method":"GET","path":"/","status":200,"duration":"8.909212ms"}
```

#### Header Logging

Enable request header logging to debug proxy configurations (especially useful for RSS http/https behavior):

```bash
./webserver -addr :8080 -store data -content content/posts -jsonl -log-headers
```

Output includes headers:
```json
{"timestamp":"2025-11-05T14:18:42Z","level":"debug","message":"Headers for GET /posts.rss","method":"GET","path":"/posts.rss","headers":{"X-Forwarded-Proto":"https","X-Forwarded-Host":"blog.example.com"}}
{"timestamp":"2025-11-05T14:18:42Z","level":"info","method":"GET","path":"/posts.rss","status":200,"duration":"8.019868ms"}
```

The header logging is particularly useful when debugging RSS feed issues related to protocol detection (http vs https) when running behind a proxy or load balancer.

**Security Note**: Header logging may expose sensitive data (e.g., Authorization headers). Only enable `-log-headers` in development environments or when debugging specific issues. Never enable it in production with public logs.

### Post Ordering

Posts in the index (both at `/posts/index.jsonld` and `/posts`) are automatically sorted by:
1. **Date Published** (descending - newest first)
2. **Title** (ascending - alphabetically for posts with the same date)

This ensures your latest content appears first while maintaining a consistent, predictable order.

## Writing Posts

Create markdown files in `content/posts/` with YAML frontmatter:

```markdown
---
title: My First Post
description: A great blog post
datePublished: 2025-11-03T00:00:00Z
author:
  name: Your Name
  type: Person
tags:
  - example
  - blog
slug: my-first-post
---

# My First Post

Your content goes here!
```

## Customizing the Index Page

You can customize the index page (homepage) by editing the `index.md` file in the `content/` directory (not in `content/posts/`).

**Note**: The first time you access the homepage, if `index.md` doesn't exist, it will be automatically created with default values. This helps you discover how to customize your blog.

### index.md Example

```markdown
---
title: My Custom Blog
description: A personal blog about technology and life
icon: 🚀
lang: en
---

Welcome to my custom blog! This is a space where I share my thoughts and experiences.
```

### Supported Fields

- `title` - The title shown in the browser tab and hero section (default: "Tens City - A Minimal Blog Platform")
- `description` - Meta description for SEO and the hero subtitle (default: "Simple, elegant blog platform built on content-addressable storage")
- `icon` - The emoji/icon shown in the hero section (default: "🏕️")
- `lang` - The language code for the page (default: "en")
- `sameAs` - Array of URLs representing the same entity (e.g., social media profiles, other websites) - used in JSON-LD for SEO

The content body (after the frontmatter) will be displayed as a message in the hero section.

### Example with Social Media Links

```markdown
---
title: My Blog
description: Thoughts on technology and design
icon: 🚀
lang: en
sameAs:
  - https://github.com/yourusername
  - https://twitter.com/yourusername
  - https://mastodon.social/@yourusername
---

Welcome! Follow me on social media for updates.
```

The `sameAs` URLs will be included in the JSON-LD structured data on your homepage, improving SEO and helping search engines understand your online presence.

## Philosophy: Tent City

- Not skyscrapers; actual encampments
- Anyone can stake a corner — no permission required
- Minimal structure that belongs to you
- No HOA, no governance tokens: just a tarp and a hook to keep your stuff dry

## Basic Shelter (Software Terms)

The "lean-to of sticks" is the absolute minimum runtime that lets something simply live.

Concrete guarantees:
- Each post gets a stable URL
- Content is static (only files)
- It is addressable (stable URL)
- Served straight off disk — no app server trying to own content

Maps to:
- Filesystem-based
- Static web hosting
- The city is literally a filesystem tree

## Principles

- Provide stable, non-transient spots
- Keep the runtime minimal and non-opinionated
- Serve static files directly from disk with a generic viewer

## Development

### Build and Run

```bash
# Build the webserver
make build

# Run tests
make test

# Format, vet, test, and build
make dev

# Start the server
./webserver -addr :8080 -store data -content content/posts -base-url http://localhost:8080
```

### Makefile Targets

- `make build` - Build webserver binary
- `make test` - Run all tests
- `make fmt` - Format Go code
- `make vet` - Run go vet
- `make check` - Format, vet, and test
- `make clean` - Clean build artifacts
- `make run-webserver` - Build and run webserver

### Server Options

- `-addr` - Server address (default: `:8080`)
- `-store` - Filesystem store directory (default: `data`)
- `-content` - Content directory for markdown posts (default: `content/posts`)
- `-base-url` - Base URL for the server (default: `http://localhost:8080`)

## Routes

### Content Routes
- `GET /` - Blog home page with post grid
- `GET /posts` - List all posts
- `GET /posts/index.jsonld` - JSON-LD index of all posts
- `GET /posts/{slug}` - Individual post as HTML
- `GET /posts/{slug}.jsonld` - Individual post as JSON-LD
- `GET /search` - Client-side search page for finding posts
- `GET /tags` - Tag cloud showing all tags
- `GET /tags/{tag}` - Posts with specific tag

### RSS Feeds
- `GET /posts.rss` - Site-wide RSS feed (all posts)
- `GET /feed.xml` - Alias for site-wide RSS feed
- `GET /rss.xml` - Alias for site-wide RSS feed
- `GET /u/{user}/posts.rss` - RSS feed for user's posts
- `GET /rss` - HTML page listing all available RSS feeds

### SEO & Discovery
- `GET /robots.txt` - Robots.txt for search engine crawlers with sitemap reference
- `GET /sitemap.xml` - XML sitemap of all pages, posts, and tags
- `GET /.well-known/security.txt` - Security contact information

### Features
- **RSS Autodiscovery** - `<link rel="alternate">` tag automatically added to homepage
- **JSON-LD with sameAs** - Add social media profiles to `content/index.md` for better SEO
- **Multiple RSS URLs** - RSS feed available at `/posts.rss`, `/feed.xml`, and `/rss.xml`
- **XML Sitemap** - Automatically generated sitemap with all pages, posts, and tags
- **Client-Side Search** - Fast search functionality for finding posts by title, description, or tags

## ActivityPub Federation

tens-city can federate with Mastodon and other ActivityPub platforms. Your blog becomes a fediverse actor that people can follow.

### Enable Federation

Set environment variables and start the server:

```bash
export ACTIVITYPUB_DOMAIN=blog.example.com
export ACTIVITYPUB_USERNAME=blogger
export ACTIVITYPUB_PUBLISH_TOKEN=$(openssl rand -base64 32)

./webserver -addr :8080 -content content/posts
```

Users can then follow `@blogger@blog.example.com` from any Mastodon instance.

### Push Posts to Followers

```bash
curl -X POST "https://blog.example.com/publish?token=$ACTIVITYPUB_PUBLISH_TOKEN"
```

New posts are delivered to all followers. Already-published posts are tracked and skipped.

See [ACTIVITYPUB.md](ACTIVITYPUB.md) for full configuration options.

## License

See LICENSE file for details.
