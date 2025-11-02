# Canonical JSON Encoding

## Overview

Tens City uses canonical JSON encoding to ensure consistent CID (Content Identifier) generation for JSON-LD documents. This guarantees that the same semantic content always produces the same CID, regardless of the order of keys in the JSON object.

## Why Canonical JSON?

JSON objects can have their keys in any order, and both JavaScript's `JSON.stringify()` and Go's `json.Marshal()` don't guarantee consistent key ordering. This means:

```javascript
// These are semantically identical but serialize differently:
{"name": "Alice", "age": 30}
{"age": 30, "name": "Alice"}
```

Without canonical encoding, these would produce different CIDs even though they represent the same data.

## Implementation

### Backend (Go)

The `internal/canonical` package provides `MarshalJSON()` which:
1. Recursively traverses the JSON structure
2. Sorts object keys alphabetically
3. Produces deterministic output

```go
import "github.com/stackdump/tens-city/internal/canonical"

canonical, err := canonical.MarshalJSON(data)
// Result: {"age":30,"name":"Alice"}
```

### Frontend (JavaScript)

The `canonicalJSON()` function in `public/tens-city.js`:
1. Recursively traverses the JSON structure
2. Sorts object keys alphabetically
3. Produces deterministic output

```javascript
const canonical = canonicalJSON(data);
// Result: {"age":30,"name":"Alice"}
```

## Usage in Tens City

### Permalink Generation

When you click the "🔗 Permalink" button:
1. Editor content is parsed as JSON
2. `canonicalJSON()` converts it to canonical form
3. The canonical JSON is URL-encoded
4. Result: consistent permalink URLs

### Auto-Save

When visiting a `?data=...` URL:
1. Data is decoded and parsed
2. `canonicalJSON()` converts it to canonical form
3. Canonical JSON is sent to `/api/save`
4. Backend uses `canonical.MarshalJSON()` before computing CID
5. Result: consistent CID generation

### API Endpoint

The `/api/save` endpoint:
1. Receives JSON-LD document
2. Uses `canonical.MarshalJSON()` to serialize
3. Passes canonical bytes to URDNA2015 normalization
4. Computes CID from normalized RDF
5. Result: same JSON always produces same CID

## Benefits

1. **Deterministic CIDs**: Same content always produces the same identifier
2. **Shareable Links**: Permalink URLs always reference the same CID
3. **Content Addressing**: CIDs accurately represent content, not formatting
4. **Cross-Platform**: Frontend and backend produce matching results

## Example

```javascript
// Different key orders, same canonical output
const obj1 = {"z": 3, "a": 1, "m": 2};
const obj2 = {"a": 1, "m": 2, "z": 3};

canonicalJSON(obj1); // '{"a":1,"m":2,"z":3}'
canonicalJSON(obj2); // '{"a":1,"m":2,"z":3}'

// Both produce the same CID when saved
```

## Testing

Run tests to verify canonical behavior:

```bash
# Backend tests
go test ./internal/canonical/... -v
go test ./cmd/webserver/... -v -run Canonical

# Full test suite
go test ./... -v
```

## Technical Details

- **Algorithm**: Alphabetic key sorting (ASCII order)
- **Recursion**: Nested objects and arrays are handled recursively
- **Consistency**: Both frontend and backend use identical algorithms
- **Integration**: Works seamlessly with URDNA2015 RDF canonicalization
