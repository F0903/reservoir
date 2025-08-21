package assertedpath

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type AssertedPath struct {
	Path string
}

func createDirs(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories '%s': %v", path, err)
	}
	return nil
}

func assertPath(path string) error {
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return nil // Path already exists, no need to create it
	}

	if err := createDirs(path); err != nil {
		return fmt.Errorf("failed to assert path '%s': %v", path, err)
	}
	return nil
}

// Asserts the existence of a directory at the specified path, creating it if necessary.
// If the directory cannot be created, it will panic.
func Assert(path string) AssertedPath {
	if err := assertPath(path); err != nil {
		panic(err)
	}

	return AssertedPath{
		Path: path,
	}
}

// Asserts the existence of a directory at the specified path, creating it if necessary.
// As opposed to Assert, this function returns an error instead of panicking.
func TryAssert(path string) (AssertedPath, error) {
	if err := assertPath(path); err != nil {
		return AssertedPath{}, err
	}

	return AssertedPath{
		Path: path,
	}, nil
}

func (ap AssertedPath) EnsureCleared() AssertedPath {
	// Just remove the path and recreate it, simpler and faster than iterating through the directory
	os.RemoveAll(ap.Path)
	if err := createDirs(ap.Path); err != nil {
		slog.Error("Failed to recreate cleared directory", "path", ap.Path, "error", err)
		panic(err)
	}
	return ap
}

func (ap *AssertedPath) String() string {
	return ap.Path
}
