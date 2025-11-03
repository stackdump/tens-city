# tens.city

## A Minimal Blog Platform 🏕️

Tens City is a simple, elegant blog platform built on markdown files and content-addressable storage. Write in markdown, publish instantly—no database, no complexity.

### Features

- **Markdown with frontmatter** - Write posts as `.md` files with YAML metadata
- **Beautiful, responsive design** - Modern card-based layout
- **Schema.org JSON-LD** - Automatic structured data generation for SEO
- **Server-side rendering** - Fast HTML generation from markdown
- **RSS feeds** - Automatic feed generation per author and site-wide
- **Static file serving** - Simple, secure blog hosting
- **No authentication required** - Pure static blog viewer

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

- `GET /` - Blog home page with post grid
- `GET /posts` - List all posts
- `GET /posts/index.jsonld` - JSON-LD index of all posts
- `GET /posts/{slug}` - Individual post as HTML
- `GET /posts/{slug}.jsonld` - Individual post as JSON-LD
- `GET /u/{user}/posts.rss` - RSS feed for user's posts

## License

See LICENSE file for details.
