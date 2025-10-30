package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackdump/tens-city/internal/seal"
)

const (
	testDBURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
)

// TestJSONLDValidation tests that pg_jsonschema correctly validates JSON-LD documents
func TestJSONLDValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start docker-compose
	t.Log("Starting docker-compose...")
	if err := startDockerCompose(t); err != nil {
		t.Fatalf("Failed to start docker-compose: %v", err)
	}
	defer cleanupDockerCompose(t)

	// Wait for database
	t.Log("Waiting for database to be ready...")
	if err := waitForDatabase(ctx, testDBURL, 30*time.Second); err != nil {
		t.Fatalf("Database failed to become ready: %v", err)
	}

	// Run migrations
	t.Log("Running database migrations...")
	if err := runMigrations(t); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Test 1: Valid JSON-LD with string @context should succeed
	t.Run("ValidJSONLD_StringContext", func(t *testing.T) {
		validJSON := json.RawMessage(`{"@context": "https://schema.org", "name": "Test"}`)
		cidStr, canonical, err := seal.SealJSONLD(validJSON)
		if err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO public.objects (cid, owner_uuid, raw, canonical)
			 VALUES ($1, $2, $3, $4)`,
			cidStr, "00000000-0000-0000-0000-000000000000", validJSON, string(canonical))
		if err != nil {
			t.Errorf("Expected insertion to succeed but got error: %v", err)
		}
	})

	// Test 2: Valid JSON-LD with object @context should succeed
	t.Run("ValidJSONLD_ObjectContext", func(t *testing.T) {
		exampleFile := filepath.Join(getRepoRoot(), "examples", "petrinet.jsonld")
		raw, err := os.ReadFile(exampleFile)
		if err != nil {
			t.Fatalf("Failed to read example file: %v", err)
		}

		cidStr, canonical, err := seal.SealJSONLD(raw)
		if err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO public.objects (cid, owner_uuid, raw, canonical)
			 VALUES ($1, $2, $3, $4)`,
			cidStr, "00000000-0000-0000-0000-000000000000", json.RawMessage(raw), string(canonical))
		if err != nil {
			t.Errorf("Expected insertion to succeed but got error: %v", err)
		}
	})

	// Test 3: Invalid JSON-LD without @context should fail
	t.Run("InvalidJSONLD_MissingContext", func(t *testing.T) {
		invalidJSON := json.RawMessage(`{"name": "Test without context"}`)
		cidStr, canonical, err := seal.SealJSONLD(invalidJSON)
		if err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO public.objects (cid, owner_uuid, raw, canonical)
			 VALUES ($1, $2, $3, $4)`,
			cidStr, "00000000-0000-0000-0000-000000000001", invalidJSON, string(canonical))
		if err == nil {
			t.Error("Expected insertion to fail for JSON without @context, but it succeeded")
		} else {
			t.Logf("Correctly rejected invalid JSON-LD: %v", err)
		}
	})

	// Test 4: Valid JSON-LD with null @context should succeed (per spec)
	t.Run("ValidJSONLD_NullContext", func(t *testing.T) {
		validJSON := json.RawMessage(`{"@context": null, "name": "Test"}`)
		cidStr, canonical, err := seal.SealJSONLD(validJSON)
		if err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO public.objects (cid, owner_uuid, raw, canonical)
			 VALUES ($1, $2, $3, $4)`,
			cidStr, "00000000-0000-0000-0000-000000000002", validJSON, string(canonical))
		if err != nil {
			t.Errorf("Expected insertion to succeed with null @context but got error: %v", err)
		}
	})

	// Test 5: Complex example (petrinet-inhibitor) should succeed
	t.Run("ValidJSONLD_ComplexExample", func(t *testing.T) {
		exampleFile := filepath.Join(getRepoRoot(), "examples", "petrinet-inhibitor.jsonld")
		raw, err := os.ReadFile(exampleFile)
		if err != nil {
			t.Fatalf("Failed to read example file: %v", err)
		}

		cidStr, canonical, err := seal.SealJSONLD(raw)
		if err != nil {
			t.Fatalf("Failed to seal: %v", err)
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO public.objects (cid, owner_uuid, raw, canonical)
			 VALUES ($1, $2, $3, $4)`,
			cidStr, "00000000-0000-0000-0000-000000000003", json.RawMessage(raw), string(canonical))
		if err != nil {
			t.Errorf("Expected insertion to succeed for complex example but got error: %v", err)
		}
	})

	// Verify the function is working
	t.Run("ValidateSchemaFunction", func(t *testing.T) {
		var isValid bool
		err := pool.QueryRow(ctx,
			`SELECT jsonb_matches_schema(
				get_jsonld_base_schema(),
				'{"@context": "https://schema.org"}'::jsonb
			)`).Scan(&isValid)
		if err != nil {
			t.Fatalf("Failed to call validation function: %v", err)
		}
		if !isValid {
			t.Error("Expected schema validation to return true for valid JSON-LD")
		}

		err = pool.QueryRow(ctx,
			`SELECT jsonb_matches_schema(
				get_jsonld_base_schema(),
				'{}'::jsonb
			)`).Scan(&isValid)
		if err != nil {
			t.Fatalf("Failed to call validation function: %v", err)
		}
		if isValid {
			t.Error("Expected schema validation to return false for JSON without @context")
		}
	})

	t.Log("All JSON-LD validation tests completed!")
}
