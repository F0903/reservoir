package services

import (
	"errors"
	"fmt"

	dbmodels "reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	coreauth "reservoir/webserver/auth"
)

const minManagedPasswordLength = 12

type UserUpdate struct {
	Username               *string
	Password               *string
	IsAdmin                *bool
	PasswordChangeRequired *bool
}

type UserService struct {
	users    *stores.UserStore
	sessions *coreauth.SessionManager
}

func (s *UserService) List() ([]dbmodels.User, error) {
	if s == nil || s.users == nil {
		return nil, ErrUserStoreUnavailable
	}

	return s.users.List()
}

func (s *UserService) CreateManagedUser(username string, password string, isAdmin bool, passwordChangeRequired *bool) (*dbmodels.User, error) {
	if s == nil || s.users == nil {
		return nil, ErrUserStoreUnavailable
	}
	if err := validateManagedPassword(password); err != nil {
		return nil, err
	}

	user, err := s.users.Create(&dbmodels.User{
		Username:               username,
		PasswordHash:           *phc.GenerateArgon2id(password),
		IsAdmin:                isAdmin,
		PasswordChangeRequired: boolOrDefault(passwordChangeRequired, true),
	})
	if err != nil {
		return nil, mapUserStoreError(err)
	}
	return user, nil
}

func (s *UserService) UpdateManagedUser(req RequestContext, id int64, update UserUpdate) (*dbmodels.User, error) {
	if s == nil || s.users == nil {
		return nil, ErrUserStoreUnavailable
	}

	user, err := s.users.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if update.Username != nil {
		user, err = s.users.UpdateUsername(id, *update.Username)
		if err != nil {
			return nil, mapUserStoreError(err)
		}
	}

	if update.IsAdmin != nil {
		user, err = s.users.UpdateAdmin(id, *update.IsAdmin)
		if err != nil {
			return nil, mapUserStoreError(err)
		}
	}

	if update.Password != nil {
		if err := validateManagedPassword(*update.Password); err != nil {
			return nil, err
		}

		user, err = s.users.UpdatePassword(
			id,
			*phc.GenerateArgon2id(*update.Password),
			boolOrDefault(update.PasswordChangeRequired, true),
		)
		if err != nil {
			return nil, mapUserStoreError(err)
		}

		s.destroySessionsForUser(req, id, true)
	}

	return user, nil
}

func (s *UserService) DeleteManagedUser(id int64) error {
	if s == nil || s.users == nil {
		return ErrUserStoreUnavailable
	}

	if err := s.users.Delete(id); err != nil {
		return mapUserStoreError(err)
	}

	s.destroySessionsForUser(RequestContext{}, id, false)
	return nil
}

func (s *UserService) destroySessionsForUser(req RequestContext, userID int64, keepCurrentSession bool) {
	if s == nil || s.sessions == nil {
		return
	}

	keepSessionID := ""
	if keepCurrentSession && req.session != nil && req.session.UserID == userID {
		keepSessionID = req.session.ID
	}

	s.sessions.DestroySessionsForUserExcept(userID, keepSessionID)
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func validateManagedPassword(password string) error {
	if password == "" {
		return ErrPasswordEmpty
	}
	if len(password) < minManagedPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

func mapUserStoreError(err error) error {
	switch {
	case errors.Is(err, stores.ErrUsernameEmpty):
		return fmt.Errorf("%w: %v", ErrUsernameEmpty, err)
	case errors.Is(err, stores.ErrUsernameTaken):
		return fmt.Errorf("%w: %v", ErrUsernameTaken, err)
	case errors.Is(err, stores.ErrUserNotFound):
		return fmt.Errorf("%w: %v", ErrUserNotFound, err)
	case errors.Is(err, stores.ErrLastAdmin):
		return fmt.Errorf("%w: %v", ErrLastAdmin, err)
	default:
		return err
	}
}
