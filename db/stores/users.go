package stores

import (
	"reservoir/db"
	"reservoir/db/models"
)

type UserStore struct {
	db db.Database
}

// Opens a connection to the user store on the main database.
func OpenUserStore() (*UserStore, error) {
	db, err := db.OpenMainDatabase()
	if err != nil {
		return nil, err
	}

	return &UserStore{db: db}, nil
}

// Saves the given user to the database. If a user with the same username already exists, it is updated.
func (s *UserStore) Save(user *models.User) error {
	return s.db.Exec(
		`
		INSERT INTO users (username, password_hash, password_reset_required)
		VALUES (?, ?, ?)
		ON CONFLICT(username) DO UPDATE SET
			username = excluded.username,
			password_hash = excluded.password_hash,
			password_reset_required = excluded.password_reset_required;
		`,
		user.Username,
		user.PasswordHash,
		user.PasswordResetRequired,
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

// Returns the user with the given ID, or nil if no such user exists.
func (s *UserStore) GetByID(id int64) (*models.User, error) {
	var user models.User
	err := s.db.Get(&user, "SELECT * FROM users WHERE id = ?", id)
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
