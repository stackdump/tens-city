package seal

import (
	"encoding/json"
	"errors"

	"github.com/piprate/json-gold/ld"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/multiformats/go-multibase"
)

// SealJSONLD takes raw JSON-LD bytes, canonicalizes using URDNA2015
// (via piprate/json-gold), computes a CIDv1 (json-ld codec) using SHA2-256,
// and returns (cidString, canonicalBytes, error).
//
// canonicalBytes are the N-Quads string returned by Normalize() encoded as UTF-8 bytes.
// The canonicalization step uses the URDNA2015 algorithm and produces deterministic output.
// The CID is encoded using base58btc (z prefix).
func SealJSONLD(raw []byte) (string, []byte, error) {
	// parse JSON-LD into a Go interface{}
	var doc interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", nil, err
	}

	// prepare processor and options
	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("") // empty base
	// Ensure URDNA2015 algorithm and n-quads output
	opts.Format = "application/n-quads"
	opts.Algorithm = "URDNA2015"

	// Normalize
	normalized, err := proc.Normalize(doc, opts)
	if err != nil {
		return "", nil, err
	}

	// Normalize returns a string containing N-Quads (normalized RDF); assert that
	nqStr, ok := normalized.(string)
	if !ok {
		return "", nil, errors.New("unexpected normalized output type")
	}
	normalizedBytes := []byte(nqStr)

	// Compute multihash (sha2-256)
	multihash, err := mh.Sum(normalizedBytes, mh.SHA2_256, -1)
	if err != nil {
		return "", nil, err
	}

	// Create CIDv1 with json-ld codec (DagJSON = 0x0129)
	c := cid.NewCidV1(cid.DagJSON, multihash)
	
	// Encode using base58btc (z prefix) for storage in filesystem
	cidStr, err := c.StringOfBase(multibase.Base58BTC)
	if err != nil {
		return "", nil, err
	}
	
	return cidStr, normalizedBytes, nil
}