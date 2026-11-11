package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appservices "reservoir/application/services"
	dbmodels "reservoir/db/models"
	"reservoir/utils/phc"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/api/auth/models"
	coreauth "reservoir/webserver/auth"
)

func TestLoginEndpointReturnsCurrentUserWhenAlreadyAuthenticated(t *testing.T) {
	store := newTestManagedUserStore(t)
	user, err := store.Create(&dbmodels.User{
		Username:     "operator",
		PasswordHash: *phc.GenerateArgon2id("current-password"),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessions := coreauth.NewSessionManager()
	session := sessions.Create(user.ID)
	services := appservices.NewContainer(nil, sessions, store, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{
		"username": "operator",
		"password": "current-password"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(session.BuildSessionCookieWithSecure(false))

	ctx, err := apitypes.CreateContext(req, services)
	if err != nil {
		t.Fatalf("failed to create API context: %v", err)
	}

	(&LoginEndpoint{}).Post(rec, req, ctx)

	// The dashboard stores the response body as the signed-in user.
	var got models.UserInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Username != "operator" || !got.IsAdmin {
		t.Fatalf("expected the current user as JSON, got status %d: %s", rec.Code, rec.Body.String())
	}
}
