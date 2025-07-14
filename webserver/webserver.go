package webserver

import (
	"apt_cacher_go/utils/http_listener"
	"context"
	"net/http"
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
	listener := http_listener.New(address, ws.mux)
	listener.ListenWithCancel(errChan, ctx)
}
