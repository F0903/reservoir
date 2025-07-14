package api

import "net/http"

type API struct {
	handler *apiHandler
}

func New() *API {
	return &API{
		handler: newAPIHandler(),
	}
}

func (api *API) RegisterHandlers(mux *http.ServeMux) error {
	mux.Handle("/api/", api.handler)
	return nil
}
