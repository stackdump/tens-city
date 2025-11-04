package httputil

import (
	"fmt"
	"net/http"
	"strings"
)

// getProxyProtocol detects the protocol from various proxy headers
// Returns the protocol (http/https) or empty string if not detected
func getProxyProtocol(r *http.Request) string {
	// Check X-Forwarded-Proto (most common)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}

	// Check X-Forwarded-Scheme (alternative to X-Forwarded-Proto)
	if scheme := r.Header.Get("X-Forwarded-Scheme"); scheme != "" {
		return scheme
	}

	// Check X-Forwarded-Ssl (on/off indicator)
	if ssl := r.Header.Get("X-Forwarded-Ssl"); ssl == "on" {
		return "https"
	}

	// Check standard Forwarded header (RFC 7239)
	if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
		// Parse "proto=https" or "proto=http" from Forwarded header
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "proto=") {
				proto := strings.TrimPrefix(part, "proto=")
				return strings.ToLower(proto)
			}
		}
	}

	return ""
}

// GetBaseURL extracts the base URL from request headers (for nginx proxy) or falls back to configured URL
// It checks headers in the following order:
// 1. X-Forwarded-Host with protocol from proxy headers (X-Forwarded-Proto, X-Forwarded-Scheme, X-Forwarded-Ssl, Forwarded)
// 2. X-Forwarded-Host alone (assumes https if protocol not detected)
// 3. Host header (with scheme based on TLS)
// 4. Falls back to the provided fallbackURL
func GetBaseURL(r *http.Request, fallbackURL string) string {
	host := r.Header.Get("X-Forwarded-Host")

	// If X-Forwarded-Host is present, try to detect protocol from proxy headers
	if host != "" {
		proto := getProxyProtocol(r)
		if proto != "" {
			return fmt.Sprintf("%s://%s", proto, host)
		}
		// Assume https if protocol not detected but host is forwarded
		return fmt.Sprintf("https://%s", host)
	}

	// Check the Host header as fallback
	if r.Host != "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	// Use configured fallback URL
	return fallbackURL
}
