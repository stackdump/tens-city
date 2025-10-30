# tens.city 

## Web Application

The web application is a single-page app for managing JSON-LD objects with GitHub authentication.

### Features
- GitHub OAuth login via Supabase Auth
- Query and view objects from the database
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

Then configure your Supabase credentials in `index.html` (see `public/README.md` for details).

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

## 4. JSON-LD Validation

The database uses the `pg_jsonschema` PostgreSQL extension to enforce JSON-LD correctness guarantees at the database layer. This ensures that all stored objects meet basic JSON-LD requirements:

- **Required `@context` field**: All JSON-LD documents must have a `@context` field (can be string, object, array, or null)
- **Schema validation**: Objects are validated against a JSON Schema before insertion
- **Database-level enforcement**: Invalid JSON-LD documents are rejected at the database level

This provides a safety net ensuring data integrity without requiring application-level validation.

For detailed information about JSON-LD validation, see [docs/JSONLD_VALIDATION.md](docs/JSONLD_VALIDATION.md).

## 5. CLI Tools

### webserver - HTTP server for JSON-LD objects
Runs a web server that provides API access to JSON-LD objects and serves the web application. Supports both filesystem and PostgreSQL/Supabase storage backends.

```bash
# Start with filesystem storage
./webserver -addr :8080 -store data -public public

# Start with PostgreSQL/Supabase storage
./webserver -addr :8080 -db <DATABASE_URL> -public public
```

**Features:**
- **Storage backends**: Filesystem or PostgreSQL/Supabase database
- **API routes**:
  - `GET /o/{cid}` - Retrieve object by CID
  - `POST /api/save` - Save JSON-LD and get CID
  - `GET /u/{user}/g/{slug}/latest` - Get latest CID for user's gist
  - `GET /u/{user}/g/{slug}/_history` - Get history for user's gist
- **Autosave**: When a logged-in user visits `?data=<json>`, valid JSON-LD is automatically saved and redirected to `?cid=<CID>`
- **Static file serving**: Serves the web application from the public directory
- **CORS support**: Enable with `-cors` flag for cross-origin requests

**Options:**
- `-addr` - Server address (default: `:8080`)
- `-db` - PostgreSQL database URL (if not set, uses filesystem)
- `-store` - Filesystem store directory (default: `data`)
- `-public` - Public directory for static files (default: `public`)
- `-cors` - Enable CORS headers (default: `true`)

### seal - Create sealed JSON-LD objects
Seals JSON-LD documents using URDNA2015 canonicalization and computes CIDv1 identifiers.

```bash
./seal -in examples/petrinet.jsonld -store data
```

See `cmd/seal/main.go` for full options including signing with Ethereum keys.

### edge - Database operations
Interact with the PostgreSQL database for importing objects, adding signatures, and querying with pg_graphql.

```bash
# Import JSON-LD to database
./edge import -db <DATABASE_URL> -file examples/petrinet.jsonld

# Import filesystem object to database
./edge import-fs -db <DATABASE_URL> -cid <CID> -store data

# Add signature to object
./edge sign -db <DATABASE_URL> -cid <CID> -store data

# Query with pg_graphql
./edge query -db <DATABASE_URL> -query '{objectsCollection{edges{node{cid}}}}'
```

See [docs/edge-cli.md](docs/edge-cli.md) for detailed documentation.

### keygen - Generate Ethereum keys
Generate or import Ethereum keystore files for signing.

```bash
./keygen -out my-key.keystore -pass mypassword
```

## 6. Quick Start

### Option 1: Web Server with Filesystem Storage

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

### Option 2: Web Server with Database Storage

1. Build the tools:
   ```bash
   go build -o seal ./cmd/seal
   go build -o edge ./cmd/edge
   go build -o webserver ./cmd/webserver
   ```

2. Start the database:
   ```bash
   docker-compose up -d
   ```

3. Run migrations:
   ```bash
   psql <DATABASE_URL> -f migrations/migrations_20251029_create_tens_city_tables.sql
   psql <DATABASE_URL> -f migrations/migrations_20251030_add_jsonld_validation.sql
   psql <DATABASE_URL> -f migrations/policies_enable_rls_and_policies.sql
   ```

4. Seal and import a JSON-LD document:
   ```bash
   ./seal -in examples/petrinet.jsonld -store data
   ./edge import-fs -db <DATABASE_URL> -cid <CID> -store data
   ```

5. Start the web server with database:
   ```bash
   ./webserver -addr :8080 -db <DATABASE_URL> -public public
   ```

6. Open your browser to `http://localhost:8080`

See [examples/workflow.sh](examples/workflow.sh) for a complete workflow example.
