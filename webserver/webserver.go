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

func (ws *WebServer) Run(address string, ctx context.Context) error {
	listener := httplistener.New(address, middleware.Harden(ws.mux))
	return listener.Run(ctx)
}
