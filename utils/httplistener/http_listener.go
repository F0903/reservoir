package httplistener

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout      = 10 * time.Second
	idleTimeout            = 2 * time.Minute
	defaultShutdownTimeout = 30 * time.Second
)

type HTTPListener struct {
	server          *http.Server
	shutdownTimeout time.Duration
}

func New(address string, handler http.Handler) *HTTPListener {
	return &HTTPListener{
		server: &http.Server{
			Addr:              address,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
		},
		shutdownTimeout: defaultShutdownTimeout,
	}
}

func (hl *HTTPListener) Run(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		if err := hl.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), hl.shutdownTimeout)
		defer cancel()

		if err := hl.server.Shutdown(shutdownCtx); err != nil {
			slog.Warn("Graceful shutdown timed out, closing remaining connections", "address", hl.server.Addr, "error", err)
			if err := hl.server.Close(); err != nil {
				return err
			}
		}
		return <-errChan
	}
}
