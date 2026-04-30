package auth

import (
	"log/slog"
	"net/http"
	"reservoir/utils/phc"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	coreauth "reservoir/webserver/auth"
)

var (
	ErrEmptyCurrentPassword = "current password must not be empty"
	ErrEmptyNewPassword     = "new password must not be empty"
	ErrMissingFields        = "both current_password and new_password fields are required"
	ErrEmptyPasswords       = "passwords must not be empty"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ChangePasswordEndpoint struct{}

func (e *ChangePasswordEndpoint) Path() string {
	return "/auth/change-password"
}

func (e *ChangePasswordEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:                      "PATCH",
			Func:                        e.Patch,
			RequiresAuth:                true,
			AllowPasswordChangeRequired: true,
		},
	}
}

func (e *ChangePasswordEndpoint) Patch(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	var req changePasswordRequest
	if !apihttp.DecodeJSON(w, r, &req) {
		return
	}

	if req.CurrentPassword == "" && req.NewPassword == "" {
		apihttp.BadRequest(w, ErrMissingFields)
		return
	}
	if req.CurrentPassword == "" {
		apihttp.BadRequest(w, ErrEmptyCurrentPassword)
		return
	}
	if req.NewPassword == "" {
		apihttp.BadRequest(w, ErrEmptyNewPassword)
		return
	}

	user, err := ctx.GetCurrentUser()
	if err != nil {
		slog.Error("Error retrieving current user", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	passwordMatch := user.PasswordHash.VerifyArgon2id(req.CurrentPassword)
	if !passwordMatch {
		slog.Warn("Failed password change attempt", "user_id", user.ID)
		apihttp.BadRequest(w, "Current password is incorrect")
		return
	}

	user.PasswordHash = *phc.GenerateArgon2id(req.NewPassword)
	user.PasswordChangeRequired = false
	if err := ctx.UserStore.Save(user); err != nil {
		slog.Error("Error updating user password", "error", err)
		apihttp.InternalServerError(w)
		return
	}
	if err := coreauth.ClearBootstrapPasswordFile(); err != nil {
		slog.Error("Error removing bootstrap password file", "error", err)
	}
	ctx.SessionManager.DestroySessionsForUserExcept(user.ID, ctx.Session.ID)

	apihttp.NoContent(w)
}
