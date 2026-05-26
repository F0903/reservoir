package services

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	dbmodels "reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	coreauth "reservoir/webserver/auth"
)

type AuthService struct {
	users    *stores.UserStore
	sessions *coreauth.SessionManager
}

type RequestContext struct {
	session *coreauth.Session
}

func (s *AuthService) RequestFromHTTP(r *http.Request) (RequestContext, bool) {
	if s == nil || s.sessions == nil {
		return RequestContext{}, false
	}

	session, ok := s.sessions.SessionFromRequest(r)
	return RequestContext{session: session}, ok
}

func (s *AuthService) CurrentUser(req RequestContext) (*dbmodels.User, error) {
	if req.session == nil {
		return nil, ErrNoSession
	}
	if s == nil || s.users == nil {
		return nil, ErrUserStoreUnavailable
	}

	user, err := s.users.GetByID(req.session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *AuthService) Login(req RequestContext, username string, password string) (*dbmodels.User, *http.Cookie, error) {
	if req.session != nil {
		return nil, nil, ErrAlreadyAuthenticated
	}
	if s == nil || s.users == nil || s.sessions == nil {
		return nil, nil, ErrUserStoreUnavailable
	}

	creds := coreauth.Credentials{Username: username, Password: password}
	user, err := creds.Authenticate(s.users)
	if err != nil {
		if errors.Is(err, coreauth.ErrInvalidCredentials) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	cookie := s.createSessionCookie(user.ID)
	return user, cookie, nil
}

func (s *AuthService) Logout(req RequestContext) (*http.Cookie, error) {
	if req.session == nil {
		return nil, ErrNoSession
	}
	if s == nil || s.sessions == nil {
		return nil, ErrUserStoreUnavailable
	}

	s.sessions.Destroy(req.session)

	cookie := req.session.BuildSessionCookie()
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0).UTC()
	cookie.MaxAge = -1
	return cookie, nil
}

func (s *AuthService) BootstrapRequired() (bool, error) {
	if s == nil || s.users == nil {
		return false, ErrUserStoreUnavailable
	}

	return coreauth.BootstrapRequired(s.users)
}

func (s *AuthService) CreateBootstrapAdmin(username string, password string) (*dbmodels.User, *http.Cookie, error) {
	if s == nil || s.users == nil || s.sessions == nil {
		return nil, nil, ErrUserStoreUnavailable
	}

	user, err := coreauth.CreateBootstrapAdmin(s.users, username, password)
	if err != nil {
		return nil, nil, mapBootstrapError(err)
	}

	cookie := s.createSessionCookie(user.ID)
	return user, cookie, nil
}

func (s *AuthService) ChangePassword(req RequestContext, currentPassword string, newPassword string) error {
	if currentPassword == "" && newPassword == "" {
		return ErrMissingPasswordFields
	}
	if currentPassword == "" {
		return ErrEmptyCurrentPassword
	}
	if newPassword == "" {
		return ErrEmptyNewPassword
	}
	if s == nil || s.users == nil || s.sessions == nil {
		return ErrUserStoreUnavailable
	}

	user, err := s.CurrentUser(req)
	if err != nil {
		return err
	}

	if !user.PasswordHash.VerifyArgon2id(currentPassword) {
		return ErrCurrentPasswordIncorrect
	}

	user.PasswordHash = *phc.GenerateArgon2id(newPassword)
	user.PasswordChangeRequired = false
	if err := s.users.Save(user); err != nil {
		return mapUserStoreError(err)
	}

	s.sessions.DestroySessionsForUserExcept(user.ID, req.session.ID)
	return nil
}

func (s *AuthService) UpdateCurrentUsername(req RequestContext, username string) (*dbmodels.User, error) {
	user, err := s.CurrentUser(req)
	if err != nil {
		return nil, err
	}

	updated, err := s.users.UpdateUsername(user.ID, username)
	if err != nil {
		return nil, mapUserStoreError(err)
	}
	return updated, nil
}

func (s *AuthService) createSessionCookie(userID int64) *http.Cookie {
	session := s.sessions.Create(userID)
	return session.BuildSessionCookie()
}

func mapBootstrapError(err error) error {
	switch {
	case errors.Is(err, coreauth.ErrBootstrapNotRequired):
		return fmt.Errorf("%w: %v", ErrBootstrapNotRequired, err)
	case errors.Is(err, coreauth.ErrBootstrapUsernameEmpty):
		return fmt.Errorf("%w: %v", ErrUsernameEmpty, err)
	case errors.Is(err, coreauth.ErrBootstrapPasswordEmpty):
		return fmt.Errorf("%w: %v", ErrPasswordEmpty, err)
	case errors.Is(err, coreauth.ErrBootstrapPasswordTooShort):
		return fmt.Errorf("%w: %v", ErrPasswordTooShort, err)
	default:
		return err
	}
}
