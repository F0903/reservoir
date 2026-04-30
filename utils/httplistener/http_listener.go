package httplistener

import (
	"context"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 2 * time.Minute
	shutdownTimeout   = 30 * time.Second
)

type HTTPListener struct {
	server *http.Server
}

func New(address string, handler http.Handler) *HTTPListener {
	return &HTTPListener{
		server: &http.Server{
			Addr:              address,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

func (hl *HTTPListener) ListenBlocking() error {
	return hl.server.ListenAndServe()
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := hl.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return <-errChan
	}
}

func (hl *HTTPListener) Listen(errChan chan error) {
	go func() {
		if err := hl.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
}

func (hl *HTTPListener) ListenWithCancel(errChan chan error, ctx context.Context) {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := hl.server.Shutdown(shutdownCtx); err != nil {
			errChan <- err
		}
	}()
	hl.Listen(errChan)
}
