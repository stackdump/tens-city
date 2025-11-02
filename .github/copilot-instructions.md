# Copilot Instructions for tens-city

## Project Overview

tens-city is a minimal, filesystem-based storage system for JSON-LD objects with stable CID (Content Identifier) addressing. The project follows a "tent city" philosophy: minimal structure, maximum stability, no unnecessary governance.

## Technology Stack

- **Language**: Go 1.24+
- **Key Dependencies**:
  - `github.com/ethereum/go-ethereum` - Ethereum key management and signing
  - `github.com/golang-jwt/jwt/v5` - JWT authentication (Supabase)
  - `github.com/ipfs/go-cid` - Content addressing (CIDv1)
  - `github.com/piprate/json-gold` - JSON-LD canonicalization (URDNA2015)

## Architecture

### Core Components

1. **webserver** (`cmd/webserver/`) - HTTP server providing:
   - API for saving/retrieving JSON-LD objects by CID
   - GitHub OAuth authentication via Supabase JWT
   - Static file serving for the web application
   - Filesystem-based storage backend

2. **seal** (`cmd/seal/`) - CLI tool for:
   - Canonicalizing JSON-LD documents (URDNA2015)
   - Computing CIDv1 identifiers
   - Optional Ethereum signature support

3. **keygen** (`cmd/keygen/`) - Ethereum keystore generation

### Internal Packages

- `internal/auth` - Supabase JWT verification and user info extraction
- `internal/canonical` - Canonical JSON encoding with sorted keys
- `internal/ethsig` - Ethereum wallet management and signing
- `internal/seal` - CID computation and JSON-LD sealing
- `internal/store` - Filesystem-based object storage
- `internal/static` - Embedded static assets

## Code Style & Conventions

### Go Conventions
- Follow standard Go conventions and `gofmt` formatting
- Use descriptive variable names (e.g., `cid`, `userInfo`, `jsonldData`)
- Keep functions focused and single-purpose
- Prefer explicit error handling over panics
- Use table-driven tests where appropriate

### Project-Specific Patterns

1. **Error Handling**: Always return errors explicitly; use HTTP status codes appropriately (400 for client errors, 500 for server errors)

2. **CID Generation**: CIDs must be deterministic - same content always produces same CID. Use canonical JSON encoding before hashing.

3. **Authentication**: 
   - JWT tokens verified via `SUPABASE_JWT_SECRET` environment variable
   - Extract user info from `app_metadata.provider` for GitHub details
   - Author verification required for delete operations

4. **Storage**: 
   - Objects stored as `{cid}.json` files
   - Directory structure: `store/{cid[:2]}/{cid[2:4]}/{cid}.json`
   - User gists: `store/u/{user}/g/{slug}/latest` → CID reference

5. **JSON-LD Validation**: 
   - Require `@context` field for valid JSON-LD
   - Support embedded `<script type="application/ld+json">` tags
   - Preserve object structure during canonicalization

## Building and Testing

### Build Commands
```bash
# Build all commands
go build ./cmd/...

# Build specific tools
go build -o seal ./cmd/seal
go build -o webserver ./cmd/webserver
go build -o keygen ./cmd/keygen
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/seal/
go test ./cmd/webserver/
```

### Running the Application
```bash
# Set required environment variable
export SUPABASE_JWT_SECRET="your-jwt-secret"

# Start web server
./webserver -addr :8080 -store data -public public

# Seal a JSON-LD document
./seal -in examples/petrinet.jsonld -store data
```

## Security Considerations

1. **JWT Verification**: Always validate JWT signatures using the Supabase secret
2. **Input Validation**: 
   - Validate JSON-LD structure before processing
   - Enforce content size limits (default: 1MB)
   - Sanitize file paths to prevent directory traversal
3. **Author Verification**: Verify object ownership before allowing deletions
4. **No Secrets in Code**: Use environment variables for sensitive configuration

## File Organization

- `cmd/` - Main applications (webserver, seal, keygen)
- `internal/` - Internal packages (not importable by external projects)
- `public/` - Static web application files
- `examples/` - Example JSON-LD documents and workflows
- `docs/` - Additional documentation

## Dependencies Management

- Use `go mod` for dependency management
- Keep dependencies minimal and well-justified
- Update dependencies carefully, testing thoroughly after updates
- Check for security vulnerabilities before adding new dependencies

## Common Tasks

### Adding New API Endpoints
1. Add handler function in `cmd/webserver/`
2. Register route in `main.go`
3. Add authentication middleware if needed
4. Add tests in `cmd/webserver/*_test.go`
5. Update README.md with new endpoint documentation

### Adding New Storage Features
1. Modify `internal/store/store.go`
2. Ensure filesystem operations are atomic
3. Add tests in `internal/store/store_test.go`
4. Consider backward compatibility with existing stored objects

### Modifying Seal/CID Logic
1. Update `internal/seal/seal.go`
2. Ensure changes maintain CID determinism
3. Add consistency tests in `internal/seal/cid_consistency_test.go`
4. Verify backward compatibility with existing CIDs

## Testing Philosophy

- Unit tests for all internal packages
- Integration tests for API endpoints
- CID consistency tests to ensure determinism
- Mock Supabase JWT tokens for auth testing
- Test both success and error paths

## Documentation

- Keep README.md up to date with API changes
- Document all exported functions and types
- Add examples for complex functionality
- Update TESTING.md for new features requiring manual testing

## Preferred Approaches

- **Filesystem operations**: Use Go's standard library (`os`, `filepath`)
- **HTTP handling**: Use standard `net/http` package
- **JSON processing**: Use `encoding/json` for standard JSON, `piprate/json-gold` for JSON-LD
- **Testing**: Use standard `testing` package, table-driven tests
- **Configuration**: Environment variables over config files

## Things to Avoid

- Adding unnecessary dependencies
- Complex abstractions when simple code suffices
- Breaking CID determinism
- Storing secrets or credentials in code or repository
- Modifying core sealing logic without extensive testing
- Adding stateful features that conflict with the filesystem-based architecture

## Questions to Consider

When making changes, ask:
1. Does this maintain CID determinism?
2. Is the filesystem storage still simple and direct?
3. Are we maintaining backward compatibility?
4. Have we added appropriate tests?
5. Is the authentication/authorization secure?
6. Does this align with the "tent city" minimalist philosophy?
