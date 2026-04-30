package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
)

const (
	DefaultAdminUsername       = "admin"
	BootstrapPasswordFilePath  = "var/bootstrap-admin-password.txt"
	legacyDefaultAdminPassword = "placeholder"
)

type BootstrapResult struct {
	Username             string
	PasswordFile         string
	Created              bool
	RotatedLegacyDefault bool
	Reissued             bool
}

type bootstrapUserStore interface {
	GetByUsername(username string) (*models.User, error)
	Count() (int, error)
	Save(user *models.User) error
}

func EnsureBootstrapAdmin() (*BootstrapResult, error) {
	users, err := stores.OpenUserStore()
	if err != nil {
		return nil, err
	}
	defer users.Close()

	return ensureBootstrapAdmin(users, BootstrapPasswordFilePath, generateBootstrapPassword)
}

func ensureBootstrapAdmin(users bootstrapUserStore, passwordFile string, generatePassword func() (string, error)) (*BootstrapResult, error) {
	admin, err := users.GetByUsername(DefaultAdminUsername)
	if err != nil {
		return nil, err
	}

	if admin == nil {
		count, err := users.Count()
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, nil
		}
		return saveBootstrapAdmin(users, passwordFile, generatePassword, &BootstrapResult{
			Username: DefaultAdminUsername,
			Created:  true,
		})
	}

	if !admin.PasswordChangeRequired {
		if err := clearBootstrapPasswordFile(passwordFile); err != nil {
			return nil, err
		}
		return nil, nil
	}

	passwordFileExists, err := bootstrapPasswordFileExists(passwordFile)
	if err != nil {
		return nil, err
	}
	if admin.PasswordHash.VerifyArgon2id(legacyDefaultAdminPassword) {
		return saveBootstrapAdmin(users, passwordFile, generatePassword, &BootstrapResult{
			Username:             DefaultAdminUsername,
			RotatedLegacyDefault: true,
		})
	}
	if !passwordFileExists {
		return saveBootstrapAdmin(users, passwordFile, generatePassword, &BootstrapResult{
			Username: DefaultAdminUsername,
			Reissued: true,
		})
	}

	return nil, nil
}

func saveBootstrapAdmin(users bootstrapUserStore, passwordFile string, generatePassword func() (string, error), result *BootstrapResult) (*BootstrapResult, error) {
	password, err := generatePassword()
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:               DefaultAdminUsername,
		PasswordHash:           *phc.GenerateArgon2id(password),
		PasswordChangeRequired: true,
	}
	if err := users.Save(user); err != nil {
		return nil, err
	}
	if err := writeBootstrapPasswordFile(passwordFile, user.Username, password); err != nil {
		return nil, err
	}

	result.PasswordFile = passwordFile
	return result, nil
}

func generateBootstrapPassword() (string, error) {
	var data [24]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data[:]), nil
}

func writeBootstrapPasswordFile(path string, username string, password string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	body := fmt.Sprintf("Reservoir bootstrap admin credentials\n\nusername: %s\npassword: %s\n\nThis file is removed after the password is changed.\n", username, password)
	return os.WriteFile(path, []byte(body), 0600)
}

func bootstrapPasswordFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ClearBootstrapPasswordFile() error {
	return clearBootstrapPasswordFile(BootstrapPasswordFilePath)
}

func clearBootstrapPasswordFile(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}
