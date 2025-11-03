# tens.city

## Static Blog Hosting

Tens City is a static blog hosting system that serves markdown blog posts with content-addressable storage. Content is managed through markdown files with YAML frontmatter, providing a simple file-based workflow.

### Features
- **Markdown with frontmatter** - Write posts as `.md` files with YAML metadata
- **Content-addressable storage** - Immutable JSON-LD objects identified by CID
- **Schema.org JSON-LD** - Automatic structured data generation
- **Server-side rendering** - HTML generation from markdown
- **RSS feeds** - Automatic feed generation per author
- **Static file serving** - Simple, fast blog hosting
- **No authentication required** - Pure static blog viewer

For details on the markdown documentation system, see [docs/markdown-docs.md](docs/markdown-docs.md).

## Documentation System

Tens City includes a comprehensive markdown documentation system with YAML frontmatter support and automatic schema.org JSON-LD mapping.

### Features
- **Markdown with YAML frontmatter** - Documents as source of truth
- **Schema.org JSON-LD** - Automatic mapping to structured data
- **Server-side rendering** - HTML generation with sanitization
- **Caching** - ETag and Last-Modified support for performance
- **Content negotiation** - Serve HTML or JSON-LD based on Accept header
- **Collection index** - `/posts/index.jsonld` with all published documents
- **Draft support** - Hide work-in-progress documents
- **RSS feeds** - Automatic feed generation per author

See [docs/markdown-docs.md](docs/markdown-docs.md) for complete documentation.

### Quick Start
```bash
# Start server
./webserver -addr :8080 -store data -content content/posts -base-url http://localhost:8080

# Access blog
# Home: http://localhost:8080 (redirects to latest post)
# Posts list: http://localhost:8080/posts
# Index: http://localhost:8080/posts/index.jsonld
# Specific post: http://localhost:8080/posts/your-slug
# JSON-LD: http://localhost:8080/posts/your-slug.jsonld
```

## 1. Ethos: tent city
- Not skyscrapers; actual encampments.
- Anyone can stake a corner — no permission required.
- Minimal structure that belongs to you — dignity-through-shelter even when janky.
- No HOA, no governance tokens: just a tarp and a hook to keep your stuff dry.

## 2. Basic shelter (software terms)
The "lean-to of sticks" is the absolute minimum runtime that lets something simply live.

Concrete guarantees:
- Each person/object gets a directory.
- The directory is static (only files).
- It is addressable (stable URL / CID).
- Served straight off disk — no app server trying to own content.

Maps to:
- Filesystem-based
- Static web hosting
- The city is literally a filesystem tree

## 3. Principles (short)
- Provide stable, non-transient spots.
- Keep the runtime minimal and non-opinionated.
- Serve static files directly from disk with a generic viewer.

## 4. CLI Tools

### webserver - HTTP server for static blog hosting
Runs a web server that serves markdown blog posts as HTML and JSON-LD using filesystem storage.

```bash
# Start with filesystem storage
./webserver -addr :8080 -store data -content content/posts -base-url http://localhost:8080
```

**Features:**
- **Blog hosting**: Serves markdown posts from the content directory
- **Read-only routes**:
  - `GET /` - Blog home page (redirects to latest post)
  - `GET /posts` - List all posts
  - `GET /posts/index.jsonld` - JSON-LD index of all posts
  - `GET /posts/{slug}` - Individual post as HTML
  - `GET /posts/{slug}.jsonld` - Individual post as JSON-LD
  - `GET /o/{cid}` - Retrieve object by CID
  - `GET /o/{cid}/markdown` - Get markdown source by CID
  - `GET /u/{user}/g/{slug}/latest` - Get latest CID for user's gist
  - `GET /u/{user}/g/{slug}/_history` - Get history for user's gist
  - `GET /u/{user}/posts.rss` - RSS feed for user's posts
- **Static file serving**: Serves the blog viewer from the public directory
- **Content-addressable storage**: All posts stored as immutable CID-addressed objects

**Options:**
- `-addr` - Server address (default: `:8080`)
- `-store` - Filesystem store directory (default: `data`)
- `-content` - Content directory for markdown posts (default: `content/posts`)
- `-base-url` - Base URL for the server (default: `http://localhost:8080`)

### seal - Create sealed JSON-LD objects
Seals JSON-LD documents using URDNA2015 canonicalization and computes CIDv1 identifiers.

```bash
./seal -in examples/petrinet.jsonld -store data
```

See `cmd/seal/main.go` for full options including signing with Ethereum keys.

### keygen - Generate Ethereum keys
Generate or import Ethereum keystore files for signing.

```bash
./keygen -out my-key.keystore -pass mypassword
```

## 5. Development with Makefile

The project includes a Makefile to streamline common development tasks:

```bash
# Build all binaries
make build

# Run tests
make test

# Format, vet, test, and build (full dev workflow)
make dev

# Clean build artifacts
make clean

# See all available targets
make help
```

**Common Makefile targets:**
- `make build` - Build all binaries (seal, keygen, webserver)
- `make test` - Run all tests
- `make fmt` - Format Go code
- `make vet` - Run go vet
- `make check` - Format, vet, and test
- `make clean` - Clean build artifacts
- `make run-webserver` - Build and run webserver
- `make run-seal` - Build and run seal with example

## 6. Quick Start

1. Build the tools:
   ```bash
   # Using Makefile (recommended)
   make build
   
   # Or manually with Go
   go build -o seal ./cmd/seal
   go build -o webserver ./cmd/webserver
   ```

2. Create markdown blog posts in `content/posts/`:
   ```bash
   mkdir -p content/posts
   # Create your first post as a .md file with YAML frontmatter
   # See docs/markdown-docs.md for frontmatter format
   ```

3. Start the web server:
   ```bash
   ./webserver -addr :8080 -store data -content content/posts -base-url http://localhost:8080
   ```

4. Open your browser to `http://localhost:8080`

The home page will automatically redirect to your latest blog post.

See [docs/markdown-docs.md](docs/markdown-docs.md) for details on writing blog posts with markdown and frontmatter.
