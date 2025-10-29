package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/stackdump/tens-city/internal/seal"
	"github.com/stackdump/tens-city/internal/store"
)

func main() {
	// CLI flags
	inPath := flag.String("in", "-", "input file (JSON-LD). Use - for stdin")
	outDir := flag.String("store", "data", "base directory to store sealed objects and containers")
	user := flag.String("user", "", "optional user login (for updating /u/{login}/g/{slug}/latest)")
	gist := flag.String("gist", "", "optional gist slug (used with --user to update latest and history)")
	pretty := flag.Bool("pretty", false, "write canonicalized normalized n-quads to a human-readable file alongside the object")
	flag.Parse()

	var data []byte
	var err error
	if *inPath == "-" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("reading stdin: %v", err)
		}
	} else {
		data, err = os.ReadFile(*inPath)
		if err != nil {
			log.Fatalf("reading file %s: %v", *inPath, err)
		}
	}

	// initialize filesystem store
	st := store.NewFSStore(*outDir)

	// run sealing pipeline: canonicalize (URDNA2015), compute CIDv1 (base32) and return canonical bytes
	cidStr, canonicalBytes, err := seal.SealJSONLD(data)
	if err != nil {
		log.Fatalf("seal failed: %v", err)
	}

	// persist raw JSON-LD and canonical form
	if err := st.SaveObject(cidStr, data, canonicalBytes); err != nil {
		log.Fatalf("store SaveObject: %v", err)
	}

	// optionally update user/gist pointer and history
	if *user != "" && *gist != "" {
		if err := st.UpdateLatest(*user, *gist, cidStr); err != nil {
			log.Fatalf("UpdateLatest failed: %v", err)
		}
		if err := st.AppendHistory(*user, *gist, cidStr); err != nil {
			log.Fatalf("AppendHistory failed: %v", err)
		}
		fmt.Printf("sealed as %s and updated /u/%s/g/%s/latest\n", cidStr, *user, *gist)
	} else {
		fmt.Printf("sealed as %s\n", cidStr)
	}

	// optionally write the canonical form to a human readable file next to the canonical payload
	if *pretty {
		hf := fmt.Sprintf("%s/canonical/%s.nq", *outDir, cidStr)
		if err := os.MkdirAll(fmt.Sprintf("%s/canonical", *outDir), 0o755); err != nil {
			log.Fatalf("mkdir canonical: %v", err)
		}
		if err := os.WriteFile(hf, canonicalBytes, 0o644); err != nil {
			log.Fatalf("write canonical file: %v", err)
		}
		fmt.Printf("wrote canonical n-quads to %s\n", hf)
	}

	// echo the saved object path
	objPath := st.ObjectPath(cidStr)
	fmt.Printf("object stored at: %s\n", objPath)

	// show first 512 bytes of canonical form on stdout for convenience
	fmt.Println("---- canonical (first 512 bytes) ----")
	r := io.LimitReader(io.NopCloser(io.NewSectionReader(os.NewFile(0, ""), 0, 1)), 0)
	_ = r // silence linter for unused; canonical printed below instead
	if len(canonicalBytes) > 0 {
		if len(canonicalBytes) > 512 {
			fmt.Printf("%s\n", canonicalBytes[:512])
		} else {
			fmt.Printf("%s\n", canonicalBytes)
		}
	}
}