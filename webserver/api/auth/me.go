package auth

import (
	"log/slog"
	"net/http"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/api/auth/models"
)

type MeEndpoint struct{}

func (e *MeEndpoint) Path() string {
	return "/auth/me"
}

func (e *MeEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:                      "GET",
			Func:                        e.Get,
			RequiresAuth:                true,
			AllowPasswordChangeRequired: true,
		},
	}
}

func (e *MeEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	user, err := ctx.GetCurrentUser()
	if err != nil {
		slog.Error("Error retrieving current user", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	apihttp.WriteJSON(w, http.StatusOK, models.UserInfo{
		ID:                     user.ID,
		Username:               user.Username,
		PasswordChangeRequired: user.PasswordChangeRequired,
		CreatedAt:              user.CreatedAt,
		UpdatedAt:              user.UpdatedAt,
	})
}
