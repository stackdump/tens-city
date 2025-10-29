package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackdump/tens-city/internal/seal"
	"github.com/stackdump/tens-city/internal/store"
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
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "import":
		importCommand()
	case "import-fs":
		importFsCommand()
	case "sign":
		signCommand()
	case "query":
		queryCommand()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("edge - Tens City database CLI tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  edge <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  import      Import JSON-LD file directly to database")
	fmt.Println("  import-fs   Import filesystem object to database")
	fmt.Println("  sign        Add signature to existing object in database")
	fmt.Println("  query       Query database using pg_graphql")
	fmt.Println()
	fmt.Println("Run 'edge <command> -h' for command-specific help")
}

// importCommand imports a JSON-LD file directly to the database
func importCommand() {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	dbURL := fs.String("db", "", "Database connection URL (required)")
	filePath := fs.String("file", "", "JSON-LD file to import (required)")
	ownerUUID := fs.String("owner", "00000000-0000-0000-0000-000000000000", "Owner UUID (default: zero UUID)")

	fs.Parse(os.Args[2:])

	if *dbURL == "" || *filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: -db and -file are required")
		fs.Usage()
		os.Exit(1)
	}

	raw, err := os.ReadFile(*filePath)
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
	cfg, err := pgxpool.ParseConfig(*dbURL)
	if err != nil {
		log.Fatalf("parse db url: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(context.Background())

	// Insert object
	_, err = tx.Exec(context.Background(),
		`INSERT INTO public.objects (cid, owner_uuid, raw, canonical, storage_path) VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (cid) DO NOTHING`,
		cidStr, *ownerUUID, json.RawMessage(raw), string(canonicalBytes), nil)
	if err != nil {
		log.Fatalf("insert object: %v", err)
	}

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

	fmt.Printf("Imported: cid=%s\n", cidStr)
}

// importFsCommand imports a filesystem object to database
func importFsCommand() {
	fs := flag.NewFlagSet("import-fs", flag.ExitOnError)
	dbURL := fs.String("db", "", "Database connection URL (required)")
	cid := fs.String("cid", "", "CID of the filesystem object to import (required)")
	storeDir := fs.String("store", "data", "Base directory of filesystem store")
	ownerUUID := fs.String("owner", "00000000-0000-0000-0000-000000000000", "Owner UUID (default: zero UUID)")

	fs.Parse(os.Args[2:])

	if *dbURL == "" || *cid == "" {
		fmt.Fprintln(os.Stderr, "Error: -db and -cid are required")
		fs.Usage()
		os.Exit(1)
	}

	// Load from filesystem store
	st := store.NewFSStore(*storeDir)

	raw, err := st.ReadObject(*cid)
	if err != nil {
		log.Fatalf("read object from store: %v", err)
	}

	canonical, err := st.ReadCanonical(*cid)
	if err != nil {
		log.Fatalf("read canonical from store: %v", err)
	}

	// Connect to Postgres (pgx)
	cfg, err := pgxpool.ParseConfig(*dbURL)
	if err != nil {
		log.Fatalf("parse db url: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(context.Background())
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(context.Background())

	// Insert object
	_, err = tx.Exec(context.Background(),
		`INSERT INTO public.objects (cid, owner_uuid, raw, canonical, storage_path) VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (cid) DO NOTHING`,
		*cid, *ownerUUID, json.RawMessage(raw), string(canonical), nil)
	if err != nil {
		log.Fatalf("insert object: %v", err)
	}

	// parse N-Quads into quads (optional)
	quads, err := parseNQuads(string(canonical))
	if err != nil {
		log.Fatalf("parse n-quads: %v", err)
	}

	// Bulk insert quads
	for _, q := range quads {
		_, err = tx.Exec(context.Background(),
			`INSERT INTO public.quads (cid, subject, predicate, object, object_is_literal, graph)
			 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
			*cid, q.Subject, q.Predicate, q.Object, q.ObjectIsLiteral, q.Graph)
		if err != nil {
			log.Fatalf("insert quad: %v", err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Printf("Imported from filesystem: cid=%s\n", *cid)
}

// signCommand adds a signature to an existing object in the database
func signCommand() {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	dbURL := fs.String("db", "", "Database connection URL (required)")
	cid := fs.String("cid", "", "CID of the object to sign (required)")
	storeDir := fs.String("store", "data", "Base directory of filesystem store")
	signature := fs.String("sig", "", "Signature to add (hex-encoded)")
	signerAddr := fs.String("addr", "", "Signer address")
	usePersonalSign := fs.Bool("personal", true, "Use personal_sign format")

	fs.Parse(os.Args[2:])

	if *dbURL == "" || *cid == "" {
		fmt.Fprintln(os.Stderr, "Error: -db and -cid are required")
		fs.Usage()
		os.Exit(1)
	}

	// If signature not provided, try to load from filesystem
	var sig, addr string
	var personalSign bool
	if *signature == "" {
		st := store.NewFSStore(*storeDir)
		meta, err := st.ReadSignature(*cid)
		if err != nil {
			log.Fatalf("no signature provided and unable to read from store: %v", err)
		}
		sig = meta.Signature
		addr = meta.SignerAddress
		personalSign = meta.UsePersonalSign
		fmt.Printf("Loaded signature from filesystem store\n")
	} else {
		sig = *signature
		addr = *signerAddr
		personalSign = *usePersonalSign
	}

	// Connect to Postgres (pgx)
	cfg, err := pgxpool.ParseConfig(*dbURL)
	if err != nil {
		log.Fatalf("parse db url: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	// Insert signature
	_, err = pool.Exec(context.Background(),
		`INSERT INTO public.signatures (cid, signer_address, signature, use_personal_sign)
		 VALUES ($1, $2, $3, $4)`,
		*cid, addr, sig, personalSign)
	if err != nil {
		log.Fatalf("insert signature: %v", err)
	}

	fmt.Printf("Added signature for cid=%s from address=%s\n", *cid, addr)
}

// queryCommand queries the database using pg_graphql
func queryCommand() {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	dbURL := fs.String("db", "", "Database connection URL (required)")
	queryStr := fs.String("query", "", "GraphQL query string (required)")
	variables := fs.String("vars", "{}", "GraphQL variables as JSON")

	fs.Parse(os.Args[2:])

	if *dbURL == "" || *queryStr == "" {
		fmt.Fprintln(os.Stderr, "Error: -db and -query are required")
		fs.Usage()
		os.Exit(1)
	}

	// Connect to Postgres (pgx)
	cfg, err := pgxpool.ParseConfig(*dbURL)
	if err != nil {
		log.Fatalf("parse db url: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	// Execute GraphQL query using pg_graphql
	var result string
	err = pool.QueryRow(context.Background(),
		`SELECT graphql.resolve($1, $2)`,
		*queryStr, *variables).Scan(&result)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}

	// Pretty print the JSON result
	var prettyJSON interface{}
	if err := json.Unmarshal([]byte(result), &prettyJSON); err == nil {
		output, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Println(result)
	}
}
