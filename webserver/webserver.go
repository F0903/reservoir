package webserver

import (
	"context"
	"net/http"
	"reservoir/utils/httplistener"
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
	listener := httplistener.New(address, ws.mux)
	listener.ListenWithCancel(errChan, ctx)
}
