package httplistener

import (
	"context"
	"net/http"
)

type HTTPListener struct {
	server *http.Server
}

func New(address string, handler http.Handler) *HTTPListener {
	return &HTTPListener{
		server: &http.Server{
			Addr:    address,
			Handler: handler,
		},
	}
}

func (hl *HTTPListener) ListenBlocking() error {
	return hl.server.ListenAndServe()
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
		if err := hl.server.Shutdown(context.Background()); err != nil {
			errChan <- err
		}
	}()
	hl.Listen(errChan)
}
