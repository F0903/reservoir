package auth

import (
	"runtime"

	"golang.org/x/crypto/argon2"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Credentials) Authenticate() bool {
	usernameMatch := c.Username == "admin" // Currently hardcoded for simplicity. Later on we should support multiple users.

	// Use between 1 and 4 threads for hashing based on CPU count.
	// We use half of the CPU cores to avoid taking resources from the proxy which is more important.
	argonThreads := min(4, max(1, runtime.NumCPU()/2))
	hashedCandidate := argon2.IDKey([]byte(c.Password), []byte("saltsticks"), 1, 64*1024, uint8(argonThreads), 32)

	userHashedPassword := "" //TODO: load from user store

	passwordMatch := string(hashedCandidate) == userHashedPassword

	return usernameMatch && passwordMatch
}
