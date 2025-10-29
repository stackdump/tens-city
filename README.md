# tens.city — Tent City Ethos

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

## CLI Tools

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

## Quick Start

1. Build the tools:
   ```bash
   go build -o seal ./cmd/seal
   go build -o edge ./cmd/edge
   go build -o keygen ./cmd/keygen
   ```

2. Start the database:
   ```bash
   docker-compose up -d
   ```

3. Run migrations:
   ```bash
   psql <DATABASE_URL> -f migrations/migrations_20251029_create_tens_city_tables.sql
   psql <DATABASE_URL> -f migrations/policies_enable_rls_and_policies.sql
   ```

4. Seal and import a JSON-LD document:
   ```bash
   ./seal -in examples/petrinet.jsonld -store data
   ./edge import-fs -db <DATABASE_URL> -cid <CID> -store data
   ```

See [examples/workflow.sh](examples/workflow.sh) for a complete workflow example.