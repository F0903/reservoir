package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/api/auth/models"
	"reservoir/webserver/auth"
)

type LoginEndpoint struct{}

func (e *LoginEndpoint) Path() string {
	return "/auth/login"
}

func (e *LoginEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "POST",
			Func:         e.Post,
			RequiresAuth: false,
		},
	}
}

func (e *LoginEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	jsonDecoder := json.NewDecoder(r.Body)
	var creds auth.Credentials
	err := jsonDecoder.Decode(&creds)
	if err != nil {
		slog.Error("Error decoding credentials JSON", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if ctx.IsAuthenticated() {
		w.Write([]byte("Already Authenticated"))
		return
	}

	user, err := creds.Authenticate()
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}
		slog.Error("Error during authentication", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	sess := auth.CreateSession(user.ID)
	http.SetCookie(w, sess.BuildSessionCookie())

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
	w.Write(userJson)
}
