package stores

import (
	"errors"
	"path/filepath"
	"reservoir/db"
	"reservoir/db/models"
	"reservoir/utils/phc"
	"testing"
)

func newTestUserStore(t *testing.T) *UserStore {
	t.Helper()

	databasePath := filepath.ToSlash(filepath.Join(t.TempDir(), "database.db"))
	database, err := db.Open(databasePath, 5000)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	store := &UserStore{db: database}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close test user store: %v", err)
		}
	})
	return store
}

func testUser(username string) *models.User {
	return &models.User{
		Username:               username,
		PasswordHash:           *phc.GenerateArgon2id("test-password"),
		PasswordChangeRequired: false,
	}
}

func TestUpdateUsernameRenamesExistingUser(t *testing.T) {
	store := newTestUserStore(t)
	if err := store.Save(testUser("admin")); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}
	user, err := store.GetByUsername("admin")
	if err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}

	updated, err := store.UpdateUsername(user.ID, " renamed ")
	if err != nil {
		t.Fatalf("UpdateUsername returned error: %v", err)
	}

	if updated.Username != "renamed" {
		t.Fatalf("expected trimmed username %q, got %q", "renamed", updated.Username)
	}
	if updated.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, updated.ID)
	}

	oldUser, err := store.GetByUsername("admin")
	if err != nil {
		t.Fatalf("failed to fetch old username: %v", err)
	}
	if oldUser != nil {
		t.Fatal("expected old username lookup to return nil")
	}
}

func TestUpdateUsernameRejectsTakenUsername(t *testing.T) {
	store := newTestUserStore(t)
	if err := store.Save(testUser("alice")); err != nil {
		t.Fatalf("failed to save first user: %v", err)
	}
	if err := store.Save(testUser("bob")); err != nil {
		t.Fatalf("failed to save second user: %v", err)
	}
	alice, err := store.GetByUsername("alice")
	if err != nil {
		t.Fatalf("failed to fetch first user: %v", err)
	}

	_, err = store.UpdateUsername(alice.ID, "BOB")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestUpdateUsernameValidatesInput(t *testing.T) {
	store := newTestUserStore(t)

	_, err := store.UpdateUsername(1, " ")
	if !errors.Is(err, ErrUsernameEmpty) {
		t.Fatalf("expected ErrUsernameEmpty, got %v", err)
	}

	_, err = store.UpdateUsername(99, "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
