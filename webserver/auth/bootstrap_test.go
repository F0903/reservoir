package auth

import (
	"errors"
	"testing"

	"reservoir/db/models"
	"reservoir/db/stores"
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

	result, err := ensureBootstrapAdmin(store)
	if err != nil {
		t.Fatalf("ensureBootstrapAdmin returned error: %v", err)
	}

	if result == nil || !result.Required {
		t.Fatalf("expected bootstrap required result, got %#v", result)
	}
	if store.saveCount != 0 {
		t.Fatalf("expected empty store bootstrap check not to save users, saved %d times", store.saveCount)
	}
}

func TestEnsureBootstrapAdminDoesNothingWhenUsersExist(t *testing.T) {
	store := newFakeBootstrapUserStore(&models.User{
		Username: DefaultAdminUsername,
		IsAdmin:  true,
	})

	result, err := ensureBootstrapAdmin(store)
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

func TestCreateBootstrapAdminCreatesFirstAdmin(t *testing.T) {
	store := newFakeBootstrapUserStore()

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
	store := newFakeBootstrapUserStore(&models.User{
		Username: "existing",
		IsAdmin:  true,
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
			store := newFakeBootstrapUserStore()

			_, err := createBootstrapAdmin(store, tt.username, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
