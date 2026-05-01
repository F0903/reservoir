package auth

import (
	"errors"
	"fmt"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	"strings"
)

const (
	DefaultAdminUsername       = "admin"
	minBootstrapPasswordLength = 12
)

var (
	ErrBootstrapNotRequired       = errors.New("bootstrap is not required")
	ErrBootstrapUsernameEmpty     = errors.New("bootstrap username must not be empty")
	ErrBootstrapPasswordEmpty     = errors.New("bootstrap password must not be empty")
	ErrBootstrapPasswordTooShort  = errors.New("bootstrap password must be at least 12 characters")
	ErrBootstrapUserCreateFailed  = errors.New("bootstrap user create failed")
	ErrBootstrapCreatedUserLookup = errors.New("bootstrap user lookup failed after create")
)

type BootstrapResult struct {
	Username string
	Required bool
}

type bootstrapUserStore interface {
	Count() (int, error)
}

type bootstrapCreateUserStore interface {
	GetByUsername(username string) (*models.User, error)
	Count() (int, error)
	CreateFirst(user *models.User) error
}

func EnsureBootstrapAdmin() (*BootstrapResult, error) {
	users, err := stores.OpenUserStore()
	if err != nil {
		return nil, err
	}
	defer users.Close()

	return ensureBootstrapAdmin(users)
}

func ensureBootstrapAdmin(users bootstrapUserStore) (*BootstrapResult, error) {
	required, err := bootstrapRequired(users)
	if err != nil {
		return nil, err
	}

	if !required {
		return nil, nil
	}

	return &BootstrapResult{
		Username: DefaultAdminUsername,
		Required: true,
	}, nil
}

func BootstrapRequired() (bool, error) {
	users, err := stores.OpenUserStore()
	if err != nil {
		return false, err
	}
	defer users.Close()

	return bootstrapRequired(users)
}

func bootstrapRequired(users interface{ Count() (int, error) }) (bool, error) {
	count, err := users.Count()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func CreateBootstrapAdmin(username string, password string) (*models.User, error) {
	users, err := stores.OpenUserStore()
	if err != nil {
		return nil, err
	}
	defer users.Close()

	return createBootstrapAdmin(users, username, password)
}

func createBootstrapAdmin(users bootstrapCreateUserStore, username string, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrBootstrapUsernameEmpty
	}
	if password == "" {
		return nil, ErrBootstrapPasswordEmpty
	}
	if len(password) < minBootstrapPasswordLength {
		return nil, ErrBootstrapPasswordTooShort
	}

	required, err := bootstrapRequired(users)
	if err != nil {
		return nil, err
	}
	if !required {
		return nil, ErrBootstrapNotRequired
	}

	user := &models.User{
		Username:               username,
		PasswordHash:           *phc.GenerateArgon2id(password),
		PasswordChangeRequired: false,
	}
	if err := users.CreateFirst(user); err != nil {
		if errors.Is(err, stores.ErrUserStoreNotEmpty) {
			return nil, ErrBootstrapNotRequired
		}
		return nil, fmt.Errorf("%w: %v", ErrBootstrapUserCreateFailed, err)
	}

	created, err := users.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBootstrapCreatedUserLookup, err)
	}
	if created == nil {
		return nil, ErrBootstrapCreatedUserLookup
	}

	return created, nil
}
