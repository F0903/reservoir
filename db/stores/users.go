package stores

import (
	"errors"
	"reservoir/db"
	"reservoir/db/models"
	"strings"
)

var (
	ErrUserStoreNotEmpty = errors.New("user store is not empty")
	ErrUserNotFound      = errors.New("user not found")
	ErrUsernameEmpty     = errors.New("username must not be empty")
	ErrUsernameTaken     = errors.New("username is already taken")
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
		INSERT INTO users (username, password_hash, password_change_required)
		VALUES (?, ?, ?)
		ON CONFLICT(username) DO UPDATE SET
			username = excluded.username,
			password_hash = excluded.password_hash,
			password_change_required = excluded.password_change_required;
		`,
		user.Username,
		user.PasswordHash,
		user.PasswordChangeRequired,
	)
}

func (s *UserStore) CreateFirst(user *models.User) error {
	result, err := s.db.ExecResult(
		`
		INSERT INTO users (username, password_hash, password_change_required)
		SELECT ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM users);
		`,
		user.Username,
		user.PasswordHash,
		user.PasswordChangeRequired,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserStoreNotEmpty
	}
	return nil
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

func (s *UserStore) UpdateUsername(id int64, username string) (*models.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrUsernameEmpty
	}

	existing, err := s.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != id {
		return nil, ErrUsernameTaken
	}

	result, err := s.db.ExecResult("UPDATE users SET username = ? WHERE id = ?", username, id)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, ErrUserNotFound
	}

	return s.GetByID(id)
}

func (s *UserStore) Count() (int, error) {
	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM users")
	return count, err
}

func (s *UserStore) Close() error {
	return s.db.Close()
}
