package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	authmodels "reservoir/webserver/api/auth/models"
	"strconv"
	"strings"
)

const minManagedPasswordLength = 12

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
	users, err := ctx.UserStore.List()
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

	if !validateManagedPassword(w, req.Password) {
		return
	}

	user, err := ctx.UserStore.Create(&models.User{
		Username:               req.Username,
		PasswordHash:           *phc.GenerateArgon2id(req.Password),
		IsAdmin:                req.IsAdmin,
		PasswordChangeRequired: passwordChangeRequiredOrDefault(req.PasswordChangeRequired, true),
	})
	if err != nil {
		writeUserStoreError(w, "creating user", err)
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

	user, err := ctx.UserStore.GetByID(id)
	if err != nil {
		slog.Error("Error loading user before update", "user_id", id, "error", err)
		apihttp.InternalServerError(w)
		return
	}
	if user == nil {
		apihttp.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if req.Username != nil {
		user, err = ctx.UserStore.UpdateUsername(id, *req.Username)
		if err != nil {
			writeUserStoreError(w, "updating username", err)
			return
		}
	}

	if req.IsAdmin != nil {
		user, err = ctx.UserStore.UpdateAdmin(id, *req.IsAdmin)
		if err != nil {
			writeUserStoreError(w, "updating admin status", err)
			return
		}
	}

	if req.Password != nil {
		if !validateManagedPassword(w, *req.Password) {
			return
		}

		user, err = ctx.UserStore.UpdatePassword(
			id,
			*phc.GenerateArgon2id(*req.Password),
			passwordChangeRequiredOrDefault(req.PasswordChangeRequired, true),
		)
		if err != nil {
			writeUserStoreError(w, "updating password", err)
			return
		}

		destroyUserSessions(ctx, id, true)
	}

	apihttp.WriteJSON(w, http.StatusOK, authmodels.FromUser(user))
}

func (e *UserEndpoint) Delete(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}

	if err := ctx.UserStore.Delete(id); err != nil {
		writeUserStoreError(w, "deleting user", err)
		return
	}

	destroyUserSessions(ctx, id, false)
	apihttp.NoContent(w)
}

func userInfoList(users []models.User) []authmodels.UserInfo {
	resp := make([]authmodels.UserInfo, 0, len(users))
	for i := range users {
		resp = append(resp, authmodels.FromUser(&users[i]))
	}
	return resp
}

func passwordChangeRequiredOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func destroyUserSessions(ctx apitypes.Context, userID int64, keepCurrentSession bool) {
	if ctx.SessionManager == nil {
		return
	}

	keepSessionID := ""
	if keepCurrentSession && ctx.Session != nil && ctx.Session.UserID == userID {
		keepSessionID = ctx.Session.ID
	}

	ctx.SessionManager.DestroySessionsForUserExcept(userID, keepSessionID)
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

func validateManagedPassword(w http.ResponseWriter, password string) bool {
	if password == "" {
		apihttp.BadRequest(w, "Password must not be empty")
		return false
	}
	if len(password) < minManagedPasswordLength {
		apihttp.BadRequest(w, "Password must be at least 12 characters")
		return false
	}
	return true
}

func writeUserStoreError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, stores.ErrUsernameEmpty):
		apihttp.BadRequest(w, "Username must not be empty")
	case errors.Is(err, stores.ErrUsernameTaken):
		apihttp.Error(w, "Username is already taken", http.StatusConflict)
	case errors.Is(err, stores.ErrUserNotFound):
		apihttp.Error(w, "User not found", http.StatusNotFound)
	case errors.Is(err, stores.ErrLastAdmin):
		apihttp.Error(w, "At least one administrator is required", http.StatusConflict)
	default:
		slog.Error("Error "+action, "error", err)
		apihttp.InternalServerError(w)
	}
}
