# Integration Tests

This directory contains integration tests for the tens-city project that test the full data ingestion pipeline from edge function to Supabase storage.

## Overview

The integration tests verify:
- Docker Compose setup with Supabase PostgreSQL
- Database migration execution
- JSON-LD data sealing and CID generation
- Data ingestion into Supabase
- JSONB indexing and querying
- @id and @type attribute indexing
- Object metadata views

## Running Integration Tests

### Prerequisites
- Docker and Docker Compose v2 installed
- Go 1.24 or later

### Run Integration Tests

To run the full integration test suite:

```bash
go test -v ./internal/integration/... -timeout 5m
```

### Skip Integration Tests

When running tests in short mode (e.g., during development), integration tests are automatically skipped:

```bash
go test -short ./...
```

## Test Details

### TestSupabaseIntegration

This test performs the following steps:

1. **Starts Docker Compose** - Spins up a Supabase PostgreSQL container
2. **Waits for Database** - Ensures the database is ready to accept connections
3. **Runs Migrations** - Executes SQL migrations to create tables and indexes
4. **Connects to Database** - Establishes a connection using pgx
5. **Seals JSON-LD** - Creates test JSON-LD data and computes CID
6. **Ingests Data** - Inserts object into the database (simulating edge function behavior)
7. **Verifies Storage** - Confirms data was stored correctly
8. **Tests Indexing** - Verifies JSONB GIN index queries work
9. **Tests @type Index** - Verifies @type field indexing
10. **Tests @id Indexing** - Verifies @id attributes are indexed and searchable
11. **Tests Views** - Verifies object_metadata view functions correctly
12. **Cleanup** - Tears down Docker containers

### Database Connection

The test connects to PostgreSQL at:
- Host: `localhost`
- Port: `5432`
- Database: `postgres`
- User: `postgres`
- Password: `postgres`

This matches the configuration in `docker-compose.yml`.

## Docker Compose

The test uses the `docker-compose.yml` file in the repository root, which defines:
- PostgreSQL 17.6 with Supabase extensions
- Container name: `pflow_db`
- Port mapping: `5432:5432`

## Migrations

The test automatically runs migrations from the `migrations/` directory:
- `migrations_20251029_create_tens_city_tables.sql` - Creates tables, indexes, and views

## Troubleshooting

### Docker not available
If you get "docker: command not found", ensure Docker is installed and in your PATH.

### Port already in use
If port 5432 is already in use, stop any existing PostgreSQL instances:
```bash
docker compose -f docker-compose.yml down -v
```

### Test timeout
If the test times out, try increasing the timeout:
```bash
go test -v ./internal/integration/... -timeout 10m
```
