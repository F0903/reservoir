package services

import "errors"

var (
	ErrServicesUnavailable      = errors.New("application services unavailable")
	ErrUserStoreUnavailable     = errors.New("user store unavailable")
	ErrCacheUnavailable         = errors.New("cache unavailable")
	ErrConfigUnavailable        = errors.New("config unavailable")
	ErrNoSession                = errors.New("no user session found")
	ErrNoLogFile                = errors.New("no log file configured")
	ErrAlreadyAuthenticated     = errors.New("already authenticated")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")
	ErrMissingPasswordFields    = errors.New("both current_password and new_password fields are required")
	ErrEmptyCurrentPassword     = errors.New("current password must not be empty")
	ErrEmptyNewPassword         = errors.New("new password must not be empty")
	ErrPasswordEmpty            = errors.New("password must not be empty")
	ErrPasswordTooShort         = errors.New("password must be at least 12 characters")
	ErrUsernameEmpty            = errors.New("username must not be empty")
	ErrUsernameTaken            = errors.New("username is already taken")
	ErrUserNotFound             = errors.New("user not found")
	ErrLastAdmin                = errors.New("cannot remove the last admin")
	ErrBootstrapNotRequired     = errors.New("bootstrap is not required")
	ErrConfigValidationFailed   = errors.New("config validation failed")
)
