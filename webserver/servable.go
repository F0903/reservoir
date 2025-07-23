package webserver

import "net/http"

type Servable interface {
	RegisterHandlers(mux *http.ServeMux) error
}
