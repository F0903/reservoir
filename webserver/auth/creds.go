package auth

import (
	"reservoir/db/stores"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Credentials) Authenticate() (bool, error) {
	users, err := stores.OpenUserStore()
	if err != nil {
		return false, err
	}
	defer users.Close()

	user, err := users.GetByUsername(c.Username)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}

	passwordMatch := user.PasswordHash.VerifyArgon2id(c.Password)
	return passwordMatch, nil
}
