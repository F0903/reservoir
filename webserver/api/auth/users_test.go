package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"reservoir/db"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	"reservoir/webserver/api/apitypes"
	coreauth "reservoir/webserver/auth"
)

func newTestManagedUserStore(t *testing.T) *stores.UserStore {
	t.Helper()

	databasePath := filepath.ToSlash(filepath.Join(t.TempDir(), "database.db"))
	database, err := db.Open(databasePath, 5000)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	store := stores.NewUserStore(database)
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close test user store: %v", err)
		}
	})
	return store
}

func TestUsersEndpointCreatesUserWithPasswordChangeRequiredByDefault(t *testing.T) {
	store := newTestManagedUserStore(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/users", bytes.NewBufferString(`{
		"username": "operator",
		"password": "generated-password",
		"is_admin": true
	}`))
	req.Header.Set("Content-Type", "application/json")

	(&UsersEndpoint{}).Post(rec, req, apitypes.Context{UserStore: store})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	created, err := store.GetByUsername("operator")
	if err != nil {
		t.Fatalf("failed to fetch created user: %v", err)
	}
	if created == nil {
		t.Fatal("expected created user")
	}
	if created.Username != "operator" {
		t.Fatalf("expected username operator, got %q", created.Username)
	}
	if !created.IsAdmin {
		t.Fatal("expected created user to be admin")
	}
	if !created.PasswordChangeRequired {
		t.Fatal("expected password change to be required by default")
	}
	if !created.PasswordHash.VerifyArgon2id("generated-password") {
		t.Fatal("expected stored password hash to verify")
	}
}

func TestUserEndpointPasswordResetDestroysOtherSessions(t *testing.T) {
	store := newTestManagedUserStore(t)
	user, err := store.Create(&models.User{
		Username:     "operator",
		PasswordHash: *phc.GenerateArgon2id("old-password"),
		IsAdmin:      false,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessions := coreauth.NewSessionManager()
	current := sessions.Create(user.ID)
	other := sessions.Create(user.ID)
	req := httptest.NewRequest(http.MethodPatch, "/api/auth/users/"+strconv.FormatInt(user.ID, 10), bytes.NewBufferString(`{
		"password": "generated-password",
		"password_change_required": false
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.FormatInt(user.ID, 10))
	rec := httptest.NewRecorder()

	(&UserEndpoint{}).Patch(rec, req, apitypes.Context{
		Session:        current,
		SessionManager: sessions,
		UserStore:      store,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if _, ok := sessions.Get(current.ID); !ok {
		t.Fatal("expected current session to remain")
	}
	if _, ok := sessions.Get(other.ID); ok {
		t.Fatal("expected other session to be destroyed")
	}

	updated, err := store.GetByID(user.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated user: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated user")
	}
	if !updated.PasswordHash.VerifyArgon2id("generated-password") {
		t.Fatal("expected password hash to be updated")
	}
	if updated.PasswordChangeRequired {
		t.Fatal("expected password change requirement to be cleared")
	}
}

func TestUserEndpointDeleteMapsLastAdminError(t *testing.T) {
	store := newTestManagedUserStore(t)
	user, err := store.Create(&models.User{
		Username:     "admin",
		PasswordHash: *phc.GenerateArgon2id("admin-password"),
		IsAdmin:      true,
	})
	if err != nil {
		t.Fatalf("failed to create test admin: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/auth/users/"+strconv.FormatInt(user.ID, 10), nil)
	req.SetPathValue("id", strconv.FormatInt(user.ID, 10))
	rec := httptest.NewRecorder()

	(&UserEndpoint{}).Delete(rec, req, apitypes.Context{
		SessionManager: coreauth.NewSessionManager(),
		UserStore:      store,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}

	remaining, err := store.GetByID(user.ID)
	if err != nil {
		t.Fatalf("failed to fetch admin after failed delete: %v", err)
	}
	if remaining == nil {
		t.Fatal("expected last admin to remain after failed delete")
	}
}
