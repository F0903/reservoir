package stores

import (
	"errors"
	"reservoir/db"
	"reservoir/db/models"
	"reservoir/utils/phc"
	"strings"
)

var (
	ErrUserStoreNotEmpty = errors.New("user store is not empty")
	ErrUserNotFound      = errors.New("user not found")
	ErrUsernameEmpty     = errors.New("username must not be empty")
	ErrUsernameTaken     = errors.New("username is already taken")
	ErrLastAdmin         = errors.New("cannot remove the last admin")
)

type UserStore struct {
	db db.Database
}

func NewUserStore(database db.Database) *UserStore {
	return &UserStore{db: database}
}

// Saves the given user to the database. If a user with the same username already exists, it is updated.
func (s *UserStore) Save(user *models.User) error {
	return s.db.WithTransaction(func(tx *db.Tx) error {
		existing, err := getUserByUsername(tx, user.Username)
		if err != nil {
			return err
		}
		if existing != nil {
			if err := ensureCanRemoveAdmin(tx, existing.IsAdmin && !user.IsAdmin); err != nil {
				return err
			}
		}

		return tx.Exec(
			`
			INSERT INTO users (username, password_hash, is_admin, password_change_required)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(username) DO UPDATE SET
				username = excluded.username,
				password_hash = excluded.password_hash,
				is_admin = excluded.is_admin,
				password_change_required = excluded.password_change_required;
			`,
			user.Username,
			user.PasswordHash,
			user.IsAdmin,
			user.PasswordChangeRequired,
		)
	})
}

func (s *UserStore) Create(user *models.User) (*models.User, error) {
	username, err := normalizeUsername(user.Username)
	if err != nil {
		return nil, err
	}
	user.Username = username

	if err := s.ensureUsernameAvailable(user.Username, 0); err != nil {
		return nil, err
	}

	result, err := s.db.ExecResult(
		`
		INSERT INTO users (username, password_hash, is_admin, password_change_required)
		VALUES (?, ?, ?, ?);
		`,
		user.Username,
		user.PasswordHash,
		user.IsAdmin,
		user.PasswordChangeRequired,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func normalizeUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", ErrUsernameEmpty
	}
	return username, nil
}

func (s *UserStore) ensureUsernameAvailable(username string, userID int64) error {
	existing, err := s.GetByUsername(username)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != userID {
		return ErrUsernameTaken
	}
	return nil
}

func (s *UserStore) CreateFirst(user *models.User) error {
	result, err := s.db.ExecResult(
		`
		INSERT INTO users (username, password_hash, is_admin, password_change_required)
		SELECT ?, ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM users);
		`,
		user.Username,
		user.PasswordHash,
		user.IsAdmin,
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

func (s *UserStore) List() ([]models.User, error) {
	users := []models.User{}
	err := s.db.Select(&users, "SELECT * FROM users ORDER BY username COLLATE NOCASE")
	return users, err
}

// Returns the user with the given username, or nil if no such user exists.
func (s *UserStore) GetByUsername(username string) (*models.User, error) {
	return getUserByUsername(&s.db, username)
}

// Returns the user with the given ID, or nil if no such user exists.
func (s *UserStore) GetByID(id int64) (*models.User, error) {
	return getUserByID(&s.db, id)
}

type userGetter interface {
	Get(dest any, query string, args ...any) error
}

func getUserByUsername(q userGetter, username string) (*models.User, error) {
	var user models.User
	err := q.Get(&user, "SELECT * FROM users WHERE username = ?", username)
	if err != nil {
		if db.IsResponseEmpty(err) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func getUserByID(q userGetter, id int64) (*models.User, error) {
	var user models.User
	err := q.Get(&user, "SELECT * FROM users WHERE id = ?", id)
	if err != nil {
		if db.IsResponseEmpty(err) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) UpdateUsername(id int64, username string) (*models.User, error) {
	username, err := normalizeUsername(username)
	if err != nil {
		return nil, err
	}

	if err := s.ensureUsernameAvailable(username, id); err != nil {
		return nil, err
	}

	result, err := s.db.ExecResult("UPDATE users SET username = ? WHERE id = ?", username, id)
	if err != nil {
		return nil, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *UserStore) UpdateAdmin(id int64, isAdmin bool) (*models.User, error) {
	var updated *models.User

	err := s.db.WithTransaction(func(tx *db.Tx) error {
		user, err := getUserByID(tx, id)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}

		if err := ensureCanRemoveAdmin(tx, user.IsAdmin && !isAdmin); err != nil {
			return err
		}

		result, err := tx.ExecResult("UPDATE users SET is_admin = ? WHERE id = ?", isAdmin, id)
		if err != nil {
			return err
		}
		if err := ensureRowsAffected(result); err != nil {
			return err
		}

		updated, err = getUserByID(tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *UserStore) UpdatePassword(id int64, passwordHash phc.PHC, passwordChangeRequired bool) (*models.User, error) {
	result, err := s.db.ExecResult(
		"UPDATE users SET password_hash = ?, password_change_required = ? WHERE id = ?",
		passwordHash,
		passwordChangeRequired,
		id,
	)
	if err != nil {
		return nil, err
	}
	if err := ensureRowsAffected(result); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *UserStore) Delete(id int64) error {
	return s.db.WithTransaction(func(tx *db.Tx) error {
		user, err := getUserByID(tx, id)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}
		if err := ensureCanRemoveAdmin(tx, user.IsAdmin); err != nil {
			return err
		}

		result, err := tx.ExecResult("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			return err
		}
		return ensureRowsAffected(result)
	})
}

func ensureRowsAffected(result interface{ RowsAffected() (int64, error) }) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func ensureCanRemoveAdmin(q userGetter, removingAdmin bool) error {
	if !removingAdmin {
		return nil
	}

	admins, err := countAdmins(q)
	if err != nil {
		return err
	}
	if admins <= 1 {
		return ErrLastAdmin
	}
	return nil
}

func (s *UserStore) Count() (int, error) {
	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM users")
	return count, err
}

func (s *UserStore) CountAdmins() (int, error) {
	return countAdmins(&s.db)
}

func countAdmins(q userGetter) (int, error) {
	var count int
	err := q.Get(&count, "SELECT COUNT(*) FROM users WHERE is_admin")
	return count, err
}

func (s *UserStore) Close() error {
	return s.db.Close()
}
