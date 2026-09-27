package web

import (
	"context"
	"net/http"
)

// ReadyProbe reports whether a dependency is ready to serve traffic.
type ReadyProbe func(ctx context.Context) error

// NewRouter builds the root router with global middleware and the
// infrastructure endpoints. Feature routes are registered on the mux by
// the caller using Go 1.22+ method and wildcard patterns.
func NewRouter(clientURL string, ready ReadyProbe) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("Hello World!"))
	})
	mux.HandleFunc(
		"GET /health/live",
		func(w http.ResponseWriter, _ *http.Request) {
			WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		},
	)
	mux.HandleFunc(
		"GET /health/ready",
		func(w http.ResponseWriter, r *http.Request) {
			if ready != nil {
				if err := ready(r.Context()); err != nil {
					WriteJSON(
						w,
						http.StatusServiceUnavailable,
						map[string]string{"status": "unavailable"},
					)
					return
				}
			}
			WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		},
	)

	// Catch-all: reproduce the legacy Express 404 body for unknown routes
	// and unsupported methods instead of the default Go 404/405 bodies.
	mux.HandleFunc("/", notFoundHandler)

	return chain(mux, requestID, cors(clientURL), requestLogger, recoverer)
}

// notFoundHandler reproduces the legacy Express 404 body so unknown routes
// and unsupported methods keep the same contract.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusNotFound, &Error{
		StatusCode: http.StatusNotFound,
		Message:    "Cannot " + r.Method + " " + r.URL.Path,
		ErrorName:  "Not Found",
	})
}
