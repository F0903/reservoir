package auth

import (
	"encoding/json"
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
		http.Error(w, "Invalid credentials", http.StatusBadRequest)
		return
	}

	if ctx.IsAuthenticated() {
		w.Write([]byte("Already authenticated"))
		return
	}

	sess := auth.CreateSession()
	http.SetCookie(w, sess.BuildSessionCookie())
	w.Write([]byte("Logged in successfully"))
}
