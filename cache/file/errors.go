package file

import (
	"errors"
)

var (
	ErrCreate = errors.New("cache file create failed")
	ErrWrite  = errors.New("cache file write failed")
	ErrRead   = errors.New("cache file read failed")
	ErrRemove = errors.New("cache file remove failed")
	ErrEmpty  = errors.New("cache file empty")
)
