package auth

import (
	"reservoir/db/stores"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Credentials) Authenticate() (bool, error) {
	usernameMatch := c.Username == "admin" // Currently hardcoded for simplicity. Later on we should support multiple users.

	users, err := stores.OpenUserStore()
	if err != nil {
		return false, err
	}
	defer users.Close()

	user, err := users.GetByUsername(c.Username)
	if err != nil {
		return false, err
	}

	passwordMatch := user.PasswordHash.VerifyArgon2id(c.Password)

	return usernameMatch && passwordMatch, nil
}
