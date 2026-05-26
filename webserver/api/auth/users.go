package auth

import (
	"errors"
	"log/slog"
	"net/http"
	appservices "reservoir/application/services"
	"reservoir/db/models"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	authmodels "reservoir/webserver/api/auth/models"
	"strconv"
	"strings"
)

type UsersEndpoint struct{}
type UserEndpoint struct{}

type createUserRequest struct {
	Username               string `json:"username"`
	Password               string `json:"password"`
	IsAdmin                bool   `json:"is_admin"`
	PasswordChangeRequired *bool  `json:"password_change_required,omitempty"`
}

type updateUserRequest struct {
	Username               *string `json:"username,omitempty"`
	Password               *string `json:"password,omitempty"`
	IsAdmin                *bool   `json:"is_admin,omitempty"`
	PasswordChangeRequired *bool   `json:"password_change_required,omitempty"`
}

func (e *UsersEndpoint) Path() string {
	return "/auth/users"
}

func (e *UsersEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:        http.MethodGet,
			Func:          e.Get,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
		{
			Method:        http.MethodPost,
			Func:          e.Post,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
	}
}

func (e *UsersEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	users, err := ctx.Services.Users.List()
	if err != nil {
		slog.Error("Error listing users", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	apihttp.WriteJSON(w, http.StatusOK, userInfoList(users))
}

func (e *UsersEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	var req createUserRequest
	if !apihttp.DecodeJSON(w, r, &req) {
		return
	}

	user, err := ctx.Services.Users.CreateManagedUser(req.Username, req.Password, req.IsAdmin, req.PasswordChangeRequired)
	if err != nil {
		writeUserServiceError(w, "creating user", err)
		return
	}

	apihttp.WriteJSON(w, http.StatusCreated, authmodels.FromUser(user))
}

func (e *UserEndpoint) Path() string {
	return "/auth/users/{id}"
}

func (e *UserEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:        http.MethodPatch,
			Func:          e.Patch,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
		{
			Method:        http.MethodDelete,
			Func:          e.Delete,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
	}
}

func (e *UserEndpoint) Patch(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	id, ok := parseUserID(w, r)
	if !ok {
		return
	}

	var req updateUserRequest
	if !apihttp.DecodeJSON(w, r, &req) {
		return
	}

	user, err := ctx.Services.Users.UpdateManagedUser(ctx.Request, id, appservices.UserUpdate{
		Username:               req.Username,
		Password:               req.Password,
		IsAdmin:                req.IsAdmin,
		PasswordChangeRequired: req.PasswordChangeRequired,
	})
	if err != nil {
		writeUserServiceError(w, "updating user", err)
		return
	}
	apihttp.WriteJSON(w, http.StatusOK, authmodels.FromUser(user))
}

func (e *UserEndpoint) Delete(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}

	if err := ctx.Services.Users.DeleteManagedUser(id); err != nil {
		writeUserServiceError(w, "deleting user", err)
		return
	}

	apihttp.NoContent(w)
}

func userInfoList(users []models.User) []authmodels.UserInfo {
	resp := make([]authmodels.UserInfo, 0, len(users))
	for i := range users {
		resp = append(resp, authmodels.FromUser(&users[i]))
	}
	return resp
}

func parseUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	rawID := strings.TrimSpace(r.PathValue("id"))
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		apihttp.BadRequest(w, "Invalid user id")
		return 0, false
	}
	return id, true
}

func writeUserServiceError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, appservices.ErrPasswordEmpty):
		apihttp.BadRequest(w, "Password must not be empty")
	case errors.Is(err, appservices.ErrPasswordTooShort):
		apihttp.BadRequest(w, "Password must be at least 12 characters")
	case errors.Is(err, appservices.ErrUsernameEmpty):
		apihttp.BadRequest(w, "Username must not be empty")
	case errors.Is(err, appservices.ErrUsernameTaken):
		apihttp.Error(w, "Username is already taken", http.StatusConflict)
	case errors.Is(err, appservices.ErrUserNotFound):
		apihttp.Error(w, "User not found", http.StatusNotFound)
	case errors.Is(err, appservices.ErrLastAdmin):
		apihttp.Error(w, "At least one administrator is required", http.StatusConflict)
	default:
		slog.Error("Error "+action, "error", err)
		apihttp.InternalServerError(w)
	}
}
