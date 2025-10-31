package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractUserFromToken(t *testing.T) {
	// Create a mock JWT payload with GitHub user info
	payload := map[string]interface{}{
		"sub":   "user-id-123",
		"email": "test@example.com",
		"user_metadata": map[string]interface{}{
			"user_name":   "testuser",
			"full_name":   "Test User",
			"provider_id": "12345678",
		},
		"app_metadata": map[string]interface{}{
			"provider": "github",
		},
	}
	
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	
	// Create a mock JWT (header.payload.signature)
	// We don't verify signature in this implementation, so we can use dummy values
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("dummy-signature"))
	
	token := strings.Join([]string{header, payloadB64, signature}, ".")
	
	// Test extraction
	userInfo, err := ExtractUserFromToken(token)
	if err != nil {
		t.Fatalf("Failed to extract user: %v", err)
	}
	
	// Validate extracted information
	if userInfo.ID != "user-id-123" {
		t.Errorf("Expected ID 'user-id-123', got '%s'", userInfo.ID)
	}
	
	if userInfo.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", userInfo.Email)
	}
	
	if userInfo.UserName != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", userInfo.UserName)
	}
	
	if userInfo.FullName != "Test User" {
		t.Errorf("Expected full name 'Test User', got '%s'", userInfo.FullName)
	}
	
	if userInfo.GitHubID != "12345678" {
		t.Errorf("Expected GitHub ID '12345678', got '%s'", userInfo.GitHubID)
	}
}

func TestExtractUserFromTokenWithBearer(t *testing.T) {
	// Create a simple token
	payload := map[string]interface{}{
		"sub":   "user-id-456",
		"email": "user@test.com",
	}
	
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("dummy-signature"))
	
	token := "Bearer " + strings.Join([]string{header, payloadB64, signature}, ".")
	
	// Test extraction with Bearer prefix
	userInfo, err := ExtractUserFromToken(token)
	if err != nil {
		t.Fatalf("Failed to extract user from Bearer token: %v", err)
	}
	
	if userInfo.ID != "user-id-456" {
		t.Errorf("Expected ID 'user-id-456', got '%s'", userInfo.ID)
	}
}

func TestExtractUserFromTokenInvalid(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"Empty token", ""},
		{"Invalid format", "not.a.valid.jwt.token"},
		{"Only two parts", "header.payload"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractUserFromToken(tt.token)
			if err == nil {
				t.Error("Expected error for invalid token, got nil")
			}
		})
	}
}

func TestExtractUserFromTokenNoUserInfo(t *testing.T) {
	// Create a token with no user identification
	payload := map[string]interface{}{
		"iat": 1234567890,
	}
	
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signature := base64.RawURLEncoding.EncodeToString([]byte("dummy-signature"))
	
	token := strings.Join([]string{header, payloadB64, signature}, ".")
	
	_, err := ExtractUserFromToken(token)
	if err == nil {
		t.Error("Expected error for token with no user info, got nil")
	}
}
