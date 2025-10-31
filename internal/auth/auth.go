package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// GitHubUserInfo contains the GitHub user information from Supabase JWT
type GitHubUserInfo struct {
	ID       string `json:"sub"`           // Supabase user ID
	Email    string `json:"email"`         // User email
	UserName string `json:"user_name"`     // GitHub username from user_metadata
	FullName string `json:"full_name"`     // GitHub full name from user_metadata
	GitHubID string `json:"provider_id"`   // GitHub user ID
}

// ExtractUserFromToken extracts GitHub user information from a Supabase JWT token
// For now, we'll do basic extraction without full verification since this is a simple app
// In production, you'd verify the JWT signature against Supabase's public key
func ExtractUserFromToken(tokenString string) (*GitHubUserInfo, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	
	// Split the JWT into its parts (header.payload.signature)
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}
	
	// Decode the payload (second part)
	// JWT uses base64url encoding without padding
	payload := parts[1]
	// Add padding if needed
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	
	// Decode base64
	decoded, err := decodeBase64URL(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}
	
	// Parse JSON payload
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}
	
	// Extract user information
	userInfo := &GitHubUserInfo{}
	
	if sub, ok := claims["sub"].(string); ok {
		userInfo.ID = sub
	}
	
	if email, ok := claims["email"].(string); ok {
		userInfo.Email = email
	}
	
	// Extract user_metadata which contains GitHub info
	if userMetadata, ok := claims["user_metadata"].(map[string]interface{}); ok {
		if userName, ok := userMetadata["user_name"].(string); ok {
			userInfo.UserName = userName
		}
		if fullName, ok := userMetadata["full_name"].(string); ok {
			userInfo.FullName = fullName
		}
		if providerID, ok := userMetadata["provider_id"].(string); ok {
			userInfo.GitHubID = providerID
		}
	}
	
	// Also try app_metadata for provider info
	if appMetadata, ok := claims["app_metadata"].(map[string]interface{}); ok {
		if provider, ok := appMetadata["provider"].(string); ok {
			if provider != "github" {
				return nil, fmt.Errorf("unsupported provider: %s", provider)
			}
		}
	}
	
	// Validate we have at least some user identification
	if userInfo.ID == "" && userInfo.Email == "" && userInfo.UserName == "" {
		return nil, fmt.Errorf("no user identification found in token")
	}
	
	return userInfo, nil
}

// decodeBase64URL decodes base64url encoded string
func decodeBase64URL(s string) ([]byte, error) {
	// Replace URL-safe characters with standard base64 characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	
	// Standard base64 decode
	return base64.StdEncoding.DecodeString(s)
}
