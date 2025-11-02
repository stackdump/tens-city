# Markdown Documentation System

The Tens City platform now includes a comprehensive markdown documentation system with YAML frontmatter support, JSON-LD mapping, and server-side rendering.

## Features

### 1. YAML Frontmatter
Documents use YAML frontmatter to define metadata:

```markdown
---
title: Getting Started
description: A guide to getting started
datePublished: 2025-11-02T00:00:00Z
dateModified: 2025-11-02T00:00:00Z
author:
  name: Tens City Team
  type: Organization
  url: https://tens.city
tags:
  - getting-started
  - tutorial
collection: guides
lang: en
slug: getting-started
draft: false
---

# Content starts here
```

### 2. Schema.org JSON-LD Mapping
Frontmatter is automatically converted to schema.org JSON-LD:
- Documents are mapped to `Article` type
- Authors are mapped to `Person` or `Organization`
- Collections are mapped to `CollectionPage` with `isPartOf` relationships
- Full support for schema.org properties: `headline`, `description`, `datePublished`, `dateModified`, `author`, `keywords`, etc.

### 3. Server Routes

#### Document List
- **GET /docs** - HTML page listing all published documents (excludes drafts)
- Returns formatted HTML with document cards showing title, description, and publication date

#### Document Index (JSON-LD)
- **GET /docs/index.jsonld** - Collection index as JSON-LD
- Returns a `CollectionPage` with `itemListElement` array
- Includes all published documents (drafts excluded)
- Example:
  ```json
  {
    "@context": "https://schema.org",
    "@type": "CollectionPage",
    "name": "Documentation Index",
    "numberOfItems": 3,
    "itemListElement": [...]
  }
  ```

#### Single Document (HTML)
- **GET /docs/:slug** - Rendered HTML document
- Includes embedded JSON-LD in `<script type="application/ld+json">` tag
- Sanitized HTML output (XSS protection)
- Interactive JSON-LD preview toggle
- Navigation links to docs list and JSON-LD endpoint

#### Single Document (JSON-LD)
- **GET /docs/:slug.jsonld** - Pure JSON-LD representation
- Returns only the structured data

#### Content Negotiation
- Supports `Accept: application/ld+json` header
- Returns JSON-LD when requested via Accept header

### 4. Caching & Performance
- **ETag support** - MD5 hash of content for cache validation
- **Last-Modified headers** - Based on file modification time
- **HTTP 304 responses** - When content hasn't changed (If-None-Match)
- **In-memory cache** - Documents cached until file changes

### 5. Security
- **HTML Sanitization** - Using bluemonday to prevent XSS
- **Safe markdown rendering** - Only safe HTML tags allowed
- **Email privacy** - Author emails not exposed in JSON-LD
- **Draft protection** - Draft documents (draft: true) not publicly accessible

### 6. Frontmatter Schema

See `schemas/frontmatter.schema.json` for the complete JSON Schema.

**Required fields:**
- `title` - Document title
- `datePublished` - ISO 8601 date-time
- `author` - Person or Organization object
- `lang` - Language code (e.g., "en", "en-US")

**Optional fields:**
- `description` - Short summary
- `dateModified` - ISO 8601 date-time
- `tags` - Array of tag strings
- `collection` - Collection name
- `draft` - Boolean (default: false)
- `slug` - URL-friendly identifier (auto-generated if omitted)
- `image` - Featured image URL
- `keywords` - SEO keywords array

**Author object:**
```yaml
author:
  name: Full Name
  email: email@example.com  # Protected, not exposed publicly
  url: https://example.com
  type: Person  # or Organization
  sameAs:
    - https://twitter.com/username
    - https://github.com/username
```

## Usage

### Starting the Server

```bash
./webserver \
  -addr :8080 \
  -store data \
  -content content/docs \
  -base-url http://localhost:8080
```

**New flags:**
- `-content` - Directory containing markdown documents (default: "content/docs")
- `-base-url` - Base URL for generating absolute URLs in JSON-LD (default: "http://localhost:8080")

### Creating Documents

1. Create a markdown file in `content/docs/`:

```markdown
---
title: My New Document
description: A description
datePublished: 2025-11-02T00:00:00Z
author:
  name: Your Name
  type: Person
lang: en
---

# My Document

Content goes here...
```

2. The slug is auto-generated from the filename, or you can specify it in frontmatter
3. Access at `/docs/your-slug`

### Draft Documents

Set `draft: true` in frontmatter to hide from public index and prevent access:

```yaml
---
title: Work in Progress
draft: true
...
---
```

## API Examples

### Get document list (HTML)
```bash
curl http://localhost:8080/docs
```

### Get collection index (JSON-LD)
```bash
curl http://localhost:8080/docs/index.jsonld
```

### Get single document (HTML)
```bash
curl http://localhost:8080/docs/getting-started
```

### Get single document (JSON-LD)
```bash
curl http://localhost:8080/docs/getting-started.jsonld
```

### Content negotiation
```bash
curl -H "Accept: application/ld+json" http://localhost:8080/docs/getting-started
```

### Check cache headers
```bash
curl -I http://localhost:8080/docs/getting-started
# Returns: ETag, Last-Modified
```

### Validate cache
```bash
curl -I -H 'If-None-Match: "etag-value"' http://localhost:8080/docs/getting-started
# Returns: 304 Not Modified (if unchanged)
```

## Implementation Details

### Packages
- `internal/markdown` - Markdown parsing, frontmatter extraction, JSON-LD mapping
- `internal/docserver` - HTTP handlers, caching, routing
- `schemas/` - JSON Schema for frontmatter validation

### Dependencies
- `github.com/yuin/goldmark` - Markdown rendering with GitHub Flavored Markdown
- `gopkg.in/yaml.v3` - YAML frontmatter parsing
- `github.com/microcosm-cc/bluemonday` - HTML sanitization
- `github.com/xeipuuv/gojsonschema` - JSON Schema validation (future use)

### Markdown Rendering
- GitHub Flavored Markdown (GFM) support
- Tables, strikethrough, task lists
- Auto-generated heading IDs
- Code syntax highlighting (via class attributes)

### Future Enhancements
This implementation provides the backend foundation. Future work includes:
- Frontend editor UI (React/Vue/Svelte)
- GitHub API integration for saving
- Client-side frontmatter validation
- Search functionality
- File tree browser
- Real-time preview
- Version history UI
- Collaborative editing

## Testing

```bash
# Test markdown package
go test ./internal/markdown/

# Test all packages
go test ./...

# Run server tests
go test ./cmd/webserver/
```

## Example Documents

See `content/docs/` for example documents:
- `getting-started.md` - Comprehensive guide
- `authentication.md` - Authentication documentation
- `jsonld-script-tag.md` - JSON-LD embedding guide

Each demonstrates proper frontmatter structure and markdown formatting.
