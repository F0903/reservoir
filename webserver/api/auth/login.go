package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/webserver/api/apitypes"
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
	json := json.NewDecoder(r.Body)
	var creds auth.Credentials
	err := json.Decode(&creds)
	if err != nil {
		slog.Error("Error decoding credentials JSON", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if ctx.IsAuthenticated() {
		w.Write([]byte("Already Authenticated"))
		return
	}

	ok, err := creds.Authenticate()
	if err != nil {
		slog.Error("Error during authentication", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !ok {
		http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
		return
	}

	sess := auth.CreateSession()
	http.SetCookie(w, sess.BuildSessionCookie())
	w.WriteHeader(http.StatusNoContent)
}
