package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// shutdownTimeout bounds the graceful shutdown drain.
const shutdownTimeout = 10 * time.Second

// Server owns the HTTP server lifecycle and timeouts.
type Server struct {
	server *http.Server
}

// NewServer builds a Server listening on the given port. The write timeout
// is generous because AI routes answer synchronously for now.
func NewServer(port int, handler http.Handler) *Server {
	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      120 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// Start serves requests until ctx is cancelled, then drains connections
// gracefully before returning.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		if err := s.server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(), shutdownTimeout,
	)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}
