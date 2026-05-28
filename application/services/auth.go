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
	users        *stores.UserStore
	sessions     *coreauth.SessionManager
	loginLimiter *coreauth.LoginAttemptLimiter
}

type RequestContext struct {
	session             *coreauth.Session
	secureSessionCookie bool
	clientAddress       string
}

func (s *AuthService) RequestFromHTTP(r *http.Request) (RequestContext, bool) {
	req := RequestContext{
		secureSessionCookie: coreauth.RequestUsesHTTPS(r),
		clientAddress:       coreauth.ClientAddress(r),
	}
	if s == nil || s.sessions == nil {
		return req, false
	}
	if r == nil {
		return req, false
	}

	session, ok := s.sessions.SessionFromRequest(r)
	req.session = session
	return req, ok
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
	if _, limited := s.loginLimiter.Check(req.clientAddress, username); limited {
		return nil, nil, ErrLoginRateLimited
	}

	creds := coreauth.Credentials{Username: username, Password: password}
	user, err := creds.Authenticate(s.users)
	if err != nil {
		if errors.Is(err, coreauth.ErrInvalidCredentials) {
			s.loginLimiter.RecordFailure(req.clientAddress, username)
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	s.loginLimiter.RecordSuccess(req.clientAddress, username)
	cookie := s.createSessionCookie(req, user.ID)
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

	cookie := req.session.BuildSessionCookieWithSecure(req.secureSessionCookie)
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

func (s *AuthService) CreateBootstrapAdmin(req RequestContext, username string, password string) (*dbmodels.User, *http.Cookie, error) {
	if s == nil || s.users == nil || s.sessions == nil {
		return nil, nil, ErrUserStoreUnavailable
	}

	user, err := coreauth.CreateBootstrapAdmin(s.users, username, password)
	if err != nil {
		return nil, nil, mapBootstrapError(err)
	}

	cookie := s.createSessionCookie(req, user.ID)
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
	if err := validateManagedPassword(newPassword); err != nil {
		return err
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

func (s *AuthService) createSessionCookie(req RequestContext, userID int64) *http.Cookie {
	session := s.sessions.Create(userID)
	return session.BuildSessionCookieWithSecure(req.secureSessionCookie)
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
