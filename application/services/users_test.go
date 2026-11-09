package services

import (
	"errors"
	"testing"

	coreauth "reservoir/webserver/auth"
)

func TestUpdateManagedUserRejectsBadPasswordBeforeSavingAnything(t *testing.T) {
	store := newAuthTestStore(t)
	user := createAuthTestUser(t, store)
	svc := &UserService{users: store, sessions: coreauth.NewSessionManager()}

	username := "renamed"
	admin := false
	password := "short"
	_, err := svc.UpdateManagedUser(RequestContext{}, user.ID, UserUpdate{
		Username: &username,
		IsAdmin:  &admin,
		Password: &password,
	})
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}

	stored, err := store.GetByID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Username != "operator" || !stored.IsAdmin {
		t.Fatalf("failed update was partly saved: username=%q admin=%v", stored.Username, stored.IsAdmin)
	}
}

func TestUpdateManagedUserSavesNothingWhenALaterWriteFails(t *testing.T) {
	store := newAuthTestStore(t)
	user := createAuthTestUser(t, store)
	svc := &UserService{users: store, sessions: coreauth.NewSessionManager()}

	username := "renamed"
	demote := false
	_, err := svc.UpdateManagedUser(RequestContext{}, user.ID, UserUpdate{Username: &username, IsAdmin: &demote})
	if !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}

	stored, err := store.GetByID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Username != "operator" || !stored.IsAdmin {
		t.Fatalf("failed update was partly saved: username=%q admin=%v", stored.Username, stored.IsAdmin)
	}
}

func TestUpdateManagedUserReportsMissingUser(t *testing.T) {
	store := newAuthTestStore(t)
	svc := &UserService{users: store, sessions: coreauth.NewSessionManager()}

	username := "renamed"
	_, err := svc.UpdateManagedUser(RequestContext{}, 99, UserUpdate{Username: &username})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUpdateManagedUserPasswordChangeEndsOtherSessions(t *testing.T) {
	store := newAuthTestStore(t)
	user := createAuthTestUser(t, store)
	sessions := coreauth.NewSessionManager()
	svc := &UserService{users: store, sessions: sessions}
	session := sessions.Create(user.ID)

	password := "a-long-enough-password"
	updated, err := svc.UpdateManagedUser(RequestContext{}, user.ID, UserUpdate{Password: &password})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if !updated.PasswordHash.VerifyArgon2id(password) || !updated.PasswordChangeRequired {
		t.Fatalf("expected the new password with a change required, got required=%v", updated.PasswordChangeRequired)
	}
	if _, ok := sessions.Get(session.ID); ok {
		t.Fatal("expected existing sessions to be ended")
	}
}

func TestUpdateManagedUserAppliesPasswordChangeRequiredOnItsOwn(t *testing.T) {
	store := newAuthTestStore(t)
	user := createAuthTestUser(t, store)
	svc := &UserService{users: store, sessions: coreauth.NewSessionManager()}

	required := true
	updated, err := svc.UpdateManagedUser(RequestContext{}, user.ID, UserUpdate{PasswordChangeRequired: &required})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !updated.PasswordChangeRequired {
		t.Fatal("expected the flag in the returned user")
	}

	stored, err := store.GetByID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.PasswordChangeRequired {
		t.Fatal("expected the flag to be saved")
	}
	if !stored.PasswordHash.VerifyArgon2id("current-password") {
		t.Fatal("expected the password to be unchanged")
	}
}
