# Authentication and Author Information

## Overview

The `/api/save` endpoint requires authentication using Supabase JWT tokens obtained via GitHub OAuth. When a user saves a JSON-LD object, their GitHub identity is automatically recorded in the saved object.

## Authentication Flow

### Frontend (JavaScript)

When saving an object, the frontend automatically includes the Supabase session token:

```javascript
const { data: { session } } = await this._supabase.auth.getSession();
const authToken = session?.access_token;

const response = await fetch('/api/save', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`,
    },
    body: canonicalData
});
```

### Backend (Go)

The server extracts user information from the JWT token and validates the request:

1. **Extract Token**: The `Authorization` header is parsed to extract the JWT token
2. **Decode JWT**: The token payload is base64-decoded and parsed
3. **Extract User Info**: GitHub user information is extracted from the token claims:
   - `sub`: Supabase user ID
   - `email`: User email address
   - `user_metadata.user_name`: GitHub username
   - `user_metadata.provider_id`: GitHub user ID

## Author Information Injection

When an object is saved, the server automatically injects author information into the JSON-LD:

```json
{
  "@context": "https://pflow.xyz/schema",
  "@id": "ipfs://z4EBG...",
  "@type": "YourObjectType",
  "author": {
    "@type": "Person",
    "name": "github-username",
    "identifier": "https://github.com/github-username",
    "id": "github:123456789"
  },
  ...your other fields...
}
```

The `author` field includes:
- `@type`: Always set to "Person"
- `name`: GitHub username (or email if username unavailable)
- `identifier`: GitHub profile URL (when username is available)
- `id`: GitHub user ID prefixed with "github:"

## Security Considerations

- **Token Validation**: The current implementation extracts user information from JWT tokens without full cryptographic verification. For production use, you should verify the token signature against Supabase's public key.
- **Authentication Required**: All save operations require a valid authentication token. Requests without authentication receive a 401 Unauthorized response.
- **User Attribution**: Author information is immutably recorded in each saved object, providing provenance and accountability.

## Testing

The authentication system includes comprehensive tests:

```bash
# Run all authentication tests
go test ./cmd/webserver -v -run TestAuth

# Test authentication requirements
go test ./cmd/webserver -v -run TestAuthenticationRequired

# Test invalid token rejection
go test ./cmd/webserver -v -run TestInvalidTokenRejected

# Test author info injection
go test ./cmd/webserver -v -run TestAuthorInfoInjection
```

## Example Usage

### Successful Save with Authentication

```bash
# Get Supabase session token from your authenticated frontend
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X POST http://localhost:8080/api/save \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "@context": "https://pflow.xyz/schema",
    "@type": "Example",
    "name": "My Object"
  }'
```

Response:
```json
{"cid": "z4EBG9j..."}
```

### Failed Save without Authentication

```bash
curl -X POST http://localhost:8080/api/save \
  -H "Content-Type: application/json" \
  -d '{
    "@context": "https://pflow.xyz/schema",
    "@type": "Example",
    "name": "My Object"
  }'
```

Response:
```
Authentication required
HTTP Status: 401
```

## Implementation Details

### New Components

1. **`internal/auth` package**: Handles JWT token parsing and user extraction
   - `ExtractUserFromToken()`: Extracts GitHub user info from Supabase JWT
   - `GitHubUserInfo`: Struct containing user identification data

2. **Updated `SaveObjectWithAuthor()` method**: Enhanced storage function that accepts author information
   - Falls back to `SaveObject()` for backwards compatibility
   - Injects author field into JSON-LD before saving

3. **Updated `handleSave()` handler**: Modified to require and validate authentication
   - Returns 401 for missing/invalid authentication
   - Extracts user info from token
   - Passes author info to storage layer

### Backwards Compatibility

The `SaveObject()` method remains available for internal use and testing, but the API endpoint now requires authentication. This ensures all user-facing saves include proper attribution.
