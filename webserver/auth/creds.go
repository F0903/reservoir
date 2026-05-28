package auth

import (
	"errors"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserStoreUnavailable = errors.New("user store unavailable")
)

var dummyPasswordHash = phc.GenerateArgon2id("reservoir dummy password")

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Credentials) Authenticate(users *stores.UserStore) (*models.User, error) {
	if users == nil {
		return nil, ErrUserStoreUnavailable
	}
	user, err := users.GetByUsername(c.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		dummyPasswordHash.VerifyArgon2id(c.Password)
		return nil, ErrInvalidCredentials
	}

	passwordMatch := user.PasswordHash.VerifyArgon2id(c.Password)
	if !passwordMatch {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}
