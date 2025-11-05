package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Logger interface for flexible logging
type Logger interface {
	LogRequest(r *http.Request, status int, duration time.Duration)
	LogError(msg string, err error)
	LogInfo(msg string)
	LogHeaders(r *http.Request)
}

// TextLogger logs in traditional text format
type TextLogger struct {
	logger *log.Logger
}

// JSONLLogger logs in JSON Lines format
type JSONLLogger struct {
	writer io.Writer
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message,omitempty"`
	Method    string                 `json:"method,omitempty"`
	Path      string                 `json:"path,omitempty"`
	Status    int                    `json:"status,omitempty"`
	Duration  string                 `json:"duration,omitempty"`
	Headers   map[string]interface{} `json:"headers,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// NewTextLogger creates a new text logger
func NewTextLogger() *TextLogger {
	return &TextLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// NewJSONLLogger creates a new JSONL logger
func NewJSONLLogger(w io.Writer) *JSONLLogger {
	if w == nil {
		w = os.Stdout
	}
	return &JSONLLogger{
		writer: w,
	}
}

// LogRequest logs an HTTP request with text format
func (l *TextLogger) LogRequest(r *http.Request, status int, duration time.Duration) {
	l.logger.Printf("%s %s - %d - %v", r.Method, r.URL.Path, status, duration)
}

// LogError logs an error with text format
func (l *TextLogger) LogError(msg string, err error) {
	l.logger.Printf("ERROR: %s: %v", msg, err)
}

// LogInfo logs an informational message with text format
func (l *TextLogger) LogInfo(msg string) {
	l.logger.Printf("INFO: %s", msg)
}

// LogHeaders logs request headers with text format
func (l *TextLogger) LogHeaders(r *http.Request) {
	l.logger.Printf("Headers for %s %s:", r.Method, r.URL.Path)
	for name, values := range r.Header {
		for _, value := range values {
			l.logger.Printf("  %s: %s", name, value)
		}
	}
}

// LogRequest logs an HTTP request with JSONL format
func (l *JSONLLogger) LogRequest(r *http.Request, status int, duration time.Duration) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "info",
		Method:    r.Method,
		Path:      r.URL.Path,
		Status:    status,
		Duration:  duration.String(),
	}
	l.writeEntry(entry)
}

// LogError logs an error with JSONL format
func (l *JSONLLogger) LogError(msg string, err error) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "error",
		Message:   msg,
		Error:     err.Error(),
	}
	l.writeEntry(entry)
}

// LogInfo logs an informational message with JSONL format
func (l *JSONLLogger) LogInfo(msg string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "info",
		Message:   msg,
	}
	l.writeEntry(entry)
}

// LogHeaders logs request headers with JSONL format
func (l *JSONLLogger) LogHeaders(r *http.Request) {
	headers := make(map[string]interface{})
	for name, values := range r.Header {
		if len(values) == 1 {
			headers[name] = values[0]
		} else {
			headers[name] = values
		}
	}
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "debug",
		Message:   fmt.Sprintf("Headers for %s %s", r.Method, r.URL.Path),
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   headers,
	}
	l.writeEntry(entry)
}

// writeEntry writes a log entry as JSON
func (l *JSONLLogger) writeEntry(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}
	fmt.Fprintf(l.writer, "%s\n", data)
}

// LoggingMiddleware creates a middleware that logs requests
func LoggingMiddleware(logger Logger, logHeaders bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Log headers if enabled (useful for debugging RSS http/https behavior)
			if logHeaders {
				logger.LogHeaders(r)
			}

			// Wrap the response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			// Log the request
			duration := time.Since(start)
			logger.LogRequest(r, wrapped.statusCode, duration)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
