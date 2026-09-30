package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// requestIDKey is the context key holding the request ID.
type requestIDKey struct{}

// Chain wraps the handler with middleware in registration order.
func Chain(
	handler http.Handler,
	middleware ...Middleware,
) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	return handler
}

// requestID assigns a unique ID to each request, honoring an incoming
// X-Request-Id header, and exposes it on the response and context.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = newRequestID()
		}

		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newRequestID generates a random hex request ID.
func newRequestID() string {
	var id [16]byte
	_, _ = rand.Read(id[:])

	return hex.EncodeToString(id[:])
}

// recoverer converts handler panics into generic 500 responses.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(
					r.Context(), "panic recovered", "error", recovered,
				)
				WriteJSON(w, http.StatusInternalServerError, &Error{
					StatusCode: http.StatusInternalServerError,
					Message:    "Internal server error",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// statusWriter captures the response status and size for logging.
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

// WriteHeader records the status before delegating.
func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write records the response size before delegating.
func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	n, err := w.ResponseWriter.Write(data)
	w.bytes += n

	return n, err
}

// requestLogger emits one structured log line per completed request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(wrapped, r)

		status := wrapped.status
		if status == 0 {
			status = http.StatusOK
		}

		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", wrapped.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", r.Context().Value(requestIDKey{}),
		)
	})
}

// cors allows the single credentialed browser origin, matching the legacy
// Express CORS configuration. Other origins receive no CORS headers.
func cors(clientURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Origin") != clientURL {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Origin", clientURL)
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			isPreflight := r.Method == http.MethodOptions &&
				r.Header.Get("Access-Control-Request-Method") != ""
			if isPreflight {
				w.Header().Set(
					"Access-Control-Allow-Methods",
					"GET,POST,PUT,PATCH,DELETE,OPTIONS",
				)
				w.Header().Set(
					"Access-Control-Allow-Headers",
					"Accept,Authorization,Content-Type,X-CSRF-Token",
				)
				w.Header().Set("Access-Control-Max-Age", "300")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
