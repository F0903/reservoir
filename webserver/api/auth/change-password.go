package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/utils/phc"
	"reservoir/webserver/api/apitypes"
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
			Method:       "PATCH",
			Func:         e.Patch,
			RequiresAuth: true,
		},
	}
}

func (e *ChangePasswordEndpoint) Patch(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" && req.NewPassword == "" {
		http.Error(w, ErrMissingFields, http.StatusBadRequest)
		return
	}
	if req.CurrentPassword == "" {
		http.Error(w, ErrEmptyCurrentPassword, http.StatusBadRequest)
		return
	}
	if req.NewPassword == "" {
		http.Error(w, ErrEmptyNewPassword, http.StatusBadRequest)
		return
	}

	user, err := ctx.GetCurrentUser()
	if err != nil {
		slog.Error("Error retrieving current user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	passwordMatch := user.PasswordHash.VerifyArgon2id(req.CurrentPassword)
	if !passwordMatch {
		slog.Warn("Failed password change attempt", "user_id", user.ID)
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	user.PasswordHash = *phc.GenerateArgon2id(req.NewPassword)
	user.PasswordResetRequired = false
	if err := ctx.UserStore.Save(user); err != nil {
		slog.Error("Error updating user password", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
