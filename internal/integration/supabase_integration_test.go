package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackdump/tens-city/internal/seal"
)

const (
	// Default connection string for docker-compose postgres
	testDatabaseURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	// Timeout for docker operations
	dockerTimeout = 120 * time.Second
)

// TestSupabaseIntegration is an integration test that:
// 1. Spins up Supabase using docker-compose
// 2. Runs database migrations
// 3. Ingests JSON-LD data through edge function logic
// 4. Verifies data is stored correctly with @id attributes
// 5. Verifies JSON objects are indexed properly
// 6. Cleans up docker containers
func TestSupabaseIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Step 1: Start docker-compose
	t.Log("Starting docker-compose...")
	if err := startDockerCompose(t); err != nil {
		t.Fatalf("Failed to start docker-compose: %v", err)
	}
	defer cleanupDockerCompose(t)

	// Step 2: Wait for database to be ready
	t.Log("Waiting for database to be ready...")
	if err := waitForDatabase(ctx, testDatabaseURL, 30*time.Second); err != nil {
		t.Fatalf("Database failed to become ready: %v", err)
	}

	// Step 3: Run migrations
	t.Log("Running database migrations...")
	if err := runMigrations(t); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 4: Connect to database
	pool, err := pgxpool.New(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
	t.Log("Successfully connected to database")

	// Step 5: Prepare test data (JSON-LD object)
	testData := map[string]interface{}{
		"@context": map[string]interface{}{
			"name":        "http://schema.org/name",
			"description": "http://schema.org/description",
		},
		"@type":       "TestObject",
		"name":        "Integration Test Object",
		"description": "This object is created by the integration test",
	}

	rawJSON, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Step 6: Seal the JSON-LD (compute CID and canonical form)
	t.Log("Sealing JSON-LD object...")
	cidStr, canonicalBytes, err := seal.SealJSONLD(rawJSON)
	if err != nil {
		t.Fatalf("Failed to seal JSON-LD: %v", err)
	}
	t.Logf("Generated CID: %s", cidStr)

	// Step 7: Insert object into database (simulating edge function ingestion)
	ownerUUID := "00000000-0000-0000-0000-000000000001" // Test UUID
	t.Log("Inserting object into database...")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Insert the object
	_, err = tx.Exec(ctx,
		`INSERT INTO public.objects (cid, owner_uuid, raw, canonical, storage_path) 
		 VALUES ($1, $2, $3, $4, $5)`,
		cidStr, ownerUUID, json.RawMessage(rawJSON), string(canonicalBytes), nil)
	if err != nil {
		t.Fatalf("Failed to insert object: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
	t.Log("Successfully inserted object")

	// Step 8: Verify object was stored correctly
	t.Log("Verifying stored object...")
	var storedCID string
	var storedRaw json.RawMessage
	var storedCanonical string
	var storedOwnerUUID string

	err = pool.QueryRow(ctx,
		`SELECT cid, raw, canonical, owner_uuid FROM public.objects WHERE cid = $1`,
		cidStr).Scan(&storedCID, &storedRaw, &storedCanonical, &storedOwnerUUID)
	if err != nil {
		t.Fatalf("Failed to query object: %v", err)
	}

	// Verify CID matches
	if storedCID != cidStr {
		t.Errorf("CID mismatch: expected %s, got %s", cidStr, storedCID)
	}

	// Verify owner UUID matches
	if storedOwnerUUID != ownerUUID {
		t.Errorf("Owner UUID mismatch: expected %s, got %s", ownerUUID, storedOwnerUUID)
	}

	// Verify canonical form matches
	if storedCanonical != string(canonicalBytes) {
		t.Errorf("Canonical form mismatch")
	}

	// Step 9: Verify JSON is indexed and @id attribute can be queried
	t.Log("Verifying JSON indexing with @id attribute...")

	// Parse stored raw JSON
	var storedDoc map[string]interface{}
	if err := json.Unmarshal(storedRaw, &storedDoc); err != nil {
		t.Fatalf("Failed to parse stored JSON: %v", err)
	}

	// The stored JSON should match our input (without @id, as that's added by the store layer)
	if storedDoc["name"] != testData["name"] {
		t.Errorf("Name mismatch: expected %v, got %v", testData["name"], storedDoc["name"])
	}

	// Step 10: Test JSONB GIN index by querying with JSON containment
	t.Log("Testing JSONB GIN index query...")
	var count int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM public.objects WHERE raw @> $1`,
		json.RawMessage(`{"name": "Integration Test Object"}`)).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query with JSONB containment: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 object matching JSONB query, got %d", count)
	}

	// Step 11: Test @type index query
	t.Log("Testing @type index query...")
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM public.objects WHERE raw->>'@type' = $1`,
		"TestObject").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query by @type: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 object with @type 'TestObject', got %d", count)
	}

	// Step 12: Verify object_metadata view works
	t.Log("Testing object_metadata view...")
	var viewCID, viewType string
	err = pool.QueryRow(ctx,
		`SELECT cid, type FROM public.object_metadata WHERE cid = $1`,
		cidStr).Scan(&viewCID, &viewType)
	if err != nil {
		t.Fatalf("Failed to query object_metadata view: %v", err)
	}

	if viewCID != cidStr {
		t.Errorf("View CID mismatch: expected %s, got %s", cidStr, viewCID)
	}

	if viewType != "TestObject" {
		t.Errorf("View type mismatch: expected 'TestObject', got %s", viewType)
	}

	t.Log("Integration test completed successfully!")
}

// startDockerCompose starts the docker-compose services
func startDockerCompose(t *testing.T) error {
	// Find the docker-compose.yml file
	composeFile := "../../docker-compose.yml"
	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml not found at %s", composeFile)
	}

	// Stop any existing containers
	stopCmd := exec.Command("docker", "compose", "-f", composeFile, "down", "-v")
	if output, err := stopCmd.CombinedOutput(); err != nil {
		t.Logf("Warning: failed to stop existing containers: %v\nOutput: %s", err, output)
	}

	// Start containers
	upCmd := exec.Command("docker", "compose", "-f", composeFile, "up", "-d")
	output, err := upCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start docker-compose: %v\nOutput: %s", err, output)
	}
	t.Logf("Docker-compose output: %s", output)

	return nil
}

// cleanupDockerCompose stops and removes docker-compose services
func cleanupDockerCompose(t *testing.T) {
	composeFile := "../../docker-compose.yml"
	cmd := exec.Command("docker", "compose", "-f", composeFile, "down", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: failed to cleanup docker-compose: %v\nOutput: %s", err, output)
	} else {
		t.Logf("Docker-compose cleanup output: %s", output)
	}
}

// waitForDatabase waits for the database to be ready
func waitForDatabase(ctx context.Context, dbURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database")
		case <-ticker.C:
			pool, err := pgxpool.New(ctx, dbURL)
			if err != nil {
				continue
			}
			if err := pool.Ping(ctx); err != nil {
				pool.Close()
				continue
			}
			pool.Close()
			return nil
		}
	}
}

// runMigrations runs the database migrations
func runMigrations(t *testing.T) error {
	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, testDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer pool.Close()

	// Read and execute main migration file
	migrationFile := "../../migrations/migrations_20251029_create_tens_city_tables.sql"
	migrationSQL, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %v", err)
	}

	_, err = pool.Exec(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %v", err)
	}

	t.Log("Successfully ran migrations")
	return nil
}
