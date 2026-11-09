package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	appservices "reservoir/application/services"
	"reservoir/db/models"
	"reservoir/utils/phc"
	"reservoir/webserver/api/apitypes"
	coreauth "reservoir/webserver/auth"
)

func TestChangePasswordEndpointRejectsShortPassword(t *testing.T) {
	store := newTestManagedUserStore(t)
	user, err := store.Create(&models.User{
		Username:               "operator",
		PasswordHash:           *phc.GenerateArgon2id("current-password"),
		IsAdmin:                true,
		PasswordChangeRequired: true,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessions := coreauth.NewSessionManager()
	session := sessions.Create(user.ID)
	services := appservices.NewContainer(nil, sessions, store, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/auth/change-password", bytes.NewBufferString(`{
		"current_password": "current-password",
		"new_password": "short"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(session.BuildSessionCookieWithSecure(false))

	ctx, err := apitypes.CreateContext(req, services)
	if err != nil {
		t.Fatalf("failed to create API context: %v", err)
	}

	(&ChangePasswordEndpoint{}).Patch(rec, req, ctx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}
