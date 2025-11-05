package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTextLogger_LogRequest(t *testing.T) {
	logger := NewTextLogger()
	req := httptest.NewRequest("GET", "/test", nil)

	// Should not panic
	logger.LogRequest(req, 200, time.Millisecond*100)
}

func TestTextLogger_LogError(t *testing.T) {
	logger := NewTextLogger()

	// Should not panic
	logger.LogError("test error", fmt.Errorf("something went wrong"))
}

func TestTextLogger_LogInfo(t *testing.T) {
	logger := NewTextLogger()

	// Should not panic
	logger.LogInfo("test info message")
}

func TestTextLogger_LogHeaders(t *testing.T) {
	logger := NewTextLogger()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("User-Agent", "Test-Agent")

	// Should not panic
	logger.LogHeaders(req)
}

func TestJSONLLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLLogger(&buf)
	req := httptest.NewRequest("GET", "/test/path", nil)

	logger.LogRequest(req, 200, time.Millisecond*100)

	// Parse the JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify fields
	if entry.Level != "info" {
		t.Errorf("Expected level 'info', got '%s'", entry.Level)
	}
	if entry.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", entry.Method)
	}
	if entry.Path != "/test/path" {
		t.Errorf("Expected path '/test/path', got '%s'", entry.Path)
	}
	if entry.Status != 200 {
		t.Errorf("Expected status 200, got %d", entry.Status)
	}
	if entry.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestJSONLLogger_LogError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLLogger(&buf)

	logger.LogError("test error", fmt.Errorf("something went wrong"))

	// Parse the JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify fields
	if entry.Level != "error" {
		t.Errorf("Expected level 'error', got '%s'", entry.Level)
	}
	if entry.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", entry.Message)
	}
	if entry.Error != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got '%s'", entry.Error)
	}
}

func TestJSONLLogger_LogInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLLogger(&buf)

	logger.LogInfo("test info message")

	// Parse the JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify fields
	if entry.Level != "info" {
		t.Errorf("Expected level 'info', got '%s'", entry.Level)
	}
	if entry.Message != "test info message" {
		t.Errorf("Expected message 'test info message', got '%s'", entry.Message)
	}
}

func TestJSONLLogger_LogHeaders(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLLogger(&buf)
	req := httptest.NewRequest("GET", "/rss", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "example.com")
	req.Header.Add("Accept", "application/xml")

	logger.LogHeaders(req)

	// Parse the JSON output
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify fields
	if entry.Level != "debug" {
		t.Errorf("Expected level 'debug', got '%s'", entry.Level)
	}
	if entry.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", entry.Method)
	}
	if entry.Path != "/rss" {
		t.Errorf("Expected path '/rss', got '%s'", entry.Path)
	}
	if entry.Headers == nil {
		t.Fatal("Expected headers to be set")
	}

	// Check specific headers
	if proto, ok := entry.Headers["X-Forwarded-Proto"].(string); !ok || proto != "https" {
		t.Errorf("Expected X-Forwarded-Proto to be 'https', got %v", entry.Headers["X-Forwarded-Proto"])
	}
	if host, ok := entry.Headers["X-Forwarded-Host"].(string); !ok || host != "example.com" {
		t.Errorf("Expected X-Forwarded-Host to be 'example.com', got %v", entry.Headers["X-Forwarded-Host"])
	}
}

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		logHeaders bool
		method     string
		path       string
		headers    map[string]string
	}{
		{
			name:       "Basic request without header logging",
			logHeaders: false,
			method:     "GET",
			path:       "/test",
			headers:    nil,
		},
		{
			name:       "Request with header logging",
			logHeaders: true,
			method:     "GET",
			path:       "/posts.rss",
			headers: map[string]string{
				"X-Forwarded-Proto": "https",
				"X-Forwarded-Host":  "example.com",
			},
		},
		{
			name:       "POST request",
			logHeaders: false,
			method:     "POST",
			path:       "/api/test",
			headers:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewJSONLLogger(&buf)

			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with logging middleware
			middleware := LoggingMiddleware(logger, tt.logHeaders)
			wrappedHandler := middleware(handler)

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.headers != nil {
				for k, v := range tt.headers {
					req.Header.Set(k, v)
				}
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(rr, req)

			// Verify response
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			// Parse log output
			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			
			// Should have at least 1 line (request log)
			if len(lines) < 1 {
				t.Fatal("Expected at least one log line")
			}

			// If headers logging is enabled, should have 2 lines (headers + request)
			if tt.logHeaders && len(lines) < 2 {
				t.Fatal("Expected at least two log lines when header logging is enabled")
			}

			// Parse the request log entry (last line)
			var requestEntry LogEntry
			lastLine := lines[len(lines)-1]
			if err := json.Unmarshal([]byte(lastLine), &requestEntry); err != nil {
				t.Fatalf("Failed to parse request log entry: %v", err)
			}

			// Verify request log
			if requestEntry.Method != tt.method {
				t.Errorf("Expected method '%s', got '%s'", tt.method, requestEntry.Method)
			}
			if requestEntry.Path != tt.path {
				t.Errorf("Expected path '%s', got '%s'", tt.path, requestEntry.Path)
			}
			if requestEntry.Status != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, requestEntry.Status)
			}

			// If headers logging is enabled, verify headers log
			if tt.logHeaders {
				var headersEntry LogEntry
				if err := json.Unmarshal([]byte(lines[0]), &headersEntry); err != nil {
					t.Fatalf("Failed to parse headers log entry: %v", err)
				}

				if headersEntry.Level != "debug" {
					t.Errorf("Expected headers log level 'debug', got '%s'", headersEntry.Level)
				}
				if headersEntry.Headers == nil {
					t.Error("Expected headers to be logged")
				}
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	// Write header
	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rw.statusCode)
	}

	// Verify the underlying response writer got the status
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected underlying response writer to have status %d, got %d", http.StatusNotFound, rr.Code)
	}
}
