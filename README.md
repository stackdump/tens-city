# tens.city 

## Web Application

The web application is a single-page app for managing JSON-LD objects with GitHub authentication.

### Features
- Post new JSON-LD objects with ownership tracking
- ACE editor for JSON editing with syntax highlighting
- Load data from embedded `<script type="application/ld+json">` tags
- Auto-update script tags when editor content changes
- Dynamic permalink anchor that updates with editor content
- Share and load JSON-LD via URL parameters

For details on the JSON-LD script tag and permalink features, see [docs/JSONLD_SCRIPT_TAG.md](docs/JSONLD_SCRIPT_TAG.md).

### Quick Start
```bash
# Serve the public directory
cd public
python3 -m http.server 8080
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

### webserver - HTTP server for JSON-LD objects
Runs a web server that provides API access to JSON-LD objects and serves the web application using filesystem storage.

```bash
# Start with filesystem storage
./webserver -addr :8080 -store data -public public
```

**Features:**
- **Storage backend**: Filesystem-based storage
- **API routes**:
  - `GET /o/{cid}` - Retrieve object by CID
  - `POST /api/save` - Save JSON-LD and get CID (requires authentication)
  - `DELETE /o/{cid}` - Delete object by CID (author only, requires authentication)
  - `GET /u/{user}/g/{slug}/latest` - Get latest CID for user's gist
  - `GET /u/{user}/g/{slug}/_history` - Get history for user's gist
- **Autosave**: When a logged-in user visits `?data=<json>`, valid JSON-LD is automatically saved and redirected to `?cid=<CID>`
- **Static file serving**: Serves the web application from the public directory
- **CORS support**: Enable with `-cors` flag for cross-origin requests
- **Security**: Server-side validation of JSON-LD structure, content size limits, and author verification for deletions

**Options:**
- `-addr` - Server address (default: `:8080`)
- `-store` - Filesystem store directory (default: `data`)
- `-public` - Public directory for static files (default: `public`)
- `-cors` - Enable CORS headers (default: `true`)
- `-max-content-mb` - Maximum content size in megabytes (default: `1`)

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

## 5. Quick Start

1. Build the tools:
   ```bash
   go build -o seal ./cmd/seal
   go build -o webserver ./cmd/webserver
   ```

2. Create and seal a JSON-LD document:
   ```bash
   ./seal -in examples/petrinet.jsonld -store data
   ```

3. Start the web server:
   ```bash
   ./webserver -addr :8080 -store data -public public
   ```

4. Open your browser to `http://localhost:8080`

See [examples/workflow.sh](examples/workflow.sh) for a complete workflow example.
