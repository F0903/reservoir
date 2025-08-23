package webserver

import (
	"context"
	"net/http"
	"reservoir/utils/httplistener"
	"reservoir/webserver/middleware"
)

type WebServer struct {
	mux *http.ServeMux
}

func New() *WebServer {
	mux := http.NewServeMux()
	return &WebServer{mux: mux}
}

func (ws *WebServer) Register(s Servable) error {
	return s.RegisterHandlers(ws.mux)
}

func (ws *WebServer) Listen(address string, errChan chan error, ctx context.Context) {
	listener := httplistener.New(address, middleware.Harden(ws.mux))
	listener.ListenWithCancel(errChan, ctx)
}
