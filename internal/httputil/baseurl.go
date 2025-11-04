package httputil

import (
	"fmt"
	"net/http"
)

// GetBaseURL extracts the base URL from request headers (for nginx proxy) or falls back to configured URL
// It checks headers in the following order:
// 1. X-Forwarded-Proto + X-Forwarded-Host (nginx proxy headers)
// 2. X-Forwarded-Host alone (assumes https)
// 3. Host header (with scheme based on TLS)
// 4. Falls back to the provided fallbackURL
func GetBaseURL(r *http.Request, fallbackURL string) string {
	// Check for X-Forwarded-Proto and X-Forwarded-Host headers from nginx
	proto := r.Header.Get("X-Forwarded-Proto")
	host := r.Header.Get("X-Forwarded-Host")

	// If both headers are present, construct the base URL
	if proto != "" && host != "" {
		return fmt.Sprintf("%s://%s", proto, host)
	}

	// Check for X-Forwarded-Host alone (assume https if proto not specified)
	if host != "" {
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
