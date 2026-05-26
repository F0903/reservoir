// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
	appservices "reservoir/application/services"
	"reservoir/db/models"
)

type Context struct {
	Request       appservices.RequestContext
	Services      *appservices.Container
	authenticated bool
}

func CreateContext(r *http.Request, services *appservices.Container) (Context, error) {
	if services == nil || services.Auth == nil {
		return Context{}, appservices.ErrServicesUnavailable
	}

	request, authenticated := services.Auth.RequestFromHTTP(r)

	return Context{
		Request:       request,
		Services:      services,
		authenticated: authenticated,
	}, nil
}

func (c *Context) IsAuthenticated() bool {
	return c.authenticated
}

func (c *Context) GetCurrentUser() (*models.User, error) {
	if c.Services == nil || c.Services.Auth == nil {
		return nil, appservices.ErrServicesUnavailable
	}
	return c.Services.Auth.CurrentUser(c.Request)
}
