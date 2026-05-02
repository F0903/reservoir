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
		IsAdmin:                true,
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

func TestCreateUserRejectsTakenUsername(t *testing.T) {
	store := newTestUserStore(t)
	if _, err := store.Create(testUser("alice")); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err := store.Create(testUser("ALICE"))
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestListUsersOrdersByUsername(t *testing.T) {
	store := newTestUserStore(t)
	if _, err := store.Create(testUser("charlie")); err != nil {
		t.Fatalf("failed to create charlie: %v", err)
	}
	if _, err := store.Create(testUser("alice")); err != nil {
		t.Fatalf("failed to create alice: %v", err)
	}

	users, err := store.List()
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" || users[1].Username != "charlie" {
		t.Fatalf("expected users to be ordered by username, got %#v", users)
	}
}

func TestUpdateAdminRejectsDemotingLastAdmin(t *testing.T) {
	store := newTestUserStore(t)
	admin, err := store.Create(testUser("admin"))
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	_, err = store.UpdateAdmin(admin.ID, false)
	if !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func TestUpdateAdminAllowsMultipleAdmins(t *testing.T) {
	store := newTestUserStore(t)
	admin, err := store.Create(testUser("admin"))
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	secondAdmin, err := store.Create(testUser("second"))
	if err != nil {
		t.Fatalf("failed to create second admin: %v", err)
	}

	updated, err := store.UpdateAdmin(secondAdmin.ID, false)
	if err != nil {
		t.Fatalf("failed to demote second admin: %v", err)
	}
	if updated.IsAdmin {
		t.Fatal("expected second user to be demoted")
	}

	admins, err := store.CountAdmins()
	if err != nil {
		t.Fatalf("failed to count admins: %v", err)
	}
	if admins != 1 {
		t.Fatalf("expected one admin after demotion, got %d", admins)
	}
	if _, err := store.GetByID(admin.ID); err != nil {
		t.Fatalf("expected first admin to remain: %v", err)
	}
}

func TestDeleteRejectsDeletingLastAdmin(t *testing.T) {
	store := newTestUserStore(t)
	admin, err := store.Create(testUser("admin"))
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	err = store.Delete(admin.ID)
	if !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func TestUpdatePasswordSetsPasswordAndRequirement(t *testing.T) {
	store := newTestUserStore(t)
	user, err := store.Create(testUser("admin"))
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	hash := *phc.GenerateArgon2id("new-password")
	updated, err := store.UpdatePassword(user.ID, hash, true)
	if err != nil {
		t.Fatalf("failed to update password: %v", err)
	}
	if !updated.PasswordChangeRequired {
		t.Fatal("expected password change to be required")
	}
	if !updated.PasswordHash.VerifyArgon2id("new-password") {
		t.Fatal("expected updated password hash to verify")
	}
}
