package services

import (
	"errors"
	"path/filepath"
	"testing"

	"reservoir/db"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	coreauth "reservoir/webserver/auth"
)

func newAuthTestStore(t *testing.T) *stores.UserStore {
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

func TestChangePasswordRejectsShortNewPassword(t *testing.T) {
	store := newAuthTestStore(t)
	user, err := store.Create(&models.User{
		Username:               "operator",
		PasswordHash:           *phc.GenerateArgon2id("current-password"),
		IsAdmin:                true,
		PasswordChangeRequired: false,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	sessions := coreauth.NewSessionManager()
	session := sessions.Create(user.ID)
	svc := &AuthService{users: store, sessions: sessions}

	err = svc.ChangePassword(RequestContext{session: session}, "current-password", "short")
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}

	unchanged, err := store.GetByID(user.ID)
	if err != nil {
		t.Fatalf("failed to reload test user: %v", err)
	}
	if !unchanged.PasswordHash.VerifyArgon2id("current-password") {
		t.Fatal("short password update changed the stored password")
	}
}
