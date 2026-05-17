package auth

import (
	"errors"
	"path/filepath"
	"testing"

	"reservoir/db"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
)

func newTestBootstrapUserStore(t *testing.T, users ...*models.User) *stores.UserStore {
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

	for _, user := range users {
		if err := store.Save(user); err != nil {
			t.Fatalf("failed to seed test user %q: %v", user.Username, err)
		}
	}
	return store
}

func TestEnsureBootstrapAdminReportsRequiredForEmptyStore(t *testing.T) {
	store := newTestBootstrapUserStore(t)

	result, err := ensureBootstrapAdmin(store)
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result == nil || !result.Required {
		t.Fatalf("expected bootstrap required result, got %#v", result)
	}
	count, err := store.Count()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected empty store bootstrap check not to create users, found %d", count)
	}
}

func TestEnsureBootstrapAdminDoesNothingWhenUsersExist(t *testing.T) {
	store := newTestBootstrapUserStore(t, &models.User{
		Username:     DefaultAdminUsername,
		PasswordHash: *phc.GenerateArgon2id("existing-password"),
		IsAdmin:      true,
	})

	result, err := ensureBootstrapAdmin(store)
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result != nil {
		t.Fatalf("expected no bootstrap result, got %#v", result)
	}
	count, err := store.Count()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected configured admin to remain unchanged, found %d users", count)
	}
}

func TestCreateBootstrapAdminCreatesFirstAdmin(t *testing.T) {
	store := newTestBootstrapUserStore(t)

	user, err := createBootstrapAdmin(store, " admin ", "generated-password")
	if err != nil {
		t.Fatalf("createBootstrapAdmin returned error: %v", err)
	}

	if user == nil {
		t.Fatal("expected created user")
	}
	if user.Username != DefaultAdminUsername {
		t.Fatalf("expected trimmed username %q, got %q", DefaultAdminUsername, user.Username)
	}
	if user.PasswordChangeRequired {
		t.Fatal("expected first-run admin password not to require immediate change")
	}
	if !user.IsAdmin {
		t.Fatal("expected bootstrap user to be an admin")
	}
	if !user.PasswordHash.VerifyArgon2id("generated-password") {
		t.Fatal("expected admin password to match chosen password")
	}
}

func TestCreateBootstrapAdminRejectsExistingUsers(t *testing.T) {
	store := newTestBootstrapUserStore(t, &models.User{
		Username:     "existing",
		PasswordHash: *phc.GenerateArgon2id("existing-password"),
		IsAdmin:      true,
	})

	_, err := createBootstrapAdmin(store, DefaultAdminUsername, "generated-password")
	if !errors.Is(err, ErrBootstrapNotRequired) {
		t.Fatalf("expected ErrBootstrapNotRequired, got %v", err)
	}
}

func TestCreateBootstrapAdminValidatesInput(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{name: "empty username", username: " ", password: "generated-password", wantErr: ErrBootstrapUsernameEmpty},
		{name: "empty password", username: DefaultAdminUsername, password: "", wantErr: ErrBootstrapPasswordEmpty},
		{name: "short password", username: DefaultAdminUsername, password: "short", wantErr: ErrBootstrapPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestBootstrapUserStore(t)

			_, err := createBootstrapAdmin(store, tt.username, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
