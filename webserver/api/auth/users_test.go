package auth

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	"reservoir/webserver/api/apitypes"
	coreauth "reservoir/webserver/auth"
	"testing"
	"time"
)

type fakeManagedUserStore struct {
	users     map[int64]*models.User
	nextID    int64
	deleteErr error
}

func newFakeManagedUserStore(users ...*models.User) *fakeManagedUserStore {
	store := &fakeManagedUserStore{
		users:  make(map[int64]*models.User),
		nextID: 1,
	}
	for _, user := range users {
		userCopy := *user
		if userCopy.ID == 0 {
			userCopy.ID = store.nextID
			store.nextID++
		}
		store.users[userCopy.ID] = &userCopy
		if userCopy.ID >= store.nextID {
			store.nextID = userCopy.ID + 1
		}
	}
	return store
}

func (s *fakeManagedUserStore) Create(user *models.User) (*models.User, error) {
	userCopy := *user
	userCopy.ID = s.nextID
	s.nextID++
	userCopy.CreatedAt = time.Now()
	userCopy.UpdatedAt = userCopy.CreatedAt
	s.users[userCopy.ID] = &userCopy
	return &userCopy, nil
}

func (s *fakeManagedUserStore) List() ([]models.User, error) {
	users := make([]models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, *user)
	}
	return users, nil
}

func (s *fakeManagedUserStore) GetByID(id int64) (*models.User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, nil
	}
	userCopy := *user
	return &userCopy, nil
}

func (s *fakeManagedUserStore) GetByUsername(username string) (*models.User, error) {
	for _, user := range s.users {
		if user.Username == username {
			userCopy := *user
			return &userCopy, nil
		}
	}
	return nil, nil
}

func (s *fakeManagedUserStore) UpdateUsername(id int64, username string) (*models.User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, stores.ErrUserNotFound
	}
	user.Username = username
	userCopy := *user
	return &userCopy, nil
}

func (s *fakeManagedUserStore) UpdateAdmin(id int64, isAdmin bool) (*models.User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, stores.ErrUserNotFound
	}
	user.IsAdmin = isAdmin
	userCopy := *user
	return &userCopy, nil
}

func (s *fakeManagedUserStore) UpdatePassword(id int64, passwordHash phc.PHC, passwordChangeRequired bool) (*models.User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, stores.ErrUserNotFound
	}
	user.PasswordHash = passwordHash
	user.PasswordChangeRequired = passwordChangeRequired
	userCopy := *user
	return &userCopy, nil
}

func (s *fakeManagedUserStore) Delete(id int64) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	if _, ok := s.users[id]; !ok {
		return stores.ErrUserNotFound
	}
	delete(s.users, id)
	return nil
}

func (s *fakeManagedUserStore) Save(user *models.User) error {
	userCopy := *user
	s.users[user.ID] = &userCopy
	return nil
}

func (s *fakeManagedUserStore) Close() error {
	return nil
}

func TestUsersEndpointCreatesUserWithPasswordChangeRequiredByDefault(t *testing.T) {
	store := newFakeManagedUserStore()
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

	created := store.users[1]
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
	store := newFakeManagedUserStore(&models.User{
		ID:           7,
		Username:     "operator",
		PasswordHash: *phc.GenerateArgon2id("old-password"),
		IsAdmin:      false,
	})
	sessions := coreauth.NewSessionManager()
	current := sessions.Create(7)
	other := sessions.Create(7)
	req := httptest.NewRequest(http.MethodPatch, "/api/auth/users/7", bytes.NewBufferString(`{
		"password": "generated-password",
		"password_change_required": false
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "7")
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
	if !store.users[7].PasswordHash.VerifyArgon2id("generated-password") {
		t.Fatal("expected password hash to be updated")
	}
	if store.users[7].PasswordChangeRequired {
		t.Fatal("expected password change requirement to be cleared")
	}
}

func TestUserEndpointDeleteMapsLastAdminError(t *testing.T) {
	store := newFakeManagedUserStore(&models.User{
		ID:       1,
		Username: "admin",
		IsAdmin:  true,
	})
	store.deleteErr = stores.ErrLastAdmin
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/users/1", nil)
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	(&UserEndpoint{}).Delete(rec, req, apitypes.Context{
		SessionManager: coreauth.NewSessionManager(),
		UserStore:      store,
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, rec.Code, rec.Body.String())
	}
	if !errors.Is(store.deleteErr, stores.ErrLastAdmin) {
		t.Fatal("expected test store to preserve last-admin error")
	}
}
