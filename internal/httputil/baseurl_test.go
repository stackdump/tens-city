package httputil

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		setupReq    func(*http.Request)
		fallbackURL string
		want        string
	}{
		{
			name: "X-Forwarded-Proto and X-Forwarded-Host",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "X-Forwarded-Proto http with X-Forwarded-Host",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "http")
				req.Header.Set("X-Forwarded-Host", "example.com")
			},
			fallbackURL: "http://localhost:8080",
			want:        "http://example.com",
		},
		{
			name: "X-Forwarded-Scheme and X-Forwarded-Host",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Scheme", "https")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "X-Forwarded-Ssl on and X-Forwarded-Host",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Ssl", "on")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "X-Forwarded-Ssl off and X-Forwarded-Host",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Ssl", "off")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city", // Falls back to assuming https
		},
		{
			name: "Forwarded header with proto=https",
			setupReq: func(req *http.Request) {
				req.Header.Set("Forwarded", "proto=https;host=tens.city")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "Forwarded header with proto=http",
			setupReq: func(req *http.Request) {
				req.Header.Set("Forwarded", "proto=http;host=example.com")
				req.Header.Set("X-Forwarded-Host", "example.com")
			},
			fallbackURL: "http://localhost:8080",
			want:        "http://example.com",
		},
		{
			name: "X-Forwarded-Host alone assumes https",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "Host header with TLS",
			setupReq: func(req *http.Request) {
				req.Host = "tens.city"
				req.TLS = &tls.ConnectionState{}
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "Host header without TLS",
			setupReq: func(req *http.Request) {
				req.Host = "localhost:8080"
			},
			fallbackURL: "http://localhost:8080",
			want:        "http://localhost:8080",
		},
		{
			name: "Fallback URL when no headers",
			setupReq: func(req *http.Request) {
				req.Host = "" // Clear the default host set by httptest.NewRequest
			},
			fallbackURL: "http://localhost:8080",
			want:        "http://localhost:8080",
		},
		{
			name: "X-Forwarded-Proto takes precedence over X-Forwarded-Scheme",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
				req.Header.Set("X-Forwarded-Scheme", "http")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "X-Forwarded-Proto takes precedence over X-Forwarded-Ssl",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "http")
				req.Header.Set("X-Forwarded-Ssl", "on")
				req.Header.Set("X-Forwarded-Host", "example.com")
			},
			fallbackURL: "http://localhost:8080",
			want:        "http://example.com",
		},
		{
			name: "Forwarded header with spaces",
			setupReq: func(req *http.Request) {
				req.Header.Set("Forwarded", "proto=https; host=tens.city")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
		{
			name: "Forwarded header with uppercase proto",
			setupReq: func(req *http.Request) {
				req.Header.Set("Forwarded", "proto=HTTPS")
				req.Header.Set("X-Forwarded-Host", "tens.city")
			},
			fallbackURL: "http://localhost:8080",
			want:        "https://tens.city",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupReq(req)
			got := GetBaseURL(req, tt.fallbackURL)
			if got != tt.want {
				t.Errorf("GetBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProxyProtocol(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func(*http.Request)
		want     string
	}{
		{
			name: "X-Forwarded-Proto https",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
			},
			want: "https",
		},
		{
			name: "X-Forwarded-Proto http",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "http")
			},
			want: "http",
		},
		{
			name: "X-Forwarded-Scheme https",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Scheme", "https")
			},
			want: "https",
		},
		{
			name: "X-Forwarded-Ssl on",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Ssl", "on")
			},
			want: "https",
		},
		{
			name: "X-Forwarded-Ssl off",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Ssl", "off")
			},
			want: "",
		},
		{
			name: "Forwarded proto=https",
			setupReq: func(req *http.Request) {
				req.Header.Set("Forwarded", "proto=https")
			},
			want: "https",
		},
		{
			name: "Forwarded proto=http",
			setupReq: func(req *http.Request) {
				req.Header.Set("Forwarded", "proto=http")
			},
			want: "http",
		},
		{
			name: "Forwarded with multiple fields",
			setupReq: func(req *http.Request) {
				req.Header.Set("Forwarded", "for=192.0.2.60;proto=https;by=203.0.113.43")
			},
			want: "https",
		},
		{
			name:     "No proxy headers",
			setupReq: func(req *http.Request) {},
			want:     "",
		},
		{
			name: "X-Forwarded-Proto takes precedence",
			setupReq: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
				req.Header.Set("X-Forwarded-Scheme", "http")
				req.Header.Set("Forwarded", "proto=http")
			},
			want: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupReq(req)
			got := getProxyProtocol(req)
			if got != tt.want {
				t.Errorf("getProxyProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}
