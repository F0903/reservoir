package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
			Method:       "GET",
			Func:         e.Get,
			RequiresAuth: true,
		},
	}
}

func (e *MeEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	user, err := ctx.GetCurrentUser()
	if err != nil {
		slog.Error("Error retrieving current user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	userJson, err := json.Marshal(models.UserInfo{
		ID:                     user.ID,
		Username:               user.Username,
		PasswordChangeRequired: user.PasswordChangeRequired,
		CreatedAt:              user.CreatedAt,
		UpdatedAt:              user.UpdatedAt,
	})

	if err != nil {
		slog.Error("Error marshaling user JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(userJson)
}
