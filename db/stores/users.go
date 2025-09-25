package stores

import (
	"reservoir/db"
	"reservoir/db/models"
)

type UserStore struct {
	db db.Database
}

func OpenUserStore() (*UserStore, error) {
	db, err := db.OpenMainDatabase()
	if err != nil {
		return nil, err
	}

	return &UserStore{db: db}, nil
}

func (s *UserStore) Save(user *models.User) error {
	return s.db.Exec(
		"INSERT INTO users (username, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?)",
		user.Username,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
}

// Returns the user with the given username, or nil if no such user exists.
func (s *UserStore) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE username = ?", username)
	if err != nil {
		if db.IsResponseEmpty(err) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) Close() error {
	return s.db.Close()
}
