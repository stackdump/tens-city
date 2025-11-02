# tens-city - Copilot Instructions

This document provides context and guidelines for GitHub Copilot when working on the tens-city project.

## Project Overview

tens-city is a content-addressable storage system for JSON-LD documents, built in Go. The project combines:
- **Filesystem-based storage** - Immutable JSON-LD objects stored by CID
- **URDNA2015 canonicalization** - Deterministic CID generation
- **Ethereum signing** - Optional cryptographic signatures
- **Web server with authentication** - JWT-based access via Supabase
- **Markdown documentation system** - Automatic schema.org JSON-LD mapping

## Technology Stack

- **Language**: Go 1.24+
- **Key Dependencies**:
  - `github.com/piprate/json-gold` - JSON-LD processing and canonicalization
  - `github.com/ipfs/go-cid` - Content addressing (CIDv1)
  - `github.com/ethereum/go-ethereum` - Ethereum key management
  - `github.com/golang-jwt/jwt/v5` - JWT authentication
  - `github.com/yuin/goldmark` - Markdown processing
  - `github.com/microcosm-cc/bluemonday` - HTML sanitization

## Architecture Principles

### Content Addressing
- All JSON-LD objects are identified by their CID (Content Identifier)
- CIDs are computed deterministically from canonicalized content
- Same logical content always produces the same CID
- Storage is immutable once written

### Authentication & Authorization
- JWT tokens from Supabase authenticate users
- Author information (GitHub ID, username) is extracted from tokens
- Ownership is enforced server-side for delete operations
- Never trust client-provided ownership claims

### Storage Model
- Filesystem-based: `{store-dir}/o/{cid}.json`
- Signatures (optional): `{store-dir}/o/{cid}.sig.json`
- User gists: `{store-dir}/u/{user}/g/{slug}/latest` → CID
- History: `{store-dir}/u/{user}/g/{slug}/_history.jsonl`

## Development Workflow

### Build System
Use the Makefile for all common operations:
- `make build` - Build all binaries
- `make test` - Run tests
- `make check` - Format, vet, and test
- `make dev` - Full development workflow

### Code Quality Requirements
- All Go code must pass `go fmt`
- All code must pass `go vet`
- Tests must pass before commits
- No security vulnerabilities in dependencies
- Follow existing code patterns

### Testing Standards
- Write tests for new functionality
- Use table-driven tests for multiple scenarios
- Ensure tests are deterministic
- Mock external dependencies
- Test error paths, not just happy paths

## Code Patterns to Follow

### Error Handling
```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process object: %w", err)
}

// Good: Check specific error types
if errors.Is(err, ErrNotFound) {
    // Handle not found
}
```

### Storage Operations
```go
// Always use the store package
cid, err := store.SaveObject(dir, data)
if err != nil {
    return err
}

// Paths are sanitized automatically
data, err := store.ReadObject(dir, cid)
```

### Authentication
```go
// Extract user info from validated token
claims := token.Claims.(jwt.MapClaims)
userInfo := auth.ExtractUserInfo(claims)
```

## Security Guidelines

### Never Do
- ❌ Log JWT secrets or private keys
- ❌ Trust client-provided ownership information
- ❌ Allow unsanitized file paths
- ❌ Skip input validation
- ❌ Commit secrets to the repository

### Always Do
- ✅ Validate all user inputs
- ✅ Check authentication before privileged operations
- ✅ Use constant-time comparisons for sensitive data
- ✅ Set appropriate file permissions (0600 for keys)
- ✅ Use environment variables for secrets

## Common Tasks

### Adding a New HTTP Endpoint
1. Define handler function in `cmd/webserver/`
2. Add route in main server setup
3. Add authentication check if needed
4. Write handler tests in `*_test.go`
5. Update README.md with API documentation

### Modifying Storage Format
1. Consider backward compatibility
2. Update read/write functions in `internal/store`
3. Add migration logic if needed
4. Update tests
5. Document the change

### Adding Dependencies
1. Check if functionality exists in stdlib first
2. Run `go get package@version`
3. Run `go mod tidy`
4. Update tests
5. Document new dependency in relevant files

## File Organization

```
tens-city/
├── cmd/                  # Command-line applications
│   ├── seal/            # CID sealing tool
│   ├── keygen/          # Ethereum key generation
│   └── webserver/       # HTTP server
├── internal/            # Private packages
│   ├── auth/           # JWT authentication
│   ├── canonical/      # Deterministic JSON
│   ├── docserver/      # Documentation serving
│   ├── ethsig/         # Ethereum signing
│   ├── markdown/       # Markdown processing
│   ├── seal/           # CID generation
│   ├── static/         # Static file serving
│   └── store/          # Filesystem storage
├── content/            # Content files
├── docs/               # Documentation
├── examples/           # Example files
├── public/             # Web application files
├── schemas/            # JSON schemas
└── Makefile           # Build automation
```

## Documentation Requirements

When making changes:
- Update README.md for user-facing changes
- Add Go doc comments for exported items
- Update API documentation for endpoint changes
- Keep examples current and tested
- Document configuration options

## Resources

- [Project README](../README.md)
- [Markdown Documentation Guide](../docs/markdown-docs.md)
- [JSON-LD Script Tag Features](../docs/jsonld-script-tag.md)
- [Testing Guide](../TESTING.md)
- [Makefile Help](../Makefile) - Run `make help`

## Working with Copilot Agents

This project has specialized agents available:
- **go-expert** - Go language and codebase expertise
- **jsonld-specialist** - JSON-LD and schema.org knowledge
- **test-specialist** - Testing patterns and coverage
- **documentation-expert** - Documentation standards

Invoke these agents for specialized tasks in their domains. They have detailed knowledge of project patterns and requirements.
