package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
)

type fakeBootstrapUserStore struct {
	users     map[string]*models.User
	saveCount int
}

func newFakeBootstrapUserStore(users ...*models.User) *fakeBootstrapUserStore {
	store := &fakeBootstrapUserStore{users: map[string]*models.User{}}
	for _, user := range users {
		userCopy := *user
		store.users[user.Username] = &userCopy
	}
	return store
}

func (s *fakeBootstrapUserStore) GetByUsername(username string) (*models.User, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, nil
	}
	userCopy := *user
	return &userCopy, nil
}

func (s *fakeBootstrapUserStore) Count() (int, error) {
	return len(s.users), nil
}

func (s *fakeBootstrapUserStore) Save(user *models.User) error {
	userCopy := *user
	s.users[user.Username] = &userCopy
	s.saveCount++
	return nil
}

func (s *fakeBootstrapUserStore) CreateFirst(user *models.User) error {
	if len(s.users) > 0 {
		return stores.ErrUserStoreNotEmpty
	}

	return s.Save(user)
}

func TestEnsureBootstrapAdminReportsRequiredForEmptyStore(t *testing.T) {
	store := newFakeBootstrapUserStore()
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

	result, err := ensureBootstrapAdmin(store, passwordFile, fixedPassword("generated-password"))
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result == nil || !result.Required {
		t.Fatalf("expected bootstrap required result, got %#v", result)
	}
	if store.saveCount != 0 {
		t.Fatalf("expected empty store bootstrap check not to save users, saved %d times", store.saveCount)
	}
	if _, err := os.Stat(passwordFile); !os.IsNotExist(err) {
		t.Fatalf("expected no bootstrap password file to be written, got %v", err)
	}
}

func TestEnsureBootstrapAdminDoesNotOverwriteConfiguredAdmin(t *testing.T) {
	store := newFakeBootstrapUserStore(&models.User{
		Username:               DefaultAdminUsername,
		PasswordHash:           *phc.GenerateArgon2id("custom-password"),
		PasswordChangeRequired: false,
	})
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

	result, err := ensureBootstrapAdmin(store, passwordFile, fixedPassword("generated-password"))
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result != nil {
		t.Fatalf("expected no bootstrap result, got %#v", result)
	}
	if store.saveCount != 0 {
		t.Fatalf("expected configured admin to remain unchanged, saved %d times", store.saveCount)
	}
}

func TestEnsureBootstrapAdminRotatesLegacyDefaultPassword(t *testing.T) {
	store := newFakeBootstrapUserStore(&models.User{
		Username:               DefaultAdminUsername,
		PasswordHash:           *phc.GenerateArgon2id(legacyDefaultAdminPassword),
		PasswordChangeRequired: true,
	})
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

	result, err := ensureBootstrapAdmin(store, passwordFile, fixedPassword("generated-password"))
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result == nil || !result.RotatedLegacyDefault {
		t.Fatalf("expected rotated legacy default result, got %#v", result)
	}
	admin := store.users[DefaultAdminUsername]
	if admin.PasswordHash.VerifyArgon2id(legacyDefaultAdminPassword) {
		t.Fatal("expected legacy default password to be rotated")
	}
	if !admin.PasswordHash.VerifyArgon2id("generated-password") {
		t.Fatal("expected admin password to match generated password")
	}
	assertPasswordFileContains(t, passwordFile, "generated-password")
}

func TestEnsureBootstrapAdminReissuesMissingBootstrapPassword(t *testing.T) {
	store := newFakeBootstrapUserStore(&models.User{
		Username:               DefaultAdminUsername,
		PasswordHash:           *phc.GenerateArgon2id("lost-password"),
		PasswordChangeRequired: true,
	})
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

	result, err := ensureBootstrapAdmin(store, passwordFile, fixedPassword("replacement-password"))
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result == nil || !result.Reissued {
		t.Fatalf("expected reissued bootstrap result, got %#v", result)
	}
	admin := store.users[DefaultAdminUsername]
	if admin.PasswordHash.VerifyArgon2id("lost-password") {
		t.Fatal("expected lost bootstrap password to be replaced")
	}
	if !admin.PasswordHash.VerifyArgon2id("replacement-password") {
		t.Fatal("expected admin password to match replacement password")
	}
	assertPasswordFileContains(t, passwordFile, "replacement-password")
}

func TestEnsureBootstrapAdminKeepsExistingBootstrapPasswordFile(t *testing.T) {
	store := newFakeBootstrapUserStore(&models.User{
		Username:               DefaultAdminUsername,
		PasswordHash:           *phc.GenerateArgon2id("existing-password"),
		PasswordChangeRequired: true,
	})
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")
	if err := os.WriteFile(passwordFile, []byte("password: existing-password"), 0600); err != nil {
		t.Fatalf("failed to write bootstrap password file: %v", err)
	}

	result, err := ensureBootstrapAdmin(store, passwordFile, fixedPassword("replacement-password"))
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result != nil {
		t.Fatalf("expected existing bootstrap password to be preserved, got %#v", result)
	}
	if store.saveCount != 0 {
		t.Fatalf("expected existing bootstrap admin to remain unchanged, saved %d times", store.saveCount)
	}
}

func TestCreateBootstrapAdminCreatesFirstAdmin(t *testing.T) {
	store := newFakeBootstrapUserStore()
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

	user, err := createBootstrapAdmin(store, " admin ", "generated-password", passwordFile)
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
	if !user.PasswordHash.VerifyArgon2id("generated-password") {
		t.Fatal("expected admin password to match chosen password")
	}
}

func TestCreateBootstrapAdminRejectsExistingUsers(t *testing.T) {
	store := newFakeBootstrapUserStore(&models.User{
		Username:               "existing",
		PasswordHash:           *phc.GenerateArgon2id("existing-password"),
		PasswordChangeRequired: false,
	})
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

	_, err := createBootstrapAdmin(store, DefaultAdminUsername, "generated-password", passwordFile)
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
			store := newFakeBootstrapUserStore()
			passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

			_, err := createBootstrapAdmin(store, tt.username, tt.password, passwordFile)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func fixedPassword(password string) func() (string, error) {
	return func() (string, error) {
		return password, nil
	}
}

func assertPasswordFileContains(t *testing.T, path string, password string) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read bootstrap password file: %v", err)
	}
	if !strings.Contains(string(body), "password: "+password) {
		t.Fatalf("expected bootstrap password file to contain generated password, got %q", string(body))
	}
}
