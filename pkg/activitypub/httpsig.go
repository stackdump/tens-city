package activitypub

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SignedPost sends a POST request with HTTP signature authentication
func (a *Actor) SignedPost(targetURL string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Accept", "application/activity+json")
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("Host", req.URL.Host)

	// Calculate digest
	digest := sha256.Sum256(body)
	digestHeader := "SHA-256=" + base64.StdEncoding.EncodeToString(digest[:])
	req.Header.Set("Digest", digestHeader)

	// Sign the request
	if err := a.signRequest(req); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// signRequest adds an HTTP signature to the request
func (a *Actor) signRequest(req *http.Request) error {
	// Headers to sign (order matters)
	signedHeaders := []string{"(request-target)", "host", "date", "digest", "content-type"}

	// Build the signing string
	var signingParts []string
	for _, header := range signedHeaders {
		var value string
		if header == "(request-target)" {
			value = fmt.Sprintf("%s %s", strings.ToLower(req.Method), req.URL.RequestURI())
		} else {
			value = req.Header.Get(header)
		}
		signingParts = append(signingParts, fmt.Sprintf("%s: %s", header, value))
	}
	signingString := strings.Join(signingParts, "\n")

	// Sign with RSA-SHA256
	hashed := sha256.Sum256([]byte(signingString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, a.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// Build signature header
	signatureHeader := fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`,
		a.KeyID(),
		strings.Join(signedHeaders, " "),
		base64.StdEncoding.EncodeToString(signature),
	)

	req.Header.Set("Signature", signatureHeader)
	return nil
}

// SignedGet sends a GET request with HTTP signature authentication
func (a *Actor) SignedGet(targetURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("Accept", "application/activity+json")
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Set("Host", req.URL.Host)

	// Sign the request (GET doesn't need digest)
	if err := a.signGetRequest(req); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

// signGetRequest adds an HTTP signature to a GET request
func (a *Actor) signGetRequest(req *http.Request) error {
	signedHeaders := []string{"(request-target)", "host", "date", "accept"}

	var signingParts []string
	for _, header := range signedHeaders {
		var value string
		if header == "(request-target)" {
			value = fmt.Sprintf("%s %s", strings.ToLower(req.Method), req.URL.RequestURI())
		} else {
			value = req.Header.Get(header)
		}
		signingParts = append(signingParts, fmt.Sprintf("%s: %s", header, value))
	}
	signingString := strings.Join(signingParts, "\n")

	hashed := sha256.Sum256([]byte(signingString))
	signature, err := rsa.SignPKCS1v15(rand.Reader, a.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	signatureHeader := fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`,
		a.KeyID(),
		strings.Join(signedHeaders, " "),
		base64.StdEncoding.EncodeToString(signature),
	)

	req.Header.Set("Signature", signatureHeader)
	return nil
}

// FetchActorInbox fetches a remote actor and returns their inbox URL
func (a *Actor) FetchActorInbox(actorURL string) (string, error) {
	resp, err := a.SignedGet(actorURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch actor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("actor fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read actor response: %w", err)
	}

	// Parse to find inbox
	// Simple JSON parsing without external dependencies
	inbox := extractJSONString(body, "inbox")
	if inbox == "" {
		return "", fmt.Errorf("no inbox found in actor response")
	}

	return inbox, nil
}

// extractJSONString extracts a string value from JSON (simple implementation)
func extractJSONString(data []byte, key string) string {
	// Look for "key": "value" pattern
	searchKey := fmt.Sprintf(`"%s"`, key)
	idx := bytes.Index(data, []byte(searchKey))
	if idx == -1 {
		return ""
	}

	// Find the colon after the key
	rest := data[idx+len(searchKey):]
	colonIdx := bytes.IndexByte(rest, ':')
	if colonIdx == -1 {
		return ""
	}

	// Find the opening quote
	rest = rest[colonIdx+1:]
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t' || rest[0] == '\n') {
		rest = rest[1:]
	}
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]

	// Find the closing quote
	endIdx := bytes.IndexByte(rest, '"')
	if endIdx == -1 {
		return ""
	}

	return string(rest[:endIdx])
}
