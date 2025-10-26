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
- docs/SEALING_PIPELINE.md — this document.

How to try locally
1. Build:
   go build ./cmd/seal

2. Seal a file:
   ./seal -in examples/petrinet.jsonld -store data -user alice -gist demo -pretty

This will:
- canonicalize and compute the CID,
- write `data/o/{cid}` (raw JSON-LD),
- write `data/o/canonical/{cid}.nq` (URDNA2015 normalized N-Quads),
- update `data/u/alice/g/demo/latest` with the CID,
- append an entry to `data/u/alice/g/demo/_history`.

Notes and next steps
- This is an intentionally small, dependency-light starter. Integrate into your HTTP server and apcore adapter by:
  - Replacing FSStore with your DB-backed store,
  - Calling SealJSONLD() as part of PUT /u/{login}/g/{slug}/index handling,
  - Recording provenance metadata (createdBy, createdAt, modelCID) in a separate metadata store or in the same container.
- Replace the storage layout with your preferred object-addressable store (IPFS, S3 keyed by CID, Postgres bytea keyed by CID).
- Consider adding signature generation (detached signature over canonical bytes) and include a `signature` block in a PetriNetSeal object. Signing keys should be stored in a secure KeyManager.
- Consider embedding the canonicalization output (or a digest) in ActivityPub objects when federating via apcore.

Security & determinism
- URDNA2015 normalization yields deterministic output given the same JSON-LD and context; ensure all contexts needed are available and stable (host local contexts at `/context/...` if necessary).
- Canonicalization depends on full expansion/resolution of @context. If external contexts are used, pin or cache them (or host your own versioned contexts).

Limitations
- The simple FSStore is not production-grade (no concurrency locks, no atomic updates). Replace with a transactional DB or object store for production.
- This code does not yet perform signature generation/verification. Add a signature sub-system that signs canonicalBytes and stores detached sig alongside the sealed object.
```
