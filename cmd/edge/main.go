package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackdump/tens-city/internal/seal" // your package
)

// parseNQuads: placeholder that parses canonical N-Quads into quads records.
// You should use an N-Quads parser or a small line parser that extracts subject/predicate/object/graph.
// Here we'll keep it conceptual: return slice of (subject,predicate,object,objectIsLiteral,graph).
type Quad struct {
	Subject         string
	Predicate       string
	Object          string
	ObjectIsLiteral bool
	Graph           *string
}

func parseNQuads(nq string) ([]Quad, error) {
	// TODO: implement a proper N-Quads parser or use an existing library.
	// For now, return empty slice to show transaction flow.
	return []Quad{}, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: ingest_to_supabase <DATABASE_URL> <jsonld-file>")
		os.Exit(2)
	}
	databaseURL := os.Args[1]
	filePath := os.Args[2]

	raw, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("read file: %v", err)
	}

	// Canonicalize + compute CID
	cidStr, canonicalBytes, err := seal.SealJSONLD(raw)
	if err != nil {
		log.Fatalf("seal failed: %v", err)
	}

	// parse N-Quads into quads (optional)
	quads, err := parseNQuads(string(canonicalBytes))
	if err != nil {
		log.Fatalf("parse n-quads: %v", err)
	}

	// Connect to Postgres (pgx)
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("parse db url: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	// owner UUID must come from the authenticated user context; here we demonstrate passing it explicitly.
	// In a real Supabase Edge Function / server you would read auth.uid() from the JWT.
	ownerUUID := "00000000-0000-0000-0000-000000000000" // replace with real authenticated uuid

	tx, err := pool.Begin(context.Background())
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(context.Background())

	// Insert object
	_, err = tx.Exec(context.Background(),
		`INSERT INTO public.objects (cid, owner_uuid, raw, canonical, storage_path) VALUES ($1, $2, $3, $4, $5)`,
		cidStr, ownerUUID, json.RawMessage(raw), string(canonicalBytes), nil)
	if err != nil {
		log.Fatalf("insert object: %v", err)
	}

	// Insert signature row if you have a signature (not shown here)
	// _, err = tx.Exec(...)

	// Bulk insert quads (if parseNQuads produced them)
	for _, q := range quads {
		_, err = tx.Exec(context.Background(),
			`INSERT INTO public.quads (cid, subject, predicate, object, object_is_literal, graph)
			 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
			cidStr, q.Subject, q.Predicate, q.Object, q.ObjectIsLiteral, q.Graph)
		if err != nil {
			log.Fatalf("insert quad: %v", err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Printf("ingested: cid=%s\n", cidStr)
}
