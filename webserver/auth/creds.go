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

// Each Argon2id check allocates 64 MB, so a burst of logins must queue instead of exhausting memory.
var hashingSlots = make(chan struct{}, 4)

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

	hashingSlots <- struct{}{}
	defer func() { <-hashingSlots }()

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
