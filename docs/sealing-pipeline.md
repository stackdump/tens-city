```markdown
# Sealing pipeline (tens.city) — implementation notes

What this delivers
- A minimal, self-contained Go sealing pipeline that:
  - Accepts JSON-LD input,
  - Canonicalizes it using URDNA2015 (via piprate/json-gold),
  - Produces deterministic N-Quads canonicalization,
  - Hashes the canonical bytes with SHA2-256 → creates a CIDv1 (raw codec),
  - Stores the raw JSON-LD and canonical form on disk under `data/o/{cid}` and `data/o/canonical/{cid}.nq`,
  - Provides simple per-user gist pointer updates: `/u/{login}/g/{slug}/latest` and `_history`.

Files added
- cmd/seal/main.go — CLI entry to run the sealing pipeline.
- internal/seal/seal.go — canonicalization + CID calculation implementation.
- internal/store/store.go — simple filesystem-backed persistence for objects and pointers.
- docs/sealing-pipeline.md — this document.

How to try locally
1. Build:
   go build ./cmd/seal
   go build ./cmd/keygen

2. Seal a file (without signing):
   ./seal -in examples/petrinet.jsonld -store data -user alice -gist demo -pretty

3. Seal and sign a file:
   # First, create a keystore
   ./keygen -out alice.keystore -pass "my-secure-passphrase"
   
   # Then seal and sign
   ./seal -in examples/petrinet.jsonld -store data -user alice -gist demo -keystore alice.keystore -pretty
   
   # You'll be prompted for your passphrase

This will:
- canonicalize and compute the CID,
- write `data/o/{cid}` (raw JSON-LD),
- write `data/o/canonical/{cid}.nq` (URDNA2015 normalized N-Quads),
- (if signing) write `data/o/signatures/{cid}.json` (signature metadata),
- update `data/u/alice/g/demo/latest` with the CID,
- append an entry to `data/u/alice/g/demo/_history`.

Ethereum Signing (NEW)
The seal CLI now supports cryptographic signing of sealed objects using Ethereum-style signatures:
- Use `-keystore <path>` to sign with an encrypted keystore (passphrase-protected)
- Use `-privkey <hex>` to sign with a raw private key (testing only)
- Signatures are stored at `data/o/signatures/{cid}.json`
- Supports both personal_sign (EIP-191) and raw signing modes
- See `docs/ethereum-signing.md` for detailed documentation

Notes and next steps
- This is an intentionally small, dependency-light starter. Integrate into your HTTP server and apcore adapter by:
  - Replacing FSStore with your DB-backed store,
  - Calling SealJSONLD() as part of PUT /u/{login}/g/{slug}/index handling,
  - Recording provenance metadata (createdBy, createdAt, modelCID) in a separate metadata store or in the same container.
- Replace the storage layout with your preferred object-addressable store (IPFS, S3 keyed by CID, Postgres bytea keyed by CID).
- Signature generation is now implemented via the ethsig package. Signatures are stored alongside sealed objects and can be verified using the VerifyEthereumSignature function.
- Consider embedding the canonicalization output (or a digest) in ActivityPub objects when federating via apcore.

Security & determinism
- URDNA2015 normalization yields deterministic output given the same JSON-LD and context; ensure all contexts needed are available and stable (host local contexts at `/context/...` if necessary).
- Canonicalization depends on full expansion/resolution of @context. If external contexts are used, pin or cache them (or host your own versioned contexts).
- Keystore files are encrypted using scrypt with standard Ethereum parameters and stored with restrictive permissions (0600).
- Never commit keystore files or private keys to version control.

Limitations
- The simple FSStore is not production-grade (no concurrency locks, no atomic updates). Replace with a transactional DB or object store for production.
- Signature verification is available programmatically but not yet exposed via CLI. Add a verify command if needed.
```
