package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reservoir/db/models"
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

func TestEnsureBootstrapAdminCreatesAdminForEmptyStore(t *testing.T) {
	store := newFakeBootstrapUserStore()
	passwordFile := filepath.Join(t.TempDir(), "bootstrap-password.txt")

	result, err := ensureBootstrapAdmin(store, passwordFile, fixedPassword("generated-password"))
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result == nil || !result.Created {
		t.Fatalf("expected created bootstrap result, got %#v", result)
	}
	admin := store.users[DefaultAdminUsername]
	if admin == nil {
		t.Fatal("expected admin user to be saved")
	}
	if !admin.PasswordChangeRequired {
		t.Fatal("expected bootstrap admin to require password change")
	}
	if !admin.PasswordHash.VerifyArgon2id("generated-password") {
		t.Fatal("expected admin password to match generated password")
	}
	assertPasswordFileContains(t, passwordFile, "generated-password")
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
