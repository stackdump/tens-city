# Edge CLI - Tens City Database Tool

The `edge` CLI provides commands for interacting with the Tens City PostgreSQL database.

## Commands

### import

Import a JSON-LD file directly to the database.

```bash
edge import -db <DATABASE_URL> -file <jsonld-file> [-owner <UUID>]
```

**Options:**
- `-db` (required): Database connection URL
- `-file` (required): Path to JSON-LD file to import
- `-owner`: Owner UUID (default: 00000000-0000-0000-0000-000000000000)

**Example:**
```bash
edge import \
  -db "postgres://postgres:postgres@localhost:5432/postgres" \
  -file examples/petrinet.jsonld
```

### import-fs

Import a filesystem object (created by the `seal` command) to the database.

```bash
edge import-fs -db <DATABASE_URL> -cid <CID> [-store <DIR>] [-owner <UUID>]
```

**Options:**
- `-db` (required): Database connection URL
- `-cid` (required): CID of the filesystem object to import
- `-store`: Base directory of filesystem store (default: "data")
- `-owner`: Owner UUID (default: 00000000-0000-0000-0000-000000000000)

**Example:**
```bash
# First create a filesystem object with seal
./seal -in examples/petrinet.jsonld -store data

# Then import it to the database
edge import-fs \
  -db "postgres://postgres:postgres@localhost:5432/postgres" \
  -cid z4EBG9jDcmuFgD2Vs1unB4caki8tPhKrdWeoEME9d35HmhBZfJQ \
  -store data
```

### sign

Add a signature to an existing object in the database.

```bash
edge sign -db <DATABASE_URL> -cid <CID> [-store <DIR>] [-sig <SIGNATURE>] [-addr <ADDRESS>] [-personal]
```

**Options:**
- `-db` (required): Database connection URL
- `-cid` (required): CID of the object to sign
- `-store`: Base directory of filesystem store (default: "data")
- `-sig`: Signature to add (hex-encoded). If not provided, will try to load from filesystem store
- `-addr`: Signer address (required if -sig is provided)
- `-personal`: Use personal_sign format (default: true)

**Example:**
```bash
# If signature exists in filesystem store
edge sign \
  -db "postgres://postgres:postgres@localhost:5432/postgres" \
  -cid z4EBG9jDcmuFgD2Vs1unB4caki8tPhKrdWeoEME9d35HmhBZfJQ \
  -store data

# Or provide signature directly
edge sign \
  -db "postgres://postgres:postgres@localhost:5432/postgres" \
  -cid z4EBG9jDcmuFgD2Vs1unB4caki8tPhKrdWeoEME9d35HmhBZfJQ \
  -sig "0x1234..." \
  -addr "0xabcd..."
```

### query

Query the database using pg_graphql.

```bash
edge query -db <DATABASE_URL> -query <GRAPHQL_QUERY> [-vars <JSON>]
```

**Options:**
- `-db` (required): Database connection URL
- `-query` (required): GraphQL query string
- `-vars`: GraphQL variables as JSON (default: "{}")

**Example:**
```bash
edge query \
  -db "postgres://postgres:postgres@localhost:5432/postgres" \
  -query '{objectsCollection{edges{node{cid}}}}'

# With variables
edge query \
  -db "postgres://postgres:postgres@localhost:5432/postgres" \
  -query 'query($cid: String!) {objectsCollection(filter: {cid: {eq: $cid}}) {edges {node {cid raw}}}}' \
  -vars '{"cid": "z4EBG9jDcmuFgD2Vs1unB4caki8tPhKrdWeoEME9d35HmhBZfJQ"}'
```

## Workflow

A typical workflow for working with tens-city:

1. **Seal a JSON-LD file to filesystem:**
   ```bash
   ./seal -in examples/petrinet.jsonld -store data
   ```

2. **Import to database:**
   ```bash
   edge import-fs -db <DB_URL> -cid <CID> -store data
   ```

3. **Add signature (optional):**
   ```bash
   edge sign -db <DB_URL> -cid <CID> -store data
   ```

4. **Query the data:**
   ```bash
   edge query -db <DB_URL> -query '{objectsCollection{edges{node{cid}}}}'
   ```

## Prerequisites

- PostgreSQL database (tested with Supabase postgres image)
- Database migrations applied (see `migrations/` directory)
- For `query` command: pg_graphql extension enabled

## Setup

1. Start the database:
   ```bash
   docker-compose up -d
   ```

2. Run migrations:
   ```bash
   psql <DB_URL> -f migrations/migrations_20251029_create_tens_city_tables.sql
   psql <DB_URL> -f migrations/policies_enable_rls_and_policies.sql
   ```

3. Build the CLI:
   ```bash
   go build -o edge ./cmd/edge
   ```
