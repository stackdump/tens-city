package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackdump/tens-city/internal/seal"
	"github.com/stackdump/tens-city/internal/store"
)

const (
	testDatabaseURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
)

// TestEdgeCLI tests the edge CLI commands end-to-end
func TestEdgeCLI(t *testing.T) {
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
	if err := waitForDatabase(ctx, testDatabaseURL, 30*time.Second); err != nil {
		t.Fatalf("Database failed to become ready: %v", err)
	}

	// Run migrations
	t.Log("Running database migrations...")
	if err := runMigrations(t); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Build the edge binary
	t.Log("Building edge binary...")
	buildCmd := exec.Command("go", "build", "-o", "edge_test", ".")
	buildCmd.Dir = filepath.Join(getRepoRoot(), "cmd", "edge")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build edge: %v\n%s", err, output)
	}
	defer os.Remove(filepath.Join(getRepoRoot(), "cmd", "edge", "edge_test"))

	edgeBinary := filepath.Join(getRepoRoot(), "cmd", "edge", "edge_test")

	// Test 1: Import JSON-LD file directly to database
	t.Log("Test 1: Import JSON-LD file directly to database")
	exampleFile := filepath.Join(getRepoRoot(), "examples", "petrinet.jsonld")
	importCmd := exec.Command(edgeBinary, "import",
		"-db", testDatabaseURL,
		"-file", exampleFile)
	if output, err := importCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to import: %v\n%s", err, output)
	} else {
		t.Logf("Import output: %s", output)
	}

	// Verify the object was imported
	pool, err := pgxpool.New(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM public.objects").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query objects: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 object, got %d", count)
	}

	// Test 2: Create filesystem object and import it to database
	t.Log("Test 2: Create filesystem object and import to database")
	tmpStore := t.TempDir()
	
	// Create a filesystem object using the seal package
	raw, err := os.ReadFile(filepath.Join(getRepoRoot(), "examples", "petrinet-inhibitor.jsonld"))
	if err != nil {
		t.Fatalf("Failed to read example file: %v", err)
	}

	st := store.NewFSStore(tmpStore)
	cidStr, canonicalBytes, err := seal.SealJSONLD(raw)
	if err != nil {
		t.Fatalf("Failed to seal: %v", err)
	}

	if err := st.SaveObject(cidStr, raw, canonicalBytes); err != nil {
		t.Fatalf("Failed to save object: %v", err)
	}
	t.Logf("Created filesystem object: %s", cidStr)

	// Import from filesystem to database
	importFsCmd := exec.Command(edgeBinary, "import-fs",
		"-db", testDatabaseURL,
		"-cid", cidStr,
		"-store", tmpStore)
	if output, err := importFsCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to import-fs: %v\n%s", err, output)
	} else {
		t.Logf("Import-fs output: %s", output)
	}

	// Verify the second object was imported
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM public.objects").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query objects: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 objects, got %d", count)
	}

	// Test 3: Add signature to object
	t.Log("Test 3: Add signature to object")
	
	// Create a test signature in filesystem
	testSig := "0x1234567890abcdef"
	testAddr := "0xabcdef1234567890"
	if err := st.SaveSignature(cidStr, testSig, testAddr, true); err != nil {
		t.Fatalf("Failed to save signature: %v", err)
	}

	// Import signature to database
	signCmd := exec.Command(edgeBinary, "sign",
		"-db", testDatabaseURL,
		"-cid", cidStr,
		"-store", tmpStore)
	if output, err := signCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to sign: %v\n%s", err, output)
	} else {
		t.Logf("Sign output: %s", output)
	}

	// Verify the signature was added
	var sigCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM public.signatures WHERE cid = $1", cidStr).Scan(&sigCount)
	if err != nil {
		t.Fatalf("Failed to query signatures: %v", err)
	}
	if sigCount != 1 {
		t.Errorf("Expected 1 signature, got %d", sigCount)
	}

	t.Log("All edge CLI tests passed!")
}

func getRepoRoot() string {
	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// The test is in cmd/edge, so go up two levels
	return filepath.Join(wd, "..", "..")
}

func startDockerCompose(t *testing.T) error {
	cmd := exec.Command("docker-compose", "up", "-d")
	cmd.Dir = getRepoRoot()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	t.Logf("Docker-compose output: %s", output)
	return nil
}

func cleanupDockerCompose(t *testing.T) {
	cmd := exec.Command("docker-compose", "down", "-v")
	cmd.Dir = getRepoRoot()
	output, _ := cmd.CombinedOutput()
	t.Logf("Docker-compose cleanup output: %s", output)
}

func waitForDatabase(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			if err := pool.Ping(ctx); err == nil {
				pool.Close()
				return nil
			}
			pool.Close()
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("database did not become ready within %v", timeout)
}

func runMigrations(t *testing.T) error {
	repoRoot := getRepoRoot()
	migrationsDir := filepath.Join(repoRoot, "migrations")

	// Connect to database
	pool, err := pgxpool.New(context.Background(), testDatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Read and execute migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".sql" {
			continue
		}

		sqlFile := filepath.Join(migrationsDir, file.Name())
		sqlBytes, err := os.ReadFile(sqlFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file.Name(), err)
		}

		if _, err := pool.Exec(context.Background(), string(sqlBytes)); err != nil {
			return fmt.Errorf("failed to execute %s: %w", file.Name(), err)
		}
		t.Logf("Executed migration: %s", file.Name())
	}

	t.Log("Successfully ran migrations")
	return nil
}
